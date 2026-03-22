// Package web provides an embedded HTTP server for the OpsDeck web dashboard.
// It reuses the same discovery and intel logic as the TUI, served via SSE
// for real-time updates.
package web

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/getopsdeck/opsdeck/internal/discovery"
	"github.com/getopsdeck/opsdeck/internal/intel"
	"github.com/getopsdeck/opsdeck/internal/monitor"
)

// SessionView is the JSON-friendly representation of a session for the web UI.
type SessionView struct {
	ID        string    `json:"id"`
	PID       int       `json:"pid"`
	CWD       string    `json:"cwd,omitempty"`
	State     string    `json:"state"`
	Project   string    `json:"project"`
	StartedAt time.Time `json:"started_at"`
	WorkingOn string    `json:"working_on"`

	// Activity data (populated on detail request).
	EditCount    int      `json:"edit_count"`
	BashCount    int      `json:"bash_count"`
	ErrorCount   int      `json:"error_count"`
	FilesChanged int      `json:"files_changed"`
	Messages     int      `json:"messages"`
	Activities   []string `json:"activities,omitempty"`
	LastRequest  string   `json:"last_request,omitempty"`

	// Git data.
	GitBranch     string `json:"git_branch,omitempty"`
	GitDirty      bool   `json:"git_dirty,omitempty"`
	GitAhead      int    `json:"git_ahead,omitempty"`
	GitBehind     int    `json:"git_behind,omitempty"`
	GitLastCommit string `json:"git_last_commit,omitempty"`

	// Cost data.
	TotalTokens int64   `json:"total_tokens"`
	EstCostUSD  float64 `json:"est_cost_usd"`
	BurnRate    float64 `json:"burn_rate"`
}

// cachedSummary holds a cached transcript summary keyed by modtime.
type cachedSummary struct {
	modTime time.Time
	summary intel.SessionSummary
	cost    intel.SessionCost
}

// Snapshot holds the current state of all sessions, refreshed periodically.
type Snapshot struct {
	mu       sync.RWMutex
	sessions []SessionView
	projects []discovery.Project
	updated  time.Time
}

// Server is the web dashboard HTTP server.
type Server struct {
	addr     string
	snapshot *Snapshot
	mux      *http.ServeMux

	// Transcript summary cache: path -> cachedSummary.
	// Avoids re-parsing unchanged transcripts every 3 seconds.
	cacheMu sync.Mutex
	cache   map[string]cachedSummary
}

// NewServer creates a new web dashboard server.
func NewServer(addr string) *Server {
	s := &Server{
		addr:     addr,
		snapshot: &Snapshot{},
		mux:      http.NewServeMux(),
		cache:    make(map[string]cachedSummary),
	}
	s.routes()
	return s
}

// routes registers all HTTP handlers.
func (s *Server) routes() {
	s.mux.HandleFunc("/", s.handleDashboard)
	s.mux.HandleFunc("/api/sessions", s.handleAPISessions)
	s.mux.HandleFunc("/api/session/", s.handleAPISessionDetail)
	s.mux.HandleFunc("/api/timeline/", s.handleAPITimeline)
	s.mux.HandleFunc("/api/brief", s.handleAPIBrief)
	s.mux.HandleFunc("/api/events", s.handleSSE)
}

// Start runs the server and the background refresh loop.
func (s *Server) Start() error {
	// Initial snapshot.
	s.refresh()

	// Background refresh every 3 seconds.
	go func() {
		ticker := time.NewTicker(3 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			s.refresh()
		}
	}()

	log.Printf("OpsDeck web dashboard: http://%s\n", s.addr)
	err := http.ListenAndServe(s.addr, s.mux)
	if err != nil && strings.Contains(err.Error(), "address already in use") {
		return fmt.Errorf("port %s is already in use. Try: opsdeck web :8080", s.addr)
	}
	return err
}

