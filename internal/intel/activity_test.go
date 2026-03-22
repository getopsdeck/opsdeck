package intel

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

// writeTranscript creates a JSONL file from individual lines.
func writeTranscript(t *testing.T, dir string, lines ...string) string {
	t.Helper()
	path := filepath.Join(dir, "test-session.jsonl")
	data := strings.Join(lines, "\n") + "\n"
	if err := os.WriteFile(path, []byte(data), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

// mustParseTime parses an RFC3339 timestamp or panics.
func mustParseTime(s string) time.Time {
	t, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		panic(err)
	}
	return t
}

// ---------------------------------------------------------------------------
// Realistic JSONL lines (based on real Claude Code transcript format)
// ---------------------------------------------------------------------------

const (
	// A real user message with string content.
	userMsgLine = `{"type":"user","message":{"role":"user","content":"Fix the auth bug in login.go"},"isMeta":false,"timestamp":"2026-03-21T10:00:00.000Z","uuid":"u1","sessionId":"sess-001","cwd":"/home/dev/myproject"}`

	// A meta user message (should be skipped for "last user message").
	userMetaLine = `{"type":"user","message":{"role":"user","content":"<local-command-caveat>auto</local-command-caveat>"},"isMeta":true,"timestamp":"2026-03-21T10:00:01.000Z","uuid":"u-meta","sessionId":"sess-001","cwd":"/home/dev/myproject"}`

	// Assistant text block.
	assistantTextLine = `{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"I will fix the auth bug by updating the token validation logic."}]},"timestamp":"2026-03-21T10:01:00.000Z","uuid":"a1","sessionId":"sess-001","cwd":"/home/dev/myproject"}`

	// Assistant with Read tool call.
	assistantReadLine = `{"type":"assistant","message":{"role":"assistant","content":[{"type":"tool_use","name":"Read","id":"tool-r1","input":{"file_path":"/home/dev/myproject/login.go"}}]},"timestamp":"2026-03-21T10:02:00.000Z","uuid":"a2","sessionId":"sess-001","cwd":"/home/dev/myproject"}`

	// Assistant with Edit tool call.
	assistantEditLine = `{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"Fixing the validation."},{"type":"tool_use","name":"Edit","id":"tool-e1","input":{"file_path":"/home/dev/myproject/login.go","old_string":"old","new_string":"new"}}]},"timestamp":"2026-03-21T10:03:00.000Z","uuid":"a3","sessionId":"sess-001","cwd":"/home/dev/myproject"}`

	// Assistant with Write tool call.
	assistantWriteLine = `{"type":"assistant","message":{"role":"assistant","content":[{"type":"tool_use","name":"Write","id":"tool-w1","input":{"file_path":"/home/dev/myproject/auth_test.go","content":"package main"}}]},"timestamp":"2026-03-21T10:04:00.000Z","uuid":"a4","sessionId":"sess-001","cwd":"/home/dev/myproject"}`

	// Assistant with Bash tool call (non-git).
	assistantBashLine = `{"type":"assistant","message":{"role":"assistant","content":[{"type":"tool_use","name":"Bash","id":"tool-b1","input":{"command":"go test ./...","description":"Run tests"}}]},"timestamp":"2026-03-21T10:05:00.000Z","uuid":"a5","sessionId":"sess-001","cwd":"/home/dev/myproject"}`

	// Assistant with Bash tool call (git commit).
	assistantGitCommitLine = `{"type":"assistant","message":{"role":"assistant","content":[{"type":"tool_use","name":"Bash","id":"tool-g1","input":{"command":"git commit -m \"fix auth bug\"","description":"Commit the fix"}}]},"timestamp":"2026-03-21T10:06:00.000Z","uuid":"a6","sessionId":"sess-001","cwd":"/home/dev/myproject"}`

	// Assistant with Bash tool call (git push).
	assistantGitPushLine = `{"type":"assistant","message":{"role":"assistant","content":[{"type":"tool_use","name":"Bash","id":"tool-g2","input":{"command":"git push origin main","description":"Push to remote"}}]},"timestamp":"2026-03-21T10:07:00.000Z","uuid":"a7","sessionId":"sess-001","cwd":"/home/dev/myproject"}`

	// Assistant with Bash tool call (gh pr create).
	assistantPRLine = `{"type":"assistant","message":{"role":"assistant","content":[{"type":"tool_use","name":"Bash","id":"tool-pr1","input":{"command":"gh pr create --title \"Fix auth\" --body \"fixes #42\"","description":"Create PR"}}]},"timestamp":"2026-03-21T10:08:00.000Z","uuid":"a8","sessionId":"sess-001","cwd":"/home/dev/myproject"}`

	// Assistant with Grep tool call.
	assistantGrepLine = `{"type":"assistant","message":{"role":"assistant","content":[{"type":"tool_use","name":"Grep","id":"tool-grep1","input":{"pattern":"validateToken","path":"/home/dev/myproject"}}]},"timestamp":"2026-03-21T10:09:00.000Z","uuid":"a9","sessionId":"sess-001","cwd":"/home/dev/myproject"}`

	// Assistant with Glob tool call.
	assistantGlobLine = `{"type":"assistant","message":{"role":"assistant","content":[{"type":"tool_use","name":"Glob","id":"tool-glob1","input":{"pattern":"**/*.go"}}]},"timestamp":"2026-03-21T10:10:00.000Z","uuid":"a10","sessionId":"sess-001","cwd":"/home/dev/myproject"}`

	// User message with tool_result containing an error.
	userToolErrorLine = `{"type":"user","message":{"role":"user","content":[{"type":"tool_result","tool_use_id":"tool-b1","is_error":true,"content":"Exit code 1\nFAIL myproject/auth [build failed]"}]},"isMeta":false,"timestamp":"2026-03-21T10:05:30.000Z","uuid":"u-err","sessionId":"sess-001","cwd":"/home/dev/myproject"}`

	// Progress line (should be largely skipped for activity extraction).
	progressLine = `{"type":"progress","data":{"type":"hook_progress","hookEvent":"PreToolUse","hookName":"PreToolUse:check","command":"bash check.sh"},"timestamp":"2026-03-21T10:01:30.000Z","uuid":"p1","sessionId":"sess-001","cwd":"/home/dev/myproject"}`

	// A second user request later in the session.
	userMsg2Line = `{"type":"user","message":{"role":"user","content":"Now add rate limiting to the endpoint"},"isMeta":false,"timestamp":"2026-03-21T10:15:00.000Z","uuid":"u2","sessionId":"sess-001","cwd":"/home/dev/myproject"}`

	// Assistant final response.
	assistantFinalLine = `{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"Done. The auth bug is fixed and rate limiting has been added."}]},"timestamp":"2026-03-21T10:16:00.000Z","uuid":"a-final","sessionId":"sess-001","cwd":"/home/dev/myproject"}`

	// Second edit to a different file.
	assistantEdit2Line = `{"type":"assistant","message":{"role":"assistant","content":[{"type":"tool_use","name":"Edit","id":"tool-e2","input":{"file_path":"/home/dev/myproject/middleware.go","old_string":"a","new_string":"b"}}]},"timestamp":"2026-03-21T10:14:00.000Z","uuid":"a-edit2","sessionId":"sess-001","cwd":"/home/dev/myproject"}`

	// Another edit to login.go (duplicate file).
	assistantEditDupLine = `{"type":"assistant","message":{"role":"assistant","content":[{"type":"tool_use","name":"Edit","id":"tool-e3","input":{"file_path":"/home/dev/myproject/login.go","old_string":"x","new_string":"y"}}]},"timestamp":"2026-03-21T10:13:00.000Z","uuid":"a-edit3","sessionId":"sess-001","cwd":"/home/dev/myproject"}`
)

// ---------------------------------------------------------------------------
// ExtractSummary tests
// ---------------------------------------------------------------------------

func TestExtractSummary_FullSession(t *testing.T) {
	dir := t.TempDir()
	path := writeTranscript(t, dir,
		userMsgLine,
		userMetaLine,
		assistantTextLine,
		assistantReadLine,
		assistantEditLine,
		assistantWriteLine,
		assistantBashLine,
		userToolErrorLine,
		assistantGitCommitLine,
		assistantGitPushLine,
		assistantPRLine,
		assistantGrepLine,
		assistantGlobLine,
		assistantEditDupLine,
		assistantEdit2Line,
		userMsg2Line,
		assistantFinalLine,
		progressLine,
	)

	summary, err := ExtractSummary(path)
	if err != nil {
		t.Fatalf("ExtractSummary() error = %v", err)
	}

	// Session metadata.
	if summary.SessionID != "sess-001" {
		t.Errorf("SessionID = %q, want %q", summary.SessionID, "sess-001")
	}
	if summary.Project != "myproject" {
		t.Errorf("Project = %q, want %q", summary.Project, "myproject")
	}

	// Counters.
	if summary.EditCount != 4 {
		t.Errorf("EditCount = %d, want 4 (Edit x3 + Write x1)", summary.EditCount)
	}
	if summary.BashCount != 4 {
		t.Errorf("BashCount = %d, want 4 (test + commit + push + pr)", summary.BashCount)
	}
	if summary.ReadCount != 1 {
		t.Errorf("ReadCount = %d, want 1", summary.ReadCount)
	}
	if summary.ErrorCount != 1 {
		t.Errorf("ErrorCount = %d, want 1", summary.ErrorCount)
	}

	// TotalMessages should count user (non-meta) + assistant messages.
	// 2 user (non-meta) + 11 assistant = 13
	// Actually count: user non-meta = u1, u-err(tool_result), u2 = 2 real text user msgs
	// assistant = textLine, readLine, editLine, writeLine, bashLine, gitCommit, gitPush, pr, grep, glob, editDup, edit2, finalLine = 13
	// We count user string messages + assistant messages
	if summary.TotalMessages < 10 {
		t.Errorf("TotalMessages = %d, want >= 10", summary.TotalMessages)
	}

	// FilesChanged should be deduplicated.
	wantFiles := map[string]bool{
		"/home/dev/myproject/login.go":     true,
		"/home/dev/myproject/auth_test.go": true,
		"/home/dev/myproject/middleware.go": true,
	}
	if len(summary.FilesChanged) != len(wantFiles) {
		t.Errorf("FilesChanged = %v (len %d), want %d unique files", summary.FilesChanged, len(summary.FilesChanged), len(wantFiles))
	}
	for _, f := range summary.FilesChanged {
		if !wantFiles[f] {
			t.Errorf("unexpected file in FilesChanged: %q", f)
		}
	}

	// Last user message should be the non-meta one.
	if summary.LastUserMsg != "Now add rate limiting to the endpoint" {
		t.Errorf("LastUserMsg = %q, want %q", summary.LastUserMsg, "Now add rate limiting to the endpoint")
	}

	// Last assistant message.
	if !strings.Contains(summary.LastAssistMsg, "auth bug is fixed") {
		t.Errorf("LastAssistMsg = %q, want it to contain 'auth bug is fixed'", summary.LastAssistMsg)
	}

	// Activities should include various types.
	typesSeen := make(map[string]bool)
	for _, a := range summary.Activities {
		typesSeen[a.Type] = true
	}
	for _, want := range []string{ActivityUserRequest, ActivityFileEdit, ActivityBashCommand, ActivityGitOp, ActivityError, ActivityToolCall} {
		if !typesSeen[want] {
			t.Errorf("Activities missing type %q, got types: %v", want, typesSeen)
		}
	}
}

func TestExtractSummary_MissingFile(t *testing.T) {
	summary, err := ExtractSummary("/nonexistent/path/transcript.jsonl")
	if err != nil {
		t.Fatalf("ExtractSummary() should not error on missing file, got %v", err)
	}
	if summary.SessionID != "" {
		t.Errorf("SessionID = %q, want empty", summary.SessionID)
	}
	if len(summary.Activities) != 0 {
		t.Errorf("Activities = %v, want empty", summary.Activities)
	}
}

func TestExtractSummary_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.jsonl")
	if err := os.WriteFile(path, []byte(""), 0644); err != nil {
		t.Fatal(err)
	}

	summary, err := ExtractSummary(path)
	if err != nil {
		t.Fatalf("ExtractSummary() error = %v", err)
	}
	if summary.TotalMessages != 0 {
		t.Errorf("TotalMessages = %d, want 0", summary.TotalMessages)
	}
}

