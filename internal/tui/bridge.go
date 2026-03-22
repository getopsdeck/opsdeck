package tui

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/getopsdeck/opsdeck/internal/discovery"
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
// as TUI Session values.
func DiscoverSessions() []Session {
	base := claudeDir()
	if base == "" {
		return nil
	}

	sessionsDir := filepath.Join(base, "sessions")
	projectsDir := filepath.Join(base, "projects")

	// 1. Scan session files.
	raw, err := discovery.ScanSessions(sessionsDir)
	if err != nil || len(raw) == 0 {
		return nil
	}

	// 2. Build index cache: cwd -> session index entries.
	indexCache := make(map[string]map[string]discovery.IndexEntry)

	// 3. Classify each session and convert.
	sessions := make([]Session, 0, len(raw))
	for _, rs := range raw {
		// Check PID liveness with reuse protection.
		alive := discovery.CheckSession(rs.PID, rs.StartedAt)

		// Find transcript for last activity.
		transcriptPath := discovery.FindTranscriptPath(projectsDir, rs.CWD, rs.ID)
		var lastActivity = rs.StartedAt // fallback
		if transcriptPath != "" {
			if t, err := discovery.ReadLastActivity(transcriptPath); err == nil && !t.IsZero() {
				lastActivity = t
			}
		}

		// Classify state.
		state := discovery.ClassifyState(alive, lastActivity)

		// Enrich from session index (cached per project).
		summary := ""
		msgCount := 0
		encoded := discovery.EncodeCWD(rs.CWD)
		if _, ok := indexCache[encoded]; !ok {
			idxPath := discovery.FindSessionIndex(projectsDir, rs.CWD)
			if idxPath != "" {
				entries, _ := discovery.ParseSessionIndex(idxPath)
				indexCache[encoded] = entries
			} else {
				indexCache[encoded] = nil
			}
		}
		if entries := indexCache[encoded]; entries != nil {
			if e, ok := entries[rs.ID]; ok {
				summary = e.Summary
				msgCount = e.MessageCount
			}
		}

		// Read actual last transcript line for preview.
		lastLine := ""
		if transcriptPath != "" {
			lastLine = readLastMeaningfulLine(transcriptPath)
		}
		if lastLine == "" && summary != "" {
			lastLine = summary
		}

		// Build WorkingOn from summary or message count.
		workingOn := ""
		if summary != "" {
			workingOn = summary
		} else if msgCount > 0 {
			workingOn = fmt.Sprintf("%d messages", msgCount)
		}

		// Convert to TUI Session.
		sessions = append(sessions, Session{
			ID:             rs.ID,
			PID:            rs.PID,
			State:          string(state),
			Project:        rs.ProjectName,
			StartedAt:      rs.StartedAt,
			WorkingOn:      workingOn,
			LastLine:       lastLine,
			TranscriptPath: transcriptPath,
		})
	}

	return sessions
}

// readLastMeaningfulLine reads the tail of a transcript JSONL and extracts
// the last human-readable content. It looks for assistant text or user messages,
// skipping tool outputs, base64 data, and system events.
func readLastMeaningfulLine(path string) string {
	const tailSize = 64 * 1024 // 64KB to catch more entries

	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil || info.Size() == 0 {
		return ""
	}

	offset := int64(0)
	if info.Size() > tailSize {
		offset = info.Size() - tailSize
	}
	if _, err := f.Seek(offset, io.SeekStart); err != nil {
		return ""
	}

	// Scan lines from tail, collect candidates.
	var lastLine string
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 256*1024), 256*1024) // handle large lines

	for scanner.Scan() {
		line := scanner.Text()
		if len(line) == 0 || line[0] != '{' {
			continue
		}

		// Quick parse for type and content.
		var entry struct {
			Type    string `json:"type"`
			Message struct {
				Role    string `json:"role"`
				Content json.RawMessage `json:"content"`
			} `json:"message"`
		}
		if json.Unmarshal([]byte(line), &entry) != nil {
			continue
		}

		// Only care about user and assistant messages.
		if entry.Type != "user" && entry.Type != "assistant" {
			continue
		}

		// Try to extract text content.
		text := extractText(entry.Message.Content)
		if text != "" {
			lastLine = text
		}
	}

	// Truncate for display.
	if len(lastLine) > 120 {
		lastLine = lastLine[:117] + "..."
	}
	return lastLine
}

// extractText tries to get readable text from a message content field.
// Content can be a string or an array of content blocks.
func extractText(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}

	// Try as string first.
	var s string
	if json.Unmarshal(raw, &s) == nil && s != "" {
		return firstLine(s)
	}

	// Try as array of content blocks.
	var blocks []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	if json.Unmarshal(raw, &blocks) == nil {
		for _, b := range blocks {
			if b.Type == "text" && b.Text != "" {
				return firstLine(b.Text)
			}
		}
	}

	return ""
}

// firstLine returns the first non-empty line of a string.
func firstLine(s string) string {
	s = strings.TrimSpace(s)
	if idx := strings.IndexByte(s, '\n'); idx >= 0 {
		s = s[:idx]
	}
	return strings.TrimSpace(s)
}
