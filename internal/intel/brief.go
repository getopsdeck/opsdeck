package intel

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/getopsdeck/opsdeck/internal/discovery"
)

// ProjectBrief summarizes activity for a single project over a time period.
type ProjectBrief struct {
	Name            string
	Path            string
	SessionCount    int
	ActiveCount     int // sessions with activity in period
	TotalEdits      int
	TotalCommands   int
	FilesChanged    []string
	KeyActivities   []string // top 5 most important activities
	NeedsAttention  bool     // any session stuck/waiting/errored
	AttentionReason string
}

// DailyBrief is the top-level daily report summarizing all Claude Code sessions.
type DailyBrief struct {
	GeneratedAt    time.Time
	Period         string // "Last 24 hours" or "Since yesterday 9am"
	Projects       []ProjectBrief
	TotalSessions  int
	ActiveSessions int
	TotalEdits     int
	TotalCommands  int
	Highlights     []string // top 3-5 most notable things
	Attention      []string // things that need the user's attention
}

// GenerateBrief scans all sessions and produces a daily brief covering
// activity since the given time. It reads session files from sessionsDir,
// locates transcripts under projectsDir, and aggregates per-project summaries.
//
// If directories do not exist, it returns a valid but empty brief.
func GenerateBrief(projectsDir, sessionsDir string, since time.Time) (DailyBrief, error) {
	brief := DailyBrief{
		GeneratedAt: time.Now(),
		Period:      formatPeriod(since),
	}

	sessions, err := discovery.ScanSessions(sessionsDir)
	if err != nil {
		return brief, fmt.Errorf("scanning sessions: %w", err)
	}

	brief.TotalSessions = len(sessions)

	projects := discovery.GroupByProject(sessions)

	for _, proj := range projects {
		pb := buildProjectBrief(proj, projectsDir, since)
		brief.Projects = append(brief.Projects, pb)
		brief.TotalEdits += pb.TotalEdits
		brief.TotalCommands += pb.TotalCommands
		brief.ActiveSessions += pb.ActiveCount

		if pb.NeedsAttention {
			brief.Attention = append(brief.Attention, pb.AttentionReason)
		}
	}

	brief.Highlights = buildHighlights(brief)

	return brief, nil
}

// buildProjectBrief produces a ProjectBrief from a discovered Project by
// examining each session's transcript.
func buildProjectBrief(proj discovery.Project, projectsDir string, since time.Time) ProjectBrief {
	pb := ProjectBrief{
		Name:         proj.Name,
		Path:         proj.Path,
		SessionCount: len(proj.Sessions),
	}

	filesSet := make(map[string]struct{})
	var allActivities []string
	var attentionReasons []string

	for _, sess := range proj.Sessions {
		transcriptPath := discovery.FindTranscriptPath(projectsDir, sess.CWD, sess.ID)

		if transcriptPath == "" {
			attentionReasons = append(attentionReasons,
				fmt.Sprintf("%s session %s (PID %d): no transcript found", proj.Name, sess.ID, sess.PID))
			continue
		}

		summary, err := ExtractRecent(transcriptPath, since)
		if err != nil {
			// Non-fatal: skip this session's activity data.
			continue
		}

		// A session is "active" if it had any edits, commands, reads, or activities.
		if summary.EditCount > 0 || summary.BashCount > 0 || summary.ReadCount > 0 || len(summary.Activities) > 0 {
			pb.ActiveCount++
		}

		pb.TotalEdits += summary.EditCount
		pb.TotalCommands += summary.BashCount

		for _, f := range summary.FilesChanged {
			filesSet[f] = struct{}{}
		}
		for _, a := range summary.Activities {
			allActivities = append(allActivities, a.Description)
		}
	}

	// Deduplicate and sort files.
	for f := range filesSet {
		pb.FilesChanged = append(pb.FilesChanged, f)
	}
	sort.Strings(pb.FilesChanged)

	// Take up to 5 key activities.
	if len(allActivities) > 5 {
		pb.KeyActivities = allActivities[:5]
	} else {
		pb.KeyActivities = allActivities
	}

	if len(attentionReasons) > 0 {
		pb.NeedsAttention = true
		pb.AttentionReason = strings.Join(attentionReasons, "; ")
	}

	return pb
}