func TestExtractSummary_MalformedLines(t *testing.T) {
	dir := t.TempDir()
	path := writeTranscript(t, dir,
		`{broken json`,
		userMsgLine,
		`not json at all`,
		assistantEditLine,
		``,
	)

	summary, err := ExtractSummary(path)
	if err != nil {
		t.Fatalf("ExtractSummary() error = %v", err)
	}
	// Should still extract from valid lines.
	if summary.EditCount != 1 {
		t.Errorf("EditCount = %d, want 1", summary.EditCount)
	}
	if summary.SessionID != "sess-001" {
		t.Errorf("SessionID = %q, want %q", summary.SessionID, "sess-001")
	}
}

func TestExtractSummary_GitOperations(t *testing.T) {
	dir := t.TempDir()
	path := writeTranscript(t, dir,
		assistantGitCommitLine,
		assistantGitPushLine,
		assistantPRLine,
	)

	summary, err := ExtractSummary(path)
	if err != nil {
		t.Fatalf("ExtractSummary() error = %v", err)
	}

	gitOps := 0
	for _, a := range summary.Activities {
		if a.Type == ActivityGitOp {
			gitOps++
		}
	}
	if gitOps != 3 {
		t.Errorf("git operations = %d, want 3 (commit, push, pr)", gitOps)
	}
}

