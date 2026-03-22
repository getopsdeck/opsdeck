package monitor

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/getopsdeck/opsdeck/internal/discovery"
)

// ---------------------------------------------------------------------------
// TestSnapshotEmpty — no sessions dir, no projects dir => empty result
// ---------------------------------------------------------------------------

func TestSnapshotEmpty(t *testing.T) {
	tmp := t.TempDir()
	sessionsDir := filepath.Join(tmp, "sessions")
	projectsDir := filepath.Join(tmp, "projects")

	sessions := Snapshot(sessionsDir, projectsDir)
	if len(sessions) != 0 {
		t.Errorf("expected 0 sessions, got %d", len(sessions))
	}
}

// ---------------------------------------------------------------------------
// TestSnapshotEmptyDirs — dirs exist but are empty => empty result
// ---------------------------------------------------------------------------

func TestSnapshotEmptyDirs(t *testing.T) {
	tmp := t.TempDir()
	sessionsDir := filepath.Join(tmp, "sessions")
	projectsDir := filepath.Join(tmp, "projects")
	os.MkdirAll(sessionsDir, 0755)
	os.MkdirAll(projectsDir, 0755)

	sessions := Snapshot(sessionsDir, projectsDir)
	if len(sessions) != 0 {
		t.Errorf("expected 0 sessions, got %d", len(sessions))
	}
}

// ---------------------------------------------------------------------------
// TestSnapshotBasicSession — one session file => one enriched Session
// ---------------------------------------------------------------------------

func TestSnapshotBasicSession(t *testing.T) {
	tmp := t.TempDir()
	sessionsDir := filepath.Join(tmp, "sessions")
	projectsDir := filepath.Join(tmp, "projects")
	os.MkdirAll(sessionsDir, 0755)
	os.MkdirAll(projectsDir, 0755)

	// Create a session file. PID 1 is always alive (init/launchd).
	sessionData := map[string]any{
		"pid":       1,
		"sessionId": "sess-abc-123",
		"cwd":       "/tmp/myproject",
		"startedAt": time.Now().Add(-10 * time.Minute).UnixMilli(),
	}
	writeJSON(t, filepath.Join(sessionsDir, "1.json"), sessionData)

	sessions := Snapshot(sessionsDir, projectsDir)
	if len(sessions) != 1 {
		t.Fatalf("expected 1 session, got %d", len(sessions))
	}

	s := sessions[0]
	if s.ID != "sess-abc-123" {
		t.Errorf("ID = %q, want %q", s.ID, "sess-abc-123")
	}
	if s.PID != 1 {
		t.Errorf("PID = %d, want 1", s.PID)
	}
	if s.CWD != "/tmp/myproject" {
		t.Errorf("CWD = %q, want %q", s.CWD, "/tmp/myproject")
	}
	if s.Project == "" {
		t.Error("Project should not be empty")
	}
	// State should be one of the valid states (busy/waiting/idle/dead).
	validStates := map[string]bool{"busy": true, "waiting": true, "idle": true, "dead": true}
	if !validStates[s.State] {
		t.Errorf("State = %q, want one of busy/waiting/idle/dead", s.State)
	}
}

// ---------------------------------------------------------------------------
// TestSnapshotSessionIndexEnrichment — session-index.json data populates
// WorkingOn and MessageCount in the enriched session.
// ---------------------------------------------------------------------------

func TestSnapshotSessionIndexEnrichment(t *testing.T) {
	tmp := t.TempDir()
	sessionsDir := filepath.Join(tmp, "sessions")
	projectsDir := filepath.Join(tmp, "projects")
	os.MkdirAll(sessionsDir, 0755)

	cwd := "/tmp/testproject"
	sessionID := "sess-enrich-001"

	// Create session file.
	writeJSON(t, filepath.Join(sessionsDir, "99.json"), map[string]any{
		"pid":       99,
		"sessionId": sessionID,
		"cwd":       cwd,
		"startedAt": time.Now().Add(-5 * time.Minute).UnixMilli(),
	})

	// Create the project dir with session-index.json.
	encoded := discovery.EncodeCWD(cwd)
	projDir := filepath.Join(projectsDir, encoded)
	os.MkdirAll(projDir, 0755)

	indexData := map[string]any{
		"version": 1,
		"entries": []map[string]any{
			{
				"sessionId":    sessionID,
				"summary":      "Implementing monitor package",
				"messageCount": 42,
			},
		},
	}
	writeJSON(t, filepath.Join(projDir, "sessions-index.json"), indexData)

	sessions := Snapshot(sessionsDir, projectsDir)
	if len(sessions) != 1 {
		t.Fatalf("expected 1 session, got %d", len(sessions))
	}

	s := sessions[0]
	if s.WorkingOn != "Implementing monitor package" {
		t.Errorf("WorkingOn = %q, want %q", s.WorkingOn, "Implementing monitor package")
	}
	if s.MessageCount != 42 {
		t.Errorf("MessageCount = %d, want 42", s.MessageCount)
	}
}

