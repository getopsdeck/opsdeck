package discovery

import "time"

const (
	// busyThreshold is the maximum time since last activity for a session
	// to be considered busy (actively working).
	busyThreshold = 30 * time.Second

	// waitingThreshold is the maximum time since last activity for a
	// session to be considered waiting (likely waiting for user input).
	waitingThreshold = 5 * time.Minute
)

// ClassifyState determines the SessionState based on process liveness and
// last activity timestamp.
//
// Decision logic:
//   - Process not alive                    -> StateDead
//   - Process alive, activity < 30s ago    -> StateBusy
//   - Process alive, activity 30s-5min ago -> StateWaiting
//   - Process alive, activity > 5min ago   -> StateIdle
//   - Process alive, no activity info      -> StateIdle
//
// StateOrphaned and StateStale are not assigned by this function; they
// require additional context from the scanner.
func ClassifyState(processAlive bool, lastActivity time.Time) SessionState {
	if !processAlive {
		return StateDead
	}

	// If we have no activity information, default to idle.
	if lastActivity.IsZero() {
		return StateIdle
	}

	elapsed := time.Since(lastActivity)

	switch {
	case elapsed <= busyThreshold:
		return StateBusy
	case elapsed <= waitingThreshold:
		return StateWaiting
	default:
		return StateIdle
	}
}
