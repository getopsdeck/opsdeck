// Package intel provides analytical intelligence over Claude Code session data.
// It transforms raw discovery data into actionable summaries and reports.
package intel

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Activity type constants.
const (
	ActivityUserRequest = "user_request"
	ActivityFileEdit    = "file_edit"
	ActivityBashCommand = "bash_command"
	ActivityGitOp       = "git_op"
	ActivityError       = "error"
	ActivityToolCall    = "tool_call"
)

// Activity represents a single discrete event extracted from a transcript.
type Activity struct {
	Timestamp   time.Time
	Type        string // one of the Activity* constants
	Description string
}

// SessionSummary holds a complete activity summary for a single session.
type SessionSummary struct {
	SessionID     string
	Project       string
	Activities    []Activity
	FilesChanged  []string
	LastUserMsg   string
	LastAssistMsg string
	TotalMessages int
	ErrorCount    int
	EditCount     int
	BashCount     int
	ReadCount     int
}

// transcriptLine is the top-level JSON structure of a transcript JSONL line.
type transcriptLine struct {
	Type      string          `json:"type"`
	IsMeta    bool            `json:"isMeta"`
	Timestamp string          `json:"timestamp"`
	UUID      string          `json:"uuid"`
	SessionID string          `json:"sessionId"`
	CWD       string          `json:"cwd"`
	Message   json.RawMessage `json:"message"`
}

// messageEnvelope holds the role and content of a message.
type messageEnvelope struct {
	Role    string          `json:"role"`
	Content json.RawMessage `json:"content"`
}

// contentBlock represents one block inside an assistant message's content array.
type contentBlock struct {
	Type  string          `json:"type"`
	Text  string          `json:"text"`
	Name  string          `json:"name"`
	ID    string          `json:"id"`
	Input json.RawMessage `json:"input"`
}

// toolResultBlock is a block inside a user tool_result message.
type toolResultBlock struct {
	Type      string `json:"type"`
	ToolUseID string `json:"tool_use_id"`
	IsError   bool   `json:"is_error"`
	Content   any    `json:"content"`
}

// toolInput captures the fields we care about from tool_use inputs.
type toolInput struct {
	FilePath    string `json:"file_path"`
	Command     string `json:"command"`
	Description string `json:"description"`
}

// ExtractSummary reads an entire transcript and returns a full SessionSummary.
// If the file does not exist or is empty, returns an empty summary and no error.
func ExtractSummary(path string) (SessionSummary, error) {
	return extractFromFile(path, time.Time{})
}

// ExtractRecent reads a transcript and returns a summary of activity after `since`.
// If the file does not exist, returns an empty summary and no error.
func ExtractRecent(path string, since time.Time) (SessionSummary, error) {
	return extractFromFile(path, since)
}

// extractFromFile is the shared implementation for ExtractSummary and ExtractRecent.
// When since is the zero time, all entries are included.
func extractFromFile(path string, since time.Time) (SessionSummary, error) {
	f, err := os.Open(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return SessionSummary{}, nil
		}
		return SessionSummary{}, fmt.Errorf("opening transcript: %w", err)
	}
	defer f.Close()

	var summary SessionSummary
	filesSet := make(map[string]struct{})
	filterByTime := !since.IsZero()

	scanner := bufio.NewScanner(f)
	// Allow very large lines (base64 content can be huge).
	scanner.Buffer(make([]byte, 0, 64*1024), 10*1024*1024)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var tl transcriptLine
		if err := json.Unmarshal(line, &tl); err != nil {
			continue // skip malformed lines
		}

		// Capture session metadata from the first valid line.
		if summary.SessionID == "" && tl.SessionID != "" {
			summary.SessionID = tl.SessionID
		}
		if summary.Project == "" && tl.CWD != "" {
			summary.Project = filepath.Base(tl.CWD)
		}

		// Parse timestamp for time filtering.
		var ts time.Time
		if tl.Timestamp != "" {
			parsed, err := time.Parse(time.RFC3339Nano, tl.Timestamp)
			if err == nil {
				ts = parsed
			}
		}

		if filterByTime && !ts.IsZero() && ts.Before(since) {
			continue
		}

		switch tl.Type {
		case "assistant":
			processAssistantLine(&summary, tl, ts, filesSet)
		case "user":
			processUserLine(&summary, tl, ts)
		}
	}

	// Build sorted, deduplicated FilesChanged.
	for f := range filesSet {
		summary.FilesChanged = append(summary.FilesChanged, f)
	}
	sort.Strings(summary.FilesChanged)

	return summary, nil
}