// refresh re-discovers all sessions and updates the snapshot.
// It uses monitor.Snapshot for core enrichment (session-index, transcript,
// git branch/dirty) and then layers on web-specific extras (extended git
// info, cached transcript summaries, cost data).
func (s *Server) refresh() {
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}

	sessionsDir := filepath.Join(home, ".claude", "sessions")
	projectsDir := filepath.Join(home, ".claude", "projects")

	// Core enrichment via the shared monitor layer.
	enriched := monitor.Snapshot(sessionsDir, projectsDir)

	var views []SessionView
	for _, ms := range enriched {
		view := SessionView{
			ID:        ms.ID,
			PID:       ms.PID,
			CWD:       ms.CWD,
			State:     ms.State,
			Project:   ms.Project,
			StartedAt: ms.StartedAt,
			WorkingOn: ms.WorkingOn,
			Messages:  ms.MessageCount,
		}

		// Extended git info (ahead/behind/last-commit) for the web detail view.
		gi := discovery.GetGitInfo(ms.CWD)
		view.GitBranch = gi.Branch
		view.GitDirty = gi.IsDirty
		view.GitAhead = gi.Ahead
		view.GitBehind = gi.Behind
		view.GitLastCommit = gi.LastCommit

		// Use cached transcript summary to avoid re-parsing unchanged files.
		if ms.TranscriptPath != "" {
			summary, cost := s.cachedExtract(ms.TranscriptPath)
			if summary.TotalMessages > 0 {
				view.EditCount = summary.EditCount
				view.BashCount = summary.BashCount
				view.ErrorCount = summary.ErrorCount
				view.FilesChanged = len(summary.FilesChanged)
				view.Messages = summary.TotalMessages
			}
			if cost.TotalTokens > 0 {
				view.TotalTokens = cost.TotalTokens
				view.EstCostUSD = cost.EstCostUSD
			}
			// Burn rate is only meaningful for active sessions.
			if ms.State == "busy" {
				view.BurnRate = intel.CalculateBurnRate(ms.TranscriptPath)
			}
		}

		views = append(views, view)
	}

	// GroupByProject still needs raw discovery.Sessions for its logic.
	raw, _ := discovery.ScanSessions(sessionsDir)
	projects := discovery.GroupByProject(raw)

	s.snapshot.mu.Lock()
	s.snapshot.sessions = views
	s.snapshot.projects = projects
	s.snapshot.updated = time.Now()
	s.snapshot.mu.Unlock()
}

// cachedExtract returns a transcript summary and cost data, using a cache
// keyed by file modification time to avoid re-parsing unchanged transcripts.
// Uses two-phase check-then-compute to avoid holding the lock during file I/O.
func (s *Server) cachedExtract(path string) (intel.SessionSummary, intel.SessionCost) {
	info, err := os.Stat(path)
	if err != nil {
		return intel.SessionSummary{}, intel.SessionCost{}
	}

	// Phase 1: check cache under lock.
	s.cacheMu.Lock()
	if cached, ok := s.cache[path]; ok && cached.modTime.Equal(info.ModTime()) {
		s.cacheMu.Unlock()
		return cached.summary, cached.cost
	}
	s.cacheMu.Unlock()

	// Phase 2: compute outside the lock.
	// Use a 24-hour window so the web dashboard shows "today" data, not all-time.
	summary, _ := intel.ExtractSummary(path)
	cost, _ := intel.ExtractCosts(path, time.Now().Add(-24*time.Hour))

	// Phase 3: store under lock.
	s.cacheMu.Lock()
	// Simple cache eviction: if cache grows beyond 100 entries, clear it.
	// This prevents unbounded memory growth for long-running web servers.
	if len(s.cache) > 100 {
		s.cache = make(map[string]cachedSummary)
	}
	s.cache[path] = cachedSummary{modTime: info.ModTime(), summary: summary, cost: cost}
	s.cacheMu.Unlock()

	return summary, cost
}

// handleAPISessions returns the current session list as JSON.
func (s *Server) handleAPISessions(w http.ResponseWriter, r *http.Request) {
	s.snapshot.mu.RLock()
	out := make([]SessionView, len(s.snapshot.sessions))
	copy(out, s.snapshot.sessions)
	s.snapshot.mu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(out)
}

