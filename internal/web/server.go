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
	"sync"
	"time"

	"github.com/getopsdeck/opsdeck/internal/discovery"
	"github.com/getopsdeck/opsdeck/internal/intel"
)

// SessionView is the JSON-friendly representation of a session for the web UI.
type SessionView struct {
	ID        string    `json:"id"`
	PID       int       `json:"pid"`
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
}

// NewServer creates a new web dashboard server.
func NewServer(addr string) *Server {
	s := &Server{
		addr:     addr,
		snapshot: &Snapshot{},
		mux:      http.NewServeMux(),
	}
	s.routes()
	return s
}

// routes registers all HTTP handlers.
func (s *Server) routes() {
	s.mux.HandleFunc("/", s.handleDashboard)
	s.mux.HandleFunc("/api/sessions", s.handleAPISessions)
	s.mux.HandleFunc("/api/session/", s.handleAPISessionDetail)
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
	return http.ListenAndServe(s.addr, s.mux)
}

// refresh re-discovers all sessions and updates the snapshot.
func (s *Server) refresh() {
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}

	sessionsDir := filepath.Join(home, ".claude", "sessions")
	projectsDir := filepath.Join(home, ".claude", "projects")

	raw, err := discovery.ScanSessions(sessionsDir)
	if err != nil {
		return
	}

	var views []SessionView
	for _, rs := range raw {
		alive := discovery.CheckSession(rs.PID, rs.StartedAt)
		transcriptPath := discovery.FindTranscriptPath(projectsDir, rs.CWD, rs.ID)

		var lastActivity = rs.StartedAt
		if transcriptPath != "" {
			if t, err := discovery.ReadLastActivity(transcriptPath); err == nil && !t.IsZero() {
				lastActivity = t
			}
		}

		state := discovery.ClassifyState(alive, lastActivity)

		workingOn := rs.Summary
		if workingOn == "" && rs.MessageCount > 0 {
			workingOn = fmt.Sprintf("%d messages", rs.MessageCount)
		}

		view := SessionView{
			ID:        rs.ID,
			PID:       rs.PID,
			State:     string(state),
			Project:   rs.ProjectName,
			StartedAt: rs.StartedAt,
			WorkingOn: workingOn,
		}

		// Lightweight stats (no full transcript parse on refresh).
		if transcriptPath != "" {
			summary, err := intel.ExtractSummary(transcriptPath)
			if err == nil && summary.TotalMessages > 0 {
				view.EditCount = summary.EditCount
				view.BashCount = summary.BashCount
				view.ErrorCount = summary.ErrorCount
				view.FilesChanged = len(summary.FilesChanged)
				view.Messages = summary.TotalMessages
			}
		}

		views = append(views, view)
	}

	projects := discovery.GroupByProject(raw)

	s.snapshot.mu.Lock()
	s.snapshot.sessions = views
	s.snapshot.projects = projects
	s.snapshot.updated = time.Now()
	s.snapshot.mu.Unlock()
}

// handleAPISessions returns the current session list as JSON.
func (s *Server) handleAPISessions(w http.ResponseWriter, r *http.Request) {
	s.snapshot.mu.RLock()
	data := s.snapshot.sessions
	s.snapshot.mu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
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
			}
			break
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(target)
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
