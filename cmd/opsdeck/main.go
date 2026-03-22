package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime/debug"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/getopsdeck/opsdeck/internal/discovery"
	"github.com/getopsdeck/opsdeck/internal/intel"
	"github.com/getopsdeck/opsdeck/internal/monitor"
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
		case "watch":
			runWatch()
			return
		case "list", "ls":
			runList()
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

	app := tui.NewApp()
	p := tea.NewProgram(app)
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// If the user pressed R to resume a session, launch it.
	if app.ResumeSessionID != "" {
		os.Args = []string{"opsdeck", "resume", app.ResumeSessionID}
		runResume()
	}
}

func printHelp() {
	fmt.Printf(`opsdeck %s — Chief of Staff for Claude Code

Usage: opsdeck [command]

Commands:
  (default)    Real-time TUI dashboard
  brief        Daily briefing across all projects
  metrics      Today vs yesterday productivity comparison
  list         Compact list of all sessions with state and branch
  costs        Token usage and estimated spend per session
  ai-brief     AI-powered morning brief via claude -p (costs tokens)
  resume <id>  Resume a Claude Code session (supports prefix match)
  watch        Monitor sessions, alert on state changes (macOS notifications)
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

// runWatch monitors sessions and alerts when any need attention.
// It prints a notification whenever a session transitions to WAITING state.
func runWatch() {
	home, _ := os.UserHomeDir()
	sessionsDir := filepath.Join(home, ".claude", "sessions")
	projectsDir := filepath.Join(home, ".claude", "projects")

	fmt.Println("Watching sessions... (Ctrl+C to stop)")

	lastStates := make(map[string]string) // session ID → last known state
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	// Immediate first check.
	checkSessions(sessionsDir, projectsDir, lastStates)

	for range ticker.C {
		checkSessions(sessionsDir, projectsDir, lastStates)
	}
}

func checkSessions(sessionsDir, projectsDir string, lastStates map[string]string) {
	sessions, err := discovery.ScanSessions(sessionsDir)
	if err != nil {
		return
	}

	for _, rs := range sessions {
		alive := discovery.CheckSession(rs.PID, rs.StartedAt)
		transcriptPath := discovery.FindTranscriptPath(projectsDir, rs.CWD, rs.ID)
		var lastActivity = rs.StartedAt
		if transcriptPath != "" {
			if t, err := discovery.ReadLastActivity(transcriptPath); err == nil && !t.IsZero() {
				lastActivity = t
			}
		}
		state := string(discovery.ClassifyState(alive, lastActivity))

		prev, seen := lastStates[rs.ID]
		lastStates[rs.ID] = state

		if !seen {
			continue // First observation — don't alert.
		}

		if state != prev {
			now := time.Now().Format("15:04:05")
			id := rs.ID
			if len(id) > 12 {
				id = id[:12]
			}

			switch state {
			case "waiting":
				// Try to get the last assistant message to show what it's waiting for.
				hint := ""
				if transcriptPath != "" {
					hint = monitor.ReadLastMeaningfulLine(transcriptPath)
					if len([]rune(hint)) > 60 {
						hint = string([]rune(hint)[:57]) + "..."
					}
				}
				if hint != "" {
					fmt.Printf("[%s] ◐ %s (%s) is now WAITING — %s\n", now, rs.ProjectName, id, hint)
				} else {
					fmt.Printf("[%s] ◐ %s (%s) is now WAITING — needs your attention\n", now, rs.ProjectName, id)
				}
				notifyMac(rs.ProjectName + " needs attention", "Session "+id+": "+hint)
			case "dead":
				fmt.Printf("[%s] ✕ %s (%s) session DIED\n", now, rs.ProjectName, id)
			case "busy":
				if prev == "waiting" {
					fmt.Printf("[%s] ● %s (%s) resumed — now BUSY\n", now, rs.ProjectName, id)
				}
			}
		}
	}
}

// notifyMac sends a macOS notification via osascript. Silent failure on Linux.
func notifyMac(title, message string) {
	exec.Command("osascript", "-e",
		fmt.Sprintf(`display notification "%s" with title "%s"`, message, title),
	).Run()
}

// runList prints a compact list of all sessions with state and project.
func runList() {
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	sessionsDir := filepath.Join(home, ".claude", "sessions")
	projectsDir := filepath.Join(home, ".claude", "projects")
	sessions, err := discovery.ScanSessions(sessionsDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	projects := discovery.GroupByProject(sessions)

	fmt.Printf("%-14s %-12s %-8s %-18s %s\n", "SESSION", "PROJECT", "STATE", "BRANCH", "STARTED")
	fmt.Println(strings.Repeat("-", 72))

	for _, p := range projects {
		for _, s := range p.Sessions {
			alive := discovery.CheckSession(s.PID, s.StartedAt)
			transcriptPath := discovery.FindTranscriptPath(projectsDir, s.CWD, s.ID)
			var lastActivity = s.StartedAt
			if transcriptPath != "" {
				if t, err := discovery.ReadLastActivity(transcriptPath); err == nil && !t.IsZero() {
					lastActivity = t
				}
			}
			state := discovery.ClassifyState(alive, lastActivity)

			gi := discovery.GetGitInfo(s.CWD)
			branch := gi.Branch
			if gi.IsDirty && branch != "" {
				branch += "*"
			}
			if branch == "" {
				branch = "-"
			}

			id := s.ID
			if len(id) > 12 {
				id = id[:12]
			}

			icon := "○"
			switch state {
			case discovery.StateBusy:
				icon = "●"
			case discovery.StateWaiting:
				icon = "◐"
			case discovery.StateDead:
				icon = "✕"
			}

			// For WAITING sessions, show how long they've been waiting.
			extra := s.StartedAt.Format("Jan 02 15:04")
			if state == discovery.StateWaiting {
				waitDur := time.Since(lastActivity)
				if waitDur.Hours() >= 24 {
					extra = fmt.Sprintf("waiting %.0fd", waitDur.Hours()/24)
				} else if waitDur.Hours() >= 1 {
					extra = fmt.Sprintf("waiting %.0fh", waitDur.Hours())
				} else {
					extra = fmt.Sprintf("waiting %.0fm", waitDur.Minutes())
				}
			}

			fmt.Printf("%s %-12s %-12s %-8s %-18s %s\n",
				icon, id, p.Name, string(state), branch, extra)
		}
	}

	fmt.Printf("\n%d sessions across %d projects\n", len(sessions), len(projects))
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
