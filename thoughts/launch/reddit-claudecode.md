# Reddit r/ClaudeCode Post

**Title:** I built a "Chief of Staff" for Claude Code — monitors all your sessions, sends you a daily brief, tracks costs, and alerts when sessions need you

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
- Groups sessions by project so you can see everything at a glance
- Shows git branch and dirty state per session
- `opsdeck brief` — plaintext daily summary across all projects
- `opsdeck metrics` — today vs yesterday productivity comparison with a Momentum score
- `opsdeck costs` — token usage and estimated spend (integrates with ccusage)
- `opsdeck web` — browser dashboard with live SSE updates, search, filters, session timeline, and cost tracking
- `opsdeck ai-brief` — AI-powered morning summary via claude -p (opt-in)
- `opsdeck watch` — monitors sessions and sends macOS desktop notifications when one needs you
- `opsdeck resume <id>` — jump back into any session (supports prefix matching)
- `opsdeck list` — compact overview of all sessions

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

Single Go binary, MIT license, works on macOS and Linux. 10K+ lines, 12
commands. Would love to know what features the community would find most useful.

GitHub: https://github.com/getopsdeck/opsdeck
