package tui

import (
	"charm.land/lipgloss/v2"
)

// Color palette — dark theme inspired by lazydocker/k9s.
var (
	// Base colors
	colorBg        = lipgloss.Color("#1a1b26") // deep navy bg
	colorFg        = lipgloss.Color("#c0caf5") // soft blue-white fg
	colorSubtle    = lipgloss.Color("#565f89") // muted comments
	colorHighlight = lipgloss.Color("#7aa2f7") // bright blue accent
	colorBorder    = lipgloss.Color("#3b4261") // dim border

	// State badge colors
	colorWaiting = lipgloss.Color("#e0af68") // amber
	colorBusy    = lipgloss.Color("#7dcfff") // cyan
	colorIdle    = lipgloss.Color("#9ece6a") // green
	colorDead    = lipgloss.Color("#f7768e") // red
	colorPaused  = lipgloss.Color("#bb9af7") // purple
	colorUnknown = lipgloss.Color("#565f89") // muted

	// Title bar color
	colorTitleBg = lipgloss.Color("#7aa2f7")
	colorTitleFg = lipgloss.Color("#1a1b26")
)

// Styles holds all the lipgloss styles used by the TUI.
type Styles struct {
	// Title bar at the very top.
	TitleBar lipgloss.Style

	// Status bar (bottom).
	StatusBar     lipgloss.Style
	StatusKey     lipgloss.Style
	StatusValue   lipgloss.Style
	StatusDivider lipgloss.Style

	// State badges.
	BadgeWaiting lipgloss.Style
	BadgeBusy    lipgloss.Style
	BadgeIdle    lipgloss.Style
	BadgeDead    lipgloss.Style
	BadgePaused  lipgloss.Style
	BadgeUnknown lipgloss.Style

	// Project header row.
	ProjectHeader lipgloss.Style

	// Session table rows.
	RowNormal   lipgloss.Style
	RowSelected lipgloss.Style

	// Detail panel (transcript preview).
	DetailPanel  lipgloss.Style
	DetailBorder lipgloss.Style
	DetailTitle  lipgloss.Style

	// Search input.
	SearchPrompt lipgloss.Style
	SearchText   lipgloss.Style

	// Help bar.
	HelpKey  lipgloss.Style
	HelpDesc lipgloss.Style
	HelpSep  lipgloss.Style
}

// DefaultStyles returns the default set of styles for the TUI.
func DefaultStyles() Styles {
	s := Styles{}

	// Title bar — bold white-on-blue, full width.
	s.TitleBar = lipgloss.NewStyle().
		Bold(true).
		Foreground(colorTitleFg).
		Background(colorTitleBg).
		Padding(0, 2)

	// Status bar — dark bg, full width at bottom.
	s.StatusBar = lipgloss.NewStyle().
		Foreground(colorFg).
		Background(lipgloss.Color("#24283b")).
		Padding(0, 1)

	s.StatusKey = lipgloss.NewStyle().
		Bold(true).
		Foreground(colorHighlight)

	s.StatusValue = lipgloss.NewStyle().
		Foreground(colorFg)

	s.StatusDivider = lipgloss.NewStyle().
		Foreground(colorSubtle).
		SetString(" | ")

	// State badges — bold colored text with matching bg tint.
	badgeBase := lipgloss.NewStyle().
		Bold(true).
		Padding(0, 1)

	s.BadgeWaiting = badgeBase.
		Foreground(lipgloss.Color("#1a1b26")).
		Background(colorWaiting)

	s.BadgeBusy = badgeBase.
		Foreground(lipgloss.Color("#1a1b26")).
		Background(colorBusy)

	s.BadgeIdle = badgeBase.
		Foreground(lipgloss.Color("#1a1b26")).
		Background(colorIdle)

	s.BadgeDead = badgeBase.
		Foreground(lipgloss.Color("#1a1b26")).
		Background(colorDead)

	s.BadgePaused = badgeBase.
		Foreground(lipgloss.Color("#1a1b26")).
		Background(colorPaused)

	s.BadgeUnknown = badgeBase.
		Foreground(lipgloss.Color("#1a1b26")).
		Background(colorUnknown)

	// Project header — bold with subtle underline effect.
	s.ProjectHeader = lipgloss.NewStyle().
		Bold(true).
		Foreground(colorHighlight).
		Padding(0, 1).
		MarginTop(1)

	// Table rows.
	s.RowNormal = lipgloss.NewStyle().
		Foreground(colorFg).
		Padding(0, 1)

	s.RowSelected = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#c0caf5")).
		Background(lipgloss.Color("#2f3549")).
		Padding(0, 1)

	// Detail panel — bordered box at the bottom.
	s.DetailPanel = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorBorder).
		Padding(0, 1)

	s.DetailBorder = lipgloss.NewStyle().
		Foreground(colorBorder)

	s.DetailTitle = lipgloss.NewStyle().
		Bold(true).
		Foreground(colorHighlight)

	// Search.
	s.SearchPrompt = lipgloss.NewStyle().
		Bold(true).
		Foreground(colorHighlight)

	s.SearchText = lipgloss.NewStyle().
		Foreground(colorFg)

	// Help bar.
	s.HelpKey = lipgloss.NewStyle().
		Bold(true).
		Foreground(colorHighlight)

	s.HelpDesc = lipgloss.NewStyle().
		Foreground(colorSubtle)

	s.HelpSep = lipgloss.NewStyle().
		Foreground(colorSubtle).
		SetString(" | ")

	return s
}

// StateIcon returns the emoji icon for a session state.
func StateIcon(state string) string {
	switch state {
	case "waiting":
		return "\u23f3" // hourglass
	case "busy":
		return "\U0001f504" // arrows
	case "idle":
		return "\U0001f634" // sleeping
	case "dead":
		return "\U0001f480" // skull
	case "paused":
		return "\u23f8\ufe0f" // pause
	default:
		return "\u2753" // question mark
	}
}

// StateBadge returns a styled badge string for a given state and count.
func (s Styles) StateBadge(state string, count int) string {
	icon := StateIcon(state)
	label := func() string {
		switch state {
		case "waiting":
			return s.BadgeWaiting.Render(icon + " " + itoa(count))
		case "busy":
			return s.BadgeBusy.Render(icon + " " + itoa(count))
		case "idle":
			return s.BadgeIdle.Render(icon + " " + itoa(count))
		case "dead":
			return s.BadgeDead.Render(icon + " " + itoa(count))
		case "paused":
			return s.BadgePaused.Render(icon + " " + itoa(count))
		default:
			return s.BadgeUnknown.Render(icon + " " + itoa(count))
		}
	}()
	return label
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
