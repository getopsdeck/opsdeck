package tui

import (
	"fmt"
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/getopsdeck/opsdeck/internal/intel"
	"github.com/getopsdeck/opsdeck/internal/tui/components"
	"github.com/getopsdeck/opsdeck/internal/tui/views"
)

// refreshMsg is sent periodically to trigger data refresh.
type refreshMsg time.Time

// App is the top-level Bubble Tea model for OpsDeck.
type App struct {
	// Terminal dimensions.
	width  int
	height int

	// Data.
	sessions    []Session
	allSessions []Session // unfiltered source of truth

	// UI state.
	keys        KeyMap
	stateFilter string // filter by state ("busy", "idle", etc.)
	searchTerm  string // search query
	searching   bool   // true when search input is active
	searchBuf   string // in-progress search text

	// Sub-components.
	dashboard views.DashboardModel
	table     components.TableModel
	statusBar components.StatusBarModel

	// Styles.
	styles Styles
}

// NewApp creates a new App model. It discovers real Claude Code sessions
// and falls back to mock data if none are found.
func NewApp() *App {
	styles := DefaultStyles()

	table := components.NewTable(StateIcon)
	statusBar := components.NewStatusBar(styles.StateBadge)
	dashboard := views.NewDashboard(table, statusBar)

	app := &App{
		keys:      DefaultKeyMap(),
		styles:    styles,
		table:     table,
		statusBar: statusBar,
		dashboard: dashboard,
	}

	// Discover real sessions. Empty is fine — dashboard shows "no sessions found".
	app.allSessions = DiscoverSessions()
	app.applyFilters()

	return app
}

// Init implements tea.Model.
func (a *App) Init() tea.Cmd {
	return tickRefresh()
}

// tickRefresh returns a command that sends a refreshMsg after 3 seconds.
func tickRefresh() tea.Cmd {
	return tea.Tick(3*time.Second, func(t time.Time) tea.Msg {
		return refreshMsg(t)
	})
}

// Update implements tea.Model.
func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		a.syncSizes()
		return a, nil

	case refreshMsg:
		// Re-discover sessions from disk. Always replace — stale data is worse
		// than empty data for a monitoring tool.
		a.allSessions = DiscoverSessions()
		a.applyFilters()
		a.statusBar.LastRefresh = time.Now()
		a.syncDashboard()
		return a, tickRefresh()

	case tea.KeyPressMsg:
		return a.handleKeyPress(msg)
	}

	return a, nil
}

// handleKeyPress processes keyboard input.
func (a *App) handleKeyPress(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	// If searching, route keys to search handler.
	if a.searching {
		return a.handleSearchKey(msg)
	}

	switch {
	case key.Matches(msg, a.keys.Quit):
		return a, tea.Quit

	case key.Matches(msg, a.keys.Up):
		a.table.MoveUp()
		if a.dashboard.ShowDetail {
			a.updateDetail()
		}
		a.syncDashboard()

	case key.Matches(msg, a.keys.Down):
		a.table.MoveDown()
		if a.dashboard.ShowDetail {
			a.updateDetail()
		}
		a.syncDashboard()

	case key.Matches(msg, a.keys.Enter):
		a.dashboard.ShowDetail = !a.dashboard.ShowDetail
		a.updateDetail()
		a.syncSizes()
		a.syncDashboard()

	case key.Matches(msg, a.keys.Search):
		a.searching = true
		a.searchBuf = ""

	case key.Matches(msg, a.keys.Refresh):
		a.allSessions = DiscoverSessions()
		a.applyFilters()
		a.statusBar.LastRefresh = time.Now()
		a.syncDashboard()

	case key.Matches(msg, a.keys.Tab):
		a.table.ProjectView = !a.table.ProjectView
		a.syncDashboard()

	case key.Matches(msg, a.keys.FilterState1):
		a.toggleFilter("waiting")
	case key.Matches(msg, a.keys.FilterState2):
		a.toggleFilter("busy")
	case key.Matches(msg, a.keys.FilterState3):
		a.toggleFilter("idle")
	case key.Matches(msg, a.keys.FilterState4):
		a.toggleFilter("dead")
	case key.Matches(msg, a.keys.FilterClear):
		a.stateFilter = ""
		a.searchTerm = ""
		a.applyFilters()
		a.syncDashboard()
	}

	return a, nil
}

