package tui

import (
	"charm.land/bubbles/v2/key"
)

// KeyMap defines all key bindings for the TUI.
type KeyMap struct {
	Up           key.Binding
	Down         key.Binding
	Enter        key.Binding
	Search       key.Binding
	Escape       key.Binding
	Refresh      key.Binding
	Quit         key.Binding
	Tab          key.Binding
	FilterState1 key.Binding // waiting
	FilterState2 key.Binding // busy
	FilterState3 key.Binding // idle
	FilterState4 key.Binding // dead
	FilterClear  key.Binding // clear filter
}

// DefaultKeyMap returns the default key bindings.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "down"),
		),
		Enter: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "toggle detail"),
		),
		Search: key.NewBinding(
			key.WithKeys("/"),
			key.WithHelp("/", "search"),
		),
		Escape: key.NewBinding(
			key.WithKeys("escape"),
			key.WithHelp("esc", "cancel"),
		),
		Refresh: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "refresh"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
		Tab: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "toggle view"),
		),
		FilterState1: key.NewBinding(
			key.WithKeys("1"),
			key.WithHelp("1", "waiting"),
		),
		FilterState2: key.NewBinding(
			key.WithKeys("2"),
			key.WithHelp("2", "busy"),
		),
		FilterState3: key.NewBinding(
			key.WithKeys("3"),
			key.WithHelp("3", "idle"),
		),
		FilterState4: key.NewBinding(
			key.WithKeys("4"),
			key.WithHelp("4", "dead"),
		),
		FilterClear: key.NewBinding(
			key.WithKeys("0"),
			key.WithHelp("0", "clear filter"),
		),
	}
}

// stateForFilter maps filter key binding index (1-4) to state string.
func stateForFilter(n int) string {
	switch n {
	case 1:
		return "waiting"
	case 2:
		return "busy"
	case 3:
		return "idle"
	case 4:
		return "dead"
	default:
		return ""
	}
}
