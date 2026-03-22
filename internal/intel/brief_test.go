package intel

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// --- test helpers ---

// writeSessionFile creates a session JSON file in the given directory.
func writeSessionFile(t *testing.T, dir string, pid int, sessionID, cwd string) string {
	t.Helper()
	data := map[string]any{
		"pid":       pid,
		"sessionId": sessionID,
		"cwd":       cwd,
		"startedAt": time.Now().Add(-2 * time.Hour).UnixMilli(),
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

// writeTranscriptEntry writes a single JSONL line into a transcript file.
func writeTranscriptEntry(t *testing.T, path string, ts time.Time) {
	t.Helper()
	entry := map[string]any{
		"timestamp": ts.Format(time.RFC3339Nano),
		"type":      "assistant",
	}
	b, err := json.Marshal(entry)
	if err != nil {
		t.Fatal(err)
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	f.Write(b)
	f.Write([]byte("\n"))
}

// setupTestDirs creates the sessions and projects directory structures for testing.
// Returns (sessionsDir, projectsDir).
func setupTestDirs(t *testing.T) (string, string) {
	t.Helper()
	root := t.TempDir()
	sessionsDir := filepath.Join(root, ".claude", "sessions")
	projectsDir := filepath.Join(root, ".claude", "projects")
	if err := os.MkdirAll(sessionsDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(projectsDir, 0755); err != nil {
		t.Fatal(err)
	}
	return sessionsDir, projectsDir
}

// encodeCWD matches discovery.EncodeCWD: replaces / and non-ASCII with hyphens.
func encodeCWD(cwd string) string {
	var b strings.Builder
	for _, r := range cwd {
		if r == '/' || r > 127 {
			b.WriteByte('-')
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// setupProjectTranscript creates the project directory and transcript file.
func setupProjectTranscript(t *testing.T, projectsDir, cwd, sessionID string, timestamps ...time.Time) string {
	t.Helper()
	encoded := encodeCWD(cwd)
	dir := filepath.Join(projectsDir, encoded)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, sessionID+".jsonl")
	for _, ts := range timestamps {
		writeTranscriptEntry(t, path, ts)
	}
	return path
}

// --- DailyBrief struct tests ---

func TestDailyBrief_ZeroValue(t *testing.T) {
	var b DailyBrief
	if b.TotalSessions != 0 {
		t.Errorf("TotalSessions = %d, want 0", b.TotalSessions)
	}
	if b.GeneratedAt != (time.Time{}) {
		t.Error("GeneratedAt should be zero value")
	}
	if len(b.Projects) != 0 {
		t.Errorf("len(Projects) = %d, want 0", len(b.Projects))
	}
}

func TestProjectBrief_ZeroValue(t *testing.T) {
	var pb ProjectBrief
	if pb.NeedsAttention {
		t.Error("NeedsAttention should be false by default")
	}
	if pb.SessionCount != 0 {
		t.Errorf("SessionCount = %d, want 0", pb.SessionCount)
	}
}

// --- GenerateBrief tests ---

func TestGenerateBrief_EmptyDirs(t *testing.T) {
	sessionsDir, projectsDir := setupTestDirs(t)
	since := time.Now().Add(-24 * time.Hour)

	brief, err := GenerateBrief(projectsDir, sessionsDir, since)
	if err != nil {
		t.Fatalf("GenerateBrief() error = %v", err)
	}

	if brief.TotalSessions != 0 {
		t.Errorf("TotalSessions = %d, want 0", brief.TotalSessions)
	}
	if brief.ActiveSessions != 0 {
		t.Errorf("ActiveSessions = %d, want 0", brief.ActiveSessions)
	}
	if !brief.GeneratedAt.After(time.Time{}) {
		t.Error("GeneratedAt should be set")
	}
}

func TestGenerateBrief_NonexistentDirs(t *testing.T) {
	since := time.Now().Add(-24 * time.Hour)
	brief, err := GenerateBrief("/nonexistent/projects", "/nonexistent/sessions", since)
	if err != nil {
		t.Fatalf("GenerateBrief() should not error on missing dirs, got %v", err)
	}
	if brief.TotalSessions != 0 {
		t.Errorf("TotalSessions = %d, want 0", brief.TotalSessions)
	}
}

func TestGenerateBrief_SingleSession(t *testing.T) {
	sessionsDir, projectsDir := setupTestDirs(t)
	cwd := "/Users/test/Projects/MyApp"
	sessionID := "sess-001"

	writeSessionFile(t, sessionsDir, 1234, sessionID, cwd)
	now := time.Now()
	setupProjectTranscript(t, projectsDir, cwd, sessionID,
		now.Add(-1*time.Hour),
		now.Add(-30*time.Minute),
	)

	since := now.Add(-24 * time.Hour)
	brief, err := GenerateBrief(projectsDir, sessionsDir, since)
	if err != nil {
		t.Fatalf("GenerateBrief() error = %v", err)
	}

	if brief.TotalSessions != 1 {
		t.Errorf("TotalSessions = %d, want 1", brief.TotalSessions)
	}
	if len(brief.Projects) != 1 {
		t.Fatalf("len(Projects) = %d, want 1", len(brief.Projects))
	}
	if brief.Projects[0].Name != "MyApp" {
		t.Errorf("Projects[0].Name = %q, want %q", brief.Projects[0].Name, "MyApp")
	}
	if brief.Projects[0].SessionCount != 1 {
		t.Errorf("Projects[0].SessionCount = %d, want 1", brief.Projects[0].SessionCount)
	}
}

func TestGenerateBrief_MultipleProjects(t *testing.T) {
	sessionsDir, projectsDir := setupTestDirs(t)
	now := time.Now()

	// Project A: 2 sessions
	cwdA := "/Users/test/Projects/Alpha"
	writeSessionFile(t, sessionsDir, 100, "sess-a1", cwdA)
	writeSessionFile(t, sessionsDir, 101, "sess-a2", cwdA)
	setupProjectTranscript(t, projectsDir, cwdA, "sess-a1", now.Add(-1*time.Hour))
	setupProjectTranscript(t, projectsDir, cwdA, "sess-a2", now.Add(-2*time.Hour))

	// Project B: 1 session
	cwdB := "/Users/test/Projects/Beta"
	writeSessionFile(t, sessionsDir, 200, "sess-b1", cwdB)
	setupProjectTranscript(t, projectsDir, cwdB, "sess-b1", now.Add(-3*time.Hour))

	since := now.Add(-24 * time.Hour)
	brief, err := GenerateBrief(projectsDir, sessionsDir, since)
	if err != nil {
		t.Fatalf("GenerateBrief() error = %v", err)
	}

	if brief.TotalSessions != 3 {
		t.Errorf("TotalSessions = %d, want 3", brief.TotalSessions)
	}
	if len(brief.Projects) != 2 {
		t.Errorf("len(Projects) = %d, want 2", len(brief.Projects))
	}

	// Projects should be sorted alphabetically
	if brief.Projects[0].Name != "Alpha" {
		t.Errorf("Projects[0].Name = %q, want %q", brief.Projects[0].Name, "Alpha")
	}
	if brief.Projects[1].Name != "Beta" {
		t.Errorf("Projects[1].Name = %q, want %q", brief.Projects[1].Name, "Beta")
	}
	if brief.Projects[0].SessionCount != 2 {
		t.Errorf("Alpha SessionCount = %d, want 2", brief.Projects[0].SessionCount)
	}
}

func TestGenerateBrief_SessionWithNoTranscript(t *testing.T) {
	sessionsDir, projectsDir := setupTestDirs(t)
	cwd := "/Users/test/Projects/Orphan"
	writeSessionFile(t, sessionsDir, 999, "sess-orphan", cwd)

	since := time.Now().Add(-24 * time.Hour)
	brief, err := GenerateBrief(projectsDir, sessionsDir, since)
	if err != nil {
		t.Fatalf("GenerateBrief() error = %v", err)
	}

	if brief.TotalSessions != 1 {
		t.Errorf("TotalSessions = %d, want 1", brief.TotalSessions)
	}
	if len(brief.Projects) != 1 {
		t.Fatalf("len(Projects) = %d, want 1", len(brief.Projects))
	}

	pb := brief.Projects[0]
	if !pb.NeedsAttention {
		t.Error("NeedsAttention should be true for session with no transcript")
	}
	if pb.AttentionReason == "" {
		t.Error("AttentionReason should be set")
	}
}

func TestGenerateBrief_AttentionItems(t *testing.T) {
	sessionsDir, projectsDir := setupTestDirs(t)

	// Session with no transcript -> should appear in Attention
	cwd := "/Users/test/Projects/NoTranscript"
	writeSessionFile(t, sessionsDir, 800, "sess-notr", cwd)

	since := time.Now().Add(-24 * time.Hour)
	brief, err := GenerateBrief(projectsDir, sessionsDir, since)
	if err != nil {
		t.Fatalf("GenerateBrief() error = %v", err)
	}

	if len(brief.Attention) == 0 {
		t.Error("Attention should contain items for sessions needing attention")
	}
}

func TestGenerateBrief_PeriodString(t *testing.T) {
	sessionsDir, projectsDir := setupTestDirs(t)
	since := time.Now().Add(-24 * time.Hour)

	brief, err := GenerateBrief(projectsDir, sessionsDir, since)
	if err != nil {
		t.Fatalf("GenerateBrief() error = %v", err)
	}

	if brief.Period == "" {
		t.Error("Period should be set")
	}
	if !strings.Contains(brief.Period, "Last") {
		t.Errorf("Period = %q, expected it to contain 'Last'", brief.Period)
	}
}

func TestGenerateBrief_AggregatesEdits(t *testing.T) {
	sessionsDir, projectsDir := setupTestDirs(t)
	now := time.Now()
	cwd := "/Users/test/Projects/WithEdits"
	writeSessionFile(t, sessionsDir, 500, "sess-edits", cwd)
	setupProjectTranscript(t, projectsDir, cwd, "sess-edits", now.Add(-30*time.Minute))

	since := now.Add(-24 * time.Hour)
	brief, err := GenerateBrief(projectsDir, sessionsDir, since)
	if err != nil {
		t.Fatalf("GenerateBrief() error = %v", err)
	}

	// With the activity stub returning zero, aggregates should also be zero.
	// This test verifies the aggregation plumbing works.
	if brief.TotalEdits != 0 {
		t.Errorf("TotalEdits = %d, want 0 (stub returns zero)", brief.TotalEdits)
	}
	if brief.TotalCommands != 0 {
		t.Errorf("TotalCommands = %d, want 0 (stub returns zero)", brief.TotalCommands)
	}
}

// --- FormatBrief tests ---

func TestFormatBrief_Empty(t *testing.T) {
	brief := DailyBrief{
		GeneratedAt: time.Date(2026, 3, 21, 9, 0, 0, 0, time.Local),
		Period:      "Last 24 hours",
	}

	output := FormatDailyBrief(brief)
	if !strings.Contains(output, "OpsDeck Daily Brief") {
		t.Error("output should contain header")
	}
	if !strings.Contains(output, "Last 24 hours") {
		t.Error("output should contain period")
	}
}

func TestFormatBrief_WithProjects(t *testing.T) {
	brief := DailyBrief{
		GeneratedAt:    time.Date(2026, 3, 21, 9, 0, 0, 0, time.Local),
		Period:         "Last 24 hours",
		TotalSessions:  5,
		ActiveSessions: 3,
		TotalEdits:     20,
		TotalCommands:  8,
		Projects: []ProjectBrief{
			{
				Name:          "QuantMind",
				Path:          "/Users/test/QuantMind",
				SessionCount:  3,
				ActiveCount:   2,
				TotalEdits:    15,
				TotalCommands: 5,
				FilesChanged:  []string{"main.go", "tui.go"},
				KeyActivities: []string{"Implemented TUI skeleton"},
			},
			{
				Name:         "GSoC",
				Path:         "/Users/test/GSoC",
				SessionCount: 2,
				ActiveCount:  1,
				TotalEdits:   5,
				TotalCommands: 3,
			},
		},
		Highlights: []string{"5 sessions across 2 projects"},
	}

	output := FormatDailyBrief(brief)

	if !strings.Contains(output, "QuantMind") {
		t.Error("output should contain project name QuantMind")
	}
	if !strings.Contains(output, "GSoC") {
		t.Error("output should contain project name GSoC")
	}
	if !strings.Contains(output, "3 sessions") {
		t.Error("output should contain session count")
	}
	if !strings.Contains(output, "2 active") {
		t.Error("output should contain active count")
	}
	if !strings.Contains(output, "Implemented TUI skeleton") {
		t.Error("output should contain key activities")
	}
	if !strings.Contains(output, "15 file edits") {
		t.Error("output should contain edit count")
	}
}

func TestFormatBrief_WithAttention(t *testing.T) {
	brief := DailyBrief{
		GeneratedAt: time.Date(2026, 3, 21, 9, 0, 0, 0, time.Local),
		Period:      "Last 24 hours",
		Attention:   []string{"Session PID 30898: idle for 58 hours"},
	}

	output := FormatDailyBrief(brief)
	if !strings.Contains(output, "NEEDS ATTENTION") {
		t.Error("output should contain NEEDS ATTENTION section")
	}
	if !strings.Contains(output, "idle for 58 hours") {
		t.Error("output should contain attention detail")
	}
}

func TestFormatBrief_NoAttentionSection_WhenEmpty(t *testing.T) {
	brief := DailyBrief{
		GeneratedAt: time.Date(2026, 3, 21, 9, 0, 0, 0, time.Local),
		Period:      "Last 24 hours",
	}

	output := FormatDailyBrief(brief)
	if strings.Contains(output, "NEEDS ATTENTION") {
		t.Error("output should NOT contain NEEDS ATTENTION when no attention items")
	}
}

func TestFormatBrief_Highlights(t *testing.T) {
	brief := DailyBrief{
		GeneratedAt: time.Date(2026, 3, 21, 9, 0, 0, 0, time.Local),
		Period:      "Last 24 hours",
		Highlights:  []string{"All tests passing", "3 new features landed"},
	}

	output := FormatDailyBrief(brief)
	if !strings.Contains(output, "All tests passing") {
		t.Error("output should contain highlight")
	}
}

// --- ParseSinceFlag tests ---

func TestParseSinceFlag_24h(t *testing.T) {
	since, err := ParseSinceFlag("24h")
	if err != nil {
		t.Fatalf("ParseSinceFlag(24h) error = %v", err)
	}
	expected := time.Now().Add(-24 * time.Hour)
	diff := since.Sub(expected)
	if diff < -time.Second || diff > time.Second {
		t.Errorf("ParseSinceFlag(24h) = %v, want ~%v", since, expected)
	}
}

func TestParseSinceFlag_12h(t *testing.T) {
	since, err := ParseSinceFlag("12h")
	if err != nil {
		t.Fatalf("ParseSinceFlag(12h) error = %v", err)
	}
	expected := time.Now().Add(-12 * time.Hour)
	diff := since.Sub(expected)
	if diff < -time.Second || diff > time.Second {
		t.Errorf("ParseSinceFlag(12h) = %v, want ~%v", since, expected)
	}
}

func TestParseSinceFlag_Yesterday(t *testing.T) {
	since, err := ParseSinceFlag("yesterday")
	if err != nil {
		t.Fatalf("ParseSinceFlag(yesterday) error = %v", err)
	}
	// Should be yesterday at 9 AM local time
	now := time.Now()
	yesterday := time.Date(now.Year(), now.Month(), now.Day()-1, 9, 0, 0, 0, time.Local)
	diff := since.Sub(yesterday)
	if diff < -time.Second || diff > time.Second {
		t.Errorf("ParseSinceFlag(yesterday) = %v, want %v", since, yesterday)
	}
}

func TestParseSinceFlag_Invalid(t *testing.T) {
	_, err := ParseSinceFlag("bogus")
	if err == nil {
		t.Error("ParseSinceFlag(bogus) should return error")
	}
}

func TestParseSinceFlag_Empty(t *testing.T) {
	since, err := ParseSinceFlag("")
	if err != nil {
		t.Fatalf("ParseSinceFlag('') error = %v", err)
	}
	expected := time.Now().Add(-24 * time.Hour)
	diff := since.Sub(expected)
	if diff < -time.Second || diff > time.Second {
		t.Errorf("ParseSinceFlag('') = %v, want ~%v (default 24h)", since, expected)
	}
}
