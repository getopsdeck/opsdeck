package views

import (
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/getopsdeck/opsdeck/internal/tui/components"
)

// DashboardModel is the main dashboard view combining all components.
type DashboardModel struct {
	Width  int
	Height int

	// Sub-components
	Table     components.TableModel
	StatusBar components.StatusBarModel

	// Detail panel
	ShowDetail   bool
	DetailTitle  string
	DetailBody   string
	DetailHeight int

	// Styles
	styles DashboardStyles
}

// DashboardStyles holds styles for the dashboard layout.
type DashboardStyles struct {
	TitleBar    lipgloss.Style
	DetailPanel lipgloss.Style
	DetailTitle lipgloss.Style
	HelpBar     lipgloss.Style
	HelpKey     lipgloss.Style
	HelpDesc    lipgloss.Style
	HelpSep     string
}

// DefaultDashboardStyles returns the default dashboard styles.
func DefaultDashboardStyles() DashboardStyles {
	return DashboardStyles{
		TitleBar: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#1a1b26")).
			Background(lipgloss.Color("#7aa2f7")).
			Padding(0, 2),
		DetailPanel: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#3b4261")).
			Foreground(lipgloss.Color("#c0caf5")).
			Padding(0, 1),
		DetailTitle: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#7aa2f7")),
		HelpBar: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#565f89")).
			Padding(0, 1),
		HelpKey: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#7aa2f7")),
		HelpDesc: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#565f89")),
		HelpSep: " \u2502 ",
	}
}

// NewDashboard creates a new dashboard view model.
func NewDashboard(table components.TableModel, statusBar components.StatusBarModel) DashboardModel {
	return DashboardModel{
		Table:        table,
		StatusBar:    statusBar,
		DetailHeight: 6,
		styles:       DefaultDashboardStyles(),
	}
}

// SetSize updates all component sizes for the given terminal dimensions.
func (m *DashboardModel) SetSize(width, height int) {
	m.Width = width
	m.Height = height

	// Reserve space: title(1) + help(1) + status(1) + detail(if shown).
	reserved := 3
	if m.ShowDetail {
		reserved += m.DetailHeight + 2 // border adds 2 lines
	}

	tableHeight := height - reserved
	if tableHeight < 3 {
		tableHeight = 3
	}

	m.Table.SetSize(width, tableHeight)
	m.StatusBar.SetSize(width)
}

// View renders the complete dashboard.
func (m DashboardModel) View() string {
	var sections []string

	// 1. Title bar.
	title := m.renderTitleBar()
	sections = append(sections, title)

	// 2. Session table (main content).
	tableView := m.Table.View()
	// Ensure the table fills its allocated height.
	tableLines := strings.Count(tableView, "\n") + 1
	targetHeight := m.Height - 3 // title + help + status
	if m.ShowDetail {
		targetHeight -= m.DetailHeight + 2
	}
	if tableLines < targetHeight {
		tableView += strings.Repeat("\n", targetHeight-tableLines)
	}
	sections = append(sections, tableView)

	// 3. Detail panel (if toggled open).
	if m.ShowDetail {
		detail := m.renderDetailPanel()
		sections = append(sections, detail)
	}

	// 4. Help bar.
	help := m.renderHelpBar()
	sections = append(sections, help)

	// 5. Status bar.
	statusView := m.StatusBar.View()
	sections = append(sections, statusView)

	return strings.Join(sections, "\n")
}

// renderTitleBar renders the top title bar.
func (m DashboardModel) renderTitleBar() string {
	title := " OpsDeck"
	subtitle := "Chief of Staff for Claude Code"
	gap := m.Width - lipgloss.Width(title) - lipgloss.Width(subtitle) - 4
	if gap < 1 {
		gap = 1
	}
	content := title + strings.Repeat(" ", gap) + subtitle
	return m.styles.TitleBar.Width(m.Width).Render(content)
}

// renderDetailPanel renders the bottom detail/transcript panel.
func (m DashboardModel) renderDetailPanel() string {
	titleLine := m.styles.DetailTitle.Render(m.DetailTitle)
	body := m.DetailBody
	if body == "" {
		body = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#565f89")).
			Render("Select a session and press Enter to view details")
	}

	content := titleLine + "\n" + body

	return m.styles.DetailPanel.
		Width(m.Width - 2). // account for border
		Height(m.DetailHeight).
		Render(content)
}

// renderHelpBar renders the help key hints.
func (m DashboardModel) renderHelpBar() string {
	type hint struct {
		key  string
		desc string
	}

	hints := []hint{
		{"j/k", "navigate"},
		{"enter", "detail"},
		{"/", "search"},
		{"1-6", "filter"},
		{"0", "clear"},
		{"tab", "view"},
		{"r", "refresh"},
		{"R", "resume"},
		{"q", "quit"},
	}

	var parts []string
	for _, h := range hints {
		parts = append(parts,
			m.styles.HelpKey.Render(h.key)+
				" "+
				m.styles.HelpDesc.Render(h.desc))
	}

	helpLine := strings.Join(parts, m.styles.HelpSep)
	return m.styles.HelpBar.Width(m.Width).Render(helpLine)
}
