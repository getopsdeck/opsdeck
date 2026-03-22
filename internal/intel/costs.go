package intel

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Pricing per million tokens (USD) as of March 2026.
// Source: https://docs.anthropic.com/en/docs/about-claude/models
var pricing = map[string]modelPricing{
	"claude-opus-4-6": {Input: 15.0, Output: 75.0, CacheWrite: 18.75, CacheRead: 1.50},
	"claude-sonnet-4-6": {Input: 3.0, Output: 15.0, CacheWrite: 3.75, CacheRead: 0.30},
	"claude-haiku-4-5": {Input: 0.80, Output: 4.0, CacheWrite: 1.0, CacheRead: 0.08},
	// Fallback for unknown models: use Sonnet pricing.
}

type modelPricing struct {
	Input      float64 // per million input tokens
	Output     float64 // per million output tokens
	CacheWrite float64 // per million cache creation tokens
	CacheRead  float64 // per million cache read tokens
}

// SessionCost holds token usage and estimated cost for a session.
type SessionCost struct {
	SessionID    string
	Project      string
	Model        string // predominant model (most messages)
	InputTokens  int64
	OutputTokens int64
	CacheWrite   int64
	CacheRead    int64
	TotalTokens  int64
	EstCostUSD   float64
	MessageCount int
}

// CostReport holds the full cost report across all sessions.
type CostReport struct {
	Sessions    []SessionCost
	TotalCost   float64
	TotalTokens int64
	Period      string
}

// transcriptUsage is the usage block inside an assistant message.
type transcriptUsage struct {
	InputTokens              int64 `json:"input_tokens"`
	OutputTokens             int64 `json:"output_tokens"`
	CacheCreationInputTokens int64 `json:"cache_creation_input_tokens"`
	CacheReadInputTokens     int64 `json:"cache_read_input_tokens"`
}

// transcriptMsg holds the fields we need from an assistant message.
type transcriptMsg struct {
	Model string           `json:"model"`
	Usage transcriptUsage  `json:"usage"`
}

// estimateCostForMessage calculates the cost of a single message given its model and usage.
func estimateCostForMessage(model string, usage transcriptUsage) float64 {
	p, ok := pricing[model]
	if !ok {
		// Default to Sonnet pricing for unknown models.
		p = pricing["claude-sonnet-4-6"]
	}

	cost := float64(usage.InputTokens) * p.Input / 1_000_000
	cost += float64(usage.OutputTokens) * p.Output / 1_000_000
	cost += float64(usage.CacheCreationInputTokens) * p.CacheWrite / 1_000_000
	cost += float64(usage.CacheReadInputTokens) * p.CacheRead / 1_000_000
	return cost
}

// ExtractCosts reads a transcript JSONL and returns token usage/cost data.
func ExtractCosts(path string, since time.Time) (SessionCost, error) {
	f, err := os.Open(path)
	if err != nil {
		return SessionCost{}, err
	}
	defer f.Close()

	var cost SessionCost
	filterByTime := !since.IsZero()

	// Track message counts per model to determine the predominant model.
	// lastModel tracks insertion order for tiebreaking.
	modelCounts := make(map[string]int)
	var lastModel string

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 10*1024*1024)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var tl transcriptLine
		if json.Unmarshal(line, &tl) != nil {
			continue
		}

		if cost.SessionID == "" && tl.SessionID != "" {
			cost.SessionID = tl.SessionID
		}
		if cost.Project == "" && tl.CWD != "" {
			cost.Project = filepath.Base(tl.CWD)
		}

		if tl.Type != "assistant" {
			continue
		}

		// Parse timestamp for time filtering.
		if filterByTime && tl.Timestamp != "" {
			ts, err := time.Parse(time.RFC3339Nano, tl.Timestamp)
			if err == nil && ts.Before(since) {
				continue
			}
		}

		var msg transcriptMsg
		if json.Unmarshal(tl.Message, &msg) != nil {
			continue
		}

		if msg.Usage.InputTokens == 0 && msg.Usage.OutputTokens == 0 {
			continue
		}

		cost.MessageCount++
		cost.InputTokens += msg.Usage.InputTokens
		cost.OutputTokens += msg.Usage.OutputTokens
		cost.CacheWrite += msg.Usage.CacheCreationInputTokens
		cost.CacheRead += msg.Usage.CacheReadInputTokens

		// Accumulate cost at the per-message model rate to handle model switches correctly.
		cost.EstCostUSD += estimateCostForMessage(msg.Model, msg.Usage)

		if msg.Model != "" {
			modelCounts[msg.Model]++
			lastModel = msg.Model
		}
	}

	cost.TotalTokens = cost.InputTokens + cost.OutputTokens + cost.CacheWrite + cost.CacheRead

	// Set Model to the predominant model (most messages) for display purposes.
	// On a tie, the last model seen wins (matches intuition: most recent work).
	var maxCount int
	for model, count := range modelCounts {
		if count > maxCount {
			maxCount = count
			cost.Model = model
		}
	}
	// Apply last-seen tiebreaker: if the last model is tied for the top count, prefer it.
	if lastModel != "" && modelCounts[lastModel] == maxCount {
		cost.Model = lastModel
	}

	return cost, nil
}