// processAssistantLine handles a transcript line of type "assistant".
func processAssistantLine(summary *SessionSummary, tl transcriptLine, ts time.Time, filesSet map[string]struct{}) {
	summary.TotalMessages++

	var msg messageEnvelope
	if err := json.Unmarshal(tl.Message, &msg); err != nil {
		return
	}

	// Content can be a string or an array of content blocks.
	var blocks []contentBlock
	if err := json.Unmarshal(msg.Content, &blocks); err != nil {
		// Try as string.
		var text string
		if err := json.Unmarshal(msg.Content, &text); err == nil {
			summary.LastAssistMsg = text
		}
		return
	}

	for _, block := range blocks {
		switch block.Type {
		case "text":
			if block.Text != "" {
				summary.LastAssistMsg = block.Text
			}
		case "tool_use":
			processToolUse(summary, block, ts, filesSet)
		}
	}
}

// processToolUse handles a single tool_use block inside an assistant message.
func processToolUse(summary *SessionSummary, block contentBlock, ts time.Time, filesSet map[string]struct{}) {
	var input toolInput
	if block.Input != nil {
		json.Unmarshal(block.Input, &input)
	}

	switch block.Name {
	case "Edit", "Write":
		summary.EditCount++
		if input.FilePath != "" {
			filesSet[input.FilePath] = struct{}{}
		}
		desc := fmt.Sprintf("Edited %s", filepath.Base(input.FilePath))
		if block.Name == "Write" {
			desc = fmt.Sprintf("Wrote %s", filepath.Base(input.FilePath))
		}
		summary.Activities = append(summary.Activities, Activity{
			Timestamp:   ts,
			Type:        ActivityFileEdit,
			Description: desc,
		})

	case "Bash":
		summary.BashCount++
		cmd := input.Command
		if isGitCommand(cmd) {
			summary.Activities = append(summary.Activities, Activity{
				Timestamp:   ts,
				Type:        ActivityGitOp,
				Description: describeGitOp(cmd, input.Description),
			})
		} else {
			desc := input.Description
			if desc == "" {
				desc = truncate(cmd, 60)
			}
			summary.Activities = append(summary.Activities, Activity{
				Timestamp:   ts,
				Type:        ActivityBashCommand,
				Description: desc,
			})
		}

	case "Read":
		summary.ReadCount++
		// Read is not tracked as an activity (too noisy).

	default:
		// Grep, Glob, and other tools: track as tool_call.
		desc := fmt.Sprintf("Used %s", block.Name)
		summary.Activities = append(summary.Activities, Activity{
			Timestamp:   ts,
			Type:        ActivityToolCall,
			Description: desc,
		})
	}
}

// processUserLine handles a transcript line of type "user".
func processUserLine(summary *SessionSummary, tl transcriptLine, ts time.Time) {
	if tl.IsMeta {
		return
	}

	var msg messageEnvelope
	if err := json.Unmarshal(tl.Message, &msg); err != nil {
		return
	}

	// Content can be a string (real user message) or an array (tool results).
	var text string
	if err := json.Unmarshal(msg.Content, &text); err == nil {
		// Skip system/control messages that look like XML commands.
		if isSystemMessage(text) {
			return
		}
		// String content -- this is a real user message.
		summary.TotalMessages++
		summary.LastUserMsg = text
		summary.Activities = append(summary.Activities, Activity{
			Timestamp:   ts,
			Type:        ActivityUserRequest,
			Description: truncate(text, 80),
		})
		return
	}

	// Array content -- check for tool_result blocks.
	var blocks []toolResultBlock
	if err := json.Unmarshal(msg.Content, &blocks); err != nil {
		return
	}

	for _, block := range blocks {
		if block.Type == "tool_result" && block.IsError {
			summary.ErrorCount++
			summary.Activities = append(summary.Activities, Activity{
				Timestamp:   ts,
				Type:        ActivityError,
				Description: "Tool returned error",
			})
		}
	}
}

