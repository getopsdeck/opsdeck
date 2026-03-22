package intel

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/getopsdeck/opsdeck/internal/discovery"
)

// WaitingSession describes a session that is waiting for user input.
type WaitingSession struct {
	SessionID    string
	ProjectName  string
	WaitingSince time.Time // when it started waiting (last activity)
	LastUserMsg  string    // last user message (truncated)
}

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

	// Secretary brief fields.
	WaitingSessions []WaitingSession // sessions waiting for user input
	OneLine         string           // one-line activity summary
	Branch          string           // git branch name
	IsDirty         bool             // git working tree has changes
	LastActive      time.Time        // most recent activity across all sessions
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
	CostEstimate   float64  // estimated spend in USD for the period
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
	var allActivities []Activity
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
		allActivities = append(allActivities, summary.Activities...)
	}

	// Deduplicate and sort files.
	for f := range filesSet {
		pb.FilesChanged = append(pb.FilesChanged, f)
	}
	sort.Strings(pb.FilesChanged)

	// Condense raw activities into human-readable summaries (max 5).
	pb.KeyActivities = SummarizeActivities(allActivities)

	if len(attentionReasons) > 0 {
		pb.NeedsAttention = true
		pb.AttentionReason = strings.Join(attentionReasons, "; ")
	}

	// Set one-line summary from all activities.
	pb.OneLine = SummarizeOneLine(allActivities)

	return pb
}

