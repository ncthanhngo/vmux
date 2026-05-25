---
title: "vmux - macOS native AI dev cockpit (v0.1 → v1.1)"
description: "Native macOS AI terminal: PTY agents + MCP proxy with Chrome DevTools MCP + Activity/Approval/Session Replay + code viewer with diff review + external editor integration. No embedded browser, no built-in editor."
status: pending
priority: P1
branch: ""
tags: [macos, swift, go, ai-agents, mcp, chrome-devtools-mcp, native, autonomous-agents]
blockedBy: []
blocks: []
created: "2026-05-24T14:53:34.523Z"
createdBy: "ck:plan"
source: skill
---

# vmux - macOS native AI dev cockpit (v0.1 → v1.1)

## Overview

Native macOS app for **autonomous AI dev sessions**: run agents (Claude Code, ...) in PTY terminals; vmux acts as MCP proxy routing tool calls to external MCP servers (primarily **Chrome DevTools MCP** for browser autonomy); observe everything via **Activity stream + Session Replay timeline**; gate dangerous ops via **Approval queue**; review agent file edits inline via **diff review**; jump to user's existing editor (VS Code/Cursor/Neovim/JetBrains) for actual editing.

**Positioning**: "AI terminal observatory" — terminal-first, MCP-driven browser autonomy, deep observability/replay, no embedded browser, no built-in editor. Target user: dev running AI agents unattended (overnight checks, autonomous PRs, prod monitoring).