func TestExtractSummary_MultipleToolsInOneMessage(t *testing.T) {
	// A single assistant message with multiple tool_use blocks.
	multiToolLine := `{"type":"assistant","message":{"role":"assistant","content":[{"type":"tool_use","name":"Read","id":"t1","input":{"file_path":"/a.go"}},{"type":"tool_use","name":"Read","id":"t2","input":{"file_path":"/b.go"}},{"type":"tool_use","name":"Edit","id":"t3","input":{"file_path":"/a.go","old_string":"x","new_string":"y"}}]},"timestamp":"2026-03-21T10:00:00.000Z","uuid":"multi","sessionId":"sess-002","cwd":"/proj"}`

	dir := t.TempDir()
	path := writeTranscript(t, dir, multiToolLine)

	summary, err := ExtractSummary(path)
	if err != nil {
		t.Fatalf("ExtractSummary() error = %v", err)
	}

	if summary.ReadCount != 2 {
		t.Errorf("ReadCount = %d, want 2", summary.ReadCount)
	}
	if summary.EditCount != 1 {
		t.Errorf("EditCount = %d, want 1", summary.EditCount)
	}
}

// ---------------------------------------------------------------------------
// ExtractRecent tests
// ---------------------------------------------------------------------------

func TestExtractRecent_FiltersByTime(t *testing.T) {
	dir := t.TempDir()
	path := writeTranscript(t, dir,
		userMsgLine,          // 10:00
		assistantEditLine,    // 10:03
		assistantBashLine,    // 10:05
		userMsg2Line,         // 10:15
		assistantEdit2Line,   // 10:14
		assistantFinalLine,   // 10:16
	)

	// Only get activity after 10:10.
	since := mustParseTime("2026-03-21T10:10:00.000Z")
	summary, err := ExtractRecent(path, since)
	if err != nil {
		t.Fatalf("ExtractRecent() error = %v", err)
	}

	// Should only see edit2 (10:14), userMsg2 (10:15), finalLine (10:16).
	if summary.EditCount != 1 {
		t.Errorf("EditCount = %d, want 1 (only edit2 after 10:10)", summary.EditCount)
	}
	if summary.BashCount != 0 {
		t.Errorf("BashCount = %d, want 0 (bash was at 10:05)", summary.BashCount)
	}
	if summary.LastUserMsg != "Now add rate limiting to the endpoint" {
		t.Errorf("LastUserMsg = %q, want rate limiting message", summary.LastUserMsg)
	}
}

