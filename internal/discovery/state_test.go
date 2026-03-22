package discovery

import (
	"testing"
	"time"
)

func TestClassifyState_DeadProcess(t *testing.T) {
	now := time.Now()
	state := ClassifyState(false, now.Add(-10*time.Second))
	if state != StateDead {
		t.Errorf("ClassifyState(alive=false) = %q, want %q", state, StateDead)
	}
}

func TestClassifyState_Busy(t *testing.T) {
	now := time.Now()
	// Activity 5 seconds ago -> busy.
	state := ClassifyState(true, now.Add(-5*time.Second))
	if state != StateBusy {
		t.Errorf("ClassifyState(alive=true, 5s ago) = %q, want %q", state, StateBusy)
	}
}

func TestClassifyState_BusyBoundary(t *testing.T) {
	now := time.Now()
	// Exactly 29 seconds ago -> still busy.
	state := ClassifyState(true, now.Add(-29*time.Second))
	if state != StateBusy {
		t.Errorf("ClassifyState(alive=true, 29s ago) = %q, want %q", state, StateBusy)
	}
}

func TestClassifyState_Waiting(t *testing.T) {
	now := time.Now()
	// Activity 2 minutes ago -> waiting.
	state := ClassifyState(true, now.Add(-2*time.Minute))
	if state != StateWaiting {
		t.Errorf("ClassifyState(alive=true, 2min ago) = %q, want %q", state, StateWaiting)
	}
}

func TestClassifyState_WaitingBoundary(t *testing.T) {
	now := time.Now()
	// Exactly 31 seconds ago -> waiting (just past busy threshold).
	state := ClassifyState(true, now.Add(-31*time.Second))
	if state != StateWaiting {
		t.Errorf("ClassifyState(alive=true, 31s ago) = %q, want %q", state, StateWaiting)
	}
}

func TestClassifyState_Idle(t *testing.T) {
	now := time.Now()
	// Activity 10 minutes ago -> idle.
	state := ClassifyState(true, now.Add(-10*time.Minute))
	if state != StateIdle {
		t.Errorf("ClassifyState(alive=true, 10min ago) = %q, want %q", state, StateIdle)
	}
}

func TestClassifyState_IdleBoundary(t *testing.T) {
	now := time.Now()
	// Exactly 5 minutes and 1 second ago -> idle.
	state := ClassifyState(true, now.Add(-5*time.Minute-1*time.Second))
	if state != StateIdle {
		t.Errorf("ClassifyState(alive=true, 5m1s ago) = %q, want %q", state, StateIdle)
	}
}

func TestClassifyState_ZeroActivity(t *testing.T) {
	// Zero time (no activity info) with alive process -> idle.
	state := ClassifyState(true, time.Time{})
	if state != StateIdle {
		t.Errorf("ClassifyState(alive=true, zero time) = %q, want %q", state, StateIdle)
	}
}

func TestClassifyState_ZeroActivityDead(t *testing.T) {
	// Zero time with dead process -> dead.
	state := ClassifyState(false, time.Time{})
	if state != StateDead {
		t.Errorf("ClassifyState(alive=false, zero time) = %q, want %q", state, StateDead)
	}
}
