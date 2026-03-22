package components

import (
	"strings"
	"testing"
	"time"
)

func stateIconStub(state string) string {
	switch state {
	case "busy":
		return "B"
	case "idle":
		return "I"
	case "dead":
		return "D"
	case "waiting":
		return "W"
	case "paused":
		return "P"
	default:
		return "?"
	}
}

func makeSessions() []TableSession {
	now := time.Now()
	return []TableSession{
		{ID: "sess_aaa", PID: 1001, State: "busy", Project: "ProjA", StartedAt: now.Add(-10 * time.Minute), WorkingOn: "task A1"},
		{ID: "sess_bbb", PID: 1002, State: "idle", Project: "ProjA", StartedAt: now.Add(-1 * time.Hour), WorkingOn: ""},
		{ID: "sess_ccc", PID: 1003, State: "busy", Project: "ProjB", StartedAt: now.Add(-5 * time.Minute), WorkingOn: "task B1"},
		{ID: "sess_ddd", PID: 1004, State: "dead", Project: "ProjB", StartedAt: now.Add(-2 * time.Hour), WorkingOn: ""},
	}
}

func TestNewTable(t *testing.T) {
	tbl := NewTable(stateIconStub)
	if !tbl.ProjectView {
		t.Error("default ProjectView should be true")
	}
	if tbl.Cursor != 0 {
		t.Errorf("default Cursor should be 0, got %d", tbl.Cursor)
	}
}

func TestTableSetSessions(t *testing.T) {
	tbl := NewTable(stateIconStub)
	sessions := makeSessions()
	tbl.SetSessions(sessions)

	if len(tbl.Sessions) != 4 {
		t.Errorf("expected 4 sessions, got %d", len(tbl.Sessions))
	}
}

func TestTableNavigation(t *testing.T) {
	tbl := NewTable(stateIconStub)
	tbl.SetSessions(makeSessions())
	tbl.SetSize(80, 20)

	// Start at 0.
	if tbl.Cursor != 0 {
		t.Fatalf("expected cursor at 0, got %d", tbl.Cursor)
	}

	// Move down.
	tbl.MoveDown()
	if tbl.Cursor != 1 {
		t.Errorf("after MoveDown, expected cursor at 1, got %d", tbl.Cursor)
	}

	// Move down to end.
	tbl.MoveDown()
	tbl.MoveDown()
	tbl.MoveDown() // past end
	if tbl.Cursor != 3 {
		t.Errorf("cursor should clamp at 3, got %d", tbl.Cursor)
	}

	// Move up.
	tbl.MoveUp()
	if tbl.Cursor != 2 {
		t.Errorf("after MoveUp, expected cursor at 2, got %d", tbl.Cursor)
	}

	// Move up past beginning.
	tbl.MoveUp()
	tbl.MoveUp()
	tbl.MoveUp() // past beginning
	if tbl.Cursor != 0 {
		t.Errorf("cursor should clamp at 0, got %d", tbl.Cursor)
	}
}

func TestSelectedSession(t *testing.T) {
	tbl := NewTable(stateIconStub)
	sessions := makeSessions()
	tbl.SetSessions(sessions)

	sel := tbl.SelectedSession()
	if sel == nil {
		t.Fatal("SelectedSession should not be nil")
	}
	if sel.ID != "sess_aaa" {
		t.Errorf("expected first session, got %q", sel.ID)
	}

	tbl.MoveDown()
	sel = tbl.SelectedSession()
	if sel.ID != "sess_bbb" {
		t.Errorf("expected second session, got %q", sel.ID)
	}
}

func TestSelectedSessionEmpty(t *testing.T) {
	tbl := NewTable(stateIconStub)
	sel := tbl.SelectedSession()
	if sel != nil {
		t.Error("SelectedSession should be nil for empty table")
	}
}

func TestTableViewGrouped(t *testing.T) {
	tbl := NewTable(stateIconStub)
	tbl.SetSessions(makeSessions())
	tbl.SetSize(100, 20)
	tbl.ProjectView = true

	view := tbl.View()

	// Should contain project headers.
	if !strings.Contains(view, "ProjA") {
		t.Error("grouped view should contain 'ProjA'")
	}
	if !strings.Contains(view, "ProjB") {
		t.Error("grouped view should contain 'ProjB'")
	}
}

func TestTableViewFlat(t *testing.T) {
	tbl := NewTable(stateIconStub)
	tbl.SetSessions(makeSessions())
	tbl.SetSize(100, 20)
	tbl.ProjectView = false

	view := tbl.View()

	// Should contain session PIDs.
	if !strings.Contains(view, "1001") {
		t.Error("flat view should contain PID 1001")
	}
	if !strings.Contains(view, "1004") {
		t.Error("flat view should contain PID 1004")
	}
}

func TestTableViewEmpty(t *testing.T) {
	tbl := NewTable(stateIconStub)
	tbl.SetSize(80, 20)

	view := tbl.View()
	if !strings.Contains(view, "No sessions found") {
		t.Error("empty table should show 'No sessions found' message")
	}
}

func TestTableCursorClampsOnSetSessions(t *testing.T) {
	tbl := NewTable(stateIconStub)
	tbl.SetSessions(makeSessions())
	tbl.Cursor = 3 // last item

	// Now set fewer sessions.
	tbl.SetSessions(makeSessions()[:2])
	if tbl.Cursor >= len(tbl.Sessions) {
		t.Errorf("cursor should be clamped, got %d with %d sessions", tbl.Cursor, len(tbl.Sessions))
	}
}

func TestShortRelativeTime(t *testing.T) {
	tests := []struct {
		ago  time.Duration
		want string
	}{
		{30 * time.Second, "<1m"},
		{5 * time.Minute, "5m"},
		{90 * time.Minute, "1h"},
		{25 * time.Hour, "1d"},
	}

	for _, tt := range tests {
		got := shortRelativeTime(time.Now().Add(-tt.ago))
		if got != tt.want {
			t.Errorf("shortRelativeTime(-%v) = %q, want %q", tt.ago, got, tt.want)
		}
	}
}

func TestPadRight(t *testing.T) {
	tests := []struct {
		s     string
		width int
		want  string
	}{
		{"abc", 6, "abc   "},
		{"abcdef", 4, "abcd"},
		{"ab", 2, "ab"},
	}
	for _, tt := range tests {
		got := padRight(tt.s, tt.width)
		if got != tt.want {
			t.Errorf("padRight(%q, %d) = %q, want %q", tt.s, tt.width, got, tt.want)
		}
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		s      string
		maxLen int
		want   string
	}{
		{"hello world", 20, "hello world"},
		{"hello world", 8, "hello..."},
		{"hello world", 3, "hel"},
		{"hello", 0, ""},
	}
	for _, tt := range tests {
		got := truncate(tt.s, tt.maxLen)
		if got != tt.want {
			t.Errorf("truncate(%q, %d) = %q, want %q", tt.s, tt.maxLen, got, tt.want)
		}
	}
}