func TestExtractRecent_NoActivityAfterSince(t *testing.T) {
	dir := t.TempDir()
	path := writeTranscript(t, dir, userMsgLine) // 10:00

	since := mustParseTime("2026-03-21T11:00:00.000Z")
	summary, err := ExtractRecent(path, since)
	if err != nil {
		t.Fatalf("ExtractRecent() error = %v", err)
	}
	if summary.TotalMessages != 0 {
		t.Errorf("TotalMessages = %d, want 0", summary.TotalMessages)
	}
}

func TestExtractRecent_MissingFile(t *testing.T) {
	since := mustParseTime("2026-03-21T10:00:00.000Z")
	summary, err := ExtractRecent("/nonexistent.jsonl", since)
	if err != nil {
		t.Fatalf("ExtractRecent() should not error on missing file, got %v", err)
	}
	if len(summary.Activities) != 0 {
		t.Errorf("Activities = %v, want empty", summary.Activities)
	}
}

// ---------------------------------------------------------------------------
// FormatBrief tests
// ---------------------------------------------------------------------------

func TestFormatBrief_TypicalSession(t *testing.T) {
	summary := SessionSummary{
		SessionID: "abc-123",
		Project:   "myproject",
		Activities: []Activity{
			{Timestamp: mustParseTime("2026-03-21T10:00:00.000Z"), Type: ActivityFileEdit, Description: "Edited login.go"},
			{Timestamp: mustParseTime("2026-03-21T10:01:00.000Z"), Type: ActivityBashCommand, Description: "Ran go test ./..."},
		},
		FilesChanged:  []string{"/home/dev/myproject/login.go", "/home/dev/myproject/auth.go"},
		LastUserMsg:   "Fix the auth bug",
		LastAssistMsg: "Done, the auth bug is fixed.",
		TotalMessages: 15,
		ErrorCount:    0,
		EditCount:     3,
		BashCount:     2,
		ReadCount:     5,
	}

	brief := FormatBrief(summary)

	// Should mention project name.
	if !strings.Contains(brief, "myproject") {
		t.Errorf("brief should contain project name, got: %s", brief)
	}
	// Should mention file count.
	if !strings.Contains(brief, "2 files") {
		t.Errorf("brief should mention '2 files', got: %s", brief)
	}
	// Should mention edit count.
	if !strings.Contains(brief, "3 edits") {
		t.Errorf("brief should mention '3 edits', got: %s", brief)
	}
	// Should not mention errors when count is 0.
	if strings.Contains(brief, "error") {
		t.Errorf("brief should not mention errors when ErrorCount=0, got: %s", brief)
	}
}