// ---------------------------------------------------------------------------
// TestSnapshotWorkingOnFallback — when summary is empty but MessageCount > 0,
// WorkingOn should show "<N> messages".
// ---------------------------------------------------------------------------

func TestSnapshotWorkingOnFallback(t *testing.T) {
	tmp := t.TempDir()
	sessionsDir := filepath.Join(tmp, "sessions")
	projectsDir := filepath.Join(tmp, "projects")
	os.MkdirAll(sessionsDir, 0755)

	cwd := "/tmp/fallbackproject"
	sessionID := "sess-fallback-001"

	writeJSON(t, filepath.Join(sessionsDir, "100.json"), map[string]any{
		"pid":       100,
		"sessionId": sessionID,
		"cwd":       cwd,
		"startedAt": time.Now().Add(-5 * time.Minute).UnixMilli(),
	})

	encoded := discovery.EncodeCWD(cwd)
	projDir := filepath.Join(projectsDir, encoded)
	os.MkdirAll(projDir, 0755)

	indexData := map[string]any{
		"version": 1,
		"entries": []map[string]any{
			{
				"sessionId":    sessionID,
				"summary":      "",
				"messageCount": 15,
			},
		},
	}
	writeJSON(t, filepath.Join(projDir, "sessions-index.json"), indexData)

	sessions := Snapshot(sessionsDir, projectsDir)
	if len(sessions) != 1 {
		t.Fatalf("expected 1 session, got %d", len(sessions))
	}

	if sessions[0].WorkingOn != "15 messages" {
		t.Errorf("WorkingOn = %q, want %q", sessions[0].WorkingOn, "15 messages")
	}
}

// ---------------------------------------------------------------------------
// TestSnapshotSessionStruct — verify all fields of Session struct exist and
// have expected zero values when not populated.
// ---------------------------------------------------------------------------

func TestSnapshotSessionStruct(t *testing.T) {
	s := Session{}

	if s.ID != "" {
		t.Error("zero Session.ID should be empty")
	}
	if s.PID != 0 {
		t.Error("zero Session.PID should be 0")
	}
	if s.CWD != "" {
		t.Error("zero Session.CWD should be empty")
	}
	if s.State != "" {
		t.Error("zero Session.State should be empty")
	}
	if s.Project != "" {
		t.Error("zero Session.Project should be empty")
	}
	if !s.StartedAt.IsZero() {
		t.Error("zero Session.StartedAt should be zero time")
	}
	if s.WorkingOn != "" {
		t.Error("zero Session.WorkingOn should be empty")
	}
	if s.LastLine != "" {
		t.Error("zero Session.LastLine should be empty")
	}
	if s.TranscriptPath != "" {
		t.Error("zero Session.TranscriptPath should be empty")
	}
	if s.GitBranch != "" {
		t.Error("zero Session.GitBranch should be empty")
	}
	if s.GitDirty {
		t.Error("zero Session.GitDirty should be false")
	}
	if s.MessageCount != 0 {
		t.Error("zero Session.MessageCount should be 0")
	}
}

// ---------------------------------------------------------------------------
// TestSnapshotTranscriptLastLine — transcript JSONL populates LastLine.
// ---------------------------------------------------------------------------

func TestSnapshotTranscriptLastLine(t *testing.T) {
	tmp := t.TempDir()
	sessionsDir := filepath.Join(tmp, "sessions")
	projectsDir := filepath.Join(tmp, "projects")
	os.MkdirAll(sessionsDir, 0755)

	cwd := "/tmp/transcripttest"
	sessionID := "sess-transcript-001"

	writeJSON(t, filepath.Join(sessionsDir, "101.json"), map[string]any{
		"pid":       101,
		"sessionId": sessionID,
		"cwd":       cwd,
		"startedAt": time.Now().Add(-2 * time.Minute).UnixMilli(),
	})

	// Create transcript JSONL in the project directory.
	encoded := discovery.EncodeCWD(cwd)
	projDir := filepath.Join(projectsDir, encoded)
	os.MkdirAll(projDir, 0755)

	// Write a transcript with a user message and an assistant message.
	ts := time.Now().Format(time.RFC3339Nano)
	lines := []string{
		`{"type":"user","timestamp":"` + ts + `","message":{"role":"user","content":"Fix the bug"}}`,
		`{"type":"assistant","timestamp":"` + ts + `","message":{"role":"assistant","content":"I will fix the bug now."}}`,
	}
	transcriptPath := filepath.Join(projDir, sessionID+".jsonl")
	os.WriteFile(transcriptPath, []byte(lines[0]+"\n"+lines[1]+"\n"), 0644)

	sessions := Snapshot(sessionsDir, projectsDir)
	if len(sessions) != 1 {
		t.Fatalf("expected 1 session, got %d", len(sessions))
	}

	s := sessions[0]
	if s.TranscriptPath != transcriptPath {
		t.Errorf("TranscriptPath = %q, want %q", s.TranscriptPath, transcriptPath)
	}
	// LastLine should contain text from the transcript.
	if s.LastLine == "" {
		t.Error("LastLine should not be empty when transcript has messages")
	}
}

