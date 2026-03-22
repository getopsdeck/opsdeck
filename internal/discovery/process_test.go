package discovery

import (
	"os"
	"testing"
	"time"
)

func TestIsProcessAlive_CurrentProcess(t *testing.T) {
	// Our own PID should be alive.
	pid := os.Getpid()
	alive := IsProcessAlive(pid)
	if !alive {
		t.Errorf("IsProcessAlive(%d) = false, want true (own PID)", pid)
	}
}

func TestIsProcessAlive_Init(t *testing.T) {
	// PID 1 (launchd/init) should always be alive.
	alive := IsProcessAlive(1)
	if !alive {
		t.Errorf("IsProcessAlive(1) = false, want true (init)")
	}
}

func TestIsProcessAlive_NonexistentPID(t *testing.T) {
	// Very high PID that almost certainly doesn't exist.
	alive := IsProcessAlive(999999999)
	if alive {
		t.Errorf("IsProcessAlive(999999999) = true, want false")
	}
}

func TestIsProcessAlive_InvalidPID(t *testing.T) {
	alive := IsProcessAlive(-1)
	if alive {
		t.Errorf("IsProcessAlive(-1) = true, want false")
	}
}

func TestIsProcessAlive_ZeroPID(t *testing.T) {
	// PID 0 should return false (or at least not crash).
	alive := IsProcessAlive(0)
	// On macOS, kill(0, 0) sends to the current process group. We just
	// want to make sure this doesn't panic.
	_ = alive
}

func TestCheckSession_AliveProcess(t *testing.T) {
	pid := os.Getpid()
	// Use a recent startedAt so PID reuse detection doesn't flag it.
	startedAt := time.Now().Add(-1 * time.Second)
	alive := CheckSession(pid, startedAt)
	if !alive {
		t.Errorf("CheckSession(own PID, recent start) = false, want true")
	}
}

func TestCheckSession_DeadProcess(t *testing.T) {
	alive := CheckSession(999999999, time.Now())
	if alive {
		t.Errorf("CheckSession(dead PID) = true, want false")
	}
}
