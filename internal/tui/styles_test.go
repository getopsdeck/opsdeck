package tui

import (
	"testing"
)

func TestDefaultStyles(t *testing.T) {
	s := DefaultStyles()

	// Verify all styles render without panic.
	_ = s.TitleBar.Render("test")
	_ = s.StatusBar.Render("test")
	_ = s.StatusKey.Render("test")
	_ = s.StatusValue.Render("test")
	_ = s.StatusDivider.String()
	_ = s.BadgeWaiting.Render("test")
	_ = s.BadgeBusy.Render("test")
	_ = s.BadgeIdle.Render("test")
	_ = s.BadgeDead.Render("test")
	_ = s.BadgePaused.Render("test")
	_ = s.BadgeUnknown.Render("test")
	_ = s.ProjectHeader.Render("test")
	_ = s.RowNormal.Render("test")
	_ = s.RowSelected.Render("test")
	_ = s.DetailPanel.Render("test")
	_ = s.DetailBorder.Render("test")
	_ = s.DetailTitle.Render("test")
	_ = s.SearchPrompt.Render("test")
	_ = s.SearchText.Render("test")
	_ = s.HelpKey.Render("test")
	_ = s.HelpDesc.Render("test")
	_ = s.HelpSep.String()
}

func TestStateBadge(t *testing.T) {
	s := DefaultStyles()

	states := []string{"waiting", "busy", "idle", "dead", "paused", "unknown"}
	for _, state := range states {
		badge := s.StateBadge(state, 5)
		if badge == "" {
			t.Errorf("StateBadge(%q, 5) returned empty string", state)
		}
	}
}
