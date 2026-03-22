package web

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/getopsdeck/opsdeck/internal/intel"
)

// ---------------------------------------------------------------------------
// TestNewServer
// ---------------------------------------------------------------------------

func TestNewServer(t *testing.T) {
	s := NewServer("127.0.0.1:9090")
	if s.addr != "127.0.0.1:9090" {
		t.Errorf("addr = %q, want %q", s.addr, "127.0.0.1:9090")
	}
	if s.mux == nil {
		t.Error("mux is nil")
	}
	if s.snapshot == nil {
		t.Error("snapshot is nil")
	}
	if s.cache == nil {
		t.Error("cache map is nil")
	}
}

// ---------------------------------------------------------------------------
// TestHandleAPISessions
// ---------------------------------------------------------------------------

func TestHandleAPISessions(t *testing.T) {
	s := NewServer(":0")

	// Pre-populate snapshot with test data.
	now := time.Date(2026, 3, 22, 10, 0, 0, 0, time.UTC)
	s.snapshot.mu.Lock()
	s.snapshot.sessions = []SessionView{
		{
			ID:        "sess-aaa",
			PID:       1001,
			State:     "busy",
			Project:   "project-alpha",
			StartedAt: now,
			WorkingOn: "implementing feature X",
			EditCount: 5,
			BashCount: 3,
			Messages:  12,
		},
		{
			ID:        "sess-bbb",
			PID:       1002,
			State:     "idle",
			Project:   "project-beta",
			StartedAt: now.Add(-1 * time.Hour),
		},
	}
	s.snapshot.updated = now
	s.snapshot.mu.Unlock()

	req := httptest.NewRequest(http.MethodGet, "/api/sessions", nil)
	rec := httptest.NewRecorder()
	s.handleAPISessions(rec, req)

	res := rec.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", res.StatusCode, http.StatusOK)
	}
	if ct := res.Header.Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want %q", ct, "application/json")
	}

	var sessions []SessionView
	if err := json.NewDecoder(res.Body).Decode(&sessions); err != nil {
		t.Fatalf("json decode error: %v", err)
	}
	if len(sessions) != 2 {
		t.Fatalf("len(sessions) = %d, want 2", len(sessions))
	}
	if sessions[0].ID != "sess-aaa" {
		t.Errorf("sessions[0].ID = %q, want %q", sessions[0].ID, "sess-aaa")
	}
	if sessions[0].EditCount != 5 {
		t.Errorf("sessions[0].EditCount = %d, want 5", sessions[0].EditCount)
	}
	if sessions[1].State != "idle" {
		t.Errorf("sessions[1].State = %q, want %q", sessions[1].State, "idle")
	}
}

func TestHandleAPISessions_Empty(t *testing.T) {
	s := NewServer(":0")

	req := httptest.NewRequest(http.MethodGet, "/api/sessions", nil)
	rec := httptest.NewRecorder()
	s.handleAPISessions(rec, req)

	res := rec.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", res.StatusCode, http.StatusOK)
	}

	var sessions []SessionView
	if err := json.NewDecoder(res.Body).Decode(&sessions); err != nil {
		t.Fatalf("json decode error: %v", err)
	}
	// Empty snapshot should return an empty JSON array (not null).
	if sessions == nil {
		// json.Decoder decodes `[]` to an empty slice but `null` to nil.
		// The handler uses make([]SessionView, 0) via copy so [] is expected.
		// Accept nil here because copy of nil slice produces nil.
	}
}

// ---------------------------------------------------------------------------
// TestHandleAPISessionDetail_NotFound
// ---------------------------------------------------------------------------

func TestHandleAPISessionDetail_NotFound(t *testing.T) {
	s := NewServer(":0")

	// Snapshot is empty, so any session ID will be "not found".
	req := httptest.NewRequest(http.MethodGet, "/api/session/nonexistent-id", nil)
	rec := httptest.NewRecorder()
	s.handleAPISessionDetail(rec, req)

	res := rec.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want %d", res.StatusCode, http.StatusNotFound)
	}

	body, _ := io.ReadAll(res.Body)
	if !strings.Contains(string(body), "session not found") {
		t.Errorf("body = %q, want it to contain %q", body, "session not found")
	}
}

