---
phase: 7
title: "AI Activity + Approval + Session Replay [v1.0 moat]"
status: pending
priority: P1
effort: "2w"
dependencies: [3, 4, 5, 6]
---

# Phase 7: AI Activity + Approval + Session Replay [v1.0 moat]

> **Design**: Activity panel, approval cards, replay scrubber use phase 4 components (grouped inset cards, capsule buttons, badge pills, segmented panel switcher) per `visuals/vmux-ui-mockup-apple.html`.

## Overview

The moat. Three integrated features for autonomous AI dev sessions: (1) **Activity stream** logs every agent action (MCP tool calls, file ops, browser actions, commands); (2) **Approval queue** gates dangerous operations; (3) **Session Replay** — scrub a timeline of an agent's run, see screenshots + network captures + console + edits at each step. Built for the "leave AI running overnight, review in morning" workflow.

Validation session 2: Session Replay confirmed as killer feature.

## Requirements

- Functional:
  - **Activity panel** (right sidebar): live stream per session — MCP tool calls with arg/result peek, screenshots inline, file edits with diff peek, command output peek, browser navigations.
  - **Approval queue**: rule-engine gates MCP `command.run` + dangerous tool calls. Modes per workspace: `watch` (log only), `gate` (block on rules), `sandbox` (block all writes/execs). Default = `watch`.
  - **Shell hook** (zsh/bash/fish): preexec/precmd report commands. OBSERVE-ONLY for command interception (locked in validation session 1); cannot block executing shell commands.
  - **Session Replay timeline**: per-session scrub bar; at any time t, show: latest screenshot, last MCP tool call, file diffs in window, commands run, console output. Click any timeline entry → jump to detail.
  - **Cost tracker** per agent session (parse Claude Code session JSONL or MCP usage).
  - **Secret redaction** in logs (token=, AWS_SECRET, password patterns).
- Non-functional: timeline scrub responsive (<100ms per step), activity event lag < 200ms, replay UI usable for 8-hour autonomous sessions (thousands of events).

## Architecture

```
sidecar/internal/
├── activity/
│   ├── event.go            # event schema, append-only log per session
│   ├── store.go            # in-memory ring + on-disk JSONL at <ws>/.vmux/sessions/<sid>/events.jsonl
│   ├── enrich.go           # attach screenshot refs, diff snapshots, command output excerpts
│   ├── stream.go           # pub-sub for UI subscribers (backpressure-safe)
│   └── redact.go           # secret-pattern scrubber
├── approval/
│   ├── rules.go            # default ruleset + per-workspace overrides
│   ├── queue.go            # pending items, decision channel
│   ├── modes.go            # watch | gate | sandbox
│   └── gate.go             # middleware applied to MCP proxy + shell hook reports
├── replay/
│   ├── index.go            # timeline index over events.jsonl
│   ├── snapshot.go         # at-time-t reconstruction (last shot, open files, etc.)
│   └── rpc.go              # replay.timeline, .at, .events, .export
└── shell_hook/
    ├── init_zsh.sh         # //go:embed
    ├── init_bash.sh
    ├── init_fish.fish
    └── reporter.go         # ingest hook events via $VMUX_SOCK

app/Vmux/Activity/
├── ActivityPanelView.swift     # live stream
├── ActivityItemView.swift
├── ApprovalSheetView.swift     # modal on dangerous action
├── ApprovalQueueView.swift     # list of pending in Activity panel
├── CostTrackerView.swift
└── ActivityModel.swift

app/Vmux/Replay/
├── ReplayTabView.swift         # new tab type "Session Replay"
├── TimelineScrubberView.swift  # horizontal scrubber, density heatmap
├── ReplayDetailView.swift      # at-time-t: screenshot + tool call + diffs
└── ReplayModel.swift
```

Approval flow:
1. Agent calls MCP tool → proxy → `approval.gate.Check(action, mode)`.
2. If `watch`: pass through, log to activity.
3. If `gate`: match rules → if dangerous, push to queue, block caller, notify UI.
4. If `sandbox`: deny all `command.run`/`file.write`/external network beyond allowlist.
5. UI decision → unblock caller; for blocked commands, return error to agent.

