package intel

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/getopsdeck/opsdeck/internal/discovery"
)

// DayMetrics holds quantified productivity metrics for a single day.
type DayMetrics struct {
	Date           string // "2026-03-22"
	ActiveSessions int    // sessions with any activity
	TotalEdits     int    // file edits across all sessions
	TotalCommands  int    // bash commands across all sessions
	TotalErrors    int    // errors encountered
	FilesChanged   int    // unique files touched
	GitOps         int    // git operations (commits, pushes, PRs)
	Projects       int    // unique projects with activity
	// Per-project breakdown
	ProjectMetrics []ProjectDayMetrics
}

// ProjectDayMetrics holds metrics for a single project on a single day.
type ProjectDayMetrics struct {
	Name     string
	Edits    int
	Commands int
	Errors   int
	Files    int
	GitOps   int
	Sessions int // sessions with activity
}

// DayComparison compares two days of metrics.
type DayComparison struct {
	Today     DayMetrics
	Yesterday DayMetrics
	// Deltas (positive = improvement)
	EditsDelta     int
	CommandsDelta  int
	FilesDelta     int
	SessionsDelta  int
	ProjectsDelta  int
	ErrorsDelta    int
	// Overall trend
	Trend string // "more productive", "similar", "less productive"
}

// ComputeDayMetrics computes metrics for a specific day across all sessions.
func ComputeDayMetrics(projectsDir, sessionsDir string, day time.Time) DayMetrics {
	start := time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, day.Location())
	end := start.Add(24 * time.Hour)

	sessions, err := discovery.ScanSessions(sessionsDir)
	if err != nil {
		return DayMetrics{Date: start.Format("2006-01-02")}
	}

	allFiles := make(map[string]bool)
	projectMap := make(map[string]*ProjectDayMetrics)
	activeSessions := 0

	for _, s := range sessions {
		transcriptPath := discovery.FindTranscriptPath(projectsDir, s.CWD, s.ID)
		if transcriptPath == "" {
			continue
		}

		summary, err := ExtractRecent(transcriptPath, start)
		if err != nil || summary.TotalMessages == 0 {
			continue
		}

		// Filter to only activities within the day window.
		dayActivities := 0
		dayEdits := 0
		dayCommands := 0
		dayErrors := 0
		dayGitOps := 0
		dayFiles := make(map[string]bool)

		for _, a := range summary.Activities {
			if a.Timestamp.Before(start) || a.Timestamp.After(end) {
				continue
			}
			dayActivities++
			switch a.Type {
			case "file_edit":
				dayEdits++
			case "bash_command":
				dayCommands++
			case "git_op":
				dayGitOps++
			case "error":
				dayErrors++
			}
		}

		// Also count sessions with reads-only as active.
		hasActivity := dayActivities > 0 || summary.ReadCount > 0

		// Count files from the session's FilesChanged list.
		if hasActivity {
			activeSessions++
			for _, f := range summary.FilesChanged {
				allFiles[f] = true
			}

			proj := s.ProjectName
			if proj == "" {
				proj = "(unknown)"
			}
			pm, ok := projectMap[proj]
			if !ok {
				pm = &ProjectDayMetrics{Name: proj}
				projectMap[proj] = pm
			}
			pm.Edits += dayEdits
			pm.Commands += dayCommands
			pm.Errors += dayErrors
			pm.Files += len(dayFiles)
			pm.GitOps += dayGitOps
			pm.Sessions++
		}
	}

	// Build project list.
	projectMetrics := make([]ProjectDayMetrics, 0, len(projectMap))
	totalEdits := 0
	totalCommands := 0
	totalErrors := 0
	totalGitOps := 0
	for _, pm := range projectMap {
		projectMetrics = append(projectMetrics, *pm)
		totalEdits += pm.Edits
		totalCommands += pm.Commands
		totalErrors += pm.Errors
		totalGitOps += pm.GitOps
	}

	return DayMetrics{
		Date:           start.Format("2006-01-02"),
		ActiveSessions: activeSessions,
		TotalEdits:     totalEdits,
		TotalCommands:  totalCommands,
		TotalErrors:    totalErrors,
		FilesChanged:   len(allFiles),
		GitOps:         totalGitOps,
		Projects:       len(projectMap),
		ProjectMetrics: projectMetrics,
	}
}