// handleAPISessionDetail returns detailed activity for a single session.
func (s *Server) handleAPISessionDetail(w http.ResponseWriter, r *http.Request) {
	sessionID := r.URL.Path[len("/api/session/"):]
	if sessionID == "" {
		http.Error(w, "missing session ID", http.StatusBadRequest)
		return
	}

	home, _ := os.UserHomeDir()
	projectsDir := filepath.Join(home, ".claude", "projects")

	// Find the session in the snapshot.
	s.snapshot.mu.RLock()
	var target *SessionView
	for i := range s.snapshot.sessions {
		if s.snapshot.sessions[i].ID == sessionID {
			copy := s.snapshot.sessions[i]
			target = &copy
			break
		}
	}
	s.snapshot.mu.RUnlock()

	if target == nil {
		http.Error(w, "session not found", http.StatusNotFound)
		return
	}

	// Get full activity details from transcript.
	raw, _ := discovery.ScanSessions(filepath.Join(home, ".claude", "sessions"))
	for _, rs := range raw {
		if rs.ID == sessionID {
			transcriptPath := discovery.FindTranscriptPath(projectsDir, rs.CWD, rs.ID)
			if transcriptPath != "" {
				summary, err := intel.ExtractSummary(transcriptPath)
				if err == nil {
					target.Activities = intel.SummarizeActivities(summary.Activities)
					if summary.LastUserMsg != "" {
						runes := []rune(summary.LastUserMsg)
						if len(runes) > 200 {
							target.LastRequest = string(runes[:197]) + "..."
						} else {
							target.LastRequest = summary.LastUserMsg
						}
					}
				}

				// Cost data for detail view.
				cost, err := intel.ExtractCosts(transcriptPath, time.Time{})
				if err == nil {
					target.TotalTokens = cost.TotalTokens
					target.EstCostUSD = cost.EstCostUSD
				}
			}
			break
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(target)
}

// handleAPITimeline returns timeline events for a session.
func (s *Server) handleAPITimeline(w http.ResponseWriter, r *http.Request) {
	sessionID := strings.TrimPrefix(r.URL.Path, "/api/timeline/")
	if sessionID == "" {
		http.Error(w, "missing session ID", http.StatusBadRequest)
		return
	}

	home, _ := os.UserHomeDir()
	projectsDir := filepath.Join(home, ".claude", "projects")
	sessionsDir := filepath.Join(home, ".claude", "sessions")

	// Find the transcript for this session.
	raw, _ := discovery.ScanSessions(sessionsDir)
	for _, rs := range raw {
		if rs.ID == sessionID {
			transcriptPath := discovery.FindTranscriptPath(projectsDir, rs.CWD, rs.ID)
			if transcriptPath == "" {
				http.Error(w, "no transcript found", http.StatusNotFound)
				return
			}

			since := time.Now().Add(-24 * time.Hour)
			timeline, err := intel.ExtractTimeline(transcriptPath, since)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(timeline)
			return
		}
	}

	http.Error(w, "session not found", http.StatusNotFound)
}

// handleAPIBrief returns the daily brief as JSON.
func (s *Server) handleAPIBrief(w http.ResponseWriter, r *http.Request) {
	home, _ := os.UserHomeDir()
	sessionsDir := filepath.Join(home, ".claude", "sessions")
	projectsDir := filepath.Join(home, ".claude", "projects")

	since := time.Now().Add(-24 * time.Hour)
	brief, err := intel.GenerateBrief(projectsDir, sessionsDir, since)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Enrich with waiting sessions, git info, and cost estimate.
	intel.EnrichBrief(&brief, projectsDir, sessionsDir, since)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(brief)
}

// handleSSE streams session updates via Server-Sent Events.
func (s *Server) handleSSE(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	// Send initial data immediately.
	s.sendSSEUpdate(w, flusher)

	for {
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
			s.sendSSEUpdate(w, flusher)
		}
	}
}

func (s *Server) sendSSEUpdate(w http.ResponseWriter, flusher http.Flusher) {
	s.snapshot.mu.RLock()
	data, _ := json.Marshal(s.snapshot.sessions)
	s.snapshot.mu.RUnlock()

	fmt.Fprintf(w, "data: %s\n\n", data)
	flusher.Flush()
}
