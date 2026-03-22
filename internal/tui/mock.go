package tui

import (
	"strings"
	"time"
)

// Session represents a Claude Code session discovered on the system.
type Session struct {
	ID        string
	PID       int
	State     string
	Project   string
	StartedAt time.Time
	WorkingOn string
	LastLine  string // last line of transcript (preview)
}

// mockSessions returns hardcoded sessions for testing the TUI layout.
func mockSessions() []Session {
	now := time.Now()
	return []Session{
		{
			ID:        "sess_a1b2c3d4e5f6",
			PID:       12401,
			State:     "busy",
			Project:   "OpsDeck",
			StartedAt: now.Add(-47 * time.Minute),
			WorkingOn: "implementing TUI skeleton with Bubble Tea v2",
			LastLine:  "> Writing internal/tui/styles.go...",
		},
		{
			ID:        "sess_f7e8d9c0b1a2",
			PID:       12533,
			State:     "busy",
			Project:   "OpsDeck",
			StartedAt: now.Add(-12 * time.Minute),
			WorkingOn: "running test suite for discovery module",
			LastLine:  "PASS: TestDiscoverSessions (0.34s)",
		},
		{
			ID:        "sess_1122334455aa",
			PID:       11987,
			State:     "idle",
			Project:   "OpsDeck",
			StartedAt: now.Add(-2 * time.Hour),
			WorkingOn: "",
			LastLine:  "Waiting for input...",
		},
		{
			ID:        "sess_aabbccddee01",
			PID:       13201,
			State:     "busy",
			Project:   "sw360-frontend",
			StartedAt: now.Add(-8 * time.Minute),
			WorkingOn: "fixing SBOM import validation",
			LastLine:  "> Editing src/components/ImportDialog.tsx",
		},
		{
			ID:        "sess_99887766ff02",
			PID:       13045,
			State:     "waiting",
			Project:   "sw360-frontend",
			StartedAt: now.Add(-22 * time.Minute),
			WorkingOn: "pending user approval for git push",
			LastLine:  "? Push to origin/fix-sbom-parser? (y/n)",
		},
		{
			ID:        "sess_deadbeef0003",
			PID:       10234,
			State:     "dead",
			Project:   "sw360-frontend",
			StartedAt: now.Add(-4 * time.Hour),
			WorkingOn: "",
			LastLine:  "Error: connection reset by peer",
		},
		{
			ID:        "sess_cafe4321fab0",
			PID:       14001,
			State:     "busy",
			Project:   "claude-opc",
			StartedAt: now.Add(-3 * time.Minute),
			WorkingOn: "implementing memory recall with vector search",
			LastLine:  "> Running PYTHONPATH=. uv run pytest tests/ -v",
		},
		{
			ID:        "sess_5678abcd0001",
			PID:       14150,
			State:     "idle",
			Project:   "claude-opc",
			StartedAt: now.Add(-35 * time.Minute),
			WorkingOn: "",
			LastLine:  "Done. 14 files changed, 238 insertions(+), 42 deletions(-)",
		},
		{
			ID:        "sess_beef1234dead",
			PID:       9871,
			State:     "dead",
			Project:   "claude-opc",
			StartedAt: now.Add(-6 * time.Hour),
			WorkingOn: "",
			LastLine:  "Session terminated: timeout after 30m idle",
		},
		{
			ID:        "sess_0000aaaa1111",
			PID:       14320,
			State:     "waiting",
			Project:   "dotfiles",
			StartedAt: now.Add(-5 * time.Minute),
			WorkingOn: "waiting for confirmation to modify .zshrc",
			LastLine:  "? Overwrite ~/.zshrc with new config? (y/n)",
		},
		{
			ID:        "sess_ffff2222bbbb",
			PID:       14455,
			State:     "paused",
			Project:   "dotfiles",
			StartedAt: now.Add(-1 * time.Hour),
			WorkingOn: "paused: reviewing brew bundle changes",
			LastLine:  "[paused by user]",
		},
		{
			ID:        "sess_3333cccc4444",
			PID:       8901,
			State:     "dead",
			Project:   "personal-site",
			StartedAt: now.Add(-12 * time.Hour),
			WorkingOn: "",
			LastLine:  "Process exited with code 137 (killed)",
		},
		{
			ID:        "sess_dddd5555eeee",
			PID:       14600,
			State:     "busy",
			Project:   "personal-site",
			StartedAt: now.Add(-1 * time.Minute),
			WorkingOn: "generating blog post from outline",
			LastLine:  "> Writing content/posts/2026-03-21-tui.md",
		},
	}
}

// groupByProject groups sessions by their project name.
// Returns ordered project names and a map of project -> sessions.
func groupByProject(sessions []Session) ([]string, map[string][]Session) {
	groups := make(map[string][]Session)
	var order []string
	seen := make(map[string]bool)

	for _, s := range sessions {
		if !seen[s.Project] {
			seen[s.Project] = true
			order = append(order, s.Project)
		}
		groups[s.Project] = append(groups[s.Project], s)
	}
	return order, groups
}

// countByState counts sessions per state.
func countByState(sessions []Session) map[string]int {
	counts := make(map[string]int)
	for _, s := range sessions {
		counts[s.State]++
	}
	return counts
}

// filterSessions returns sessions matching the given state filter and search term.
// Empty stateFilter or searchTerm means no filtering on that dimension.
func filterSessions(sessions []Session, stateFilter, searchTerm string) []Session {
	if stateFilter == "" && searchTerm == "" {
		return sessions
	}

	var result []Session
	for _, s := range sessions {
		if stateFilter != "" && s.State != stateFilter {
			continue
		}
		if searchTerm != "" && !matchesSearch(s, searchTerm) {
			continue
		}
		result = append(result, s)
	}
	return result
}

// matchesSearch performs a case-insensitive substring match across session fields.
// Uses strings.Contains and strings.ToLower for proper Unicode handling.
func matchesSearch(s Session, term string) bool {
	t := strings.ToLower(term)
	return strings.Contains(strings.ToLower(s.ID), t) ||
		strings.Contains(strings.ToLower(s.Project), t) ||
		strings.Contains(strings.ToLower(s.State), t) ||
		strings.Contains(strings.ToLower(s.WorkingOn), t) ||
		strings.Contains(strings.ToLower(s.LastLine), t) ||
		strings.Contains(strings.ToLower(itoa(s.PID)), t)
}