// CompareDays computes metrics for today and yesterday and returns a comparison.
func CompareDays(projectsDir, sessionsDir string) DayComparison {
	now := time.Now()
	today := ComputeDayMetrics(projectsDir, sessionsDir, now)
	yesterday := ComputeDayMetrics(projectsDir, sessionsDir, now.Add(-24*time.Hour))

	editsDelta := today.TotalEdits - yesterday.TotalEdits
	commandsDelta := today.TotalCommands - yesterday.TotalCommands
	filesDelta := today.FilesChanged - yesterday.FilesChanged
	sessionsDelta := today.ActiveSessions - yesterday.ActiveSessions
	projectsDelta := today.Projects - yesterday.Projects
	errorsDelta := today.TotalErrors - yesterday.TotalErrors

	// Simple trend calculation based on edits + commands.
	todayScore := today.TotalEdits + today.TotalCommands
	yesterdayScore := yesterday.TotalEdits + yesterday.TotalCommands

	trend := "similar"
	if yesterdayScore == 0 && todayScore > 0 {
		trend = "more productive"
	} else if yesterdayScore > 0 {
		ratio := float64(todayScore) / float64(yesterdayScore)
		if ratio > 1.2 {
			trend = "more productive"
		} else if ratio < 0.8 {
			trend = "less productive"
		}
	}

	return DayComparison{
		Today:         today,
		Yesterday:     yesterday,
		EditsDelta:    editsDelta,
		CommandsDelta: commandsDelta,
		FilesDelta:    filesDelta,
		SessionsDelta: sessionsDelta,
		ProjectsDelta: projectsDelta,
		ErrorsDelta:   errorsDelta,
		Trend:         trend,
	}
}

// FormatComparison renders a day comparison as a readable report.
func FormatComparison(cmp DayComparison) string {
	var b strings.Builder

	b.WriteString("=== OpsDeck Productivity Report ===\n")
	b.WriteString(fmt.Sprintf("Today (%s) vs Yesterday (%s)\n\n", cmp.Today.Date, cmp.Yesterday.Date))

	// Trend headline with context.
	today := time.Now()
	isWeekend := today.Weekday() == time.Saturday || today.Weekday() == time.Sunday
	yesterdayOutlier := cmp.Yesterday.TotalEdits > 3*cmp.Today.TotalEdits && cmp.Yesterday.TotalEdits > 50

	switch cmp.Trend {
	case "more productive":
		b.WriteString("Trend: UP -- You're more productive than yesterday.\n\n")
	case "less productive":
		reason := ""
		if isWeekend {
			reason = " (weekend)"
		} else if yesterdayOutlier {
			reason = " (yesterday was unusually active)"
		}
		b.WriteString(fmt.Sprintf("Trend: DOWN -- Less activity than yesterday%s.\n\n", reason))
	default:
		b.WriteString("Trend: STEADY -- Similar activity level.\n\n")
	}

	// Comparison table.
	b.WriteString(fmt.Sprintf("%-20s %10s %10s %10s\n", "Metric", "Today", "Yesterday", "Delta"))
	b.WriteString(strings.Repeat("-", 52) + "\n")

	writeRow := func(name string, today, yesterday, delta int) {
		deltaStr := formatDelta(delta)
		b.WriteString(fmt.Sprintf("%-20s %10d %10d %10s\n", name, today, yesterday, deltaStr))
	}

	writeRow("Active sessions", cmp.Today.ActiveSessions, cmp.Yesterday.ActiveSessions, cmp.SessionsDelta)
	writeRow("File edits", cmp.Today.TotalEdits, cmp.Yesterday.TotalEdits, cmp.EditsDelta)
	writeRow("Commands run", cmp.Today.TotalCommands, cmp.Yesterday.TotalCommands, cmp.CommandsDelta)
	writeRow("Files changed", cmp.Today.FilesChanged, cmp.Yesterday.FilesChanged, cmp.FilesDelta)
	writeRow("Projects active", cmp.Today.Projects, cmp.Yesterday.Projects, cmp.ProjectsDelta)
	writeRow("Errors", cmp.Today.TotalErrors, cmp.Yesterday.TotalErrors, -cmp.ErrorsDelta) // fewer errors = positive

	// Per-project today breakdown, sorted by activity.
	if len(cmp.Today.ProjectMetrics) > 0 {
		sorted := make([]ProjectDayMetrics, len(cmp.Today.ProjectMetrics))
		copy(sorted, cmp.Today.ProjectMetrics)
		sort.Slice(sorted, func(i, j int) bool {
			return sorted[i].Edits+sorted[i].Commands > sorted[j].Edits+sorted[j].Commands
		})
		b.WriteString("\nToday by project:\n")
		for _, pm := range sorted {
			b.WriteString(fmt.Sprintf("  %-20s %d edits, %d commands, %d files\n",
				pm.Name, pm.Edits, pm.Commands, pm.Files))
		}
	}

	return b.String()
}

