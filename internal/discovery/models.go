// Package discovery scans for active Claude Code sessions on the local machine.
// It reads session files, checks process liveness, parses transcripts, and
// determines each session's current state.
package discovery

import "time"

// SessionState describes the lifecycle state of a Claude Code session.
type SessionState string

const (
	// StateBusy means the session's process is alive and had activity within
	// the last 30 seconds.
	StateBusy SessionState = "busy"

	// StateWaiting means the process is alive but activity was between 30
	// seconds and 5 minutes ago (likely waiting for user input).
	StateWaiting SessionState = "waiting"

	// StateIdle means the process is alive but no activity for over 5
	// minutes.
	StateIdle SessionState = "idle"

	// StateDead means the session's process is no longer running.
	StateDead SessionState = "dead"

)

// AttentionReason flags diagnostic issues that are separate from lifecycle state.
// Per Codex review: don't mix lifecycle states with data-integrity warnings.
type AttentionReason string

const (
	// AttentionOrphaned means a transcript exists but no session file was found.
	AttentionOrphaned AttentionReason = "orphaned"

	// AttentionStale means the PID was reused by a different process.
	AttentionStale AttentionReason = "stale"

	// AttentionNoTranscript means a session file exists but no transcript was found.
	AttentionNoTranscript AttentionReason = "no-transcript"
)

// IsAlive reports whether the state indicates an active, running session.
func (s SessionState) IsAlive() bool {
	switch s {
	case StateBusy, StateWaiting, StateIdle:
		return true
	default:
		return false
	}
}

// Session represents a single Claude Code session discovered on disk.
type Session struct {
	// ID is the session UUID from the session file (sessionId field).
	ID string

	// PID is the process ID of the Claude Code process.
	PID int

	// CWD is the working directory the session was started in.
	CWD string

	// ProjectName is the basename of CWD (or the git root if available).
	ProjectName string

	// StartedAt is when the session was created (from the session file).
	StartedAt time.Time

	// State is the classified lifecycle state.
	State SessionState

	// LastActivity is the timestamp of the most recent transcript entry.
	// Zero value means no activity information is available.
	LastActivity time.Time

	// Summary is the human-readable summary from sessions-index.json, if
	// available. Empty string if not found.
	Summary string

	// MessageCount is the number of messages from sessions-index.json.
	// Zero if not available.
	MessageCount int

	// Attention holds diagnostic flags (orphaned, stale, no-transcript).
	// Empty slice means no issues.
	Attention []AttentionReason
}

// Project groups sessions that share the same working directory.
type Project struct {
	// Name is the project identifier. Normally the basename of the shared CWD
	// (e.g. "QuantMind"). When multiple projects share the same basename,
	// GroupByProject expands it to "parent/basename" (e.g. "work/api") to
	// prevent identity collisions.
	Name string

	// Path is the full filesystem path to the project directory.
	Path string

	// Sessions contains all discovered sessions for this project.
	Sessions []Session
}
