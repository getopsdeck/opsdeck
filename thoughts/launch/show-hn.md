# Show HN: OpsDeck — Chief of Staff for Claude Code sessions

**Title:** Show HN: OpsDeck — Chief of Staff for all your Claude Code sessions

---

I run 8–15 Claude Code sessions simultaneously across 6+ projects. The
workflow is great until you lose track of which session is waiting for your
input, which one is still churning, and which one crashed 20 minutes ago. I
kept alt-tabbing between terminals looking for the blinking cursor.

OpsDeck is a single Go binary that reads Claude Code's session files (the same
`~/.claude/sessions/` directory Claude Code already writes) and gives you a
complete operational picture:

- **TUI dashboard** — real-time terminal view of all sessions, grouped by
  project, with vi-style navigation and state filters
- **Web dashboard** — browser-based view with SSE live updates, search bar,
  clickable filters, session timeline visualization, git branch per session,
  and cost tracking
- **Daily brief** (`opsdeck brief`) — what happened across all projects
- **Productivity metrics** (`opsdeck metrics`) — today vs yesterday comparison
  with a Momentum score
- **Cost analytics** (`opsdeck costs`) — token usage and spend per session,
  integrates with ccusage for accurate pricing from LiteLLM
- **AI brief** (`opsdeck ai-brief`) — natural language morning summary via
  `claude -p` (opt-in, costs tokens)
- **Watch mode** (`opsdeck watch`) — monitors sessions and sends macOS desktop
  notifications when a session needs your attention
- **Resume** (`opsdeck resume <id>`) — jump back into any session with prefix
  matching

The design constraint I cared most about: it is strictly read-only. It checks
process liveness with `kill -0` (which only tests whether the PID exists — no
signal is sent). It never writes to session files, never sends data anywhere, no
telemetry, no network access. If you use claude-squad or any other session
manager, OpsDeck runs alongside it without interfering. Zero config — no YAML,
no env vars, no API keys. Just run `opsdeck`.

Tech stack: Go 1.26, Bubble Tea v2, Lip Gloss v2 (the Charm TUI ecosystem).
stdlib net/http + SSE for the web dashboard. Single static binary, no runtime
dependencies. 10K+ lines, 45% test coverage. Works on macOS and Linux.

GitHub: https://github.com/getopsdeck/opsdeck

Install:
```
go install github.com/getopsdeck/opsdeck/cmd/opsdeck@latest
```

Or grab a binary from the [releases page](https://github.com/getopsdeck/opsdeck/releases).

MIT licensed. Happy to hear what features would be useful.
