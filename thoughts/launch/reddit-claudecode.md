# Reddit r/ClaudeCode Post

**Title:** I built a dashboard to manage multiple Claude Code sessions — free,
open source, read-only

---

If you're running more than a few Claude Code sessions at once, you've probably
hit this problem: you open a new task in a new session, go back to something
else, and 10 minutes later you can't remember which session was waiting on you
and which one was still running.

I kept toggling between terminal tabs looking for the cursor. Built OpsDeck to
fix it.

---

**What it does**

- Scans all your Claude Code sessions automatically — no setup required
- Shows each session's state in real time: **BUSY**, **WAITING**, **IDLE**, or **DEAD**
- Groups sessions by project so you can see everything for a given repo at a glance
- Updates every 3 seconds; vi-style keyboard navigation (j/k, /, 1-4 for state filters)
- `opsdeck brief` — plaintext daily summary across all projects
- `opsdeck metrics` — today vs yesterday productivity comparison with a Momentum score
- `opsdeck costs` — token usage and estimated spend per session ($X.XX today)
- `opsdeck web` — browser dashboard with live updates, search, clickable filters, session timeline, and cost tracking
- `opsdeck ai-brief` — AI-powered morning summary via claude -p (opt-in)

---

**What it does NOT do**

- It will not touch your sessions. Read-only. Period.
- No network calls. No telemetry. No config files.
- Doesn't conflict with claude-squad or anything else you're already using

---

**Install**

```bash
go install github.com/getopsdeck/opsdeck/cmd/opsdeck@latest
```

Or grab a binary from [GitHub Releases](https://github.com/getopsdeck/opsdeck/releases).

Then just run `opsdeck`. That's it.

---

Single Go binary, MIT license, works on macOS and Linux. Also has `opsdeck web`
for a browser-based dashboard with live SSE updates. Cost analytics (token
spend per session) coming next. Would love to know what features the community
would find most useful.

GitHub: https://github.com/getopsdeck/opsdeck