// EnrichBrief populates the secretary-format fields on an existing DailyBrief:
// waiting sessions, git branch/dirty state, cost estimates, and last-active times.
// It re-scans sessions to determine live state. This is a separate pass so the
// core GenerateBrief stays fast and testable without filesystem side effects.
func EnrichBrief(brief *DailyBrief, projectsDir, sessionsDir string, since time.Time) {
	sessions, err := discovery.ScanSessions(sessionsDir)
	if err != nil {
		return
	}

	projects := discovery.GroupByProject(sessions)

	// Build a lookup from project name to its sessions for waiting detection.
	projectSessions := make(map[string][]discovery.Session)
	for _, proj := range projects {
		projectSessions[proj.Name] = proj.Sessions
	}

	for i := range brief.Projects {
		pb := &brief.Projects[i]

		// Git branch info: use the project path for git queries.
		if pb.Path != "" {
			gi := discovery.GetGitInfo(pb.Path)
			pb.Branch = gi.Branch
			pb.IsDirty = gi.IsDirty
		}

		// Detect waiting sessions and track last activity.
		if sessions, ok := projectSessions[pb.Name]; ok {
			for _, sess := range sessions {
				transcriptPath := discovery.FindTranscriptPath(projectsDir, sess.CWD, sess.ID)
				if transcriptPath == "" {
					continue
				}

				// Get last activity time.
				lastActivity, err := discovery.ReadLastActivity(transcriptPath)
				if err != nil || lastActivity.IsZero() {
					lastActivity = sess.StartedAt
				}

				// Track the most recent activity for this project.
				if lastActivity.After(pb.LastActive) {
					pb.LastActive = lastActivity
				}

				// Check if the session is in a waiting/idle state (alive but inactive).
				alive := discovery.CheckSession(sess.PID, sess.StartedAt)
				state := discovery.ClassifyState(alive, lastActivity)

				if state == discovery.StateWaiting || state == discovery.StateIdle {
					// Read transcript to get the last user message.
					summary, err := ExtractRecent(transcriptPath, time.Time{})
					lastMsg := ""
					if err == nil && summary.LastUserMsg != "" {
						lastMsg = truncate(summary.LastUserMsg, 40)
					}

					ws := WaitingSession{
						SessionID:    sess.ID,
						ProjectName:  pb.Name,
						WaitingSince: lastActivity,
						LastUserMsg:  lastMsg,
					}
					pb.WaitingSessions = append(pb.WaitingSessions, ws)
				}

				// Also try git info from session CWD if project path is empty.
				if pb.Branch == "" && sess.CWD != "" {
					gi := discovery.GetGitInfo(sess.CWD)
					pb.Branch = gi.Branch
					pb.IsDirty = gi.IsDirty
				}
			}
		}
	}

	// Cost estimate.
	costReport, err := GenerateCostReport(projectsDir, sessionsDir, since)
	if err == nil {
		brief.CostEstimate = costReport.TotalCost
	}
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

// FormatDailyBriefVerbose renders a DailyBrief as a detailed plaintext report.
// This is the original format, kept for the --verbose flag.
func FormatDailyBriefVerbose(brief DailyBrief) string {
	var b strings.Builder

	b.WriteString("=== OpsDeck Daily Brief ===\n")
	b.WriteString(fmt.Sprintf("Period: %s\n", brief.Period))

	if brief.TotalSessions > 0 {
		// Executive summary: one-line overview.
		b.WriteString(fmt.Sprintf("You worked across %d projects, made %d edits, ran %d commands.\n",
			len(brief.Projects), brief.TotalEdits, brief.TotalCommands))
		b.WriteString(fmt.Sprintf("Sessions: %d active of %d total\n",
			brief.ActiveSessions, brief.TotalSessions))
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

// FormatDailyBrief renders a DailyBrief in the secretary format -- concise,
// action-oriented, suitable for a CEO who needs to know what to do next.
func FormatDailyBrief(brief DailyBrief) string {
	var b strings.Builder

	// Header with day-of-week.
	dayName := brief.GeneratedAt.Format("Monday")
	dateFmt := brief.GeneratedAt.Format("Jan 2")
	projectCount := len(brief.Projects)

	b.WriteString("=== OpsDeck Morning Brief ===\n")
	if brief.TotalSessions > 0 {
		b.WriteString(fmt.Sprintf("%s, %s | %d projects, %d sessions\n",
			dayName, dateFmt, projectCount, brief.TotalSessions))
	} else {
		b.WriteString(fmt.Sprintf("%s, %s\n", dayName, dateFmt))
		b.WriteString("\nNo sessions found.\n")
		return b.String()
	}

	// --- NEEDS YOUR ATTENTION: waiting sessions ---
	var waitingLines []string
	for _, p := range brief.Projects {
		for _, ws := range p.WaitingSessions {
			waitDur := formatWaitDuration(time.Since(ws.WaitingSince))
			id := ws.SessionID
			if len(id) > 8 {
				id = id[:8]
			}
			line := fmt.Sprintf("  ! %s (%s) -- waiting %s", p.Name, id, waitDur)
			if ws.LastUserMsg != "" {
				msg := truncate(ws.LastUserMsg, 40)
				line += fmt.Sprintf(" -- last: %q", msg)
			}
			waitingLines = append(waitingLines, line)
		}
	}

	if len(waitingLines) > 5 {
		waitingLines = waitingLines[:5]
	}
	if len(waitingLines) > 0 {
		b.WriteString("\nNEEDS YOUR ATTENTION:\n")
		for _, line := range waitingLines {
			b.WriteString(line + "\n")
		}
	}

	// --- PROJECT UPDATES: active projects with one-line summaries ---
	var activeLines []string
	var idleLines []string

	// Find the longest project name for alignment.
	maxNameLen := 0
	for _, p := range brief.Projects {
		if len(p.Name) > maxNameLen {
			maxNameLen = len(p.Name)
		}
	}
	if maxNameLen < 8 {
		maxNameLen = 8
	}

	for _, p := range brief.Projects {
		if p.ActiveCount == 0 && p.TotalEdits == 0 {
			// Idle project.
			lastActiveStr := "unknown"
			if !p.LastActive.IsZero() {
				lastActiveStr = formatTimeAgo(p.LastActive)
			}
			sessInfo := ""
			if p.SessionCount > 0 {
				sessInfo = fmt.Sprintf("%d sessions, ", p.SessionCount)
			}
			idleLines = append(idleLines,
				fmt.Sprintf("  %-*s -- %slast active %s", maxNameLen, p.Name, sessInfo, lastActiveStr))
			continue
		}

		// Active project line.
		summary := p.OneLine
		if summary == "" {
			summary = defaultOneLine(p)
		}

		editInfo := ""
		if p.TotalEdits > 0 {
			fileCount := len(p.FilesChanged)
			if fileCount > 0 {
				editInfo = fmt.Sprintf(", %d edits across %d files", p.TotalEdits, fileCount)
			} else {
				editInfo = fmt.Sprintf(", %d edits", p.TotalEdits)
			}
		}

		branchInfo := ""
		if p.Branch != "" {
			branchStr := p.Branch
			if p.IsDirty {
				branchStr += "*"
			}
			branchInfo = fmt.Sprintf(" [%s]", branchStr)
		}

		activeLines = append(activeLines,
			fmt.Sprintf("  %-*s -- %s%s%s", maxNameLen, p.Name, summary, editInfo, branchInfo))
	}

	if len(activeLines) > 0 {
		b.WriteString("\nPROJECT UPDATES:\n")
		for _, line := range activeLines {
			b.WriteString(line + "\n")
		}
	}

	if len(idleLines) > 0 {
		b.WriteString("\nIDLE (no activity):\n")
		for _, line := range idleLines {
			b.WriteString(line + "\n")
		}
	}

	// --- Cost estimate ---
	if brief.CostEstimate > 0 {
		b.WriteString(fmt.Sprintf("\nTODAY'S SPEND: ~$%.0f estimated\n", brief.CostEstimate))
	}

	return b.String()
}

// defaultOneLine produces a fallback one-line summary from a ProjectBrief
// when OneLine was not explicitly set.
func defaultOneLine(p ProjectBrief) string {
	if len(p.KeyActivities) > 0 {
		// Use the first key activity, truncated.
		return truncate(p.KeyActivities[0], 60)
	}
	if p.TotalEdits > 0 {
		return "active"
	}
	return "no notable activity"
}

// formatWaitDuration formats a duration into a human-readable wait string.
func formatWaitDuration(d time.Duration) string {
	if d < time.Minute {
		return "just now"
	}
	if d < time.Hour {
		mins := int(d.Minutes())
		return fmt.Sprintf("%d min", mins)
	}
	if d < 24*time.Hour {
		hours := int(d.Hours())
		if hours == 1 {
			return "1 hour"
		}
		return fmt.Sprintf("%d hours", hours)
	}
	days := int(d.Hours() / 24)
	if days == 1 {
		return "1 day"
	}
	return fmt.Sprintf("%d days", days)
}

// formatTimeAgo formats a timestamp as a human-readable "X ago" string.
func formatTimeAgo(t time.Time) string {
	d := time.Since(t)
	if d < time.Hour {
		mins := int(d.Minutes())
		if mins <= 0 {
			return "just now"
		}
		return fmt.Sprintf("%d min ago", mins)
	}
	if d < 24*time.Hour {
		hours := int(d.Hours())
		if hours == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", hours)
	}
	days := int(d.Hours() / 24)
	if days == 1 {
		return "1 day ago"
	}
	return fmt.Sprintf("%d days ago", days)
}

// SummarizeOneLine condenses a list of activities into a single short line
// suitable for the secretary brief. It prioritizes user requests for context,
// then falls back to describing what tools did.
func SummarizeOneLine(activities []Activity) string {
	if len(activities) == 0 {
		return ""
	}

	// Collect user requests -- they give the best context.
	var userRequests []string
	var gitOps []string
	editCount := 0
	bashCount := 0

	for _, a := range activities {
		switch a.Type {
		case ActivityUserRequest:
			userRequests = append(userRequests, a.Description)
		case ActivityGitOp:
			gitOps = append(gitOps, a.Description)
		case ActivityFileEdit:
			editCount++
		case ActivityBashCommand:
			bashCount++
		}
	}

	// Build a description of what happened (not what the user said).
	var parts []string

	// Git operations first — most meaningful.
	if len(gitOps) > 0 {
		parts = append(parts, summarizeGit(gitOps))
	}

	// Edit/command counts.
	if editCount > 0 {
		parts = append(parts, fmt.Sprintf("%d edits", editCount))
	}
	if bashCount > 0 {
		parts = append(parts, fmt.Sprintf("%d commands", bashCount))
	}

	if len(parts) > 0 {
		return strings.Join(parts, ", ")
	}

	// Last resort: use the most recent user request for context.
	if len(userRequests) > 0 {
		return truncate(userRequests[len(userRequests)-1], 60)
	}

	return "explored codebase"
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
