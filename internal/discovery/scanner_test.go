package discovery

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// writeSessionFile creates a session JSON file in the given directory.
func writeSessionFile(t *testing.T, dir string, pid int, sessionID, cwd string, startedAt int64) string {
	t.Helper()
	data := map[string]any{
		"pid":       pid,
		"sessionId": sessionID,
		"cwd":       cwd,
		"startedAt": startedAt,
	}
	b, err := json.Marshal(data)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, fmt.Sprintf("%d.json", pid))
	if err := os.WriteFile(path, b, 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestScanSessions_Empty(t *testing.T) {
	dir := t.TempDir()
	sessions, err := ScanSessions(dir)
	if err != nil {
		t.Fatalf("ScanSessions() error = %v", err)
	}
	if len(sessions) != 0 {
		t.Errorf("len(sessions) = %d, want 0", len(sessions))
	}
}

func TestScanSessions_NonexistentDir(t *testing.T) {
	sessions, err := ScanSessions("/nonexistent/path/to/sessions")
	if err != nil {
		t.Fatalf("ScanSessions() should not error on missing dir, got %v", err)
	}
	if len(sessions) != 0 {
		t.Errorf("len(sessions) = %d, want 0", len(sessions))
	}
}

func TestScanSessions_ValidFiles(t *testing.T) {
	dir := t.TempDir()
	writeSessionFile(t, dir, 100, "sess-aaa", "/home/user/projectA", 1774054268177)
	writeSessionFile(t, dir, 200, "sess-bbb", "/home/user/projectB", 1774054268200)
	writeSessionFile(t, dir, 300, "sess-ccc", "/home/user/projectA", 1774054268300)

	sessions, err := ScanSessions(dir)
	if err != nil {
		t.Fatalf("ScanSessions() error = %v", err)
	}
	if len(sessions) != 3 {
		t.Fatalf("len(sessions) = %d, want 3", len(sessions))
	}

	// Verify fields are parsed correctly.
	found := map[string]bool{}
	for _, s := range sessions {
		found[s.ID] = true
		switch s.ID {
		case "sess-aaa":
			if s.PID != 100 {
				t.Errorf("sess-aaa PID = %d, want 100", s.PID)
			}
			if s.CWD != "/home/user/projectA" {
				t.Errorf("sess-aaa CWD = %q", s.CWD)
			}
			if s.ProjectName != "projectA" {
				t.Errorf("sess-aaa ProjectName = %q, want %q", s.ProjectName, "projectA")
			}
		case "sess-bbb":
			if s.PID != 200 {
				t.Errorf("sess-bbb PID = %d, want 200", s.PID)
			}
		}
	}

	if !found["sess-aaa"] || !found["sess-bbb"] || !found["sess-ccc"] {
		t.Errorf("missing sessions: found = %v", found)
	}
}

func TestScanSessions_MalformedJSON(t *testing.T) {
	dir := t.TempDir()
	// Write a valid one.
	writeSessionFile(t, dir, 100, "sess-good", "/home/user/proj", 1774054268177)
	// Write a malformed one.
	if err := os.WriteFile(filepath.Join(dir, "999.json"), []byte("{broken json"), 0644); err != nil {
		t.Fatal(err)
	}
	// Write a non-json file.
	if err := os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("not json"), 0644); err != nil {
		t.Fatal(err)
	}

	sessions, err := ScanSessions(dir)
	if err != nil {
		t.Fatalf("ScanSessions() error = %v", err)
	}
	// Should still return the valid session, skipping bad files.
	if len(sessions) != 1 {
		t.Errorf("len(sessions) = %d, want 1", len(sessions))
	}
}

func TestScanSessions_MissingFields(t *testing.T) {
	dir := t.TempDir()
	// Write a file missing sessionId.
	data := map[string]any{
		"pid":       500,
		"cwd":       "/home/user/proj",
		"startedAt": 1774054268177,
	}
	b, _ := json.Marshal(data)
	if err := os.WriteFile(filepath.Join(dir, "500.json"), b, 0644); err != nil {
		t.Fatal(err)
	}

	sessions, err := ScanSessions(dir)
	if err != nil {
		t.Fatalf("ScanSessions() error = %v", err)
	}
	// Missing sessionId should skip the file.
	if len(sessions) != 0 {
		t.Errorf("len(sessions) = %d, want 0 (missing sessionId)", len(sessions))
	}
}

func TestGroupByProject(t *testing.T) {
	sessions := []Session{
		{ID: "s1", CWD: "/home/user/projectA", ProjectName: "projectA"},
		{ID: "s2", CWD: "/home/user/projectB", ProjectName: "projectB"},
		{ID: "s3", CWD: "/home/user/projectA", ProjectName: "projectA"},
	}

	projects := GroupByProject(sessions)
	if len(projects) != 2 {
		t.Fatalf("len(projects) = %d, want 2", len(projects))
	}

	projectMap := map[string]Project{}
	for _, p := range projects {
		projectMap[p.Name] = p
	}

	pa, ok := projectMap["projectA"]
	if !ok {
		t.Fatal("projectA not found")
	}
	if len(pa.Sessions) != 2 {
		t.Errorf("projectA sessions = %d, want 2", len(pa.Sessions))
	}
	if pa.Path != "/home/user/projectA" {
		t.Errorf("projectA path = %q, want %q", pa.Path, "/home/user/projectA")
	}

	pb, ok := projectMap["projectB"]
	if !ok {
		t.Fatal("projectB not found")
	}
	if len(pb.Sessions) != 1 {
		t.Errorf("projectB sessions = %d, want 1", len(pb.Sessions))
	}
}

