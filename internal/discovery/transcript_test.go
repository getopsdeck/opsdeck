package discovery

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestReadLastActivity_ValidTranscript(t *testing.T) {
	dir := t.TempDir()
	projectDir := filepath.Join(dir, "-Users-jason-project")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Write a JSONL transcript with timestamps.
	lines := `{"type":"user","timestamp":"2026-03-21T10:00:00.000Z","uuid":"aaa"}
{"type":"assistant","timestamp":"2026-03-21T10:01:00.000Z","uuid":"bbb"}
{"type":"user","timestamp":"2026-03-21T10:05:30.000Z","uuid":"ccc"}
`
	transcriptPath := filepath.Join(projectDir, "sess-123.jsonl")
	if err := os.WriteFile(transcriptPath, []byte(lines), 0644); err != nil {
		t.Fatal(err)
	}

	activity, err := ReadLastActivity(transcriptPath)
	if err != nil {
		t.Fatalf("ReadLastActivity() error = %v", err)
	}

	expected, _ := time.Parse(time.RFC3339Nano, "2026-03-21T10:05:30.000Z")
	if !activity.Equal(expected) {
		t.Errorf("activity = %v, want %v", activity, expected)
	}
}

func TestReadLastActivity_MissingFile(t *testing.T) {
	activity, err := ReadLastActivity("/nonexistent/path/transcript.jsonl")
	if err != nil {
		t.Fatalf("ReadLastActivity() should not error on missing file, got %v", err)
	}
	if !activity.IsZero() {
		t.Errorf("activity = %v, want zero time", activity)
	}
}

func TestReadLastActivity_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.jsonl")
	if err := os.WriteFile(path, []byte(""), 0644); err != nil {
		t.Fatal(err)
	}

	activity, err := ReadLastActivity(path)
	if err != nil {
		t.Fatalf("ReadLastActivity() error = %v", err)
	}
	if !activity.IsZero() {
		t.Errorf("activity = %v, want zero time", activity)
	}
}

func TestReadLastActivity_NoTimestamp(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "notimestamp.jsonl")
	lines := `{"type":"progress","data":{"type":"hook_progress"}}
{"type":"progress","data":{"type":"hook_progress"}}
`
	if err := os.WriteFile(path, []byte(lines), 0644); err != nil {
		t.Fatal(err)
	}

	activity, err := ReadLastActivity(path)
	if err != nil {
		t.Fatalf("ReadLastActivity() error = %v", err)
	}
	if !activity.IsZero() {
		t.Errorf("activity = %v, want zero time", activity)
	}
}

func TestReadLastActivity_MalformedLines(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.jsonl")
	lines := `{broken json
{"type":"user","timestamp":"2026-03-21T10:00:00.000Z","uuid":"aaa"}
not json at all
`
	if err := os.WriteFile(path, []byte(lines), 0644); err != nil {
		t.Fatal(err)
	}

	activity, err := ReadLastActivity(path)
	if err != nil {
		t.Fatalf("ReadLastActivity() error = %v", err)
	}
	// Should still find the valid timestamp.
	expected, _ := time.Parse(time.RFC3339Nano, "2026-03-21T10:00:00.000Z")
	if !activity.Equal(expected) {
		t.Errorf("activity = %v, want %v", activity, expected)
	}
}

func TestReadLastActivity_LastPromptEntry(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "lastprompt.jsonl")
	// A transcript ending with a last-prompt entry (no timestamp).
	lines := `{"type":"user","timestamp":"2026-03-21T10:00:00.000Z","uuid":"aaa"}
{"type":"last-prompt","lastPrompt":"hello","sessionId":"sess-1"}
`
	if err := os.WriteFile(path, []byte(lines), 0644); err != nil {
		t.Fatal(err)
	}

	activity, err := ReadLastActivity(path)
	if err != nil {
		t.Fatalf("ReadLastActivity() error = %v", err)
	}
	// Should return the timestamp from the user entry, not zero.
	expected, _ := time.Parse(time.RFC3339Nano, "2026-03-21T10:00:00.000Z")
	if !activity.Equal(expected) {
		t.Errorf("activity = %v, want %v", activity, expected)
	}
}

func TestFindTranscriptPath(t *testing.T) {
	dir := t.TempDir()
	// Simulate projects dir structure.
	projectDir := filepath.Join(dir, "-Users-jason-project")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatal(err)
	}
	transcriptPath := filepath.Join(projectDir, "sess-abc.jsonl")
	if err := os.WriteFile(transcriptPath, []byte(`{"type":"user"}`), 0644); err != nil {
		t.Fatal(err)
	}

	result := FindTranscriptPath(dir, "/Users/jason/project", "sess-abc")
	if result != transcriptPath {
		t.Errorf("FindTranscriptPath() = %q, want %q", result, transcriptPath)
	}
}

func TestFindTranscriptPath_Missing(t *testing.T) {
	dir := t.TempDir()
	result := FindTranscriptPath(dir, "/Users/jason/project", "nonexistent")
	if result != "" {
		t.Errorf("FindTranscriptPath() = %q, want empty string", result)
	}
}

func TestEncodeCWD(t *testing.T) {
	tests := []struct {
		cwd  string
		want string
	}{
		{"/Users/jason/project", "-Users-jason-project"},
		{"/home/user/my-project", "-home-user-my-project"},
		{"/", "-"},
	}
	for _, tt := range tests {
		got := EncodeCWD(tt.cwd)
		if got != tt.want {
			t.Errorf("EncodeCWD(%q) = %q, want %q", tt.cwd, got, tt.want)
		}
	}
}
