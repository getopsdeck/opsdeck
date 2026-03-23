# Changelog

## v1.2.0 — UI Polish
- Header gradient, 44px touch targets, shimmer loading skeletons
- Session rows hover lift, smooth detail panel transitions
- Mobile responsive at 768px breakpoint
- List sorts by state priority (busy/waiting first)
- `list --active/-a` shows only busy/waiting sessions
- Web auto-opens browser on startup
- Watch notifications play sound on macOS

## v1.1.0 — Status + JSON + Refactor
- `opsdeck status` — one-line summary in 34ms (for shell prompts)
- `list --json` — scriptable output for piping to jq
- Refactored list to use shared monitor.Snapshot
- New tests: TUI timeline, web module, cache eviction

## v1.0.x — PM Polish Pass
- Secretary-style brief: ATTENTION shows what AI was doing, not user messages
- QUICK ACTION matches recommendation (sorted by wait time)
- Only WAITING sessions in ATTENTION (not IDLE)
- Friendly errors: unknown commands, TTY missing, port in use
- Privacy copy qualified for ai-brief
- Colored list output with TTY detection
- Metrics: Files changed fixed, trend context (weekend/outlier)
- Git tag shown in brief (e.g. [main* v1.0.2])

## v1.0.0 — Production Ready
- 12 commands, 12K+ lines, PM reviewed
- Secretary-style `opsdeck brief` with recommendations
- Full 4-agent code review: 8 critical bugs fixed

## v0.9.0 — Burn Rate + ccusage
- Session burn rate ($/hr) in web dashboard
- Full ccusage passthrough for accurate cost reporting
- TUI: press R to resume selected session

## v0.8.0 — Shared Monitor Layer
- `internal/monitor/` eliminates TUI/web data drift
- `opsdeck watch` with macOS desktop notifications
- `opsdeck list` with git branch and wait duration

## v0.7.0 — Resume + List
- `opsdeck resume <id>` with prefix matching
- `opsdeck list` compact session overview
- ccusage integration for cost reporting

## v0.6.0 — Timeline + Git
- Session timeline visualization (web dashboard)
- Git integration: branch, dirty state, last commit per session
- Web search bar and clickable state filters

## v0.5.0 — Full Feature Set
- `opsdeck costs` — token usage and estimated spend
- `opsdeck ai-brief` — AI morning brief via claude -p
- `opsdeck web` — browser dashboard with SSE
- Rich detail panel with activity summary
- Cost badge in web dashboard header

## v0.1.0 — Initial Release
- TUI dashboard with Tokyo Night theme
- `opsdeck brief` — daily briefing
- `opsdeck metrics` — productivity comparison
- Session discovery, PID checking, state classification
- CI green on macOS + Linux

## v1.3.0 — Export, Clean, Favicon
- `opsdeck export` saves brief to markdown file
- `opsdeck clean` lists dead sessions for cleanup
- SVG favicon for web dashboard browser tab
- GitHub badges in README (CI, release, license, Go report)
- Status command shows daily cost ($XXX today)
- Web Morning Brief card opens by default
- Help reorganized into categories
- CHANGELOG.md added
- Opus plan + Sonnet implement workflow: 5/5 success rate
