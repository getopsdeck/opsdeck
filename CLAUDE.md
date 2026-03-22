# OpsDeck

**Chief of Staff for Claude Code** — monitor all your Claude Code sessions from a single terminal.

## Project Status

v0.1.0 released + v0.3 web dashboard shipped.
GitHub: https://github.com/getopsdeck/opsdeck
CI green (macOS + Linux). 7,000+ lines of Go code.

### What's Done
- Session discovery (PID checking, transcript reading, state classification)
- Bubble Tea v2 TUI dashboard with Tokyo Night theme
- `opsdeck brief` — daily briefing across all projects
- `opsdeck metrics` — today vs yesterday productivity comparison with Momentum score
- `opsdeck web` — browser-based dashboard with SSE real-time updates
- Rich detail panel (Enter in TUI / click in web) with activity summary
- `--version` flag with ldflags + runtime/debug.ReadBuildInfo fallback
- Hero GIF recorded with VHS (demo.gif)
- v0.1.0 tagged release with 4 binaries (darwin/linux × amd64/arm64)
- Launch materials ready (thoughts/launch/)
- Real data working (tested against 19 real Claude Code sessions, 7 projects)

### Architecture
```
cmd/opsdeck/main.go          — entry point, brief/metrics/web subcommands
internal/discovery/           — session scanning, PID checking, transcript reading
internal/intel/               — activity extraction, daily brief, metrics, summarize
internal/tui/                 — Bubble Tea TUI (app, bridge, mock, styles, keymap)
internal/tui/components/      — reusable TUI components (table, statusbar)
internal/tui/views/           — composite views (dashboard)
internal/web/                 — embedded HTTP server, SSE, JSON API
```

### Key Design Decisions
- Pure observer: never mutates Claude Code data
- Defensive parsing: missing/malformed files return zero values, never error
- SessionState (busy/waiting/idle/dead) separate from AttentionReason
- Non-ASCII path encoding: Claude Code replaces non-ASCII chars with hyphens
- Web dashboard uses SSE (not WebSockets) for real-time updates
- Transcript summary cache by modtime to avoid re-parsing every 3s
- Direct-to-main workflow (no PR overhead for solo project)
- Codex is standing consultant — consult on all design decisions

### Roadmap
- v0.4: Cost analytics (token usage per session)
- v0.5: AI morning brief via `claude -p` (opt-in, costs tokens)
- Launch campaign: Show HN + r/ClaudeCode + X (Tuesday-Thursday optimal)
- Future: Homebrew tap (needs HOMEBREW_TAP_GITHUB_TOKEN secret)

### Competitive Context
- Main competitor: claude-squad (6.3K stars, tmux-based, no analytics)
- Our positioning: "Chief of Staff — not just monitoring, but intelligence"
- Key differentiators: daily brief, metrics, web dashboard, activity detail panel
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
opsdeck web     # web dashboard at localhost:7070
opsdeck version # show version info
```

## User Preferences
- User is CEO/mentor — does not write code, only makes decisions
- Work autonomously: web search or Codex when blocked, only ask user for choices
- Always consult Codex on design decisions before implementing
- Commit directly to main (no PR overhead unless breaking change)
- No Claude attribution in commits/PRs
- macOS provenance fix: terminal added to Developer Tools
