package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime/debug"
	"sort"
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
		case "status":
			runStatus()
			return
		case "list", "ls":
			runList()
			return
		case "resume":
			runResume()
			return
		case "export":
			runExport()
			return
		case "clean":
		runClean()
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
		default:
			fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", os.Args[1])
			printHelp()
			os.Exit(1)
		}
	}

	app := tui.NewApp()
	p := tea.NewProgram(app)
	if _, err := p.Run(); err != nil {
		if strings.Contains(err.Error(), "TTY") {
			fmt.Fprintln(os.Stderr, "Error: TUI requires a terminal. Try one of:")
			fmt.Fprintln(os.Stderr, "  opsdeck web     — browser dashboard")
			fmt.Fprintln(os.Stderr, "  opsdeck list    — compact session list")
			fmt.Fprintln(os.Stderr, "  opsdeck brief   — daily briefing")
			fmt.Fprintln(os.Stderr, "  opsdeck help    — all commands")
		} else {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		}
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
  status       One-line summary (for shell prompts / tmux status bars)
  list         Compact list of all sessions with state and branch
  costs        Token usage and estimated spend per session
  ai-brief     AI-powered morning brief via claude -p (costs tokens)
  clean        Show dead sessions that can be cleaned up
  export       Export daily brief to markdown file
  resume <id>  Resume a Claude Code session (supports prefix match)
  watch        Monitor sessions, alert on state changes (macOS notifications)
  web [addr]   Web dashboard at addr (default: localhost:7070)
  version      Show version information
  help         Show this help message

Flags:
  brief --since <duration>   Only include activity from last N hours (e.g. 2h, 48h)
  brief --verbose (-V)       Use detailed format instead of secretary brief
  list --json                Output session data as JSON for scripting
  list --active (-a)         Show only busy and waiting sessions
  costs --since <duration>   Only include costs from last N hours

OpsDeck reads Claude Code session data from ~/.claude/ and is strictly
read-only — it never modifies your sessions. All data stays local
(except ai-brief, which pipes data through claude -p).

GitHub: https://github.com/getopsdeck/opsdeck
`, version)
}

// runWatch monitors sessions and alerts when any need attention.
// It prints a notification whenever a session transitions to WAITING state.
func runWatch() {
	home, _ := os.UserHomeDir()
	sessionsDir := filepath.Join(home, ".claude", "sessions")
	projectsDir := filepath.Join(home, ".claude", "projects")

	// Show current state summary before watching.
	sessions, _ := discovery.ScanSessions(sessionsDir)
	busy, waiting, idle, dead := 0, 0, 0, 0
	for _, s := range sessions {
		alive := discovery.CheckSession(s.PID, s.StartedAt)
		tp := discovery.FindTranscriptPath(projectsDir, s.CWD, s.ID)
		la := s.StartedAt
		if tp != "" {
			if t, err := discovery.ReadLastActivity(tp); err == nil && !t.IsZero() {
				la = t
			}
		}
		switch discovery.ClassifyState(alive, la) {
		case discovery.StateBusy:
			busy++
		case discovery.StateWaiting:
			waiting++
		case discovery.StateIdle:
			idle++
		case discovery.StateDead:
			dead++
		}
	}
	fmt.Printf("Watching %d sessions (● %d busy, ◐ %d waiting, ○ %d idle, ✕ %d dead)\n", len(sessions), busy, waiting, idle, dead)
	fmt.Println("Alerts on state changes. Ctrl+C to stop.")

	lastStates := make(map[string]string)
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	// Immediate first check (populates lastStates without alerting).
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
		fmt.Sprintf(`display notification "%s" with title "%s" sound name "Submarine"`, message, title),
	).Run()
}

// runStatus prints a one-line summary suitable for shell prompts or status bars.
func runStatus() {
	home, _ := os.UserHomeDir()
	sessionsDir := filepath.Join(home, ".claude", "sessions")
	projectsDir := filepath.Join(home, ".claude", "projects")

	sessions, _ := discovery.ScanSessions(sessionsDir)
	busy, waiting, idle := 0, 0, 0
	var topWait string
	var topWaitDur time.Duration

	for _, s := range sessions {
		alive := discovery.CheckSession(s.PID, s.StartedAt)
		tp := discovery.FindTranscriptPath(projectsDir, s.CWD, s.ID)
		la := s.StartedAt
		if tp != "" {
			if t, err := discovery.ReadLastActivity(tp); err == nil && !t.IsZero() {
				la = t
			}
		}
		state := discovery.ClassifyState(alive, la)
		switch state {
		case discovery.StateBusy:
			busy++
		case discovery.StateWaiting:
			waiting++
			d := time.Since(la)
			if d > topWaitDur {
				topWaitDur = d
				topWait = s.ProjectName
			}
		case discovery.StateIdle:
			idle++
		}
	}

	parts := []string{fmt.Sprintf("● %d busy", busy)}
	if waiting > 0 {
		parts = append(parts, fmt.Sprintf("◐ %d waiting", waiting))
	}
	parts = append(parts, fmt.Sprintf("○ %d idle", idle))

	line := strings.Join(parts, " | ")

	if topWait != "" {
		var waitStr string
		if topWaitDur.Hours() >= 24 {
			waitStr = fmt.Sprintf("%.0fd", topWaitDur.Hours()/24)
		} else if topWaitDur.Minutes() >= 60 {
			waitStr = fmt.Sprintf("%.0fh", topWaitDur.Hours())
		} else {
			waitStr = fmt.Sprintf("%.0fm", topWaitDur.Minutes())
		}
		line += fmt.Sprintf(" | %s waiting %s", topWait, waitStr)
	}

	fmt.Println(line)
}

// isTTY reports whether stdout is connected to a terminal.
func isTTY() bool {
	fi, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}

// runList prints a compact list of all sessions with state and project.
func runList() {
	// Check for --json flag.
	jsonOutput := false
	for _, arg := range os.Args[2:] {
		if arg == "--json" {
			jsonOutput = true
		}
	}

	useColor := isTTY()
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	sessionsDir := filepath.Join(home, ".claude", "sessions")
	projectsDir := filepath.Join(home, ".claude", "projects")

	// Use monitor.Snapshot for enriched data (same as web/TUI).
	enriched := monitor.Snapshot(sessionsDir, projectsDir)

	if jsonOutput {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		enc.Encode(enriched)
		return
	}

	if len(enriched) == 0 {
		fmt.Println("No Claude Code sessions found.")
		fmt.Println("Start a session with 'claude' and OpsDeck will detect it automatically.")
		return
	}

	// Check for --active flag: only show busy/waiting.
	activeOnly := false
	for _, arg := range os.Args[2:] {
		if arg == "--active" || arg == "-a" {
			activeOnly = true
		}
	}

	// Sort: busy first, then waiting, then idle, then dead.
	statePriority := map[string]int{"busy": 0, "waiting": 1, "idle": 2, "dead": 3}
	sort.Slice(enriched, func(i, j int) bool {
		pi := statePriority[enriched[i].State]
		pj := statePriority[enriched[j].State]
		if pi != pj {
			return pi < pj
		}
		return enriched[i].Project < enriched[j].Project
	})

	fmt.Printf("%-14s %-12s %-8s %-18s %s\n", "SESSION", "PROJECT", "STATE", "BRANCH", "STARTED")
	fmt.Println(strings.Repeat("-", 72))

	displayed := 0
	for _, s := range enriched {
		if activeOnly && s.State != "busy" && s.State != "waiting" {
			continue
		}

		id := s.ID
		if len(id) > 12 {
			id = id[:12]
		}

		branch := s.GitBranch
		if s.GitDirty && branch != "" {
			branch += "*"
		}
		if branch == "" {
			branch = "-"
		}

		icon := "○"
		color := ""
		reset := ""
		if useColor {
			reset = "\033[0m"
			switch s.State {
			case "busy":
				color = "\033[32m"
			case "waiting":
				color = "\033[33m"
			case "idle":
				color = "\033[90m"
			case "dead":
				color = "\033[31m"
			}
		}
		switch s.State {
		case "busy":
			icon = "●"
		case "waiting":
			icon = "◐"
		case "dead":
			icon = "✕"
		}

		extra := s.StartedAt.Format("Jan 02 15:04")
		if s.State == "waiting" {
			// Show how long waiting. LastLine has the timestamp context.
			// Use TranscriptPath to get last activity.
			if s.TranscriptPath != "" {
				if la, err := discovery.ReadLastActivity(s.TranscriptPath); err == nil && !la.IsZero() {
					waitDur := time.Since(la)
					if waitDur.Hours() >= 24 {
						extra = fmt.Sprintf("waiting %.0fd", waitDur.Hours()/24)
					} else if waitDur.Minutes() >= 60 {
						extra = fmt.Sprintf("waiting %.0fh", waitDur.Hours())
					} else {
						extra = fmt.Sprintf("waiting %.0fm", waitDur.Minutes())
					}
				}
			}
		}

		fmt.Printf("%s%s %-12s %-12s %-8s %-18s %s%s\n",
			color, icon, id, s.Project, s.State, branch, extra, reset)
		displayed++
	}

	fmt.Printf("\n%d sessions (%d shown)\n", len(enriched), displayed)
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
	// Parse flags from remaining args.
	sinceValue := ""
	verbose := false
	for i := 2; i < len(os.Args); i++ {
		switch {
		case os.Args[i] == "--since" && i+1 < len(os.Args):
			sinceValue = os.Args[i+1]
		case len(os.Args[i]) > 8 && os.Args[i][:8] == "--since=":
			sinceValue = os.Args[i][8:]
		case os.Args[i] == "--verbose" || os.Args[i] == "-V":
			verbose = true
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

	// Enrich brief with cost data and per-session details for the secretary format.
	if !verbose {
		intel.EnrichBrief(&brief, projectsDir, sessionsDir, since)
	}

	if verbose {
		fmt.Print(intel.FormatDailyBriefVerbose(brief))
	} else {
		fmt.Print(intel.FormatDailyBrief(brief))
	}
}

// runClean lists dead sessions that could be cleaned up. It is strictly
// read-only and never deletes anything.
func runClean() {
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	sessionsDir := filepath.Join(home, ".claude", "sessions")
	projectsDir := filepath.Join(home, ".claude", "projects")

	enriched := monitor.Snapshot(sessionsDir, projectsDir)

	var dead []monitor.Session
	for _, s := range enriched {
		if s.State == "dead" {
			dead = append(dead, s)
		}
	}

	if len(dead) == 0 {
		fmt.Println("No dead sessions found.")
		return
	}

	// Sort dead sessions by start time, oldest first.
	sort.Slice(dead, func(i, j int) bool {
		return dead[i].StartedAt.Before(dead[j].StartedAt)
	})

	fmt.Printf("%d dead session(s) found:\n\n", len(dead))
	fmt.Printf("%-36s %-20s %s\n", "SESSION ID", "PROJECT", "STARTED")
	fmt.Println(strings.Repeat("-", 72))
	for _, s := range dead {
		fmt.Printf("%-36s %-20s %s\n", s.ID, s.Project, s.StartedAt.Format("2006-01-02 15:04"))
	}
	fmt.Println()
	fmt.Println("To clean up, remove session files from ~/.claude/sessions/")
}

// runExport generates a daily brief and writes it to a markdown file in the
// current directory named opsdeck-brief-YYYY-MM-DD.md.
func runExport() {
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: cannot determine home directory: %v\n", err)
		os.Exit(1)
	}

	sessionsDir := filepath.Join(home, ".claude", "sessions")
	projectsDir := filepath.Join(home, ".claude", "projects")

	// Use a 24-hour window, matching the default brief behaviour.
	since, _ := intel.ParseSinceFlag("24h")

	brief, err := intel.GenerateBrief(projectsDir, sessionsDir, since)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error generating brief: %v\n", err)
		os.Exit(1)
	}

	intel.EnrichBrief(&brief, projectsDir, sessionsDir, since)

	content := intel.FormatDailyBrief(brief)

	filename := fmt.Sprintf("opsdeck-brief-%s.md", time.Now().Format("2006-01-02"))
	if err := os.WriteFile(filename, []byte(content), 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Brief exported to %s\n", filename)
}