// TestGroupByProject_CollidingBasenames verifies that sessions with the same
// basename but different CWDs are separated into distinct projects, with names
// expanded to parent/basename.
func TestGroupByProject_CollidingBasenames(t *testing.T) {
	sessions := []Session{
		{ID: "s1", CWD: "/home/user/work/api", ProjectName: "api"},
		{ID: "s2", CWD: "/home/user/personal/api", ProjectName: "api"},
		{ID: "s3", CWD: "/home/user/work/api", ProjectName: "api"},
	}

	projects := GroupByProject(sessions)
	if len(projects) != 2 {
		t.Fatalf("len(projects) = %d, want 2", len(projects))
	}

	projectMap := map[string]Project{}
	for _, p := range projects {
		projectMap[p.Name] = p
	}

	// Colliding basenames should be expanded to parent/basename.
	workAPI, ok := projectMap["work/api"]
	if !ok {
		t.Fatalf("expected project 'work/api', got projects: %v", projectNames(projects))
	}
	if len(workAPI.Sessions) != 2 {
		t.Errorf("work/api sessions = %d, want 2", len(workAPI.Sessions))
	}
	if workAPI.Path != "/home/user/work/api" {
		t.Errorf("work/api path = %q, want %q", workAPI.Path, "/home/user/work/api")
	}

	personalAPI, ok := projectMap["personal/api"]
	if !ok {
		t.Fatalf("expected project 'personal/api', got projects: %v", projectNames(projects))
	}
	if len(personalAPI.Sessions) != 1 {
		t.Errorf("personal/api sessions = %d, want 1", len(personalAPI.Sessions))
	}
	if personalAPI.Path != "/home/user/personal/api" {
		t.Errorf("personal/api path = %q, want %q", personalAPI.Path, "/home/user/personal/api")
	}
}

// TestGroupByProject_MixedCollision verifies that only colliding basenames get
// expanded; unique basenames stay short.
func TestGroupByProject_MixedCollision(t *testing.T) {
	sessions := []Session{
		{ID: "s1", CWD: "/home/user/work/api", ProjectName: "api"},
		{ID: "s2", CWD: "/home/user/personal/api", ProjectName: "api"},
		{ID: "s3", CWD: "/home/user/GitHub/QuantMind", ProjectName: "QuantMind"},
	}

	projects := GroupByProject(sessions)
	if len(projects) != 3 {
		t.Fatalf("len(projects) = %d, want 3", len(projects))
	}

	projectMap := map[string]Project{}
	for _, p := range projects {
		projectMap[p.Name] = p
	}

	// "api" collides, so both should be expanded.
	if _, ok := projectMap["work/api"]; !ok {
		t.Errorf("expected 'work/api', got: %v", projectNames(projects))
	}
	if _, ok := projectMap["personal/api"]; !ok {
		t.Errorf("expected 'personal/api', got: %v", projectNames(projects))
	}
	// "QuantMind" is unique, stays as-is.
	if _, ok := projectMap["QuantMind"]; !ok {
		t.Errorf("expected 'QuantMind', got: %v", projectNames(projects))
	}
}

// TestGroupByProject_CollidingSessionNamesUpdated verifies that after
// disambiguation, each session's ProjectName is updated to match the project.
func TestGroupByProject_CollidingSessionNamesUpdated(t *testing.T) {
	sessions := []Session{
		{ID: "s1", CWD: "/home/user/work/api", ProjectName: "api"},
		{ID: "s2", CWD: "/home/user/personal/api", ProjectName: "api"},
	}

	projects := GroupByProject(sessions)

	for _, p := range projects {
		for _, s := range p.Sessions {
			if s.ProjectName != p.Name {
				t.Errorf("session %s ProjectName = %q, want %q (matching project)", s.ID, s.ProjectName, p.Name)
			}
		}
	}
}

// TestGroupByProject_EmptyCWD verifies that sessions with empty CWD still
// group under "(unknown)" without panicking.
func TestGroupByProject_EmptyCWD(t *testing.T) {
	sessions := []Session{
		{ID: "s1", CWD: "", ProjectName: ""},
		{ID: "s2", CWD: "", ProjectName: ""},
	}

	projects := GroupByProject(sessions)
	if len(projects) != 1 {
		t.Fatalf("len(projects) = %d, want 1", len(projects))
	}
	if projects[0].Name != "(unknown)" {
		t.Errorf("name = %q, want %q", projects[0].Name, "(unknown)")
	}
	if len(projects[0].Sessions) != 2 {
		t.Errorf("sessions = %d, want 2", len(projects[0].Sessions))
	}
}

// TestGroupByProject_RootPath verifies a CWD at the filesystem root does not
// panic when expanding (no parent component available).
func TestGroupByProject_RootPath(t *testing.T) {
	sessions := []Session{
		{ID: "s1", CWD: "/api", ProjectName: "api"},
		{ID: "s2", CWD: "/home/user/work/api", ProjectName: "api"},
	}

	projects := GroupByProject(sessions)
	if len(projects) != 2 {
		t.Fatalf("len(projects) = %d, want 2", len(projects))
	}

	projectMap := map[string]Project{}
	for _, p := range projects {
		projectMap[p.Name] = p
	}

	// /api has no parent dir to expand with, falls back to just "api".
	// /home/user/work/api expands to "work/api".
	// Since both can't be "api", the root one stays "api" and the other expands.
	if _, ok := projectMap["api"]; !ok {
		t.Errorf("expected 'api' for root path, got: %v", projectNames(projects))
	}
	if _, ok := projectMap["work/api"]; !ok {
		t.Errorf("expected 'work/api', got: %v", projectNames(projects))
	}
}

// projectNames is a test helper that returns all project names for diagnostics.
func projectNames(projects []Project) []string {
	names := make([]string, len(projects))
	for i, p := range projects {
		names[i] = p.Name
	}
	return names
}
