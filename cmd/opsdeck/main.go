package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime/debug"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/getopsdeck/opsdeck/internal/discovery"
	"github.com/getopsdeck/opsdeck/internal/intel"
	"github.com/getopsdeck/opsdeck/internal/tui"
	"github.com/getopsdeck/opsdeck/internal/web"
)

// Set via ldflags at build time.
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func init() {
	// When installed via `go install`, ldflags are not set.
	// Fall back to build info embedded by the Go toolchain.
	if version == "dev" {
		if info, ok := debug.ReadBuildInfo(); ok {
			if info.Main.Version != "" && info.Main.Version != "(devel)" {
				version = info.Main.Version
			}
			for _, s := range info.Settings {
				if s.Key == "vcs.revision" && len(s.Value) >= 7 {
					commit = s.Value[:7]
				}
				if s.Key == "vcs.time" {
					date = s.Value
				}
			}
		}
	}
}

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "version", "--version", "-v":
			fmt.Printf("opsdeck %s (commit %s, built %s)\n", version, commit, date)
			return
		case "help", "--help", "-h":
			printHelp()
			return
		case "brief":
			runBrief()
			return
		case "metrics":
			intel.RunMetricsReport()
			return
		case "costs":
			intel.RunCostReport()
			return
		case "ai-brief":
			intel.RunAIBrief()
			return
		case "resume":
			runResume()
			return
		case "web", "serve":
			addr := "localhost:7070"
			if len(os.Args) > 2 {
				addr = os.Args[2]
			}
			srv := web.NewServer(addr)
			if err := srv.Start(); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			return
		}
	}

	p := tea.NewProgram(tui.NewApp())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func printHelp() {
	fmt.Printf(`opsdeck %s — Chief of Staff for Claude Code

Usage: opsdeck [command]

Commands:
  (default)    Real-time TUI dashboard
  brief        Daily briefing across all projects
  metrics      Today vs yesterday productivity comparison
  costs        Token usage and estimated spend per session
  ai-brief     AI-powered morning brief via claude -p (costs tokens)
  resume <id>  Resume a Claude Code session (supports prefix match)
  web [addr]   Web dashboard at addr (default: localhost:7070)
  version      Show version information
  help         Show this help message

Flags:
  brief --since <duration>   Only include activity from last N hours (e.g. 2h, 48h)
  costs --since <duration>   Only include costs from last N hours

OpsDeck reads Claude Code session data from ~/.claude/ and is strictly
read-only — it never modifies your sessions or sends data anywhere.

GitHub: https://github.com/getopsdeck/opsdeck
`, version)
}

// runResume opens a Claude Code session by ID. It finds the session's working
// directory and launches `claude --resume <id>` there.
func runResume() {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "Usage: opsdeck resume <session-id>")
		fmt.Fprintln(os.Stderr, "  Tip: use opsdeck brief or opsdeck web to find session IDs")
		os.Exit(1)
	}

	sessionID := os.Args[2]
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	sessionsDir := filepath.Join(home, ".claude", "sessions")
	sessions, err := discovery.ScanSessions(sessionsDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error scanning sessions: %v\n", err)
		os.Exit(1)
	}

	// Find matching session (supports prefix match).
	var match *discovery.Session
	for i := range sessions {
		if sessions[i].ID == sessionID || strings.HasPrefix(sessions[i].ID, sessionID) {
			if match != nil {
				fmt.Fprintf(os.Stderr, "Error: ambiguous session ID prefix %q — matches multiple sessions\n", sessionID)
				os.Exit(1)
			}
			match = &sessions[i]
		}
	}

	if match == nil {
		fmt.Fprintf(os.Stderr, "Error: session %q not found\n", sessionID)
		os.Exit(1)
	}

	claudePath, err := exec.LookPath("claude")
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error: 'claude' CLI not found in PATH")
		os.Exit(1)
	}

	fmt.Printf("Resuming session %s in %s\n", match.ID, match.CWD)

	cmd := exec.Command(claudePath, "--resume", match.ID)
	cmd.Dir = match.CWD
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
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