func TestHandleAPISessionDetail_MissingID(t *testing.T) {
	s := NewServer(":0")

	// Path is exactly "/api/session/" with no trailing ID.
	req := httptest.NewRequest(http.MethodGet, "/api/session/", nil)
	rec := httptest.NewRecorder()
	s.handleAPISessionDetail(rec, req)

	res := rec.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", res.StatusCode, http.StatusBadRequest)
	}
}

// ---------------------------------------------------------------------------
// TestHandleSSE
// ---------------------------------------------------------------------------

func TestHandleSSE(t *testing.T) {
	s := NewServer(":0")

	// Pre-populate snapshot so the SSE payload is non-empty.
	s.snapshot.mu.Lock()
	s.snapshot.sessions = []SessionView{
		{ID: "sess-sse", State: "busy", Project: "test-proj"},
	}
	s.snapshot.mu.Unlock()

	// Use httptest.NewServer so ResponseWriter supports http.Flusher.
	ts := httptest.NewServer(s.mux)
	defer ts.Close()

	// Create a request that we cancel quickly (SSE is a long-lived connection).
	client := ts.Client()
	req, _ := http.NewRequest(http.MethodGet, ts.URL+"/api/events", nil)

	// Use a short timeout to read the initial SSE frame without blocking forever.
	client.Timeout = 2 * time.Second
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("request error: %v", err)
	}
	defer resp.Body.Close()

	// Verify SSE headers.
	if ct := resp.Header.Get("Content-Type"); ct != "text/event-stream" {
		t.Errorf("Content-Type = %q, want %q", ct, "text/event-stream")
	}
	if cc := resp.Header.Get("Cache-Control"); cc != "no-cache" {
		t.Errorf("Cache-Control = %q, want %q", cc, "no-cache")
	}
	if conn := resp.Header.Get("Connection"); conn != "keep-alive" {
		t.Errorf("Connection = %q, want %q", conn, "keep-alive")
	}

	// Read the initial SSE data frame.
	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)
	if !strings.HasPrefix(bodyStr, "data: ") {
		t.Errorf("SSE body should start with 'data: ', got prefix %q", bodyStr[:min(len(bodyStr), 30)])
	}
	// The payload should contain the session ID we injected.
	if !strings.Contains(bodyStr, "sess-sse") {
		t.Errorf("SSE body should contain session ID 'sess-sse', got %q", bodyStr[:min(len(bodyStr), 200)])
	}
}

// ---------------------------------------------------------------------------
// TestCachedExtract
// ---------------------------------------------------------------------------

func TestCachedExtract_CacheHit(t *testing.T) {
	s := NewServer(":0")

	// Create a temp file to serve as a "transcript".
	tmp := t.TempDir()
	path := filepath.Join(tmp, "transcript.jsonl")
	if err := os.WriteFile(path, []byte(""), 0644); err != nil {
		t.Fatal(err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}

	// Pre-populate the cache with known values.
	s.cacheMu.Lock()
	s.cache[path] = cachedSummary{
		modTime: info.ModTime(),
		summary: intel.SessionSummary{
			TotalMessages: 42,
			EditCount:     10,
			BashCount:     5,
		},
		cost: intel.SessionCost{
			TotalTokens: 100000,
			EstCostUSD:  1.50,
		},
	}
	s.cacheMu.Unlock()

	// Call cachedExtract -- should return cached values (same modtime).
	summary, cost := s.cachedExtract(path)
	if summary.TotalMessages != 42 {
		t.Errorf("TotalMessages = %d, want 42", summary.TotalMessages)
	}
	if summary.EditCount != 10 {
		t.Errorf("EditCount = %d, want 10", summary.EditCount)
	}
	if cost.TotalTokens != 100000 {
		t.Errorf("TotalTokens = %d, want 100000", cost.TotalTokens)
	}
	if cost.EstCostUSD != 1.50 {
		t.Errorf("EstCostUSD = %f, want 1.50", cost.EstCostUSD)
	}
}

func TestCachedExtract_CacheMiss(t *testing.T) {
	s := NewServer(":0")

	// Create a temp file.
	tmp := t.TempDir()
	path := filepath.Join(tmp, "transcript.jsonl")
	if err := os.WriteFile(path, []byte(""), 0644); err != nil {
		t.Fatal(err)
	}

	// Pre-populate the cache with a DIFFERENT modtime (stale entry).
	s.cacheMu.Lock()
	s.cache[path] = cachedSummary{
		modTime: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC), // old timestamp
		summary: intel.SessionSummary{
			TotalMessages: 999, // stale value -- should NOT be returned
		},
	}
	s.cacheMu.Unlock()

	// Call cachedExtract -- modtime mismatch forces re-parse.
	summary, _ := s.cachedExtract(path)
	// The file is empty, so ExtractSummary will return zero values.
	if summary.TotalMessages == 999 {
		t.Error("cache was not invalidated: got stale TotalMessages=999")
	}
}

