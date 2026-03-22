# Show HN: OpsDeck — real-time dashboard for all your Claude Code sessions

**Title:** Show HN: OpsDeck — real-time dashboard for all your Claude Code sessions

---

I run 8–12 Claude Code sessions at a time across several projects. The workflow
is great until you lose track of which session is waiting for your input, which
one is still churning, and which one silently died 20 minutes ago. I kept
alt-tabbing between terminals looking for the blinking cursor.

OpsDeck is a single Go binary that reads Claude Code's session files (the same
`~/.claude/sessions/` directory Claude Code already writes) and gives you a
real-time terminal dashboard. It classifies each session as BUSY (active in the
last 30s), WAITING (30s–5min, likely needs you), IDLE (5min+), or DEAD (process
gone). Sessions are grouped by project so you can see at a glance what's
happening across your whole codebase. Other commands:
- `opsdeck brief` — plaintext daily summary across all projects
- `opsdeck metrics` — today vs yesterday productivity comparison
- `opsdeck costs` — token usage and estimated spend per session
- `opsdeck web` — browser dashboard with real-time SSE, search, filters, session timeline, and cost tracking

The design constraint I cared most about: it is strictly read-only. It checks
process liveness with `kill -0` (which only tests whether the PID exists — no
signal is sent). It never writes to session files, never sends data anywhere, no
telemetry, no network access. If you use claude-squad or any other session
manager, OpsDeck runs alongside it without interfering. Zero config — no YAML,
no env vars, no API keys. Just run `opsdeck`.

Tech stack: Go 1.26, Bubble Tea v2, Lip Gloss v2 (the Charm TUI ecosystem).
Single static binary, no runtime dependencies. Works on macOS and Linux.

GitHub: https://github.com/getopsdeck/opsdeck

Install:
```
go install github.com/getopsdeck/opsdeck/cmd/opsdeck@latest
```

Or grab a binary from the [releases page](https://github.com/getopsdeck/opsdeck/releases).

MIT licensed. It also ships with `opsdeck web` — a browser-based dashboard
with real-time SSE updates, same data, same dark theme. Cost analytics (token
spend per session) is next. Happy to hear what features would be useful.
