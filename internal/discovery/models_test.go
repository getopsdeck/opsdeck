package discovery

import (
	"testing"
	"time"
)

func TestSessionStateConstants(t *testing.T) {
	tests := []struct {
		state SessionState
		want  string
	}{
		{StateBusy, "busy"},
		{StateWaiting, "waiting"},
		{StateIdle, "idle"},
		{StateDead, "dead"},
	}

	for _, tt := range tests {
		if string(tt.state) != tt.want {
			t.Errorf("state = %q, want %q", tt.state, tt.want)
		}
	}
}

func TestAttentionReasonConstants(t *testing.T) {
	tests := []struct {
		reason AttentionReason
		want   string
	}{
		{AttentionOrphaned, "orphaned"},
		{AttentionStale, "stale"},
		{AttentionNoTranscript, "no-transcript"},
	}

	for _, tt := range tests {
		if string(tt.reason) != tt.want {
			t.Errorf("reason = %q, want %q", tt.reason, tt.want)
		}
	}
}

func TestSessionStruct(t *testing.T) {
	now := time.Now()
	s := Session{
		ID:           "abc-123",
		PID:          42,
		CWD:          "/home/user/project",
		ProjectName:  "project",
		StartedAt:    now,
		State:        StateBusy,
		LastActivity: now,
		Summary:      "Working on feature X",
		MessageCount: 10,
		Attention:    []AttentionReason{AttentionStale},
	}

	if s.ID != "abc-123" {
		t.Errorf("ID = %q, want %q", s.ID, "abc-123")
	}
	if s.PID != 42 {
		t.Errorf("PID = %d, want %d", s.PID, 42)
	}
	if s.State != StateBusy {
		t.Errorf("State = %q, want %q", s.State, StateBusy)
	}
	if len(s.Attention) != 1 || s.Attention[0] != AttentionStale {
		t.Errorf("Attention = %v, want [stale]", s.Attention)
	}
}

func TestProjectStruct(t *testing.T) {
	p := Project{
		Name: "myproject",
		Path: "/home/user/myproject",
		Sessions: []Session{
			{ID: "s1", State: StateBusy},
			{ID: "s2", State: StateDead},
		},
	}

	if p.Name != "myproject" {
		t.Errorf("Name = %q, want %q", p.Name, "myproject")
	}
	if len(p.Sessions) != 2 {
		t.Fatalf("len(Sessions) = %d, want 2", len(p.Sessions))
	}
}

func TestSessionStateIsAlive(t *testing.T) {
	alive := []SessionState{StateBusy, StateWaiting, StateIdle}
	notAlive := []SessionState{StateDead}

	for _, s := range alive {
		if !s.IsAlive() {
			t.Errorf("%q.IsAlive() = false, want true", s)
		}
	}
	for _, s := range notAlive {
		if s.IsAlive() {
			t.Errorf("%q.IsAlive() = true, want false", s)
		}
	}
}