// handleSearchKey processes keys while in search mode.
func (a *App) handleSearchKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, a.keys.Escape):
		a.searching = false
		a.searchBuf = ""
		// Don't clear the active searchTerm (esc just closes input)

	case msg.String() == "enter":
		a.searching = false
		a.searchTerm = a.searchBuf
		a.applyFilters()
		a.syncDashboard()

	case msg.String() == "backspace":
		if len(a.searchBuf) > 0 {
			// Drop the last rune, not the last byte, for Unicode safety.
			runes := []rune(a.searchBuf)
			a.searchBuf = string(runes[:len(runes)-1])
		}

	default:
		// Append printable characters.
		if msg.Key().Text != "" {
			a.searchBuf += msg.Key().Text
		}
	}

	return a, nil
}

// toggleFilter sets or clears a state filter.
func (a *App) toggleFilter(state string) {
	if a.stateFilter == state {
		a.stateFilter = ""
	} else {
		a.stateFilter = state
	}
	a.applyFilters()
	a.syncDashboard()
}

// applyFilters rebuilds the filtered session list.
func (a *App) applyFilters() {
	filtered := filterSessions(a.allSessions, a.stateFilter, a.searchTerm)
	a.sessions = filtered

	// Convert to table sessions.
	tableSessions := make([]components.TableSession, len(filtered))
	for i, s := range filtered {
		tableSessions[i] = components.TableSession{
			ID:             s.ID,
			PID:            s.PID,
			State:          s.State,
			Project:        s.Project,
			StartedAt:      s.StartedAt,
			WorkingOn:      s.WorkingOn,
			LastLine:       s.LastLine,
			TranscriptPath: s.TranscriptPath,
			GitBranch:      s.GitBranch,
			GitDirty:       s.GitDirty,
		}
	}
	a.table.SetSessions(tableSessions)

	// Update status bar counts (always from all sessions).
	counts := countByState(a.allSessions)
	a.statusBar.Counts = counts
	a.statusBar.Total = len(a.allSessions)
	a.statusBar.Filter = a.stateFilter
	a.statusBar.SearchTerm = a.searchTerm
}