// formatDelta formats an integer delta with +/- prefix.
func formatDelta(d int) string {
	if d > 0 {
		return fmt.Sprintf("+%d", d)
	}
	if d < 0 {
		return fmt.Sprintf("%d", d)
	}
	return "0"
}

// CalculateMomentum computes the Momentum score from Codex's recommendation.
// Four components: Outcome, Validation, Flow, Drag.
func CalculateMomentum(metrics DayMetrics) (score float64, components map[string]float64) {
	components = make(map[string]float64)

	// Outcome: artifacts produced (edits, git ops).
	outcome := float64(metrics.TotalEdits)*1.0 + float64(metrics.GitOps)*5.0
	outcome = math.Min(outcome, 100) // cap at 100
	components["outcome"] = outcome

	// Validation: commands run (often tests).
	validation := float64(metrics.TotalCommands) * 2.0
	validation = math.Min(validation, 100)
	components["validation"] = validation

	// Flow: sessions active, projects engaged.
	flow := float64(metrics.ActiveSessions)*10.0 + float64(metrics.Projects)*15.0
	flow = math.Min(flow, 100)
	components["flow"] = flow

	// Drag: errors reduce the score.
	drag := float64(metrics.TotalErrors) * 5.0
	drag = math.Min(drag, 100)
	components["drag"] = drag

	// Weighted composite: outcome 40%, validation 25%, flow 20%, drag 15% (subtracted).
	score = outcome*0.4 + validation*0.25 + flow*0.2 - drag*0.15
	if score < 0 {
		score = 0
	}
	score = math.Min(score, 100)

	return score, components
}

// FormatMomentum renders the Momentum score as a brief display.
func FormatMomentum(score float64, components map[string]float64) string {
	bar := renderBar(score, 20)
	return fmt.Sprintf("Momentum: %.0f/100 %s\n  Outcome: %.0f | Validation: %.0f | Flow: %.0f | Drag: -%.0f",
		score, bar,
		components["outcome"], components["validation"],
		components["flow"], components["drag"])
}

// renderBar creates a simple ASCII progress bar.
func renderBar(value float64, width int) string {
	filled := int(value / 100.0 * float64(width))
	if filled > width {
		filled = width
	}
	if filled < 0 {
		filled = 0
	}
	return "[" + strings.Repeat("#", filled) + strings.Repeat("-", width-filled) + "]"
}

// RunMetricsReport generates and prints the full metrics comparison report.
func RunMetricsReport() {
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return
	}

	sessionsDir := filepath.Join(home, ".claude", "sessions")
	projectsDir := filepath.Join(home, ".claude", "projects")

	cmp := CompareDays(projectsDir, sessionsDir)
	fmt.Print(FormatComparison(cmp))

	fmt.Println()
	todayScore, todayComponents := CalculateMomentum(cmp.Today)
	fmt.Println(FormatMomentum(todayScore, todayComponents))

	if cmp.Yesterday.TotalEdits+cmp.Yesterday.TotalCommands > 0 {
		yesterdayScore, _ := CalculateMomentum(cmp.Yesterday)
		delta := todayScore - yesterdayScore
		fmt.Printf("\nYesterday: %.0f/100 | Change: %s\n", yesterdayScore, formatDelta(int(delta)))
	}
}