// ---------------------------------------------------------------------------
// TestSnapshotLastLineFallsBackToSummary — when transcript has no meaningful
// lines, LastLine should fall back to the session index summary.
// ---------------------------------------------------------------------------

func TestSnapshotLastLineFallsBackToSummary(t *testing.T) {
	tmp := t.TempDir()
	sessionsDir := filepath.Join(tmp, "sessions")
	projectsDir := filepath.Join(tmp, "projects")
	os.MkdirAll(sessionsDir, 0755)

	cwd := "/tmp/fallbackline"
	sessionID := "sess-fallback-line-001"

	writeJSON(t, filepath.Join(sessionsDir, "102.json"), map[string]any{
		"pid":       102,
		"sessionId": sessionID,
		"cwd":       cwd,
		"startedAt": time.Now().Add(-5 * time.Minute).UnixMilli(),
	})

	encoded := discovery.EncodeCWD(cwd)
	projDir := filepath.Join(projectsDir, encoded)
	os.MkdirAll(projDir, 0755)

	// Session index with summary but no transcript file.
	indexData := map[string]any{
		"version": 1,
		"entries": []map[string]any{
			{
				"sessionId":    sessionID,
				"summary":      "Working on tests",
				"messageCount": 5,
			},
		},
	}
	writeJSON(t, filepath.Join(projDir, "sessions-index.json"), indexData)

	sessions := Snapshot(sessionsDir, projectsDir)
	if len(sessions) != 1 {
		t.Fatalf("expected 1 session, got %d", len(sessions))
	}

	// No transcript means LastLine should fall back to summary.
	if sessions[0].LastLine != "Working on tests" {
		t.Errorf("LastLine = %q, want %q", sessions[0].LastLine, "Working on tests")
	}
}

// ---------------------------------------------------------------------------
// TestSnapshotMultipleSessions — multiple session files all appear.
// ---------------------------------------------------------------------------

func TestSnapshotMultipleSessions(t *testing.T) {
	tmp := t.TempDir()
	sessionsDir := filepath.Join(tmp, "sessions")
	projectsDir := filepath.Join(tmp, "projects")
	os.MkdirAll(sessionsDir, 0755)
	os.MkdirAll(projectsDir, 0755)

	for i := 1; i <= 3; i++ {
		writeJSON(t, filepath.Join(sessionsDir, itoa(i)+".json"), map[string]any{
			"pid":       i,
			"sessionId": "sess-multi-" + itoa(i),
			"cwd":       "/tmp/multi",
			"startedAt": time.Now().Add(-time.Duration(i) * time.Minute).UnixMilli(),
		})
	}

	sessions := Snapshot(sessionsDir, projectsDir)
	if len(sessions) != 3 {
		t.Errorf("expected 3 sessions, got %d", len(sessions))
	}
}

// ---------------------------------------------------------------------------
// TestSnapshotIndexCachePerProject — sessions sharing the same CWD should
// share the same session-index.json lookup (no redundant reads).
// This test verifies correctness: both sessions get their index data.
// ---------------------------------------------------------------------------

