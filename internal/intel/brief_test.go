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

// --- FormatDailyBriefVerbose tests (old format, renamed) ---

func TestFormatBriefVerbose_Empty(t *testing.T) {
	brief := DailyBrief{
		GeneratedAt: time.Date(2026, 3, 21, 9, 0, 0, 0, time.Local),
		Period:      "Last 24 hours",
	}

	output := FormatDailyBriefVerbose(brief)
	if !strings.Contains(output, "OpsDeck Daily Brief") {
		t.Error("output should contain header")
	}
	if !strings.Contains(output, "Last 24 hours") {
		t.Error("output should contain period")
	}
}

func TestFormatBriefVerbose_WithProjects(t *testing.T) {
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

	output := FormatDailyBriefVerbose(brief)

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

func TestFormatBriefVerbose_WithAttention(t *testing.T) {
	brief := DailyBrief{
		GeneratedAt: time.Date(2026, 3, 21, 9, 0, 0, 0, time.Local),
		Period:      "Last 24 hours",
		Attention:   []string{"Session PID 30898: idle for 58 hours"},
	}

	output := FormatDailyBriefVerbose(brief)
	if !strings.Contains(output, "NEEDS ATTENTION") {
		t.Error("output should contain NEEDS ATTENTION section")
	}
	if !strings.Contains(output, "idle for 58 hours") {
		t.Error("output should contain attention detail")
	}
}

func TestFormatBriefVerbose_NoAttentionSection_WhenEmpty(t *testing.T) {
	brief := DailyBrief{
		GeneratedAt: time.Date(2026, 3, 21, 9, 0, 0, 0, time.Local),
		Period:      "Last 24 hours",
	}

	output := FormatDailyBriefVerbose(brief)
	if strings.Contains(output, "NEEDS ATTENTION") {
		t.Error("output should NOT contain NEEDS ATTENTION when no attention items")
	}
}

func TestFormatBriefVerbose_Highlights(t *testing.T) {
	brief := DailyBrief{
		GeneratedAt: time.Date(2026, 3, 21, 9, 0, 0, 0, time.Local),
		Period:      "Last 24 hours",
		Highlights:  []string{"All tests passing", "3 new features landed"},
	}

	output := FormatDailyBriefVerbose(brief)
	if !strings.Contains(output, "All tests passing") {
		t.Error("output should contain highlight")
	}
}

// --- FormatDailyBrief (secretary format) tests ---

func TestFormatDailyBrief_Header(t *testing.T) {
	brief := DailyBrief{
		GeneratedAt:   time.Date(2026, 3, 23, 9, 0, 0, 0, time.Local),
		Period:        "Last 24 hours",
		TotalSessions: 16,
		Projects: []ProjectBrief{
			{Name: "Alpha", ActiveCount: 1, TotalEdits: 5},
			{Name: "Beta", ActiveCount: 0},
		},
	}

	output := FormatDailyBrief(brief)

	if !strings.Contains(output, "OpsDeck Morning Brief") {
		t.Error("header should say 'OpsDeck Morning Brief'")
	}
	// Should include day-of-week.
	if !strings.Contains(output, "Monday") {
		t.Errorf("header should contain day-of-week, got:\n%s", output)
	}
	// Should show project and session count.
	if !strings.Contains(output, "2 projects") {
		t.Errorf("header should contain project count, got:\n%s", output)
	}
	if !strings.Contains(output, "16 sessions") {
		t.Errorf("header should contain session count, got:\n%s", output)
	}
}

func TestFormatDailyBrief_WaitingSessions(t *testing.T) {
	now := time.Now()
	brief := DailyBrief{
		GeneratedAt:   now,
		TotalSessions: 2,
		Projects: []ProjectBrief{
			{
				Name:       "QuantMind",
				ActiveCount: 1,
				TotalEdits: 10,
				WaitingSessions: []WaitingSession{
					{
						SessionID:   "b901962b-1234-5678-9abc-def012345678",
						WaitingSince: now.Add(-8 * 24 * time.Hour),
						LastUserMsg: "let's go",
					},
				},
			},
		},
	}

	output := FormatDailyBrief(brief)

	if !strings.Contains(output, "NEEDS YOUR ATTENTION") {
		t.Error("should contain NEEDS YOUR ATTENTION section")
	}
	if !strings.Contains(output, "QuantMind") {
		t.Error("should contain project name in waiting section")
	}
	if !strings.Contains(output, "b901962b") {
		t.Error("should contain truncated session ID")
	}
	if !strings.Contains(output, "8 days") {
		t.Errorf("should contain wait duration '8 days', got:\n%s", output)
	}
	if !strings.Contains(output, "let's go") {
		t.Error("should contain last user message")
	}
}

func TestFormatDailyBrief_NoWaitingSessions(t *testing.T) {
	brief := DailyBrief{
		GeneratedAt:   time.Now(),
		TotalSessions: 1,
		Projects: []ProjectBrief{
			{Name: "Alpha", ActiveCount: 1, TotalEdits: 5},
		},
	}

	output := FormatDailyBrief(brief)

	if strings.Contains(output, "NEEDS YOUR ATTENTION") {
		t.Error("should NOT contain NEEDS YOUR ATTENTION when no waiting sessions")
	}
}

func TestFormatDailyBrief_ProjectUpdates(t *testing.T) {
	brief := DailyBrief{
		GeneratedAt:   time.Now(),
		TotalSessions: 3,
		Projects: []ProjectBrief{
			{
				Name:       "OpsDeck",
				ActiveCount: 1,
				TotalEdits: 151,
				OneLine:    "shipped v0.9.0",
				Branch:     "main",
				IsDirty:    true,
				FilesChanged: make([]string, 35),
			},
			{
				Name:       "FusionSQL",
				ActiveCount: 1,
				TotalEdits: 3,
				OneLine:    "debugging session loss issue",
				Branch:     "dev",
				IsDirty:    true,
			},
		},
	}

	output := FormatDailyBrief(brief)

	if !strings.Contains(output, "PROJECT UPDATES") {
		t.Error("should contain PROJECT UPDATES section")
	}
	if !strings.Contains(output, "OpsDeck") {
		t.Error("should contain project name OpsDeck")
	}
	if !strings.Contains(output, "shipped v0.9.0") {
		t.Error("should contain one-line summary")
	}
	if !strings.Contains(output, "151 edits") {
		t.Error("should contain edit count in project line")
	}
	if !strings.Contains(output, "main*") {
		t.Error("should show branch with dirty marker")
	}
	if !strings.Contains(output, "dev*") {
		t.Error("should show branch with dirty marker for FusionSQL")
	}
}

func TestFormatDailyBrief_IdleProjects(t *testing.T) {
	now := time.Now()
	brief := DailyBrief{
		GeneratedAt:   now,
		TotalSessions: 3,
		Projects: []ProjectBrief{
			{Name: "ActiveProject", ActiveCount: 1, TotalEdits: 5, OneLine: "working"},
			{
				Name:         "IdleProject",
				SessionCount: 2,
				ActiveCount:  0,
				TotalEdits:   0,
				LastActive:   now.Add(-3 * 24 * time.Hour),
			},
		},
	}

	output := FormatDailyBrief(brief)

	if !strings.Contains(output, "IDLE") {
		t.Errorf("should contain IDLE section, got:\n%s", output)
	}
	if !strings.Contains(output, "IdleProject") {
		t.Error("should contain idle project name")
	}
	if !strings.Contains(output, "3 days ago") {
		t.Errorf("should contain last active time '3 days ago', got:\n%s", output)
	}
}

func TestFormatDailyBrief_CostLine(t *testing.T) {
	brief := DailyBrief{
		GeneratedAt:   time.Now(),
		TotalSessions: 1,
		CostEstimate:  270.50,
		Projects: []ProjectBrief{
			{Name: "Alpha", ActiveCount: 1, TotalEdits: 5},
		},
	}

	output := FormatDailyBrief(brief)

	if !strings.Contains(output, "SPEND") {
		t.Errorf("should contain SPEND line, got:\n%s", output)
	}
	if !strings.Contains(output, "$271") && !strings.Contains(output, "$270") {
		t.Errorf("should contain cost estimate near $270, got:\n%s", output)
	}
}

func TestFormatDailyBrief_NoCostLine_WhenZero(t *testing.T) {
	brief := DailyBrief{
		GeneratedAt:   time.Now(),
		TotalSessions: 1,
		CostEstimate:  0,
		Projects: []ProjectBrief{
			{Name: "Alpha", ActiveCount: 1, TotalEdits: 5},
		},
	}

	output := FormatDailyBrief(brief)

	if strings.Contains(output, "SPEND") {
		t.Error("should NOT contain SPEND line when cost is zero")
	}
}

func TestFormatDailyBrief_Empty(t *testing.T) {
	brief := DailyBrief{
		GeneratedAt: time.Now(),
	}

	output := FormatDailyBrief(brief)

	if !strings.Contains(output, "No sessions found") {
		t.Errorf("empty brief should say 'No sessions found', got:\n%s", output)
	}
}

func TestFormatDailyBrief_NoRawCounts(t *testing.T) {
	brief := DailyBrief{
		GeneratedAt:    time.Now(),
		TotalSessions:  5,
		ActiveSessions: 3,
		TotalEdits:     20,
		TotalCommands:  8,
		Projects: []ProjectBrief{
			{Name: "Alpha", ActiveCount: 1, TotalEdits: 5, OneLine: "working"},
		},
	}

	output := FormatDailyBrief(brief)

	// The old format had lines like "You worked across N projects, made X edits, ran Y commands"
	// and "N sessions had activity in period". The new format should not have these.
	if strings.Contains(output, "ran 8 commands") {
		t.Error("should NOT contain raw command count in new format")
	}
	if strings.Contains(output, "sessions had activity") {
		t.Error("should NOT contain 'sessions had activity' noise")
	}
}

// --- SummarizeOneLine tests ---

func TestSummarizeOneLine_Empty(t *testing.T) {
	result := SummarizeOneLine(nil)
	if result != "" {
		t.Errorf("SummarizeOneLine(nil) = %q, want empty", result)
	}
}

func TestSummarizeOneLine_SingleGitOp(t *testing.T) {
	activities := []Activity{
		{Type: ActivityGitOp, Description: "Committed, pushed, created PR"},
	}
	result := SummarizeOneLine(activities)
	if result == "" {
		t.Error("should return non-empty for git op")
	}
}

func TestSummarizeOneLine_EditsOnly(t *testing.T) {
	activities := []Activity{
		{Type: ActivityFileEdit, Description: "Edited main.go"},
		{Type: ActivityFileEdit, Description: "Edited tui.go"},
		{Type: ActivityFileEdit, Description: "Edited brief.go"},
	}
	result := SummarizeOneLine(activities)
	if result == "" {
		t.Error("should return non-empty for edits")
	}
	// Should be concise, not list every file.
	if len(result) > 80 {
		t.Errorf("one-line summary too long (%d chars): %q", len(result), result)
	}
}

func TestSummarizeOneLine_MixedActivities(t *testing.T) {
	activities := []Activity{
		{Type: ActivityUserRequest, Description: "Fix the auth bug"},
		{Type: ActivityFileEdit, Description: "Edited auth.go"},
		{Type: ActivityBashCommand, Description: "Ran tests"},
		{Type: ActivityGitOp, Description: "Git commit"},
	}
	result := SummarizeOneLine(activities)
	if result == "" {
		t.Error("should return non-empty for mixed activities")
	}
	// Should prioritize activity (git/edits) over user messages.
	if !strings.Contains(strings.ToLower(result), "commit") {
		t.Errorf("should describe activity (git/edits), got: %q", result)
	}
}

// --- formatWaitDuration tests ---

func TestFormatWaitDuration_Minutes(t *testing.T) {
	result := formatWaitDuration(15 * time.Minute)
	if result != "15 min" {
		t.Errorf("formatWaitDuration(15m) = %q, want %q", result, "15 min")
	}
}

func TestFormatWaitDuration_Hours(t *testing.T) {
	result := formatWaitDuration(3 * time.Hour)
	if result != "3 hours" {
		t.Errorf("formatWaitDuration(3h) = %q, want %q", result, "3 hours")
	}
}

func TestFormatWaitDuration_OneHour(t *testing.T) {
	result := formatWaitDuration(1 * time.Hour)
	if result != "1 hour" {
		t.Errorf("formatWaitDuration(1h) = %q, want %q", result, "1 hour")
	}
}

func TestFormatWaitDuration_Days(t *testing.T) {
	result := formatWaitDuration(8 * 24 * time.Hour)
	if result != "8 days" {
		t.Errorf("formatWaitDuration(8d) = %q, want %q", result, "8 days")
	}
}

func TestFormatWaitDuration_OneDay(t *testing.T) {
	result := formatWaitDuration(1 * 24 * time.Hour)
	if result != "1 day" {
		t.Errorf("formatWaitDuration(1d) = %q, want %q", result, "1 day")
	}
}

func TestFormatWaitDuration_LessThanMinute(t *testing.T) {
	result := formatWaitDuration(30 * time.Second)
	if result != "just now" {
		t.Errorf("formatWaitDuration(30s) = %q, want %q", result, "just now")
	}
}

// --- formatTimeAgo tests ---

func TestFormatTimeAgo_Days(t *testing.T) {
	result := formatTimeAgo(time.Now().Add(-3 * 24 * time.Hour))
	if result != "3 days ago" {
		t.Errorf("formatTimeAgo(-3d) = %q, want %q", result, "3 days ago")
	}
}

func TestFormatTimeAgo_Hours(t *testing.T) {
	result := formatTimeAgo(time.Now().Add(-5 * time.Hour))
	if result != "5 hours ago" {
		t.Errorf("formatTimeAgo(-5h) = %q, want %q", result, "5 hours ago")
	}
}

func TestFormatTimeAgo_Minutes(t *testing.T) {
	result := formatTimeAgo(time.Now().Add(-20 * time.Minute))
	if result != "20 min ago" {
		t.Errorf("formatTimeAgo(-20m) = %q, want %q", result, "20 min ago")
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