**Positioning vs cmux**: cmux = terminal + scriptable embedded browser. vmux = same terminal + MCP proxy (no embedded browser) + Activity/Approval/**Session Replay** + diff review + external editor integration. Replay + observability is the moat.

## Stack (locked)

- Shell: SwiftUI + AppKit. Distribution: DMG + Developer ID notarization. macOS 14.0+ (Sonoma). No sandbox.
- Backend: Go sidecar process. IPC: JSON-RPC over Unix socket.
- Terminal: SwiftTerm.
- **Browser automation**: external Chrome (user-installed Chrome.app or Chromium variant) controlled via **Chrome DevTools MCP** (Google official). vmux auto-installs MCP servers; spawns Chrome per-workspace with isolated `user-data-dir`. **No bundled Chromium.** **No embedded browser pane.**
- MCP architecture: vmux Go sidecar is an **MCP proxy** — agents connect to vmux, vmux routes calls to upstream MCP servers + logs to Activity.
- Code viewer (v1.0): CodeMirror 6 read-only in WKWebView; tree-sitter highlighting.
- Markdown viewer: markdown-it + Shiki + KaTeX + Mermaid in WKWebView.
- Editor: **external** (VS Code/Cursor/Neovim/JetBrains/Sublime/Zed) via 1-click "Open in editor" integration. **No built-in editor.**
- Chrome import: Swift (Keychain decrypt for cookies) + Go (SQLite for bookmarks/history) → merge into per-workspace Chrome profile.

## Phases

| Phase | Name | Status |
|-------|------|--------|
| 1 | [Project setup & build pipeline](./phase-01-project-setup-build-pipeline.md) | Done |
| 2 | [Go sidecar foundation (IPC + PTY + workspace)](./phase-02-go-sidecar-foundation-ipc-pty-workspace.md) | Done |
| 3 | [MCP proxy + auto-install external MCP servers (headless)](./phase-03-mcp-server-cdp-browser-wrapper-headless.md) | Pending |
| 4 | [Design system foundation (Apple HIG, SwiftUI)](./phase-04-design-system-foundation.md) | Pending |
| 5 | [Native Swift shell + terminal pane [v0.1 ship]](./phase-05-native-swift-shell-terminal-pane.md) | Pending |
| 6 | [Browser session orchestration + screenshot preview [v0.2]](./phase-06-browser-session-orchestration-screenshot-preview.md) | Pending |
| 7 | [AI Activity + Approval + Session Replay [v1.0 moat]](./phase-07-ai-activity-approval-session-replay.md) | Pending |
| 8 | [Chrome import (cookies/bookmarks/history)](./phase-08-chrome-import-cookies-bookmarks-history.md) | Pending |
| 9 | [Markdown + Code viewer + Diff review + Splits + Notifications [v1.0]](./phase-09-markdown-code-viewer-diff-review.md) | Pending |
| 10 | [External editor integration [v1.1]](./phase-10-external-editor-integration.md) | Pending |

**Design reference:** `visuals/vmux-ui-mockup-apple.html` is the locked visual source of truth (Apple HIG). Phase 4 codifies it as a SwiftUI design system; phases 5/6/7/9 consume it.

## Release milestones

- **v0.1 (4–5 weeks)**: phases 1–5. Internal beta. Design system + terminal + spawn agent + workspace + file tree.
- **v0.2 (+1 week)**: phase 6. Browser session orchestration via MCP + screenshot preview pane.
- **v1.0 (+3–4 weeks)**: phases 7–9. Activity + Approval + **Session Replay** + Chrome import + Markdown + Code viewer + Diff review. Public DMG.
- **v1.1 (+1 week)**: phase 10. External editor integration polish.

**Total to v1.0: ~9–11 weeks single dev.** (Design system adds ~1 week to v0.1 but de-risks all downstream UI by eliminating per-phase restyling.)

## Cross-cutting technical decisions

1. **IPC**: JSON-RPC 2.0 over Unix socket at `~/Library/Application Support/vmux/sidecar.sock`. Bidirectional (Swift→Go calls + Go→Swift events).
2. **MCP architecture**: vmux is an MCP proxy. Upstream MCP servers (Chrome DevTools MCP, Playwright MCP, etc.) installed into `~/Library/Application Support/vmux/mcp-servers/` via `npx`/`uvx`. vmux exposes a unified MCP endpoint to agents.
3. **Browser**: external Chrome (or Brave/Arc/Edge fallback) spawned by Chrome DevTools MCP with per-workspace `--user-data-dir=<ws>/.vmux/chrome-profile`. Headless mode default for autonomy.
4. **Auto-config Claude Code MCP**: opt-in flow with backup + diff preview. Never silent.
5. **Shell hook injection**: write embedded scripts to `~/Library/Application Support/vmux/shell-init/` and inject via `ZDOTDIR`/`BASH_ENV`. **Observe-only**; cannot block executing shell commands.
6. **Workspace config**: `<workspace>/.vmux/workspace.json` — git-shareable (no secrets).
7. **Session storage**: `<workspace>/.vmux/sessions/<sid>/events.jsonl` for activity, `shots/` for screenshots.
8. **Logs**: `~/Library/Logs/vmux/`.

## Dependencies

None (greenfield project). External runtime dependencies (user machine):
- Chrome (or Chromium variant: Brave/Arc/Edge). Wizard offers install help if absent.
- Node.js (for npx-based MCP servers) OR Python (for uvx). vmux detects and uses whichever is available.

## Open questions (revisit before each phase)

- Phase 3: when user has neither node nor python — install one or bundle a minimal runtime? Likely show install link.
- Phase 6: long autonomous sessions (>8h) — UI virtualization strategy needs benchmarking.
- Phase 9: Cursor CLI sometimes missing from PATH — auto-install helper link or rely on user.

## Validation Log

### Session 1 — 2026-05-24

Critical-questions interview, 8 questions, all Recommended:

1. **Browser pane render** → CGS attach (later reversed by session 2)
2. **Shell gating** → MCP true-gate + Shell observe-only
3. **MCP auto-config** → Opt-in with backup + diff preview
4. **macOS deployment target** → macOS 14.0+ (Sonoma)
5. **Chromium per workspace** → 1 instance / ws + idle-stop 10min (later reframed by session 2)
6. **PTY transport** → JSON-RPC + base64 for v0.1
7. **v0.1 agent scope** → Claude Code only
8. **Yjs CRDT** → Node sidecar process (later removed by session 2)

### Session 2 — 2026-05-24 (architecture pivot)

Discussion-driven decisions, no formal interview. Major architectural reframe:

1. **Cut embedded browser pane** → vmux orchestrates external Chrome via Chrome DevTools MCP. App size ~40MB vs ~200MB; effort phase 5 cut from 3w to 4-5d. Justified by: user use case is autonomous agent (headless preferred), DevTools-level network/screenshot/auto-login already provided by MCP server, multi-monitor dev OK with separate Chrome window.
2. **Cut built-in editor (phase 9 LSP + phase 10 vim/collab)** → integrate with user's external editor instead. 6-8 weeks effort cut. Justified by: dev keeps existing editor (Cursor/VS Code/nvim) with their config; vmux's role is verify, not edit; viewer + diff review covers 90% of "watch AI" need.
3. **Add code viewer + diff review** to phase 8 → fills "view AI code without leaving vmux" need without competing with full editors.
4. **Add Session Replay** to phase 6 → killer feature for autonomous AI sessions; scrub timeline of agent actions with screenshots + network + edits + console at each step.
5. **vmux auto-installs MCP servers** (Chrome DevTools MCP primary, Playwright MCP alt) via npx/uvx into managed dir.
6. **Phase 10 deleted** (vim + collaborative editing). Replaced by simpler diff staging in phase 8 (accept/reject hunks, no CRDT).
7. **Positioning shift**: "AI terminal observatory" — moat is observability + replay + diff review, not embedded surfaces.

### Whole-Plan Consistency Sweep — 2026-05-24

- Phase 3 fully rewritten as MCP proxy + installer.
- Phase 5 fully rewritten as browser session orchestration + screenshot preview.
- Phase 6 expanded with Session Replay subsystem.
- Phase 8 expanded with code viewer + diff review.
- Phase 9 fully rewritten as external editor integration.
- Phase 10 deleted.
- plan.md overview + stack + milestones + decisions reconciled.
- App size estimate updated: ~40MB.
- Effort estimate updated: ~8-10 weeks to v1.0 (was 12-16).
- No remaining contradictions.

### Session 3 — 2026-05-25 (design system added)

UI design approved. The Apple-style mockup `visuals/vmux-ui-mockup-apple.html` is now the **locked visual source of truth** (macOS HIG: vibrancy sidebars, SF Pro/SF Mono, system semantic colors, compact 40px titlebar, segmented controls, capsule buttons, floating card panes, accent-fill sidebar selection).

Changes:
1. **New phase 4 "Design system foundation"** — codifies the mockup as a reusable SwiftUI design system (tokens, `NSVisualEffectView` vibrancy bridge, component library). Belongs to v0.1 (built before the shell).
2. **Renumbered phases 4→5 (shell), 5→6 (browser), 6→7 (activity/replay), 7→8 (chrome import), 8→9 (viewers), 9→10 (editor integration).** Files renamed; frontmatter + H1 + dependencies updated.
3. **Dependency edges added**: shell(5), browser(6), activity(7), viewers(9) now depend on design system (4) and reuse its components — no per-phase restyling.
4. **Theme**: automatic light/dark via SwiftUI environment + system accent color. No manual toggle in product (mockup toggle is preview-only).
5. v0.1 milestone now phases 1–5; total-to-v1.0 ~9–11 weeks.

### Whole-Plan Consistency Sweep — 2026-05-25

- Phase numbering 1–10 contiguous; plan.md table links match renamed files (verified).
- All UI phases reference phase 4 design system + mockup path.
- Phase 5 stale refs fixed: `Theme.swift` removed (handled by DesignSystem), `/* phase 5 */` comment → `/* phase 6 */`, "macOS 13+" → "macOS 14+".
- Dependency arrays reconciled to new numbering (5→[2,4]; 6→[3,4,5]; 7→[3,4,5,6]; 8→[6]; 9→[4,5,6,7]; 10→[9]).
- Milestones + effort updated for the inserted phase.
- No remaining contradictions.

**Recommendation:** Plan eligible for implementation. Run `/clear` before `/ck:cook`.
