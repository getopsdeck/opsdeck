package main

import (
	"fmt"
	"os"
	"path/filepath"

	tea "charm.land/bubbletea/v2"
	"github.com/getopsdeck/opsdeck/internal/intel"
	"github.com/getopsdeck/opsdeck/internal/tui"
)

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "brief":
			runBrief()
			return
		case "metrics":
			intel.RunMetricsReport()
			return
		}
	}

	p := tea.NewProgram(tui.NewApp())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// runBrief handles the "opsdeck brief" subcommand. It generates a daily brief
// and prints it to stdout, suitable for scripts and cron jobs.
func runBrief() {
	// Parse --since flag from remaining args.
	sinceValue := ""
	for i := 2; i < len(os.Args); i++ {
		if os.Args[i] == "--since" && i+1 < len(os.Args) {
			sinceValue = os.Args[i+1]
			break
		}
		// Also support --since=value form.
		if len(os.Args[i]) > 8 && os.Args[i][:8] == "--since=" {
			sinceValue = os.Args[i][8:]
			break
		}
	}

	since, err := intel.ParseSinceFlag(sinceValue)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: cannot determine home directory: %v\n", err)
		os.Exit(1)
	}

	sessionsDir := filepath.Join(home, ".claude", "sessions")
	projectsDir := filepath.Join(home, ".claude", "projects")

	brief, err := intel.GenerateBrief(projectsDir, sessionsDir, since)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error generating brief: %v\n", err)
		os.Exit(1)
	}

	fmt.Print(intel.FormatDailyBrief(brief))
}
