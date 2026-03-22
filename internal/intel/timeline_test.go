package intel

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestExtractTimeline_BasicEvents(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.jsonl")

	ts1 := "2026-03-22T10:00:00Z"
	ts2 := "2026-03-22T10:00:05Z"
	ts3 := "2026-03-22T10:00:10Z"
	ts4 := "2026-03-22T10:00:15Z"

	content := `{"type":"user","sessionId":"sess1","cwd":"/tmp/proj","timestamp":"` + ts1 + `","message":{"role":"user","content":"hello"}}
{"type":"assistant","sessionId":"sess1","timestamp":"` + ts2 + `","message":{"role":"assistant","content":[{"type":"text","text":"hi there"}]}}
{"type":"assistant","sessionId":"sess1","timestamp":"` + ts3 + `","message":{"role":"assistant","content":[{"type":"tool_use","name":"Edit","id":"t1","input":{}}]}}
{"type":"user","sessionId":"sess1","timestamp":"` + ts4 + `","message":{"role":"user","content":[{"type":"tool_result","tool_use_id":"t1","is_error":true}]}}
`
	os.WriteFile(path, []byte(content), 0644)

	tl, err := ExtractTimeline(path, time.Time{})
	if err != nil {
		t.Fatalf("ExtractTimeline() error: %v", err)
	}

	if tl.SessionID != "sess1" {
		t.Errorf("SessionID = %q, want %q", tl.SessionID, "sess1")
	}
	if tl.Project != "proj" {
		t.Errorf("Project = %q, want %q", tl.Project, "proj")
	}
	if len(tl.Events) != 4 {
		t.Fatalf("Events count = %d, want 4", len(tl.Events))
	}

	// Check event types.
	wantTypes := []string{"user", "text", "tool", "error"}
	for i, want := range wantTypes {
		if tl.Events[i].Type != want {
			t.Errorf("Events[%d].Type = %q, want %q", i, tl.Events[i].Type, want)
		}
	}

	// Check tool name.
	if tl.Events[2].Tool != "Edit" {
		t.Errorf("Events[2].Tool = %q, want %q", tl.Events[2].Tool, "Edit")
	}

	// Check durations (5s gaps between events).
	if tl.Events[0].Duration != 5 {
		t.Errorf("Events[0].Duration = %d, want 5", tl.Events[0].Duration)
	}
	if tl.Events[1].Duration != 5 {
		t.Errorf("Events[1].Duration = %d, want 5", tl.Events[1].Duration)
	}
}

func TestExtractTimeline_SinceFilter(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.jsonl")

	content := `{"type":"user","sessionId":"s1","cwd":"/tmp","timestamp":"2026-03-22T08:00:00Z","message":{"role":"user","content":"early"}}
{"type":"user","sessionId":"s1","timestamp":"2026-03-22T12:00:00Z","message":{"role":"user","content":"late"}}
`
	os.WriteFile(path, []byte(content), 0644)

	since, _ := time.Parse(time.RFC3339, "2026-03-22T10:00:00Z")
	tl, _ := ExtractTimeline(path, since)

	if len(tl.Events) != 1 {
		t.Fatalf("Expected 1 event after since filter, got %d", len(tl.Events))
	}
}

func TestExtractTimeline_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.jsonl")
	os.WriteFile(path, []byte(""), 0644)

	tl, err := ExtractTimeline(path, time.Time{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tl.Events) != 0 {
		t.Errorf("expected 0 events, got %d", len(tl.Events))
	}
}

func TestExtractTimeline_MissingFile(t *testing.T) {
	_, err := ExtractTimeline("/nonexistent/path.jsonl", time.Time{})
	if err == nil {
		t.Error("expected error for missing file")
	}
}

func TestExtractTimeline_SkipsSystemMessages(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.jsonl")

	content := `{"type":"user","sessionId":"s1","cwd":"/tmp","timestamp":"2026-03-22T10:00:00Z","message":{"role":"user","content":"<command-name>/model</command-name>"}}
{"type":"user","sessionId":"s1","timestamp":"2026-03-22T10:00:05Z","message":{"role":"user","content":"real message"}}
`
	os.WriteFile(path, []byte(content), 0644)

	tl, _ := ExtractTimeline(path, time.Time{})
	if len(tl.Events) != 1 {
		t.Fatalf("expected 1 event (system message skipped), got %d", len(tl.Events))
	}
	if tl.Events[0].Type != "user" {
		t.Errorf("expected user event, got %s", tl.Events[0].Type)
	}
}

func TestExtractTimeline_SkipsMetaMessages(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.jsonl")

	content := `{"type":"user","isMeta":true,"sessionId":"s1","cwd":"/tmp","timestamp":"2026-03-22T10:00:00Z","message":{"role":"user","content":"meta stuff"}}
{"type":"user","sessionId":"s1","timestamp":"2026-03-22T10:00:05Z","message":{"role":"user","content":"real stuff"}}
`
	os.WriteFile(path, []byte(content), 0644)

	tl, _ := ExtractTimeline(path, time.Time{})
	if len(tl.Events) != 1 {
		t.Fatalf("expected 1 event (meta skipped), got %d", len(tl.Events))
	}
}
