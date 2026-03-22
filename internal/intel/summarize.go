package intel

import (
	"fmt"
	"strings"
)

// SummarizeActivities condenses a list of raw activities into at most 5
// human-readable bullet points suitable for an executive summary.
//
// The algorithm:
//  1. User requests appear verbatim (they are what the user asked for).
//  2. File edits are grouped: "Edited N files (a.go, b.go, ...)"
//  3. Bash commands are grouped: "Ran N commands"
//  4. Git operations are grouped: "Committed and pushed changes" / "Created PR"
//  5. Errors are grouped: "Encountered N errors"
//  6. Tool calls (Grep, Glob, etc.) are omitted -- they are noise.
//
// Results are capped at 5 items with user requests taking priority.
func SummarizeActivities(activities []Activity) []string {
	if len(activities) == 0 {
		return nil
	}

	var userRequests []string
	var editDescs []string
	var bashDescs []string
	var gitDescs []string
	var errorCount int
	var toolCallCount int

	for _, a := range activities {
		switch a.Type {
		case ActivityUserRequest:
			userRequests = append(userRequests, a.Description)
		case ActivityFileEdit:
			editDescs = append(editDescs, a.Description)
		case ActivityBashCommand:
			bashDescs = append(bashDescs, a.Description)
		case ActivityGitOp:
			gitDescs = append(gitDescs, a.Description)
		case ActivityError:
			errorCount++
		case ActivityToolCall:
			// Omit tool calls from summary -- they are internal noise.
			toolCallCount++
		}
	}

	var result []string

	// User requests go first -- they are the "what".
	for _, req := range userRequests {
		result = append(result, req)
	}

	// Grouped edits.
	if len(editDescs) > 0 {
		result = append(result, summarizeEdits(editDescs))
	}

	// Grouped bash commands.
	if len(bashDescs) > 0 {
		result = append(result, summarizeBash(bashDescs))
	}

	// Grouped git operations.
	if len(gitDescs) > 0 {
		result = append(result, summarizeGit(gitDescs))
	}

	// Errors.
	if errorCount > 0 {
		if errorCount == 1 {
			result = append(result, "Encountered 1 error")
		} else {
			result = append(result, fmt.Sprintf("Encountered %d errors", errorCount))
		}
	}

	// Fallback: if the session had only tool calls (Grep/Glob/Agent) and nothing
	// else generated a summary line, show a minimal activity label instead of
	// returning nothing (which would render as "No activity").
	if len(result) == 0 && toolCallCount > 0 {
		result = append(result, fmt.Sprintf("Explored codebase (%d tool calls)", toolCallCount))
	}

	// Cap at 5.
	if len(result) > 5 {
		result = result[:5]
	}

	return result
}

// summarizeEdits condenses edit activities into a single line.
// The count reflects unique files, not raw edit operations.
func summarizeEdits(descs []string) string {
	// Extract unique file names from descriptions like "Edited foo.go" / "Wrote bar.go".
	seen := make(map[string]struct{})
	var unique []string
	for _, d := range descs {
		name := d
		for _, prefix := range []string{"Edited ", "Wrote "} {
			if strings.HasPrefix(d, prefix) {
				name = strings.TrimPrefix(d, prefix)
				break
			}
		}
		if _, ok := seen[name]; !ok {
			seen[name] = struct{}{}
			unique = append(unique, name)
		}
	}

	n := len(unique)
	if n <= 3 {
		return fmt.Sprintf("Edited %d files (%s)", n, strings.Join(unique, ", "))
	}
	shown := strings.Join(unique[:2], ", ")
	return fmt.Sprintf("Edited %d files (%s, +%d more)", n, shown, n-2)
}

// summarizeBash condenses bash command activities into a single line.
func summarizeBash(descs []string) string {
	n := len(descs)
	if n == 1 {
		return fmt.Sprintf("Ran 1 command: %s", descs[0])
	}
	return fmt.Sprintf("Ran %d commands", n)
}

// summarizeGit condenses git operations into a single human-readable line.
func summarizeGit(descs []string) string {
	hasCommit := false
	hasPush := false
	hasPR := false
	for _, d := range descs {
		dl := strings.ToLower(d)
		if strings.Contains(dl, "commit") {
			hasCommit = true
		}
		if strings.Contains(dl, "push") {
			hasPush = true
		}
		// Use precise patterns to avoid false positives from words containing
		// the substring "pr" (e.g. "sprint", "approve", "compress").
		if strings.Contains(dl, "gh pr create") || strings.Contains(dl, "pull request") || strings.Contains(dl, "create pr") {
			hasPR = true
		}
	}

	var parts []string
	if hasCommit {
		parts = append(parts, "committed")
	}
	if hasPush {
		parts = append(parts, "pushed")
	}
	if hasPR {
		parts = append(parts, "created PR")
	}
	if len(parts) == 0 {
		return fmt.Sprintf("Performed %d git operations", len(descs))
	}

	summary := strings.Join(parts, ", ")
	// Capitalize first letter.
	if len(summary) > 0 {
		summary = strings.ToUpper(summary[:1]) + summary[1:]
	}
	return summary
}
