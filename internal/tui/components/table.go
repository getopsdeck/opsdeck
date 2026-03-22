package components

import (
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// TableSession is the data needed to render a session row.
type TableSession struct {
	ID        string
	PID       int
	State     string
	Project   string
	StartedAt time.Time
	WorkingOn string
	LastLine  string
}

// TableModel is the Bubble Tea model for the session table.
type TableModel struct {
	// Data
	Sessions    []TableSession
	ProjectView bool // true = grouped by project, false = flat list

	// State
	Cursor int
	Offset int // scroll offset for viewport
	Height int // visible row count
	Width  int

	// Rendering
	styles TableStyles
}

// TableStyles holds styles for the table.
type TableStyles struct {
	ProjectHeader lipgloss.Style
	RowNormal     lipgloss.Style
	RowSelected   lipgloss.Style
	StateIcon     func(string) string
	Subtle        lipgloss.Style
	Truncated     lipgloss.Style
}

// DefaultTableStyles returns the default table styles.
func DefaultTableStyles(stateIconFn func(string) string) TableStyles {
	return TableStyles{
		ProjectHeader: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#7aa2f7")).
			Padding(0, 1).
			MarginTop(1),
		RowNormal: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#c0caf5")).
			Padding(0, 1),
		RowSelected: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#c0caf5")).
			Background(lipgloss.Color("#2f3549")).
			Padding(0, 1),
		StateIcon: stateIconFn,
		Subtle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#565f89")),
		Truncated: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#565f89")),
	}
}

// NewTable creates a new table model.
func NewTable(stateIconFn func(string) string) TableModel {
	return TableModel{
		ProjectView: true,
		styles:      DefaultTableStyles(stateIconFn),
	}
}

// SetSize updates the table dimensions.
func (m *TableModel) SetSize(width, height int) {
	m.Width = width
	m.Height = height
}

// SetSessions replaces the session list.
func (m *TableModel) SetSessions(sessions []TableSession) {
	m.Sessions = sessions
	if m.Cursor >= len(sessions) {
		m.Cursor = max(0, len(sessions)-1)
	}
}

// SelectedSession returns the currently selected session, or nil.
func (m *TableModel) SelectedSession() *TableSession {
	if len(m.Sessions) == 0 {
		return nil
	}
	if m.Cursor < 0 || m.Cursor >= len(m.Sessions) {
		return nil
	}
	s := m.Sessions[m.Cursor]
	return &s
}

// MoveUp moves the cursor up by one row.
func (m *TableModel) MoveUp() {
	if m.Cursor > 0 {
		m.Cursor--
	}
}

// MoveDown moves the cursor down by one row.
func (m *TableModel) MoveDown() {
	if m.Cursor < len(m.Sessions)-1 {
		m.Cursor++
	}
}

// Update handles messages for the table.
func (m TableModel) Update(msg tea.Msg) (TableModel, tea.Cmd) {
	return m, nil
}

// View renders the session table.
func (m TableModel) View() string {
	if len(m.Sessions) == 0 {
		empty := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#565f89")).
			Padding(2, 4).
			Render("No sessions found. Start some Claude Code instances!")
		return empty
	}

	var b strings.Builder

	if m.ProjectView {
		m.renderGrouped(&b)
	} else {
		m.renderFlat(&b)
	}

	return b.String()
}

// renderGrouped renders sessions grouped by project.
func (m TableModel) renderGrouped(b *strings.Builder) {
	// Determine project order and grouping.
	type group struct {
		name     string
		sessions []int // indices into m.Sessions
	}

	var groups []group
	seen := make(map[string]int)

	for i, s := range m.Sessions {
		idx, ok := seen[s.Project]
		if !ok {
			idx = len(groups)
			seen[s.Project] = idx
			groups = append(groups, group{name: s.Project})
		}
		groups[idx].sessions = append(groups[idx].sessions, i)
	}

	for _, g := range groups {
		// Project header.
		header := m.styles.ProjectHeader.Render(
			"\u2501\u2501 " + g.name + " (" + itoa(len(g.sessions)) + ")")
		b.WriteString(header)
		b.WriteString("\n")

		// Rows.
		for _, idx := range g.sessions {
			b.WriteString(m.renderRow(idx))
			b.WriteString("\n")
		}
	}
}

// renderFlat renders sessions as a flat list.
func (m TableModel) renderFlat(b *strings.Builder) {
	for i := range m.Sessions {
		b.WriteString(m.renderRow(i))
		b.WriteString("\n")
	}
}

// renderRow renders a single session row.
func (m TableModel) renderRow(idx int) string {
	s := m.Sessions[idx]
	selected := idx == m.Cursor
	style := m.styles.RowNormal
	if selected {
		style = m.styles.RowSelected
	}

	icon := m.styles.StateIcon(s.State)

	// PID
	pid := itoa(s.PID)

	// Relative time
	ago := shortRelativeTime(s.StartedAt)

	// Truncated session ID (first 12 chars).
	sid := s.ID
	if len(sid) > 16 {
		sid = sid[:16]
	}

	// Working on (truncated to fit).
	workingOn := s.WorkingOn
	if workingOn == "" {
		workingOn = m.styles.Subtle.Render("--")
	}

	// Build the row columns with fixed widths.
	pointer := "  "
	if selected {
		pointer = "\u25b6 " // right-pointing triangle
	}

	// Format: [pointer] [icon] [PID 6w] [time 6w] [SID 16w] [workingOn ...]
	row := pointer + icon + "  " +
		padRight(pid, 7) +
		m.styles.Subtle.Render(padRight(ago, 8)) +
		m.styles.Truncated.Render(padRight(sid, 18)) +
		truncate(workingOn, max(20, m.Width-60))

	return style.Width(m.Width).Render(row)
}

// shortRelativeTime formats a time as a compact relative string.
func shortRelativeTime(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "<1m"
	case d < time.Hour:
		return itoa(int(d.Minutes())) + "m"
	case d < 24*time.Hour:
		return itoa(int(d.Hours())) + "h"
	default:
		return itoa(int(d.Hours()/24)) + "d"
	}
}

// padRight pads a string with spaces to the given width.
func padRight(s string, width int) string {
	if len(s) >= width {
		return s[:width]
	}
	return s + strings.Repeat(" ", width-len(s))
}

// truncate cuts a string to maxLen, adding ellipsis if needed.
func truncate(s string, maxLen int) string {
	if maxLen <= 0 {
		return ""
	}
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