// updateDetail refreshes the detail panel content based on current selection.
// When a transcript is available, it shows a rich activity summary from the
// intel module instead of just basic metadata.
func (a *App) updateDetail() {
	sel := a.table.SelectedSession()
	if sel == nil {
		a.dashboard.DetailTitle = ""
		a.dashboard.DetailBody = ""
		return
	}

	a.dashboard.DetailTitle = StateIcon(sel.State) + "  " + sel.ID + "  (PID " + itoa(sel.PID) + ")"

	var lines []string
	lines = append(lines, a.styles.StatusKey.Render("Project: ")+sel.Project)
	lines = append(lines, a.styles.StatusKey.Render("State:   ")+sel.State)
	if sel.WorkingOn != "" {
		lines = append(lines, a.styles.StatusKey.Render("Task:    ")+sel.WorkingOn)
	}

	// Try to extract rich activity data from transcript.
	if sel.TranscriptPath != "" {
		summary, err := intel.ExtractSummary(sel.TranscriptPath)
		if err == nil && summary.TotalMessages > 0 {
			lines = append(lines, "")

			// Stats line.
			var stats []string
			if summary.EditCount > 0 {
				stats = append(stats, fmt.Sprintf("%d edits", summary.EditCount))
			}
			if summary.BashCount > 0 {
				stats = append(stats, fmt.Sprintf("%d commands", summary.BashCount))
			}
			if len(summary.FilesChanged) > 0 {
				stats = append(stats, fmt.Sprintf("%d files", len(summary.FilesChanged)))
			}
			if summary.ErrorCount > 0 {
				stats = append(stats, fmt.Sprintf("%d errors", summary.ErrorCount))
			}
			stats = append(stats, fmt.Sprintf("%d messages", summary.TotalMessages))
			lines = append(lines, a.styles.StatusKey.Render("Stats:   ")+strings.Join(stats, " | "))

			// Condensed activity bullets.
			condensed := intel.SummarizeActivities(summary.Activities)
			if len(condensed) > 0 {
				lines = append(lines, "")
				lines = append(lines, a.styles.HelpDesc.Render("Activity:"))
				for _, bullet := range condensed {
					lines = append(lines, "  * "+bullet)
				}
			}

			// Last user request.
			if summary.LastUserMsg != "" {
				lines = append(lines, "")
				lines = append(lines, a.styles.HelpDesc.Render("Last request:"))
				lines = append(lines, "  "+truncateRunes(summary.LastUserMsg, 100))
			}

			a.dashboard.DetailBody = strings.Join(lines, "\n")
			a.dashboard.DetailHeight = len(lines) + 1 // +1 for title
			a.syncSizes()
			return
		}
	}

	// Fallback: basic detail with last output line.
	lines = append(lines, "")
	lines = append(lines, a.styles.HelpDesc.Render("Last output:"))
	lines = append(lines, "  "+sel.LastLine)

	a.dashboard.DetailBody = strings.Join(lines, "\n")
	a.dashboard.DetailHeight = len(lines) + 1
	a.syncSizes()
}

// truncateRunes truncates a string to maxLen runes (not bytes), adding "..."
// if truncated. This is Unicode-safe unlike byte slicing.
func truncateRunes(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return strings.ReplaceAll(s, "\n", " ")
	}
	if maxLen <= 3 {
		return string(runes[:maxLen])
	}
	return strings.ReplaceAll(string(runes[:maxLen-3]), "\n", " ") + "..."
}

// syncSizes propagates terminal dimensions to all sub-components.
func (a *App) syncSizes() {
	a.dashboard.SetSize(a.width, a.height)
	a.table.SetSize(a.width, a.height-6)
	a.statusBar.SetSize(a.width)
}

// syncDashboard copies sub-component state into the dashboard for rendering.
func (a *App) syncDashboard() {
	a.dashboard.Table = a.table
	a.dashboard.StatusBar = a.statusBar
}

// View implements tea.Model.
func (a *App) View() tea.View {
	if a.width == 0 {
		// No size yet — show loading.
		v := tea.NewView("Loading OpsDeck...")
		v.AltScreen = true
		return v
	}

	// Sync before rendering.
	a.syncDashboard()

	var content string

	if a.searching {
		content = a.viewWithSearch()
	} else {
		content = a.dashboard.View()
	}

	v := tea.NewView(content)
	v.AltScreen = true
	return v
}

// viewWithSearch renders the dashboard with a search input overlay on the status bar.
func (a *App) viewWithSearch() string {
	// Render dashboard without the normal status bar.
	dashView := a.dashboard.View()

	// Replace the last line (status bar) with the search input.
	lines := strings.Split(dashView, "\n")
	if len(lines) > 0 {
		searchLine := a.styles.SearchPrompt.Render(" /") +
			a.styles.SearchText.Render(a.searchBuf) +
			lipgloss.NewStyle().
				Foreground(lipgloss.Color("#7aa2f7")).
				Render("\u2588") // block cursor

		// Pad to full width.
		searchBar := lipgloss.NewStyle().
			Background(lipgloss.Color("#24283b")).
			Width(a.width).
			Render(searchLine)

		lines[len(lines)-1] = searchBar
	}

	return strings.Join(lines, "\n")
}
