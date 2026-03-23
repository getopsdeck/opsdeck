// Package mcp provides an MCP (Model Context Protocol) server for OpsDeck.
// It exposes session data, daily briefs, cost reports, and session detail
// as MCP tools that Claude Code (or any MCP client) can call over stdio.
//
// Usage:
//
//	srv := mcp.NewServer()
//	srv.Run(ctx, &sdkmcp.StdioTransport{})
package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/getopsdeck/opsdeck/internal/intel"
	"github.com/getopsdeck/opsdeck/internal/monitor"
)

// sessionsDirs returns the standard Claude Code directories for sessions and
// projects. Both TUI and CLI use the same paths.
func sessionsDirs() (sessionsDir, projectsDir string) {
	home, _ := os.UserHomeDir()
	sessionsDir = filepath.Join(home, ".claude", "sessions")
	projectsDir = filepath.Join(home, ".claude", "projects")
	return
}

// parseSinceDuration parses a duration string like "24h" or "12h" and returns
// the corresponding time.Time (now minus duration). Empty string defaults to 24h.
func parseSinceDuration(s string) time.Time {
	if s == "" {
		s = "24h"
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		d = 24 * time.Hour
	}
	return time.Now().Add(-d)
}

// --- Input types for MCP tools ---

// getSessionsInput has no fields (tool takes no parameters).
type getSessionsInput struct{}

// briefInput is the input for the get_brief tool.
type briefInput struct {
	Since string `json:"since,omitempty" jsonschema:"Time window, e.g. 24h or 12h. Defaults to 24h."`
}

// costsInput is the input for the get_costs tool.
type costsInput struct {
	Since string `json:"since,omitempty" jsonschema:"Time window, e.g. 24h or 12h. Defaults to 24h."`
}

// sessionDetailInput is the input for the get_session_detail tool.
type sessionDetailInput struct {
	SessionID string `json:"session_id" jsonschema:"required,The session ID to look up."`
}

// NewServer creates a configured MCP server with all OpsDeck tools registered.
// The returned server is ready to be connected to a transport via Run.
func NewServer() *sdkmcp.Server {
	srv := sdkmcp.NewServer(&sdkmcp.Implementation{
		Name:    "opsdeck",
		Title:   "OpsDeck MCP Server",
		Version: "0.1.0",
	}, nil)

	// -- get_sessions --
	sdkmcp.AddTool(srv, &sdkmcp.Tool{
		Name:        "get_sessions",
		Description: "List all Claude Code sessions with their state, project, branch, and current activity.",
	}, handleGetSessions)

	// -- get_brief --
	sdkmcp.AddTool(srv, &sdkmcp.Tool{
		Name:        "get_brief",
		Description: "Generate a daily briefing of all Claude Code session activity. Returns a formatted summary suitable for a CEO.",
	}, handleGetBrief)

	// -- get_costs --
	sdkmcp.AddTool(srv, &sdkmcp.Tool{
		Name:        "get_costs",
		Description: "Generate a cost report showing token usage and estimated costs across all sessions.",
	}, handleGetCosts)

	// -- get_session_detail --
	sdkmcp.AddTool(srv, &sdkmcp.Tool{
		Name:        "get_session_detail",
		Description: "Get detailed activity summary for a specific session by ID, including files changed, tool usage counts, and last messages.",
	}, handleGetSessionDetail)

	return srv
}

// handleGetSessions returns all sessions as JSON.
func handleGetSessions(_ context.Context, _ *sdkmcp.CallToolRequest, _ getSessionsInput) (*sdkmcp.CallToolResult, any, error) {
	sessionsDir, projectsDir := sessionsDirs()
	sessions := monitor.Snapshot(sessionsDir, projectsDir)

	if sessions == nil {
		sessions = []monitor.Session{}
	}

	data, err := json.MarshalIndent(sessions, "", "  ")
	if err != nil {
		return nil, nil, fmt.Errorf("marshaling sessions: %w", err)
	}

	return &sdkmcp.CallToolResult{
		Content: []sdkmcp.Content{
			&sdkmcp.TextContent{Text: string(data)},
		},
	}, nil, nil
}

// handleGetBrief generates and returns the daily brief.
func handleGetBrief(_ context.Context, _ *sdkmcp.CallToolRequest, input briefInput) (*sdkmcp.CallToolResult, any, error) {
	sessionsDir, projectsDir := sessionsDirs()
	since := parseSinceDuration(input.Since)

	brief, err := intel.GenerateBrief(projectsDir, sessionsDir, since)
	if err != nil {
		return nil, nil, fmt.Errorf("generating brief: %w", err)
	}

	intel.EnrichBrief(&brief, projectsDir, sessionsDir, since)
	text := intel.FormatDailyBrief(brief)

	return &sdkmcp.CallToolResult{
		Content: []sdkmcp.Content{
			&sdkmcp.TextContent{Text: text},
		},
	}, nil, nil
}

// handleGetCosts generates and returns the cost report.
func handleGetCosts(_ context.Context, _ *sdkmcp.CallToolRequest, input costsInput) (*sdkmcp.CallToolResult, any, error) {
	sessionsDir, projectsDir := sessionsDirs()
	since := parseSinceDuration(input.Since)

	report, err := intel.GenerateCostReport(projectsDir, sessionsDir, since)
	if err != nil {
		return nil, nil, fmt.Errorf("generating cost report: %w", err)
	}

	text := intel.FormatCostReport(report)

	return &sdkmcp.CallToolResult{
		Content: []sdkmcp.Content{
			&sdkmcp.TextContent{Text: text},
		},
	}, nil, nil
}

// handleGetSessionDetail finds a session by ID and returns its activity summary.
func handleGetSessionDetail(_ context.Context, _ *sdkmcp.CallToolRequest, input sessionDetailInput) (*sdkmcp.CallToolResult, any, error) {
	sessionsDir, projectsDir := sessionsDirs()
	sessions := monitor.Snapshot(sessionsDir, projectsDir)

	// Find the session by exact or prefix match.
	var found *monitor.Session
	for i := range sessions {
		if sessions[i].ID == input.SessionID {
			found = &sessions[i]
			break
		}
	}
	// Try prefix match if exact match failed.
	if found == nil {
		for i := range sessions {
			if len(sessions[i].ID) >= len(input.SessionID) &&
				sessions[i].ID[:len(input.SessionID)] == input.SessionID {
				found = &sessions[i]
				break
			}
		}
	}

	if found == nil {
		return nil, nil, fmt.Errorf("session %q not found", input.SessionID)
	}

	if found.TranscriptPath == "" {
		return &sdkmcp.CallToolResult{
			Content: []sdkmcp.Content{
				&sdkmcp.TextContent{Text: fmt.Sprintf("Session %s found but has no transcript.", found.ID)},
			},
		}, nil, nil
	}

	summary, err := intel.ExtractSummary(found.TranscriptPath)
	if err != nil {
		return nil, nil, fmt.Errorf("extracting summary for session %s: %w", found.ID, err)
	}

	summary.SessionID = found.ID
	summary.Project = found.Project
	text := intel.FormatBrief(summary)

	return &sdkmcp.CallToolResult{
		Content: []sdkmcp.Content{
			&sdkmcp.TextContent{Text: text},
		},
	}, nil, nil
}