func TestSnapshotIndexCachePerProject(t *testing.T) {
	tmp := t.TempDir()
	sessionsDir := filepath.Join(tmp, "sessions")
	projectsDir := filepath.Join(tmp, "projects")
	os.MkdirAll(sessionsDir, 0755)

	cwd := "/tmp/sharedproject"
	encoded := discovery.EncodeCWD(cwd)
	projDir := filepath.Join(projectsDir, encoded)
	os.MkdirAll(projDir, 0755)

	// Two sessions in the same project.
	writeJSON(t, filepath.Join(sessionsDir, "200.json"), map[string]any{
		"pid": 200, "sessionId": "sess-shared-1",
		"cwd": cwd, "startedAt": time.Now().Add(-5 * time.Minute).UnixMilli(),
	})
	writeJSON(t, filepath.Join(sessionsDir, "201.json"), map[string]any{
		"pid": 201, "sessionId": "sess-shared-2",
		"cwd": cwd, "startedAt": time.Now().Add(-3 * time.Minute).UnixMilli(),
	})

	// Session index with entries for both.
	indexData := map[string]any{
		"version": 1,
		"entries": []map[string]any{
			{"sessionId": "sess-shared-1", "summary": "Task A", "messageCount": 10},
			{"sessionId": "sess-shared-2", "summary": "Task B", "messageCount": 20},
		},
	}
	writeJSON(t, filepath.Join(projDir, "sessions-index.json"), indexData)

	sessions := Snapshot(sessionsDir, projectsDir)
	if len(sessions) != 2 {
		t.Fatalf("expected 2 sessions, got %d", len(sessions))
	}

	// Build a map for easier lookup.
	byID := make(map[string]Session, len(sessions))
	for _, s := range sessions {
		byID[s.ID] = s
	}

	if s, ok := byID["sess-shared-1"]; ok {
		if s.WorkingOn != "Task A" {
			t.Errorf("sess-shared-1 WorkingOn = %q, want %q", s.WorkingOn, "Task A")
		}
		if s.MessageCount != 10 {
			t.Errorf("sess-shared-1 MessageCount = %d, want 10", s.MessageCount)
		}
	} else {
		t.Error("sess-shared-1 not found in results")
	}

	if s, ok := byID["sess-shared-2"]; ok {
		if s.WorkingOn != "Task B" {
			t.Errorf("sess-shared-2 WorkingOn = %q, want %q", s.WorkingOn, "Task B")
		}
		if s.MessageCount != 20 {
			t.Errorf("sess-shared-2 MessageCount = %d, want 20", s.MessageCount)
		}
	} else {
		t.Error("sess-shared-2 not found in results")
	}
}

// ---------------------------------------------------------------------------
// TestReadLastMeaningfulLine — verify the extracted function works correctly.
// ---------------------------------------------------------------------------

func TestReadLastMeaningfulLine(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "test.jsonl")

	ts := time.Now().Format(time.RFC3339Nano)
	content := `{"type":"user","timestamp":"` + ts + `","message":{"role":"user","content":"Hello world"}}` + "\n" +
		`{"type":"assistant","timestamp":"` + ts + `","message":{"role":"assistant","content":"Hi there!"}}` + "\n"

	os.WriteFile(path, []byte(content), 0644)

	result := ReadLastMeaningfulLine(path)
	if result != "Hi there!" {
		t.Errorf("ReadLastMeaningfulLine = %q, want %q", result, "Hi there!")
	}
}

func TestReadLastMeaningfulLineEmpty(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "empty.jsonl")
	os.WriteFile(path, []byte(""), 0644)

	result := ReadLastMeaningfulLine(path)
	if result != "" {
		t.Errorf("ReadLastMeaningfulLine empty file = %q, want empty", result)
	}
}

func TestReadLastMeaningfulLineNonexistent(t *testing.T) {
	result := ReadLastMeaningfulLine("/nonexistent/path/test.jsonl")
	if result != "" {
		t.Errorf("ReadLastMeaningfulLine nonexistent = %q, want empty", result)
	}
}

func TestReadLastMeaningfulLineSkipsToolOutput(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "tool.jsonl")

	ts := time.Now().Format(time.RFC3339Nano)
	content := `{"type":"assistant","timestamp":"` + ts + `","message":{"role":"assistant","content":"Doing work"}}` + "\n" +
		`{"type":"tool_result","timestamp":"` + ts + `","message":{"role":"tool","content":"ok"}}` + "\n"

	os.WriteFile(path, []byte(content), 0644)

	result := ReadLastMeaningfulLine(path)
	// Should return the assistant message, not the tool result.
	if result != "Doing work" {
		t.Errorf("ReadLastMeaningfulLine = %q, want %q", result, "Doing work")
	}
}

func TestReadLastMeaningfulLineTruncation(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "long.jsonl")

	ts := time.Now().Format(time.RFC3339Nano)
	longText := ""
	for i := 0; i < 200; i++ {
		longText += "a"
	}
	content := `{"type":"assistant","timestamp":"` + ts + `","message":{"role":"assistant","content":"` + longText + `"}}` + "\n"
	os.WriteFile(path, []byte(content), 0644)

	result := ReadLastMeaningfulLine(path)
	if len(result) > 120 {
		t.Errorf("ReadLastMeaningfulLine should truncate to <=120 chars, got %d", len(result))
	}
	if len(result) > 3 && result[len(result)-3:] != "..." {
		t.Error("truncated result should end with '...'")
	}
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func writeJSON(t *testing.T, path string, v any) {
	t.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	s := ""
	for n > 0 {
		s = string(rune('0'+n%10)) + s
		n /= 10
	}
	return s
}
