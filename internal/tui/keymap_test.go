package tui

import (
	"testing"
)

func TestDefaultKeyMap(t *testing.T) {
	km := DefaultKeyMap()

	bindings := []struct {
		name    string
		keys    []string
		enabled bool
	}{
		{"Up", km.Up.Keys(), km.Up.Enabled()},
		{"Down", km.Down.Keys(), km.Down.Enabled()},
		{"Enter", km.Enter.Keys(), km.Enter.Enabled()},
		{"Search", km.Search.Keys(), km.Search.Enabled()},
		{"Escape", km.Escape.Keys(), km.Escape.Enabled()},
		{"Refresh", km.Refresh.Keys(), km.Refresh.Enabled()},
		{"Quit", km.Quit.Keys(), km.Quit.Enabled()},
		{"Tab", km.Tab.Keys(), km.Tab.Enabled()},
		{"FilterState1", km.FilterState1.Keys(), km.FilterState1.Enabled()},
		{"FilterState2", km.FilterState2.Keys(), km.FilterState2.Enabled()},
		{"FilterState3", km.FilterState3.Keys(), km.FilterState3.Enabled()},
		{"FilterState4", km.FilterState4.Keys(), km.FilterState4.Enabled()},
		{"FilterClear", km.FilterClear.Keys(), km.FilterClear.Enabled()},
	}

	for _, b := range bindings {
		if len(b.keys) == 0 {
			t.Errorf("binding %q has no keys", b.name)
		}
		if !b.enabled {
			t.Errorf("binding %q should be enabled by default", b.name)
		}
	}
}

func TestStateForFilter(t *testing.T) {
	tests := []struct {
		n    int
		want string
	}{
		{1, "waiting"},
		{2, "busy"},
		{3, "idle"},
		{4, "dead"},
		{0, ""},
		{5, ""},
	}

	for _, tt := range tests {
		got := stateForFilter(tt.n)
		if got != tt.want {
			t.Errorf("stateForFilter(%d) = %q, want %q", tt.n, got, tt.want)
		}
	}
}
