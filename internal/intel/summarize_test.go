package intel

import (
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// SummarizeActivities tests
// ---------------------------------------------------------------------------

func TestSummarizeActivities_GroupsEdits(t *testing.T) {
	activities := []Activity{
		{Type: ActivityFileEdit, Description: "Edited login.go"},
		{Type: ActivityFileEdit, Description: "Edited login.go"},
		{Type: ActivityFileEdit, Description: "Wrote auth_test.go"},
		{Type: ActivityFileEdit, Description: "Edited middleware.go"},
	}

	result := SummarizeActivities(activities)

	// Should condense into something like "Edited 4 files (login.go, auth_test.go, middleware.go)"
	found := false
	for _, s := range result {
		if strings.Contains(s, "4 files") || strings.Contains(s, "4 file") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected a grouped edit summary mentioning 4 files, got: %v", result)
	}
}

func TestSummarizeActivities_GroupsBashCommands(t *testing.T) {
	activities := []Activity{
		{Type: ActivityBashCommand, Description: "Run tests"},
		{Type: ActivityBashCommand, Description: "Install dependencies"},
		{Type: ActivityBashCommand, Description: "Build project"},
	}

	result := SummarizeActivities(activities)

	found := false
	for _, s := range result {
		if strings.Contains(s, "3 commands") || strings.Contains(s, "3 command") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected a grouped command summary mentioning 3 commands, got: %v", result)
	}
}

func TestSummarizeActivities_GroupsGitOps(t *testing.T) {
	activities := []Activity{
		{Type: ActivityGitOp, Description: "Git commit"},
		{Type: ActivityGitOp, Description: "Git push"},
		{Type: ActivityGitOp, Description: "Create PR"},
	}

	result := SummarizeActivities(activities)

	found := false
	for _, s := range result {
		if strings.Contains(s, "Git") || strings.Contains(s, "git") || strings.Contains(s, "commit") || strings.Contains(s, "Committed") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected a git ops summary, got: %v", result)
	}
}

func TestSummarizeActivities_UserRequestsAppearVerbatim(t *testing.T) {
	activities := []Activity{
		{Type: ActivityUserRequest, Description: "Fix the auth bug in login.go"},
		{Type: ActivityFileEdit, Description: "Edited login.go"},
		{Type: ActivityUserRequest, Description: "Now add rate limiting"},
	}

	result := SummarizeActivities(activities)

	// User requests should appear as-is (they are the "what the user asked for").
	foundAuth := false
	foundRate := false
	for _, s := range result {
		if strings.Contains(s, "Fix the auth bug") {
			foundAuth = true
		}
		if strings.Contains(s, "rate limiting") {
			foundRate = true
		}
	}
	if !foundAuth {
		t.Errorf("expected user request 'Fix the auth bug' to appear, got: %v", result)
	}
	if !foundRate {
		t.Errorf("expected user request 'rate limiting' to appear, got: %v", result)
	}
}

func TestSummarizeActivities_OmitsToolCalls(t *testing.T) {
	activities := []Activity{
		{Type: ActivityToolCall, Description: "Used Grep"},
		{Type: ActivityToolCall, Description: "Used Glob"},
		{Type: ActivityToolCall, Description: "Used ToolSearch"},
		{Type: ActivityUserRequest, Description: "Find the bug"},
	}

	result := SummarizeActivities(activities)

	// Tool calls (Grep, Glob, etc.) should NOT appear as individual items.
	for _, s := range result {
		if strings.Contains(s, "Used Grep") || strings.Contains(s, "Used Glob") || strings.Contains(s, "Used ToolSearch") {
			t.Errorf("raw tool call should not appear in summary, got: %v", result)
		}
	}
}

func TestSummarizeActivities_LimitsToFive(t *testing.T) {
	activities := []Activity{
		{Type: ActivityUserRequest, Description: "Task 1"},
		{Type: ActivityUserRequest, Description: "Task 2"},
		{Type: ActivityUserRequest, Description: "Task 3"},
		{Type: ActivityUserRequest, Description: "Task 4"},
		{Type: ActivityUserRequest, Description: "Task 5"},
		{Type: ActivityUserRequest, Description: "Task 6"},
		{Type: ActivityUserRequest, Description: "Task 7"},
		{Type: ActivityFileEdit, Description: "Edited a.go"},
		{Type: ActivityBashCommand, Description: "Run tests"},
		{Type: ActivityGitOp, Description: "Git commit"},
	}

	result := SummarizeActivities(activities)

	if len(result) > 5 {
		t.Errorf("SummarizeActivities should return at most 5 items, got %d: %v", len(result), result)
	}
}

func TestSummarizeActivities_Empty(t *testing.T) {
	result := SummarizeActivities(nil)
	if len(result) != 0 {
		t.Errorf("SummarizeActivities(nil) should return empty, got: %v", result)
	}
}

func TestSummarizeActivities_ErrorsIncluded(t *testing.T) {
	activities := []Activity{
		{Type: ActivityError, Description: "Tool returned error"},
		{Type: ActivityError, Description: "Tool returned error"},
		{Type: ActivityError, Description: "Tool returned error"},
	}

	result := SummarizeActivities(activities)

	found := false
	for _, s := range result {
		if strings.Contains(s, "3 errors") || strings.Contains(s, "error") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected error summary, got: %v", result)
	}
}

func TestSummarizeActivities_MixedRealSession(t *testing.T) {
	// Simulate a realistic session: user request, many edits, some bash, git ops, tool calls.
	activities := []Activity{
		{Type: ActivityUserRequest, Description: "Fix the auth bug in login.go"},
		{Type: ActivityToolCall, Description: "Used Grep"},
		{Type: ActivityToolCall, Description: "Used Glob"},
		{Type: ActivityFileEdit, Description: "Edited login.go"},
		{Type: ActivityFileEdit, Description: "Edited login.go"},
		{Type: ActivityFileEdit, Description: "Wrote auth_test.go"},
		{Type: ActivityBashCommand, Description: "Run tests"},
		{Type: ActivityError, Description: "Tool returned error"},
		{Type: ActivityBashCommand, Description: "Run tests again"},
		{Type: ActivityFileEdit, Description: "Edited middleware.go"},
		{Type: ActivityGitOp, Description: "Git commit"},
		{Type: ActivityGitOp, Description: "Git push"},
		{Type: ActivityGitOp, Description: "Create PR"},
		{Type: ActivityUserRequest, Description: "Now add rate limiting"},
		{Type: ActivityToolCall, Description: "Used Grep"},
		{Type: ActivityFileEdit, Description: "Edited handler.go"},
	}

	result := SummarizeActivities(activities)

	// Must contain user requests.
	foundUserReq := false
	for _, s := range result {
		if strings.Contains(s, "Fix the auth bug") || strings.Contains(s, "rate limiting") {
			foundUserReq = true
			break
		}
	}
	if !foundUserReq {
		t.Errorf("expected user requests in summary, got: %v", result)
	}

	// Must NOT contain raw tool calls.
	for _, s := range result {
		if strings.Contains(s, "Used Grep") || strings.Contains(s, "Used Glob") {
			t.Errorf("raw tool calls should not appear, got: %v", result)
		}
	}

	// At most 5 items.
	if len(result) > 5 {
		t.Errorf("should have at most 5 items, got %d: %v", len(result), result)
	}
}

func TestSummarizeActivities_SingleCommand(t *testing.T) {
	activities := []Activity{
		{Type: ActivityBashCommand, Description: "Run tests"},
	}

	result := SummarizeActivities(activities)

	if len(result) != 1 {
		t.Fatalf("expected 1 result, got %d: %v", len(result), result)
	}
	// Single command should show the description.
	if !strings.Contains(result[0], "Run tests") {
		t.Errorf("single command should include description, got: %s", result[0])
	}
}