// estimateCost calculates the estimated USD cost for a SessionCost using its Model field.
// It is used by tests and kept as a helper for single-model sessions.
func estimateCost(c SessionCost) float64 {
	usage := transcriptUsage{
		InputTokens:              c.InputTokens,
		OutputTokens:             c.OutputTokens,
		CacheCreationInputTokens: c.CacheWrite,
		CacheReadInputTokens:     c.CacheRead,
	}
	return estimateCostForMessage(c.Model, usage)
}

// GenerateCostReport scans all sessions and produces a cost report.
func GenerateCostReport(projectsDir, sessionsDir string, since time.Time) (CostReport, error) {
	report := CostReport{}

	if !since.IsZero() {
		report.Period = fmt.Sprintf("Since %s", since.Format("Jan 02 15:04"))
	} else {
		report.Period = "All time"
	}

	// Find all transcript files.
	entries, err := os.ReadDir(projectsDir)
	if err != nil {
		return report, nil
	}

	for _, projectEntry := range entries {
		if !projectEntry.IsDir() {
			continue
		}

		projectPath := filepath.Join(projectsDir, projectEntry.Name())
		files, err := os.ReadDir(projectPath)
		if err != nil {
			continue
		}

		for _, f := range files {
			if f.IsDir() || !strings.HasSuffix(f.Name(), ".jsonl") {
				continue
			}

			path := filepath.Join(projectPath, f.Name())
			cost, err := ExtractCosts(path, since)
			if err != nil || cost.TotalTokens == 0 {
				continue
			}

			report.Sessions = append(report.Sessions, cost)
			report.TotalCost += cost.EstCostUSD
			report.TotalTokens += cost.TotalTokens
		}
	}

	// Sort by cost descending.
	sort.Slice(report.Sessions, func(i, j int) bool {
		return report.Sessions[i].EstCostUSD > report.Sessions[j].EstCostUSD
	})

	return report, nil
}

// FormatCostReport renders a cost report as human-readable text.
func FormatCostReport(report CostReport) string {
	var b strings.Builder

	b.WriteString("=== OpsDeck Cost Report ===\n")
	b.WriteString(fmt.Sprintf("Period: %s\n", report.Period))
	b.WriteString(fmt.Sprintf("Total: $%.2f estimated across %d sessions\n\n", report.TotalCost, len(report.Sessions)))

	// Group by project.
	byProject := make(map[string][]SessionCost)
	projectCosts := make(map[string]float64)
	for _, s := range report.Sessions {
		byProject[s.Project] = append(byProject[s.Project], s)
		projectCosts[s.Project] += s.EstCostUSD
	}

	// Sort projects by cost.
	projects := make([]string, 0, len(byProject))
	for p := range byProject {
		projects = append(projects, p)
	}
	sort.Slice(projects, func(i, j int) bool {
		return projectCosts[projects[i]] > projectCosts[projects[j]]
	})

	for _, project := range projects {
		sessions := byProject[project]
		b.WriteString(fmt.Sprintf("--- %s ($%.2f) ---\n", project, projectCosts[project]))

		for _, s := range sessions {
			id := s.SessionID
			if len(id) > 12 {
				id = id[:12]
			}
			b.WriteString(fmt.Sprintf("  %s  %-20s  %s tokens  $%.4f\n",
				id, s.Model, formatTokens(s.TotalTokens), s.EstCostUSD))
		}
		b.WriteString("\n")
	}

	return b.String()
}

// formatTokens formats a token count with K/M suffixes.
func formatTokens(n int64) string {
	if n >= 1_000_000 {
		return fmt.Sprintf("%.1fM", float64(n)/1_000_000)
	}
	if n >= 1_000 {
		return fmt.Sprintf("%.1fK", float64(n)/1_000)
	}
	return fmt.Sprintf("%d", n)
}

// RunCostReport is the entrypoint for the `opsdeck costs` subcommand.
func RunCostReport() {
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return
	}

	sessionsDir := filepath.Join(home, ".claude", "sessions")
	projectsDir := filepath.Join(home, ".claude", "projects")

	since := time.Now().Add(-24 * time.Hour)
	report, err := GenerateCostReport(projectsDir, sessionsDir, since)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return
	}

	fmt.Print(FormatCostReport(report))
}