Replay model:
- Each session = ordered events.jsonl (append-only).
- Timeline densitized into "moments" (group sub-second events).
- `replay.at(sessionId, t)` returns reconstructed state: latest screenshot ref, files open by agent in window, commands ran in window, console errors in window.

## Related Code Files

- Create: `sidecar/internal/activity/*.go`, `approval/*.go`, `replay/*.go`, `shell_hook/*`
- Create: `app/Vmux/Activity/*.swift`, `app/Vmux/Replay/*.swift`
- Modify: `sidecar/internal/mcp/proxy.go` — wrap every call with approval+activity middleware
- Modify: `sidecar/internal/pty/session.go` — inject shell hook env when spawning shells
- Modify: `app/Vmux/Sidebar/PanelSwitcher.swift` — Activity becomes default panel when agent active

## Implementation Steps

1. Event schema (`shared/activity-schema.md`): `{ts, sessionId, kind, actor, summary, refs, riskLevel}`. Kinds: `tool_call`, `file_edit`, `command_run`, `browser_nav`, `screenshot`, `console`, `attention`, `cost`.
2. Activity store: append-only JSONL + in-memory ring (last 5k); on-demand load from disk for older.
3. Approval rules default set (rm -rf, sudo, curl|sh, dd, mkfs, network to non-localhost beyond allowlist).
4. Mode UI: workspace settings dropdown `watch/gate/sandbox` + per-rule overrides.
5. Shell hooks: write embedded scripts to `~/Library/Application Support/vmux/shell-init/`; PTY spawn sets `ZDOTDIR`/`BASH_ENV`/`fish_function_path`. Document **observe-only** clearly in UI.
6. MCP proxy middleware: every call → enrich event → log → gate-check → forward → log result → return.
7. Cost tracker: tail Claude Code session JSONL OR MCP usage events; map to model pricing table; expose in CostTrackerView.
8. Activity panel UI: LazyVStack of items grouped by session; per-item peek (screenshot thumb, diff lines, command stdout first N lines).
9. Replay tab: scrubber renders event density; click/drag → set t; detail view assembles `replay.at(t)` payload.
10. Export session: "Export as report" → markdown + screenshots zip for sharing/post-mortem.

## Todo List

- [ ] Activity events captured for every MCP call
- [ ] Shell hook events for command preexec/postexec
- [ ] Approval blocks dangerous command in `gate` mode
- [ ] Mode toggle persists per workspace
- [ ] Session JSONL persisted; reload on app restart
- [ ] Replay scrubber smooth on 5k-event session
- [ ] Cost tracker shows non-zero for active Claude session
- [ ] Secret redaction works on common patterns
- [ ] Export session bundle works

## Success Criteria

- [ ] User runs claude unattended 4h → next morning opens replay → scrubs entire session in <30s, spots a failed deploy step from a glance
- [ ] Blocking `rm -rf node_modules` from agent works with clear UI prompt
- [ ] No false positives in default `gate` ruleset on typical npm/git workflows

## Risk Assessment

- **Shell hook is OBSERVE-ONLY, not a true gate** (locked validation session 1). Must surface this UI-wise: shell command activity items show "logged" badge, MCP tool calls show "gated" badge.
- **Replay UI complexity**: large sessions (10k+ events) may lag. Mitigation: density heatmap pre-computed; virtualized list; jump-by-moment instead of per-event.
- **Cost tracker breaking on JSONL format change**: Anthropic may change schema. Mitigation: defensive parser; fall back to MCP usage events.
- **Disk usage**: long sessions + screenshots → GB. Mitigation: per-workspace cap (default 1GB); GC oldest sessions after N days.

## Security Considerations

- Activity log may capture command lines containing secrets — `redact.go` scrubs known patterns; user can mark events to never-export.
- Approval rules file workspace-trusted; never auto-loaded from network sources.
- Hook scripts signed at sidecar build; verified on each spawn.
- Replay export bundle includes opt-in redaction pass (defaults strict).