func TestFormatBrief_WithErrors(t *testing.T) {
	summary := SessionSummary{
		SessionID:  "abc-123",
		Project:    "myproject",
		ErrorCount: 3,
		EditCount:  1,
	}

	brief := FormatBrief(summary)
	if !strings.Contains(brief, "3 errors") {
		t.Errorf("brief should mention '3 errors', got: %s", brief)
	}
}

func TestFormatBrief_EmptySummary(t *testing.T) {
	summary := SessionSummary{}
	brief := FormatBrief(summary)
	// Should not panic and should produce something.
	if brief == "" {
		t.Error("FormatBrief() returned empty string for empty summary")
	}
}

// ---------------------------------------------------------------------------
// Edge cases
// ---------------------------------------------------------------------------

func TestExtractSummary_SkipsBase64Content(t *testing.T) {
	// Simulate a tool_result with base64 image data (very long line).
	// The parser should not choke on this.
	longContent := strings.Repeat("ABCDEFGH", 100000) // 800KB of fake base64
	base64Line := `{"type":"user","message":{"role":"user","content":[{"type":"tool_result","tool_use_id":"t1","content":[{"type":"image","source":{"data":"` + longContent + `"}}]}]},"timestamp":"2026-03-21T10:00:00.000Z","uuid":"big","sessionId":"sess-003","cwd":"/proj"}`

	dir := t.TempDir()
	path := writeTranscript(t, dir,
		base64Line,
		assistantEditLine,
	)

	summary, err := ExtractSummary(path)
	if err != nil {
		t.Fatalf("ExtractSummary() error = %v", err)
	}
	if summary.EditCount != 1 {
		t.Errorf("EditCount = %d, want 1", summary.EditCount)
	}
}

func TestExtractSummary_ToolCallTypes(t *testing.T) {
	dir := t.TempDir()
	path := writeTranscript(t, dir,
		assistantGrepLine,
		assistantGlobLine,
	)

	summary, err := ExtractSummary(path)
	if err != nil {
		t.Fatalf("ExtractSummary() error = %v", err)
	}

	// Grep and Glob are tracked as tool_call activities.
	toolCalls := 0
	for _, a := range summary.Activities {
		if a.Type == ActivityToolCall {
			toolCalls++
		}
	}
	if toolCalls != 2 {
		t.Errorf("tool_call activities = %d, want 2 (Grep + Glob)", toolCalls)
	}
}

func TestExtractSummary_UserContentAsList(t *testing.T) {
	// User message with content as a list (tool_result blocks).
	// Should not count as a "user request".
	dir := t.TempDir()
	path := writeTranscript(t, dir, userToolErrorLine)

	summary, err := ExtractSummary(path)
	if err != nil {
		t.Fatalf("ExtractSummary() error = %v", err)
	}

	// Tool error results should bump ErrorCount.
	if summary.ErrorCount != 1 {
		t.Errorf("ErrorCount = %d, want 1", summary.ErrorCount)
	}

	// Should not set LastUserMsg from a tool_result-only message.
	if summary.LastUserMsg != "" {
		t.Errorf("LastUserMsg = %q, want empty (no real user text)", summary.LastUserMsg)
	}
}

func TestActivity_Constants(t *testing.T) {
	// Verify the activity type constants are defined.
	types := []string{
		ActivityUserRequest,
		ActivityFileEdit,
		ActivityBashCommand,
		ActivityGitOp,
		ActivityError,
		ActivityToolCall,
	}
	for _, typ := range types {
		if typ == "" {
			t.Errorf("activity type constant is empty")
		}
	}
}
