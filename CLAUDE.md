# OpsDeck

**Chief of Staff for Claude Code** — monitor all your Claude Code sessions from a single terminal.

## Project Status

v0.5.0 released. Full feature set shipped.
GitHub: https://github.com/getopsdeck/opsdeck
CI green (macOS + Linux). 9,500+ lines of Go code.

### What's Done
- Session discovery (PID checking, transcript reading, state classification)
- Bubble Tea v2 TUI dashboard with Tokyo Night theme
- `opsdeck brief` — daily briefing across all projects
- `opsdeck metrics` — today vs yesterday productivity comparison with Momentum score
- `opsdeck costs` — token usage and estimated spend per session/project
- `opsdeck ai-brief` — AI-powered morning brief via claude -p (opt-in)
- `opsdeck web` — browser dashboard with SSE, search, filter, cost indicator, timeline
- `opsdeck help` — all available commands
- Rich detail panel (TUI + web) with activity summary and cost estimate
- Session timeline visualization (colored segments for tool/text/user/error events)
- Hero GIF + web dashboard screenshot in README
- v0.1.0 + v0.5.0 tagged releases with 4 binaries each
- Launch materials ready (thoughts/launch/)

### Architecture
```
cmd/opsdeck/main.go          — entry point, 8 subcommands
internal/discovery/           — session scanning, PID checking, transcript reading
internal/intel/               — activity, brief, metrics, costs, timeline, ai-brief, summarize
internal/tui/                 — Bubble Tea TUI (app, bridge, mock, styles, keymap)
internal/tui/components/      — reusable TUI components (table, statusbar)
internal/tui/views/           — composite views (dashboard)
internal/web/                 — embedded HTTP server, SSE, JSON API, timeline
```

### Key Design Decisions
- Pure observer: never mutates Claude Code data
- Defensive parsing: missing/malformed files return zero values, never error
- SessionState (busy/waiting/idle/dead) separate from AttentionReason
- Web dashboard uses SSE (not WebSockets) for real-time updates
- Transcript summary + cost cache by modtime (two-phase lock to avoid I/O under mutex)
- Per-message cost accumulation (handles mixed-model sessions correctly)
- XSS protection via escapeHtml on all innerHTML interpolations
- AI brief pipes prompt via stdin (not argv) to avoid ARG_MAX and process list exposure
- Direct-to-main workflow (no PR overhead for solo project)

### Code Quality
- 4-agent code review completed: web (critic), intel (critic), discovery (critic), overall (Codex)
- 8 critical bugs found and fixed (race condition, XSS, cost mispricing, PID reuse, cache lock)
- 33 cost analytics tests, comprehensive discovery + intel test suites

### Roadmap
- Session timeline in TUI (currently web-only)
- GitHub integration (branch, PR, CI status per session)
- Shared monitor layer (internal/monitor/) to eliminate web/TUI drift
- Homebrew tap (needs HOMEBREW_TAP_GITHUB_TOKEN secret)
- Launch campaign: Show HN + r/ClaudeCode + X (Tuesday-Thursday optimal)

### Competitive Context
- Main competitor: claude-squad (6.3K stars, tmux-based, no analytics)
- Our positioning: "Chief of Staff — not just monitoring, but intelligence"
- Key differentiators: daily brief, metrics, cost analytics, web dashboard, timeline, AI brief
- Target: 5K+ GitHub stars

## Tech Stack
- Go 1.26 + Bubble Tea v2 (charm.land/bubbletea/v2)
- Lip Gloss v2 + Bubbles v2 (Charm ecosystem)
- stdlib net/http + SSE for web dashboard
- VHS (charmbracelet/vhs) for terminal recording
- GitHub org: getopsdeck
- Module path: github.com/getopsdeck/opsdeck

## Build and Test
```bash
make build      # compile
make test       # race detector
make lint       # go vet
opsdeck         # TUI dashboard
opsdeck brief   # daily briefing
opsdeck metrics # productivity comparison
opsdeck costs   # token usage and spend
opsdeck ai-brief # AI morning brief (costs tokens)
opsdeck web     # web dashboard at localhost:7070
opsdeck version # show version info
opsdeck help    # list all commands
```

## User Preferences
- User is CEO/mentor — does not write code, only makes decisions
- Work autonomously: Codex or web search when blocked, never ask user
- Only stop for major strategic decisions
- Commit directly to main (no PR overhead unless breaking change)
- No Claude attribution in commits/PRs
- Use multiple subagents for parallel review and research
- macOS provenance fix: terminal added to Developer Tools
