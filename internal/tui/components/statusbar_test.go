package components

import (
	"strings"
	"testing"
	"time"
)

func badgeStub(state string, count int) string {
	return "[" + state + ":" + itoa(count) + "]"
}

func TestNewStatusBar(t *testing.T) {
	sb := NewStatusBar(badgeStub)
	if sb.Counts == nil {
		t.Error("Counts should be initialized")
	}
	if sb.LastRefresh.IsZero() {
		t.Error("LastRefresh should be set")
	}
}

func TestStatusBarView(t *testing.T) {
	sb := NewStatusBar(badgeStub)
	sb.SetSize(120)
	sb.Counts = map[string]int{
		"waiting": 3,
		"busy":    5,
		"idle":    2,
		"dead":    4,
		"paused":  1,
	}
	sb.Total = 15
	sb.LastRefresh = time.Now()

	view := sb.View()

	// Should contain all state badges.
	if !strings.Contains(view, "[waiting:3]") {
		t.Error("view should contain waiting badge")
	}
	if !strings.Contains(view, "[busy:5]") {
		t.Error("view should contain busy badge")
	}
	if !strings.Contains(view, "[idle:2]") {
		t.Error("view should contain idle badge")
	}
	if !strings.Contains(view, "[dead:4]") {
		t.Error("view should contain dead badge")
	}
	if !strings.Contains(view, "[paused:1]") {
		t.Error("view should contain paused badge")
	}

	// Should contain total.
	if !strings.Contains(view, "15") {
		t.Error("view should contain total count")
	}
}

func TestStatusBarWithFilter(t *testing.T) {
	sb := NewStatusBar(badgeStub)
	sb.SetSize(120)
	sb.Counts = map[string]int{"busy": 2}
	sb.Total = 2
	sb.Filter = "busy"
	sb.LastRefresh = time.Now()

	view := sb.View()
	if !strings.Contains(view, "filter:") {
		t.Error("view should show active filter")
	}
	if !strings.Contains(view, "busy") {
		t.Error("view should show filter value 'busy'")
	}
}

func TestStatusBarWithSearch(t *testing.T) {
	sb := NewStatusBar(badgeStub)
	sb.SetSize(120)
	sb.Counts = map[string]int{"busy": 2}
	sb.Total = 2
	sb.SearchTerm = "OpsDeck"
	sb.LastRefresh = time.Now()

	view := sb.View()
	if !strings.Contains(view, "search:") {
		t.Error("view should show active search")
	}
	if !strings.Contains(view, "OpsDeck") {
		t.Error("view should show search term")
	}
}

func TestStatusBarZeroWidth(t *testing.T) {
	sb := NewStatusBar(badgeStub)
	view := sb.View()
	if view != "" {
		t.Error("zero-width status bar should return empty string")
	}
}

func TestRelativeTime(t *testing.T) {
	tests := []struct {
		ago  time.Duration
		want string
	}{
		{500 * time.Millisecond, "just now"},
		{30 * time.Second, "30s ago"},
		{5 * time.Minute, "5m ago"},
		{3 * time.Hour, "3h ago"},
	}

	for _, tt := range tests {
		got := relativeTime(time.Now().Add(-tt.ago))
		if got != tt.want {
			t.Errorf("relativeTime(-%v) = %q, want %q", tt.ago, got, tt.want)
		}
	}
}
