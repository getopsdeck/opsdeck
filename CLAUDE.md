# OpsDeck

**Chief of Staff for Claude Code** — monitor all your Claude Code sessions from a single terminal.

## Project Status

v0.1 is live on GitHub: https://github.com/getopsdeck/opsdeck
CI green (macOS + Linux). 6,000+ lines of Go code.

### What's Done
- Session discovery (PID checking, transcript reading, state classification)
- Bubble Tea v2 TUI dashboard with Tokyo Night theme
- `opsdeck brief` — daily briefing across all projects
- `opsdeck metrics` — today vs yesterday productivity comparison with Momentum score
- Real data working (tested against 19 real Claude Code sessions, 7 projects)

### Recently Merged
- PR #2: fix/project-identity — multi-level disambiguation for same-name projects
- PR #3: fix/table-viewport — viewport clipping in both flat and grouped view
- PR #4: improve/brief-quality — CEO-friendly brief output with SummarizeActivities

### Current Focus
- Launch prep: GoReleaser release, Hero GIF, README polish
- Next: v0.3 web dashboard

### Architecture
```
cmd/opsdeck/main.go          — entry point, brief/metrics subcommands
internal/discovery/           — session scanning, PID checking, transcript reading
internal/intel/               — activity extraction, daily brief, metrics, summarize
internal/tui/                 — Bubble Tea TUI (app, bridge, mock, styles, keymap)
internal/tui/components/      — reusable TUI components (table, statusbar)
internal/tui/views/           — composite views (dashboard)
```

### Key Design Decisions
- Pure observer: never mutates Claude Code data
- Defensive parsing: missing/malformed files return zero values, never error
- SessionState (busy/waiting/idle/dead) separate from AttentionReason
- Non-ASCII path encoding: Claude Code replaces non-ASCII chars with hyphens
- In-memory only for v0.1 (no SQLite yet)
- Codex is standing consultant — consult on all design decisions

### Roadmap
- v0.3: Web dashboard (embedded HTTP server)
- v0.4: AI morning brief via `claude -p` (opt-in, costs tokens)
- Future: Hero GIF, launch campaign (Show HN, r/ClaudeCode)

### Competitive Context
- Main competitor: claude-squad (6.3K stars, tmux-based, no analytics)
- Our positioning: "most reliable way to see which sessions need your attention"
- Target: 5K+ GitHub stars

## Tech Stack
- Go 1.26 + Bubble Tea v2 (charm.land/bubbletea/v2)
- Lip Gloss v2 + Bubbles v2 (Charm ecosystem)
- GitHub org: getopsdeck
- Module path: github.com/getopsdeck/opsdeck

## Build and Test
```bash
make build    # compile
make test     # race detector
make lint     # go vet
opsdeck       # TUI dashboard
opsdeck brief # daily briefing
opsdeck metrics # productivity comparison
```

## User Preferences
- User is CEO/mentor — does not write code, only makes decisions
- Work autonomously: web search or Codex when blocked, only ask user for choices
- Always consult Codex on design decisions before implementing
- Submit PRs for all changes (Codex auto-reviews via GitHub)
- No Claude attribution in commits/PRs
- macOS provenance fix: terminal added to Developer Tools
