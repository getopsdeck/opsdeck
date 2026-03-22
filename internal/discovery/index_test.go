package discovery

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseSessionIndex_Valid(t *testing.T) {
	dir := t.TempDir()
	projectDir := filepath.Join(dir, "-Users-jason-project")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatal(err)
	}

	indexJSON := `{
  "version": 1,
  "entries": [
    {
      "sessionId": "sess-aaa",
      "fullPath": "/Users/jason/.claude/projects/-Users-jason-project/sess-aaa.jsonl",
      "fileMtime": 1774054268177,
      "firstPrompt": "Hello",
      "summary": "Worked on feature X",
      "messageCount": 15,
      "created": "2026-03-21T10:00:00.000Z",
      "modified": "2026-03-21T10:30:00.000Z",
      "gitBranch": "main"
    },
    {
      "sessionId": "sess-bbb",
      "fullPath": "/Users/jason/.claude/projects/-Users-jason-project/sess-bbb.jsonl",
      "fileMtime": 1774054268200,
      "firstPrompt": "Fix bug",
      "summary": "Fixed auth bug",
      "messageCount": 8,
      "created": "2026-03-21T11:00:00.000Z",
      "modified": "2026-03-21T11:15:00.000Z",
      "gitBranch": "fix/auth"
    }
  ]
}`

	indexPath := filepath.Join(projectDir, "sessions-index.json")
	if err := os.WriteFile(indexPath, []byte(indexJSON), 0644); err != nil {
		t.Fatal(err)
	}

	entries, err := ParseSessionIndex(indexPath)
	if err != nil {
		t.Fatalf("ParseSessionIndex() error = %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("len(entries) = %d, want 2", len(entries))
	}

	e := entries["sess-aaa"]
	if e.Summary != "Worked on feature X" {
		t.Errorf("sess-aaa summary = %q, want %q", e.Summary, "Worked on feature X")
	}
	if e.MessageCount != 15 {
		t.Errorf("sess-aaa messageCount = %d, want 15", e.MessageCount)
	}

	e2 := entries["sess-bbb"]
	if e2.Summary != "Fixed auth bug" {
		t.Errorf("sess-bbb summary = %q, want %q", e2.Summary, "Fixed auth bug")
	}
}

func TestParseSessionIndex_MissingFile(t *testing.T) {
	entries, err := ParseSessionIndex("/nonexistent/sessions-index.json")
	if err != nil {
		t.Fatalf("ParseSessionIndex() should not error on missing file, got %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("len(entries) = %d, want 0", len(entries))
	}
}

func TestParseSessionIndex_MalformedJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sessions-index.json")
	if err := os.WriteFile(path, []byte("{broken json"), 0644); err != nil {
		t.Fatal(err)
	}

	entries, err := ParseSessionIndex(path)
	if err != nil {
		t.Fatalf("ParseSessionIndex() should not error on malformed JSON, got %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("len(entries) = %d, want 0", len(entries))
	}
}

func TestParseSessionIndex_EmptyEntries(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sessions-index.json")
	if err := os.WriteFile(path, []byte(`{"version":1,"entries":[]}`), 0644); err != nil {
		t.Fatal(err)
	}

	entries, err := ParseSessionIndex(path)
	if err != nil {
		t.Fatalf("ParseSessionIndex() error = %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("len(entries) = %d, want 0", len(entries))
	}
}

func TestParseSessionIndex_UnknownFields(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sessions-index.json")
	indexJSON := `{
  "version": 1,
  "entries": [{
    "sessionId": "sess-1",
    "summary": "test",
    "messageCount": 5,
    "unknownField": "should not break",
    "anotherNewField": 42
  }]
}`
	if err := os.WriteFile(path, []byte(indexJSON), 0644); err != nil {
		t.Fatal(err)
	}

	entries, err := ParseSessionIndex(path)
	if err != nil {
		t.Fatalf("ParseSessionIndex() error = %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("len(entries) = %d, want 1", len(entries))
	}
	if entries["sess-1"].Summary != "test" {
		t.Errorf("summary = %q, want %q", entries["sess-1"].Summary, "test")
	}
}

func TestFindSessionIndex(t *testing.T) {
	dir := t.TempDir()
	projectDir := filepath.Join(dir, "-Users-jason-project")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatal(err)
	}
	indexPath := filepath.Join(projectDir, "sessions-index.json")
	if err := os.WriteFile(indexPath, []byte(`{"version":1,"entries":[]}`), 0644); err != nil {
		t.Fatal(err)
	}

	result := FindSessionIndex(dir, "/Users/jason/project")
	if result != indexPath {
		t.Errorf("FindSessionIndex() = %q, want %q", result, indexPath)
	}
}

func TestFindSessionIndex_Missing(t *testing.T) {
	dir := t.TempDir()
	result := FindSessionIndex(dir, "/Users/jason/project")
	if result != "" {
		t.Errorf("FindSessionIndex() = %q, want empty", result)
	}
}
