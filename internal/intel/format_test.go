package intel

import (
	"strings"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// FormatDailyBrief executive summary tests
// ---------------------------------------------------------------------------

func TestFormatDailyBrief_HasExecutiveSummary(t *testing.T) {
	brief := DailyBrief{
		GeneratedAt:    time.Date(2026, 3, 21, 9, 0, 0, 0, time.Local),
		Period:         "Last 24 hours",
		TotalSessions:  5,
		ActiveSessions: 3,
		TotalEdits:     20,
		TotalCommands:  8,
		Projects: []ProjectBrief{
			{Name: "Alpha", SessionCount: 3, ActiveCount: 2, TotalEdits: 15, TotalCommands: 5},
			{Name: "Beta", SessionCount: 2, ActiveCount: 1, TotalEdits: 5, TotalCommands: 3},
		},
	}

	output := FormatDailyBrief(brief)

	// Should have a one-line executive summary near the top.
	if !strings.Contains(output, "2 projects") {
		t.Errorf("executive summary should mention project count, got:\n%s", output)
	}
	if !strings.Contains(output, "20 edits") {
		t.Errorf("executive summary should mention edit count, got:\n%s", output)
	}
	if !strings.Contains(output, "8 commands") {
		t.Errorf("executive summary should mention command count, got:\n%s", output)
	}
}

func TestFormatDailyBrief_NoRawToolNames(t *testing.T) {
	brief := DailyBrief{
		GeneratedAt:    time.Date(2026, 3, 21, 9, 0, 0, 0, time.Local),
		Period:         "Last 24 hours",
		TotalSessions:  1,
		ActiveSessions: 1,
		Projects: []ProjectBrief{
			{
				Name:         "MyApp",
				SessionCount: 1,
				ActiveCount:  1,
				KeyActivities: []string{
					"Fix the auth bug in login.go",
					"Edited 5 files (login.go, auth.go, +3 more)",
					"Ran 2 commands",
				},
			},
		},
	}

	output := FormatDailyBrief(brief)

	// Should NOT contain raw tool names like "Used Grep", "Used Glob".
	rawToolNames := []string{"Used Grep", "Used Glob", "Used ToolSearch", "Used Agent"}
	for _, raw := range rawToolNames {
		if strings.Contains(output, raw) {
			t.Errorf("output should not contain raw tool name %q, got:\n%s", raw, output)
		}
	}
}

func TestFormatDailyBrief_ProjectUsesCondensedActivities(t *testing.T) {
	brief := DailyBrief{
		GeneratedAt:    time.Date(2026, 3, 21, 9, 0, 0, 0, time.Local),
		Period:         "Last 24 hours",
		TotalSessions:  1,
		ActiveSessions: 1,
		Projects: []ProjectBrief{
			{
				Name:          "MyApp",
				SessionCount:  1,
				ActiveCount:   1,
				TotalEdits:    5,
				TotalCommands: 3,
				KeyActivities: []string{
					"Fix the auth bug",
					"Edited 5 files",
				},
			},
		},
	}

	output := FormatDailyBrief(brief)

	// Key activities should appear as bullet points.
	if !strings.Contains(output, "Fix the auth bug") {
		t.Errorf("output should contain condensed activity, got:\n%s", output)
	}
}

// ---------------------------------------------------------------------------
// FormatBrief (single session) improvement tests
// ---------------------------------------------------------------------------

func TestFormatBrief_UsesCondensedActivities(t *testing.T) {
	summary := SessionSummary{
		SessionID: "abc-123",
		Project:   "myproject",
		Activities: []Activity{
			{Type: ActivityUserRequest, Description: "Fix the auth bug in login.go"},
			{Type: ActivityToolCall, Description: "Used Grep"},
			{Type: ActivityToolCall, Description: "Used Glob"},
			{Type: ActivityFileEdit, Description: "Edited login.go"},
			{Type: ActivityFileEdit, Description: "Edited login.go"},
			{Type: ActivityFileEdit, Description: "Wrote auth_test.go"},
			{Type: ActivityBashCommand, Description: "Run tests"},
			{Type: ActivityGitOp, Description: "Git commit"},
			{Type: ActivityGitOp, Description: "Git push"},
		},
		FilesChanged:  []string{"/home/dev/myproject/login.go", "/home/dev/myproject/auth_test.go"},
		LastUserMsg:   "Fix the auth bug",
		LastAssistMsg: "Done.",
		TotalMessages: 10,
		EditCount:     3,
		BashCount:     1,
		ReadCount:     2,
	}

	output := FormatBrief(summary)

	// Should NOT contain raw "[tool_call] Used Grep" style lines.
	if strings.Contains(output, "[tool_call]") {
		t.Errorf("FormatBrief should not contain raw [tool_call] type labels, got:\n%s", output)
	}
	if strings.Contains(output, "Used Grep") {
		t.Errorf("FormatBrief should not contain raw 'Used Grep', got:\n%s", output)
	}
	if strings.Contains(output, "Used Glob") {
		t.Errorf("FormatBrief should not contain raw 'Used Glob', got:\n%s", output)
	}

	// Should contain the user request verbatim.
	if !strings.Contains(output, "Fix the auth bug") {
		t.Errorf("FormatBrief should show user requests, got:\n%s", output)
	}
}

func TestFormatBrief_NoRawActivityTypes(t *testing.T) {
	summary := SessionSummary{
		SessionID: "abc-123",
		Project:   "myproject",
		Activities: []Activity{
			{Type: ActivityUserRequest, Description: "Do something"},
			{Type: ActivityFileEdit, Description: "Edited foo.go"},
			{Type: ActivityBashCommand, Description: "Run tests"},
		},
		EditCount: 1,
		BashCount: 1,
	}

	output := FormatBrief(summary)

	// Should not contain raw type labels like [user_request], [file_edit], etc.
	rawTypes := []string{"[user_request]", "[file_edit]", "[bash_command]", "[git_op]", "[error]", "[tool_call]"}
	for _, raw := range rawTypes {
		if strings.Contains(output, raw) {
			t.Errorf("FormatBrief should not contain raw type %q, got:\n%s", raw, output)
		}
	}
}
