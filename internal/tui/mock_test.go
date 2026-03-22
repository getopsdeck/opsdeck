package tui

import (
	"testing"
	"time"
)

func TestMockSessions(t *testing.T) {
	sessions := mockSessions()
	if len(sessions) == 0 {
		t.Fatal("mockSessions() returned empty slice")
	}

	// Verify all sessions have required fields.
	for i, s := range sessions {
		if s.ID == "" {
			t.Errorf("session %d has empty ID", i)
		}
		if s.PID == 0 {
			t.Errorf("session %d has zero PID", i)
		}
		if s.State == "" {
			t.Errorf("session %d has empty State", i)
		}
		if s.Project == "" {
			t.Errorf("session %d has empty Project", i)
		}
		if s.StartedAt.IsZero() {
			t.Errorf("session %d has zero StartedAt", i)
		}
	}
}

func TestGroupByProject(t *testing.T) {
	sessions := mockSessions()
	order, groups := groupByProject(sessions)

	if len(order) == 0 {
		t.Fatal("groupByProject returned no projects")
	}

	// Every session must appear in exactly one group.
	total := 0
	for _, name := range order {
		g, ok := groups[name]
		if !ok {
			t.Errorf("project %q in order but not in groups map", name)
			continue
		}
		total += len(g)
	}

	if total != len(sessions) {
		t.Errorf("grouped sessions count %d != original %d", total, len(sessions))
	}

	// Order should be stable (first-seen order).
	if order[0] != sessions[0].Project {
		t.Errorf("first project should be %q, got %q", sessions[0].Project, order[0])
	}
}

func TestCountByState(t *testing.T) {
	sessions := mockSessions()
	counts := countByState(sessions)

	total := 0
	for _, c := range counts {
		total += c
	}

	if total != len(sessions) {
		t.Errorf("countByState total %d != session count %d", total, len(sessions))
	}

	// We know mock data has at least one busy and one dead session.
	if counts["busy"] == 0 {
		t.Error("expected at least one busy session in mock data")
	}
	if counts["dead"] == 0 {
		t.Error("expected at least one dead session in mock data")
	}
}

func TestFilterSessionsByState(t *testing.T) {
	sessions := mockSessions()

	busy := filterSessions(sessions, "busy", "")
	for _, s := range busy {
		if s.State != "busy" {
			t.Errorf("filtered for 'busy' but got state %q", s.State)
		}
	}

	if len(busy) == 0 {
		t.Error("expected at least one busy session in mock data")
	}

	// Empty filter returns all.
	all := filterSessions(sessions, "", "")
	if len(all) != len(sessions) {
		t.Errorf("empty filter should return all sessions: got %d, want %d", len(all), len(sessions))
	}
}

func TestFilterSessionsBySearch(t *testing.T) {
	sessions := mockSessions()

	// Search by project name.
	results := filterSessions(sessions, "", "OpsDeck")
	if len(results) == 0 {
		t.Error("search for 'OpsDeck' should return results")
	}
	for _, s := range results {
		if s.Project != "OpsDeck" {
			// It must match somewhere else in the session
			if !matchesSearch(s, "OpsDeck") {
				t.Errorf("session %s should match 'OpsDeck'", s.ID)
			}
		}
	}

	// Case-insensitive search.
	upper := filterSessions(sessions, "", "OPSDECK")
	if len(upper) != len(results) {
		t.Errorf("case-insensitive search should match same count: got %d, want %d", len(upper), len(results))
	}
}

func TestFilterCombined(t *testing.T) {
	sessions := mockSessions()

	// Filter by state AND search.
	results := filterSessions(sessions, "busy", "OpsDeck")
	for _, s := range results {
		if s.State != "busy" {
			t.Errorf("expected state 'busy', got %q", s.State)
		}
		if !matchesSearch(s, "OpsDeck") {
			t.Errorf("session %s should match 'OpsDeck'", s.ID)
		}
	}
}

func TestMatchesSearch(t *testing.T) {
	s := Session{
		ID:        "sess_abc123",
		PID:       12345,
		State:     "busy",
		Project:   "MyProject",
		StartedAt: time.Now(),
		WorkingOn: "implementing feature",
		LastLine:  "Done writing code",
	}

	tests := []struct {
		term string
		want bool
	}{
		{"abc123", true},
		{"12345", true},
		{"MyProject", true},
		{"myproject", true}, // case insensitive
		{"implementing", true},
		{"writing code", true},
		{"nonexistent", false},
		{"", true}, // empty matches everything via filterSessions short-circuit
	}

	for _, tt := range tests {
		if tt.term == "" {
			continue // empty is handled by filterSessions, not matchesSearch
		}
		got := matchesSearch(s, tt.term)
		if got != tt.want {
			t.Errorf("matchesSearch(%q) = %v, want %v", tt.term, got, tt.want)
		}
	}
}

func TestStateIcon(t *testing.T) {
	states := []string{"waiting", "busy", "idle", "dead", "paused", "unknown"}
	for _, state := range states {
		icon := StateIcon(state)
		if icon == "" {
			t.Errorf("StateIcon(%q) returned empty string", state)
		}
	}
}

func TestItoa(t *testing.T) {
	tests := []struct {
		n    int
		want string
	}{
		{0, "0"},
		{1, "1"},
		{42, "42"},
		{-1, "-1"},
		{1000, "1000"},
	}
	for _, tt := range tests {
		got := itoa(tt.n)
		if got != tt.want {
			t.Errorf("itoa(%d) = %q, want %q", tt.n, got, tt.want)
		}
	}
}
