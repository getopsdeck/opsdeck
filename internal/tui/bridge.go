package tui

import (
	"os"
	"path/filepath"

	"github.com/getopsdeck/opsdeck/internal/monitor"
)

// claudeDir returns the path to ~/.claude.
func claudeDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".claude")
}

// DiscoverSessions scans for real Claude Code sessions and returns them
// as TUI Session values. It delegates to monitor.Snapshot for all
// enrichment logic, then converts to the TUI-specific Session type.
func DiscoverSessions() []Session {
	base := claudeDir()
	if base == "" {
		return nil
	}

	sessionsDir := filepath.Join(base, "sessions")
	projectsDir := filepath.Join(base, "projects")

	enriched := monitor.Snapshot(sessionsDir, projectsDir)
	if len(enriched) == 0 {
		return nil
	}

	sessions := make([]Session, len(enriched))
	for i, ms := range enriched {
		sessions[i] = Session{
			ID:             ms.ID,
			PID:            ms.PID,
			State:          ms.State,
			Project:        ms.Project,
			StartedAt:      ms.StartedAt,
			WorkingOn:      ms.WorkingOn,
			LastLine:        ms.LastLine,
			TranscriptPath: ms.TranscriptPath,
			GitBranch:      ms.GitBranch,
			GitDirty:       ms.GitDirty,
		}
	}

	return sessions
}
