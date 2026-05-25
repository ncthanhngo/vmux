---
phase: 5
title: "Native Swift shell + terminal pane [v0.1 ship]"
status: pending
priority: P1
effort: "1.5w"
dependencies: [2, 4]
---

# Phase 5: Native Swift shell + terminal pane [v0.1 ship]

> **Design**: consume the phase 4 design system (tokens, vibrancy, components). Visual source of truth: `visuals/vmux-ui-mockup-apple.html`. Reuse `VibrancyView`, `CardPane`, `SidebarRow`, `StatusBar`, `SegmentedControl` — do not re-style.

## Overview

Build the first usable native UI: left sidebar (workspaces), center area with tabs (terminal only for now), right sidebar (file tree, view-only). Wire SwiftTerm to the Go sidecar's PTY sessions. End state: open a workspace, spawn `claude` in a tab, type, see output, browse files in the tree. Ship as v0.1 internal beta.

## Requirements

- Functional:
  - Workspace sidebar: add/remove workspace (folder picker), select, show branch + dir.
  - Tab bar in center: new tab spawns shell or selected agent. Switch tabs. Close tabs.
  - Terminal pane: SwiftTerm bound to PTY session via sidecar RPC.
  - File tree (right sidebar): lazy-load workspace files, git status indicators (●/+), click reveal in Finder, double-click open in default editor (no in-app editor yet).
  - Persist workspace list + open tabs across app restart.
- Non-functional: input latency < 16ms (1 frame). Smooth scrolling 60fps with 10k-line scrollback.

## Architecture

```
app/Vmux/
├── App/
│   ├── VmuxApp.swift           # @main, SidecarLauncher
│   └── AppState.swift          # ObservableObject, workspace registry
├── Sidecar/
│   ├── SidecarClient.swift     # JSON-RPC client over Unix socket
│   └── RpcTypes.swift          # Codable types matching sidecar schema
├── Workspace/
│   ├── WorkspaceListView.swift
│   └── WorkspaceModel.swift
├── Terminal/
│   ├── TerminalPaneView.swift  # NSViewRepresentable wrapping SwiftTerm
│   ├── PtySession.swift        # binds SwiftTerm ↔ SidecarClient
│   └── AgentLauncher.swift     # presets: shell, claude, codex
├── FileTree/
│   ├── FileTreeView.swift
│   └── FileTreeModel.swift     # lazy nodes, git status
├── Layout/
│   ├── MainSplitView.swift     # 3-region shell from DesignSystem; recursive panes
│   └── TabBarView.swift
└── (Theme handled by phase 4 DesignSystem module — no local Theme.swift)
```

Tab content protocol: `enum TabContent { case terminal(PtySession), case browser(PageId) /* phase 6 */ }`.

## Related Code Files

- Create: all files listed above (Swift PascalCase).
- Reuse (phase 4): `DesignSystem/` — `VibrancyView`, `CardPane`, `SidebarRow`, `StatusBar`, `SegmentedControl`, color/typography tokens.
- Dependencies: `SwiftTerm` via SwiftPM (`migueldeicaza/SwiftTerm`), `swift-log`.

## Implementation Steps

1. `SidecarClient`: connect on app launch (retry 5x with backoff). Codable RPC types. Notification dispatch via Combine `PassthroughSubject`.
2. `MainSplitView` using `NSSplitViewController` wrapped: 3 columns, persisted sizes via UserDefaults.
3. `WorkspaceListView`: add via NSOpenPanel folder picker → RPC `workspace.open` → store in AppState.
4. `TabBarView`: + button menu (Shell / Claude / Codex). Close button per tab. Drag-reorder (basic).
5. `TerminalPaneView`: `NSViewRepresentable<TerminalView>`. On appear → RPC `pty.spawn` → bind data flow. Resize observer → RPC `pty.resize`.
6. `AgentLauncher`: presets map agent name → command + env. Claude: `claude` (cwd=workspace). Codex: `codex`.
7. `FileTreeView`: `OutlineView`-backed; nodes lazy load via sidecar `workspace.listDir`. Subscribe to `workspace.gitChanged` notifications.
8. Keyboard shortcuts: ⌘T new tab, ⌘W close tab, ⌘1-9 switch tab, ⌘N new workspace, ⌘\ toggle left sidebar, ⌘B toggle right sidebar.
9. Persist: workspaces list + last open tabs per workspace (serialize to UserDefaults).
10. Empty state UX: first launch shows "Open a folder to start".

## Todo List

- [ ] App connects to sidecar reliably on launch
- [ ] Add workspace, persists across restart
- [ ] Spawn shell tab, type, see prompt
- [ ] Spawn claude tab in workspace; full Claude Code session usable
- [ ] File tree shows workspace, git status badges
- [ ] Resize + keyboard shortcuts work
- [ ] No crash after 30min stress session

## Success Criteria

- [ ] Daily-driver internal use for 1 week without major bugs (5+ team members)
- [ ] Cold launch → ready < 1.5s
- [ ] Memory < 200MB with 3 workspaces + 5 terminal tabs idle
- [ ] DMG installs and runs on clean macOS 14+ box

## Risk Assessment

- **SwiftTerm gaps**: some escape sequences may be incomplete. Mitigation: test with `claude`, `vim`, `htop`, `tmux` — file SwiftTerm issues or patch fork.
- **Sidecar reconnect**: if sidecar crashes, UI must recover. Mitigation: auto-respawn + restore sessions from sidecar persistence (future) — for v0.1, show error banner + reset state.
- **NSSplitViewController in SwiftUI**: integration friction. Mitigation: accept AppKit-heavy host for the layout shell; SwiftUI for inner views only.

## Security Considerations

- Sidecar socket auth: app and sidecar share a launch-time secret token (passed via env), verified on first RPC.
- File tree paths sanitized server-side (workspace-rooted).
