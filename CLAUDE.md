# OpsDeck

**Chief of Staff for Claude Code** — monitor all your sessions from a single terminal.

## Project Status

v1.3.0 released + MCP server in development.
GitHub: https://github.com/getopsdeck/opsdeck
CI green (macOS + Linux). 13,000+ lines of Go, 47% test coverage.

### Architecture
```
cmd/opsdeck/main.go          — 15 commands (Dashboard, Reports, Actions, Advanced)
internal/discovery/           — session scanning, PID checking, state classification, git info
internal/monitor/             — shared enrichment layer (session-index + transcript + git)
internal/intel/               — brief, metrics, costs, timeline, ai-brief, summarize
internal/tui/                 — Bubble Tea TUI dashboard
internal/tui/components/      — table, statusbar
internal/tui/views/           — dashboard layout
internal/web/                 — HTTP + SSE web dashboard with keyboard shortcuts
internal/mcp/                 — MCP server (in progress)
```

### Commands (15)
```
Dashboard:  opsdeck, web
Reports:    brief, status, list, metrics, costs, export
Actions:    resume, watch, clean
Advanced:   ai-brief, version, help
```

### Key Design Decisions
- Pure observer: never mutates Claude Code data
- Secretary-style brief: actions first, user messages last, QUICK ACTION matches recommendation
- ccusage integration for accurate cost reporting (our estimates as fallback)
- Shared monitor layer: TUI and web use same data source
- Web dashboard: SSE, keyboard shortcuts (/ j k Enter Esc ?), auto-opens browser
- Per-message cost accumulation for mixed-model sessions
- Codex as standing consultant (6 consultations, 4 code reviews)

### Cost Tracking
- `opsdeck costs` delegates to ccusage (npm) for accurate LiteLLM pricing
- Built-in estimates as fallback when ccusage unavailable
- Web dashboard shows daily cost badge and per-session burn rate ($/hr)
- Brief includes "TODAY'S SPEND" with link to accurate pricing

## Build and Test
```bash
make build && make test && make lint
```

## User Preferences
- CEO/mentor — does not write code, only makes decisions
- Work autonomously: Codex or web search when blocked
- Only stop for major strategic decisions
- Commit directly to main
- No Claude attribution
- Use Opus for planning, Sonnet for implementation