// buildHighlights produces the top-level highlight strings from the brief.
func buildHighlights(brief DailyBrief) []string {
	var hl []string

	projectCount := len(brief.Projects)
	if projectCount > 0 && brief.TotalSessions > 0 {
		hl = append(hl, fmt.Sprintf("%d sessions across %d projects",
			brief.TotalSessions, projectCount))
	}

	if brief.ActiveSessions > 0 {
		hl = append(hl, fmt.Sprintf("%d sessions had activity in period",
			brief.ActiveSessions))
	}

	if brief.TotalEdits > 0 {
		hl = append(hl, fmt.Sprintf("%d total file edits", brief.TotalEdits))
	}

	// Gather top activities across all projects.
	for _, p := range brief.Projects {
		for _, a := range p.KeyActivities {
			if len(hl) >= 5 {
				break
			}
			hl = append(hl, a)
		}
	}

	return hl
}

// formatPeriod produces a human-readable period string from a since timestamp.
func formatPeriod(since time.Time) string {
	duration := time.Since(since)
	hours := duration.Hours()

	if hours <= 25 {
		return fmt.Sprintf("Last %d hours (%s \u2192 %s)",
			int(math.Round(hours)),
			since.Format("Jan 2 15:04"),
			time.Now().Format("Jan 2 15:04"))
	}
	days := int(math.Round(hours / 24))
	return fmt.Sprintf("Last %d days (%s \u2192 %s)",
		days,
		since.Format("Jan 2"),
		time.Now().Format("Jan 2"))
}

// FormatDailyBrief renders a DailyBrief as a human-readable plaintext report
// suitable for stdout or piping into other tools.
func FormatDailyBrief(brief DailyBrief) string {
	var b strings.Builder

	b.WriteString("=== OpsDeck Daily Brief ===\n")
	b.WriteString(fmt.Sprintf("Period: %s\n", brief.Period))

	if brief.TotalSessions > 0 {
		b.WriteString(fmt.Sprintf("Active: %d of %d sessions across %d projects\n",
			brief.ActiveSessions, brief.TotalSessions, len(brief.Projects)))
	} else {
		b.WriteString("No sessions found.\n")
	}

	b.WriteString("\n")

	// Per-project sections.
	for _, p := range brief.Projects {
		b.WriteString(fmt.Sprintf("--- %s (%d sessions, %d active) ---\n",
			p.Name, p.SessionCount, p.ActiveCount))

		for _, a := range p.KeyActivities {
			b.WriteString(fmt.Sprintf("  * %s\n", a))
		}

		// Stats line.
		stats := []string{}
		if p.TotalEdits > 0 {
			stats = append(stats, fmt.Sprintf("%d file edits", p.TotalEdits))
		}
		if p.TotalCommands > 0 {
			stats = append(stats, fmt.Sprintf("%d commands", p.TotalCommands))
		}
		if len(p.FilesChanged) > 0 {
			stats = append(stats, fmt.Sprintf("%d files changed", len(p.FilesChanged)))
		}
		if len(stats) > 0 {
			b.WriteString(fmt.Sprintf("  %s\n", strings.Join(stats, ", ")))
		}

		if len(p.KeyActivities) == 0 && len(stats) == 0 {
			b.WriteString("  No activity in period\n")
		}

		b.WriteString("\n")
	}

	// Highlights.
	if len(brief.Highlights) > 0 {
		for _, h := range brief.Highlights {
			b.WriteString(fmt.Sprintf("  %s\n", h))
		}
		b.WriteString("\n")
	}

	// Attention section.
	if len(brief.Attention) > 0 {
		b.WriteString("\u26a0 NEEDS ATTENTION:\n")
		for _, a := range brief.Attention {
			b.WriteString(fmt.Sprintf("  * %s\n", a))
		}
		b.WriteString("\n")
	}

	return b.String()
}

// ParseSinceFlag parses the --since flag value into a time.Time.
// Supported formats:
//   - "" (empty) -> 24 hours ago (default)
//   - "24h", "12h", "48h" etc -> duration ago
//   - "yesterday" -> yesterday at 9:00 AM local time
func ParseSinceFlag(value string) (time.Time, error) {
	if value == "" {
		return time.Now().Add(-24 * time.Hour), nil
	}

	if value == "yesterday" {
		now := time.Now()
		return time.Date(now.Year(), now.Month(), now.Day()-1, 9, 0, 0, 0, time.Local), nil
	}

	// Try parsing as a Go duration string (e.g., "24h", "12h", "48h").
	d, err := time.ParseDuration(value)
	if err == nil {
		return time.Now().Add(-d), nil
	}

	return time.Time{}, fmt.Errorf("unrecognized --since value: %q (use e.g. 24h, 12h, yesterday)", value)
}
