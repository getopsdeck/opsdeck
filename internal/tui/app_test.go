package tui

import (
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/getopsdeck/opsdeck/internal/intel"
)

func TestRenderMiniTimeline_Empty(t *testing.T) {
	got := renderMiniTimeline(nil, 20)
	if got != "" {
		t.Errorf("expected empty string for nil events, got %q", got)
	}

	got = renderMiniTimeline([]intel.TimelineEvent{}, 20)
	if got != "" {
		t.Errorf("expected empty string for empty events, got %q", got)
	}
}

func TestRenderMiniTimeline_SingleEvent(t *testing.T) {
	// A single event has zero duration (start == end), so the function
	// should return an empty string.
	events := []intel.TimelineEvent{
		{Timestamp: time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC), Type: "tool"},
	}

	got := renderMiniTimeline(events, 20)
	if got != "" {
		t.Errorf("expected empty string for single event (zero duration), got %q", got)
	}
}

func TestRenderMiniTimeline_ToolEvents(t *testing.T) {
	base := time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC)
	events := []intel.TimelineEvent{
		{Timestamp: base, Type: "tool"},
		{Timestamp: base.Add(5 * time.Minute), Type: "tool"},
		{Timestamp: base.Add(10 * time.Minute), Type: "tool"},
	}

	got := renderMiniTimeline(events, 10)

	// The bar must contain at least one full-block character for tool events.
	if !strings.ContainsRune(got, '█') {
		t.Errorf("expected tool character '█' in timeline, got %q", got)
	}

	// Every cell should be either tool (█) or idle (░); no other types present.
	for _, r := range got {
		if r != '█' && r != '░' {
			t.Errorf("unexpected rune %q in tool-only timeline %q", string(r), got)
		}
	}
}

func TestRenderMiniTimeline_MixedEvents(t *testing.T) {
	base := time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC)
	events := []intel.TimelineEvent{
		{Timestamp: base, Type: "user"},
		{Timestamp: base.Add(10 * time.Minute), Type: "tool"},
		{Timestamp: base.Add(20 * time.Minute), Type: "text"},
		{Timestamp: base.Add(30 * time.Minute), Type: "error"},
	}

	got := renderMiniTimeline(events, 20)

	// Each event type should produce its corresponding character somewhere.
	checks := []struct {
		evType string
		char   rune
	}{
		{"user", '▒'},
		{"tool", '█'},
		{"text", '▓'},
		{"error", 'X'},
	}

	for _, c := range checks {
		if !strings.ContainsRune(got, c.char) {
			t.Errorf("expected %s character %q in mixed timeline, got %q", c.evType, string(c.char), got)
		}
	}

	// Gaps between events should be filled with idle character.
	if !strings.ContainsRune(got, '░') {
		t.Errorf("expected idle character '░' in gaps between events, got %q", got)
	}
}

func TestRenderMiniTimeline_Width(t *testing.T) {
	base := time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC)
	events := []intel.TimelineEvent{
		{Timestamp: base, Type: "tool"},
		{Timestamp: base.Add(1 * time.Hour), Type: "text"},
	}

	widths := []int{1, 5, 10, 50, 100}
	for _, w := range widths {
		got := renderMiniTimeline(events, w)
		runeCount := utf8.RuneCountInString(got)
		if runeCount != w {
			t.Errorf("renderMiniTimeline(events, %d) produced %d runes, want %d", w, runeCount, w)
		}
	}
}

func TestRenderMiniTimeline_ZeroWidth(t *testing.T) {
	base := time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC)
	events := []intel.TimelineEvent{
		{Timestamp: base, Type: "tool"},
		{Timestamp: base.Add(1 * time.Hour), Type: "text"},
	}

	got := renderMiniTimeline(events, 0)
	if got != "" {
		t.Errorf("expected empty string for zero width, got %q", got)
	}

	got = renderMiniTimeline(events, -1)
	if got != "" {
		t.Errorf("expected empty string for negative width, got %q", got)
	}
}

func TestRenderMiniTimeline_SameTimestamp(t *testing.T) {
	// All events at the same instant means totalDur == 0 -> empty string.
	ts := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	events := []intel.TimelineEvent{
		{Timestamp: ts, Type: "tool"},
		{Timestamp: ts, Type: "user"},
		{Timestamp: ts, Type: "text"},
	}

	got := renderMiniTimeline(events, 20)
	if got != "" {
		t.Errorf("expected empty string when all events share timestamp, got %q", got)
	}
}

func TestRenderMiniTimeline_EventPositioning(t *testing.T) {
	// With two events: first at position 0, last at position width-1.
	base := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	events := []intel.TimelineEvent{
		{Timestamp: base, Type: "user"},
		{Timestamp: base.Add(1 * time.Hour), Type: "error"},
	}

	got := renderMiniTimeline(events, 10)
	runes := []rune(got)

	// First cell should be the first event's character.
	if runes[0] != '▒' {
		t.Errorf("first cell should be user '▒', got %q", string(runes[0]))
	}

	// Last cell should be the last event's character (clamped to width-1).
	if runes[len(runes)-1] != 'X' {
		t.Errorf("last cell should be error 'X', got %q", string(runes[len(runes)-1]))
	}
}
