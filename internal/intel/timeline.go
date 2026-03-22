package intel

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// TimelineEvent represents a single point on a session timeline.
type TimelineEvent struct {
	Timestamp time.Time `json:"timestamp"`
	Type      string    `json:"type"`      // "user", "tool", "text", "error", "idle"
	Tool      string    `json:"tool,omitempty"`
	Duration  int       `json:"duration"`  // seconds until next event
}

// SessionTimeline holds the complete timeline for a session.
type SessionTimeline struct {
	SessionID string          `json:"session_id"`
	Project   string          `json:"project"`
	StartedAt time.Time       `json:"started_at"`
	EndedAt   time.Time       `json:"ended_at"`
	Events    []TimelineEvent `json:"events"`
}

// ExtractTimeline reads a transcript and builds a timeline of events.
func ExtractTimeline(path string, since time.Time) (SessionTimeline, error) {
	f, err := os.Open(path)
	if err != nil {
		return SessionTimeline{}, err
	}
	defer f.Close()

	var tl SessionTimeline
	filterByTime := !since.IsZero()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 10*1024*1024)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var entry transcriptLine
		if json.Unmarshal(line, &entry) != nil {
			continue
		}

		if tl.SessionID == "" && entry.SessionID != "" {
			tl.SessionID = entry.SessionID
		}
		if tl.Project == "" && entry.CWD != "" {
			tl.Project = filepath.Base(entry.CWD)
		}

		// Parse timestamp.
		if entry.Timestamp == "" {
			continue
		}
		ts, err := time.Parse(time.RFC3339Nano, entry.Timestamp)
		if err != nil {
			continue
		}

		if filterByTime && ts.Before(since) {
			continue
		}

		if tl.StartedAt.IsZero() || ts.Before(tl.StartedAt) {
			tl.StartedAt = ts
		}
		if ts.After(tl.EndedAt) {
			tl.EndedAt = ts
		}

		switch entry.Type {
		case "user":
			if entry.IsMeta {
				continue
			}
			// Check if it's a tool result or real user message.
			var msg messageEnvelope
			if json.Unmarshal(entry.Message, &msg) != nil {
				continue
			}

			// String content = real user message.
			var text string
			if json.Unmarshal(msg.Content, &text) == nil && !isSystemMessage(text) {
				tl.Events = append(tl.Events, TimelineEvent{
					Timestamp: ts,
					Type:      "user",
				})
			}

			// Array content = check for tool results with errors.
			var blocks []toolResultBlock
			if json.Unmarshal(msg.Content, &blocks) == nil {
				for _, b := range blocks {
					if b.Type == "tool_result" && b.IsError {
						tl.Events = append(tl.Events, TimelineEvent{
							Timestamp: ts,
							Type:      "error",
						})
					}
				}
			}

		case "assistant":
			var msg messageEnvelope
			if json.Unmarshal(entry.Message, &msg) != nil {
				continue
			}

			var blocks []contentBlock
			if json.Unmarshal(msg.Content, &blocks) != nil {
				continue
			}

			for _, b := range blocks {
				switch b.Type {
				case "text":
					if b.Text != "" {
						tl.Events = append(tl.Events, TimelineEvent{
							Timestamp: ts,
							Type:      "text",
						})
					}
				case "tool_use":
					tl.Events = append(tl.Events, TimelineEvent{
						Timestamp: ts,
						Type:      "tool",
						Tool:      b.Name,
					})
				}
			}
		}
	}

	// Calculate durations between events.
	for i := 0; i < len(tl.Events)-1; i++ {
		d := tl.Events[i+1].Timestamp.Sub(tl.Events[i].Timestamp)
		tl.Events[i].Duration = int(d.Seconds())
	}
	if len(tl.Events) > 0 {
		tl.Events[len(tl.Events)-1].Duration = 0
	}

	return tl, nil
}