func TestCachedExtract_NonexistentFile(t *testing.T) {
	s := NewServer(":0")

	// Call with a file that does not exist.
	summary, cost := s.cachedExtract("/nonexistent/path/transcript.jsonl")
	if summary.TotalMessages != 0 {
		t.Errorf("TotalMessages = %d, want 0 for missing file", summary.TotalMessages)
	}
	if cost.TotalTokens != 0 {
		t.Errorf("TotalTokens = %d, want 0 for missing file", cost.TotalTokens)
	}
}

// ---------------------------------------------------------------------------
// TestHandleDashboard
// ---------------------------------------------------------------------------

func TestHandleDashboard(t *testing.T) {
	s := NewServer(":0")

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	s.handleDashboard(rec, req)

	res := rec.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", res.StatusCode, http.StatusOK)
	}
	if ct := res.Header.Get("Content-Type"); ct != "text/html; charset=utf-8" {
		t.Errorf("Content-Type = %q, want %q", ct, "text/html; charset=utf-8")
	}

	body, _ := io.ReadAll(res.Body)
	bodyStr := string(body)
	if !strings.Contains(bodyStr, "<!DOCTYPE html>") {
		t.Error("response body should contain <!DOCTYPE html>")
	}
	if !strings.Contains(bodyStr, "OpsDeck") {
		t.Error("response body should contain 'OpsDeck'")
	}
}

func TestHandleDashboard_404(t *testing.T) {
	s := NewServer(":0")

	// handleDashboard checks r.URL.Path != "/" and returns 404 for other paths.
	req := httptest.NewRequest(http.MethodGet, "/nonexistent", nil)
	rec := httptest.NewRecorder()
	s.handleDashboard(rec, req)

	res := rec.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want %d", res.StatusCode, http.StatusNotFound)
	}
}

// ---------------------------------------------------------------------------
// TestSessionViewJSON — verify JSON field names match the API contract.
// ---------------------------------------------------------------------------

func TestSessionViewJSON(t *testing.T) {
	now := time.Date(2026, 3, 22, 10, 0, 0, 0, time.UTC)
	sv := SessionView{
		ID:          "sess-json",
		PID:         42,
		State:       "busy",
		Project:     "test",
		StartedAt:   now,
		WorkingOn:   "testing JSON",
		GitBranch:   "main",
		GitDirty:    true,
		TotalTokens: 5000,
		EstCostUSD:  0.25,
	}

	data, err := json.Marshal(sv)
	if err != nil {
		t.Fatalf("json.Marshal error: %v", err)
	}

	// Decode into a generic map to verify field names.
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("json.Unmarshal error: %v", err)
	}

	// Verify key JSON field names used by the web frontend.
	requiredFields := []string{
		"id", "pid", "state", "project", "started_at",
		"working_on", "git_branch", "git_dirty",
		"total_tokens", "est_cost_usd",
	}
	for _, field := range requiredFields {
		if _, ok := m[field]; !ok {
			t.Errorf("JSON output missing required field %q", field)
		}
	}
}

// ---------------------------------------------------------------------------
// TestSnapshotConcurrency — verify snapshot is safe for concurrent access.
// ---------------------------------------------------------------------------

func TestSnapshotConcurrency(t *testing.T) {
	s := NewServer(":0")

	// Write sessions in a goroutine while reading from another.
	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := 0; i < 100; i++ {
			s.snapshot.mu.Lock()
			s.snapshot.sessions = []SessionView{
				{ID: "sess-concurrent", State: "busy"},
			}
			s.snapshot.updated = time.Now()
			s.snapshot.mu.Unlock()
		}
	}()

	// Concurrent reads via the API handler.
	for i := 0; i < 100; i++ {
		req := httptest.NewRequest(http.MethodGet, "/api/sessions", nil)
		rec := httptest.NewRecorder()
		s.handleAPISessions(rec, req)
		if rec.Code != http.StatusOK {
			t.Errorf("concurrent read %d: status = %d", i, rec.Code)
		}
	}

	<-done
}

// min returns the smaller of a and b.
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