// FormatBrief renders a single SessionSummary as a human-readable text block.
func FormatBrief(summary SessionSummary) string {
	var b strings.Builder

	project := summary.Project
	if project == "" {
		project = "(unknown)"
	}
	b.WriteString(fmt.Sprintf("Session: %s | Project: %s\n", summary.SessionID, project))

	// Stats line.
	parts := []string{}
	if summary.EditCount > 0 {
		parts = append(parts, fmt.Sprintf("%d edits", summary.EditCount))
	}
	if summary.BashCount > 0 {
		parts = append(parts, fmt.Sprintf("%d commands", summary.BashCount))
	}
	if summary.ReadCount > 0 {
		parts = append(parts, fmt.Sprintf("%d reads", summary.ReadCount))
	}
	if len(summary.FilesChanged) > 0 {
		parts = append(parts, fmt.Sprintf("%d files changed", len(summary.FilesChanged)))
	}
	if summary.TotalMessages > 0 {
		parts = append(parts, fmt.Sprintf("%d messages", summary.TotalMessages))
	}
	if summary.ErrorCount > 0 {
		parts = append(parts, fmt.Sprintf("%d errors", summary.ErrorCount))
	}
	if len(parts) > 0 {
		b.WriteString(strings.Join(parts, " | ") + "\n")
	}

	// Activities -- condensed into human-readable summaries.
	condensed := SummarizeActivities(summary.Activities)
	if len(condensed) > 0 {
		b.WriteString("Activities:\n")
		for _, line := range condensed {
			b.WriteString(fmt.Sprintf("  * %s\n", line))
		}
	}

	// Last messages.
	if summary.LastUserMsg != "" {
		b.WriteString(fmt.Sprintf("Last request: %s\n", truncate(summary.LastUserMsg, 100)))
	}
	if summary.LastAssistMsg != "" {
		b.WriteString(fmt.Sprintf("Last response: %s\n", truncate(summary.LastAssistMsg, 100)))
	}

	return b.String()
}

// isGitCommand checks if a bash command is a git or gh command.
func isGitCommand(cmd string) bool {
	trimmed := strings.TrimSpace(cmd)
	return strings.HasPrefix(trimmed, "git ") || strings.HasPrefix(trimmed, "gh ")
}

// describeGitOp produces a human-readable description for a git command.
func describeGitOp(cmd, description string) string {
	if description != "" {
		return description
	}
	trimmed := strings.TrimSpace(cmd)
	if strings.HasPrefix(trimmed, "git commit") {
		return "Git commit"
	}
	if strings.HasPrefix(trimmed, "git push") {
		return "Git push"
	}
	if strings.HasPrefix(trimmed, "gh pr") {
		return "Create PR"
	}
	return truncate(trimmed, 60)
}

// truncate shortens a string to maxLen characters, adding "..." if truncated.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

// isSystemMessage checks if a user message is a system/control message
// (local commands, task notifications, etc.) that should not appear as
// user requests in activity summaries.
func isSystemMessage(text string) bool {
	trimmed := strings.TrimSpace(text)
	// Local slash-command results (e.g., /model output).
	if strings.HasPrefix(trimmed, "<command-name>") {
		return true
	}
	if strings.HasPrefix(trimmed, "<local-command") {
		return true
	}
	// Task notifications from subagents.
	if strings.HasPrefix(trimmed, "<task-notification>") {
		return true
	}
	return false
}
