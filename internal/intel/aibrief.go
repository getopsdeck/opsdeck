package intel

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

const aiBriefPrompt = `You are OpsDeck, a Chief of Staff for Claude Code sessions. Below is structured session data from the last 24 hours across all projects. Write a concise morning brief (3-5 paragraphs) for a busy CEO who manages multiple projects.

Focus on:
1. What was accomplished (edits, PRs, deployments)
2. What needs attention right now (errors, stuck sessions, waiting states)
3. A momentum assessment (more or less productive than usual?)
4. Top priority for today

Be direct, specific, and actionable. Use project names and numbers. No fluff.

Here is the data:
`

// RunAIBrief generates a structured brief, then uses claude -p to produce
// a natural language summary. This is opt-in because it costs tokens.
func RunAIBrief() {
	// Check if claude CLI is available.
	claudePath, err := exec.LookPath("claude")
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error: 'claude' CLI not found in PATH. AI brief requires Claude Code installed.")
		os.Exit(1)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	sessionsDir := filepath.Join(home, ".claude", "sessions")
	projectsDir := filepath.Join(home, ".claude", "projects")

	since := time.Now().Add(-24 * time.Hour)
	brief, err := GenerateBrief(projectsDir, sessionsDir, since)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error generating brief: %v\n", err)
		os.Exit(1)
	}

	structuredBrief := FormatDailyBrief(brief)

	// Also include cost data.
	costReport, _ := GenerateCostReport(projectsDir, sessionsDir, since)
	costText := FormatCostReport(costReport)

	// Build the prompt.
	prompt := aiBriefPrompt + structuredBrief + "\n\nCost data:\n" + costText

	// Run claude -p with the prompt.
	fmt.Fprintln(os.Stderr, "Generating AI brief (this costs tokens)...")

	cmd := exec.Command(claudePath, "-p", prompt)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Pass through environment but disable any interactive features.
	cmd.Env = append(os.Environ(), "CLAUDE_CODE_ENTRYPOINT=cli")

	if err := cmd.Run(); err != nil {
		// If claude -p fails, fall back to structured brief.
		fmt.Fprintln(os.Stderr, "\nAI summary failed, showing structured brief instead:")
		fmt.Print(structuredBrief)
	}
}

