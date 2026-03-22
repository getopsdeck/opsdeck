package components

import (
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// StatusBarModel is the Bubble Tea model for the bottom status bar.
type StatusBarModel struct {
	Width       int
	Counts      map[string]int
	Total       int
	LastRefresh time.Time
	Filter      string // active state filter, empty = none
	SearchTerm  string // active search query
	styles      StatusBarStyles
}

// StatusBarStyles holds the styles for the status bar.
type StatusBarStyles struct {
	Bar       lipgloss.Style
	Key       lipgloss.Style
	Value     lipgloss.Style
	Divider   string
	BadgeFn   func(state string, count int) string
	Separator string
}

// DefaultStatusBarStyles returns the default styling for the status bar.
func DefaultStatusBarStyles(badgeFn func(string, int) string) StatusBarStyles {
	return StatusBarStyles{
		Bar: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#c0caf5")).
			Background(lipgloss.Color("#24283b")).
			Padding(0, 1),
		Key: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#7aa2f7")),
		Value: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#c0caf5")),
		Divider:   " \u2502 ", // thin vertical bar
		BadgeFn:   badgeFn,
		Separator: "  ",
	}
}

// NewStatusBar creates a new status bar model.
func NewStatusBar(badgeFn func(string, int) string) StatusBarModel {
	return StatusBarModel{
		Counts:      make(map[string]int),
		LastRefresh: time.Now(),
		styles:      DefaultStatusBarStyles(badgeFn),
	}
}

// SetSize updates the status bar width.
func (m *StatusBarModel) SetSize(width int) {
	m.Width = width
}

// Update handles messages for the status bar. Currently a no-op passthrough.
func (m StatusBarModel) Update(msg tea.Msg) (StatusBarModel, tea.Cmd) {
	return m, nil
}

// View renders the status bar.
func (m StatusBarModel) View() string {
	if m.Width <= 0 {
		return ""
	}

	var parts []string

	// State badges — always in a fixed order.
	states := []string{"waiting", "busy", "idle", "dead", "paused"}
	for _, state := range states {
		count := m.Counts[state]
		parts = append(parts, m.styles.BadgeFn(state, count))
	}

	left := strings.Join(parts, m.styles.Separator)

	// Right side: total, filter, last refresh.
	var rightParts []string

	if m.SearchTerm != "" {
		rightParts = append(rightParts,
			m.styles.Key.Render("search:")+
				m.styles.Value.Render(" \""+m.SearchTerm+"\""))
	}

	if m.Filter != "" {
		rightParts = append(rightParts,
			m.styles.Key.Render("filter:")+
				m.styles.Value.Render(" "+m.Filter))
	}

	rightParts = append(rightParts,
		m.styles.Key.Render("total:")+
			m.styles.Value.Render(" "+itoa(m.Total)))

	rightParts = append(rightParts,
		m.styles.Key.Render("refresh:")+
			m.styles.Value.Render(" "+relativeTime(m.LastRefresh)))

	right := strings.Join(rightParts, m.styles.Divider)

	// Calculate gap between left and right.
	leftW := lipgloss.Width(left)
	rightW := lipgloss.Width(right)
	gap := m.Width - leftW - rightW - 2 // 2 for padding
	if gap < 1 {
		gap = 1
	}

	bar := left + strings.Repeat(" ", gap) + right

	return m.styles.Bar.Width(m.Width).Render(bar)
}

// relativeTime formats a time as a relative duration string.
func relativeTime(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < time.Second:
		return "just now"
	case d < time.Minute:
		s := int(d.Seconds())
		return itoa(s) + "s ago"
	case d < time.Hour:
		m := int(d.Minutes())
		return itoa(m) + "m ago"
	default:
		h := int(d.Hours())
		return itoa(h) + "h ago"
	}
}

func itoa(n int) string {
	if n < 0 {
		return "-" + uitoa(uint(-n))
	}
	return uitoa(uint(n))
}

func uitoa(n uint) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}
