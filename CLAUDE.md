# OpsDeck

A Go + Bubble Tea v2 TUI for monitoring all local Claude Code sessions.
"Chief of Staff for Claude Code" — discover, summarize, query, resume.

## Tech Stack
- Go 1.26 + Bubble Tea v2 (charm.land/bubbletea/v2)
- Lip Gloss v2 + Bubbles v2 (Charm ecosystem)
- SQLite planned for v0.2 (currently in-memory only)

## Architecture
```
cmd/opsdeck/main.go          — entry point
internal/discovery/           — session scanning, PID checking, transcript reading
internal/tui/                 — Bubble Tea TUI (app, bridge, mock, styles, keymap)
internal/tui/components/      — reusable TUI components (table, statusbar)
internal/tui/views/           — composite views (dashboard)
```

## Data Sources (read-only)
1. ~/.claude/sessions/*.json — active session PIDs
2. ~/.claude/projects/<path>/sessions-index.json — session metadata
3. Transcript .jsonl files — last activity timestamps
4. Process table — PID liveness via kill(pid, 0) + start time comparison

## Key Design Decisions
- Pure observer: never mutates Claude Code data
- Defensive parsing: missing/malformed files return zero values, never error
- SessionState (busy/waiting/idle/dead) is separate from AttentionReason (orphaned/stale/no-transcript)
- Non-ASCII path encoding: Claude Code replaces non-ASCII chars with hyphens in project dir names
- Fallback transcript discovery: if direct path fails, scan all project dirs

## Build and Test
```bash
make build    # compile
make test     # race detector
make lint     # go vet
```

## Current Status: v0.1 MVP
- Auto-discovers real sessions
- Groups by project
- State classification working
- TUI with mock and real data
- README, Makefile, .goreleaser ready
- Pending: Codex code review feedback, GitHub repo creation
