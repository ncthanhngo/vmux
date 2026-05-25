---
phase: 9
title: "Markdown viewer + Code viewer + Diff review + Splits + Notifications [v1.0]"
status: pending
priority: P1
effort: "1.5w"
dependencies: [4, 5, 6, 7]
---

# Phase 9: Markdown viewer + Code viewer + Diff review + Splits + Notifications [v1.0]

> **Design**: viewers/diff/splits use phase 4 `CardPane`, recursive split layout, capsule hunk buttons, code/diff theme tokens per `visuals/vmux-ui-mockup-apple.html`.

## Overview

v1.0 polish phase, **expanded with code viewer + diff review** (validation session 2): flexible split panes; Warp-style markdown rendering; **read-only code viewer with syntax highlighting**; **inline diff review** for agent file edits (accept/reject hunks without leaving vmux); macOS notifications; status bar; right-sidebar panel switcher with more panels.

Read-only stance is deliberate: vmux verifies AI work; editing happens in user's external editor (phase 9).

## Requirements

- Functional:
  - **Recursive split panes** in center: ⌘D (right), ⌘⇧D (down), drag-to-resize, persisted per workspace.
  - **Markdown viewer**: opens `.md` files; markdown-it + Shiki + KaTeX + Mermaid. View / Source / Split modes.
  - **Code viewer (read-only)**: CodeMirror 6 read-only mode, tree-sitter syntax for 20+ languages, themes match app. Search-in-file (⌘F), go-to-line (⌘G), select+copy, "Open in editor" button (phase 9).
  - **Diff review pane**: when agent edits file, viewer shows inline diff (red/green hunks). Per-hunk Accept/Reject (⌘⏎ / ⌘⌫). "Accept all" / "Reject all" toolbar buttons. Pending hunks tracked in workspace.
  - **System notifications** (UserNotifications): badge dot on workspace, pane border highlight, macOS notification on agent attention.
  - **Status bar**: branch, dirty count, listening port chips, active agent state.
  - **Panel switcher**: Files / Git / AI Activity / Ports / Tasks / Bookmarks / History / Browser Sessions.
- Non-functional: markdown render < 100ms; code viewer open large file (10k lines) < 500ms; diff hunk accept < 50ms round-trip.

## Architecture

```
app/Vmux/Markdown/
├── MarkdownPaneView.swift            # WKWebView host
├── MarkdownRenderer.swift            # bundled JS pipeline
└── Resources/markdown-runtime/       # markdown-it + Shiki + KaTeX + mermaid esbuild bundle

app/Vmux/CodeViewer/
├── CodeViewerPaneView.swift          # WKWebView host for CM6 read-only
├── CodeViewerSession.swift           # file path, language, dirty (when agent edits)
├── DiffReviewToolbar.swift           # accept/reject controls
├── EditorMessageBridge.swift         # JS ↔ Swift
└── Resources/code-viewer-runtime/    # CM6 read-only + tree-sitter + themes esbuild bundle

app/Vmux/Notifications/
├── NotificationCenterBridge.swift
├── AttentionDetector.swift
└── BadgeOverlay.swift

app/Vmux/StatusBar/
├── StatusBarView.swift
└── PortsIndicator.swift

app/Vmux/Sidebar/
├── PanelSwitcher.swift
├── GitPanelView.swift
├── PortsPanelView.swift
├── TasksPanelView.swift
├── BookmarksPanelView.swift          # reads from Chrome profile via sidecar
├── HistoryPanelView.swift
└── BrowserSessionsPanel.swift        # from phase 5

sidecar/internal/diffreview/
├── pending.go            # tracks pending hunks per file
├── hunk.go               # parse unified diff, identify hunks, apply individual ones
└── rpc.go                # diffReview.list, .accept, .reject
```

Diff review flow:
1. Agent calls MCP `file.write` (mediated by vmux) OR external editor changes file.
2. Sidecar `diffreview` computes diff vs last-known-clean; in `review` workspace mode, stages hunks pending (writes go to `<file>.vmux-staged`).
3. UI shows code viewer with inline diff decorations; per-hunk Accept commits to actual file, Reject discards.
4. In `auto-apply` mode, writes pass through directly (default for low-risk workspaces).

Recursive split pane: replace fixed center with `PaneNode` tree (`leaf(tab)` or `split(direction, [PaneNode])`); drag-divider; drag-tab to spawn split. Persisted in workspace state.

## Related Code Files

- Create: `app/Vmux/Markdown/*.swift`, `code-viewer-build/` (Node project: CM6 read-only + tree-sitter wasm + themes esbuild)
- Create: `app/Vmux/CodeViewer/*.swift`
- Create: `app/Vmux/Notifications/*.swift`, `app/Vmux/StatusBar/*.swift`
- Create: `app/Vmux/Sidebar/PanelSwitcher.swift`, `GitPanelView.swift`, `PortsPanelView.swift`, `TasksPanelView.swift`, `BookmarksPanelView.swift`, `HistoryPanelView.swift`
- Create: `sidecar/internal/diffreview/*.go`
- Modify: `sidecar/internal/mcp/tools/file.go` — route `file.write` through diffreview (review mode)
- Modify: `app/Vmux/Layout/MainSplitView.swift` — recursive PaneNode

## Implementation Steps

1. Build `markdown-runtime` bundle (markdown-it + Shiki + KaTeX + mermaid via esbuild) → `Resources/markdown-runtime/`.
2. Build `code-viewer-runtime` bundle: CM6 + `@codemirror/lang-*` + tree-sitter wasm + themes. Read-only config (no edit commands, no LSP).
3. WKWebView shells for markdown + code viewer with shared message bridge protocol (load file, theme switch, scroll position).
4. Split-pane refactor: PaneNode model, recursive layout view, drag-divider with min-width clamps.
5. Diff review backend: `diffreview.pending`, integrate with MCP `file.write` middleware in review mode.
6. Code viewer diff decorations: receive hunks list, render inline + and - lines with gutter markers, per-hunk floating Accept/Reject buttons.
7. Attention detector: tail PTY for `\? `, prompt patterns, idle > 30s after user input → attention event.
8. Notification bridge: request authorization; map attention to `UNNotificationRequest`; dedupe.
9. Status bar: subscribe to workspace + session state.
10. Sidebar panels:
    - Ports: `lsof` polling via sidecar (2s).
    - Tasks: parse package.json scripts, Makefile, justfile, Taskfile.yml.
    - Git: `git status --porcelain=v2 -z` + "AI generate commit msg" button.
    - Bookmarks/History: read from per-workspace Chrome profile (sidecar exposes RPCs).

## Todo List

- [ ] Splits work recursively, persist across restart
- [ ] Markdown render matches GitHub quality
- [ ] Code viewer: tree-sitter highlighting, 20+ languages
- [ ] Diff review accept/reject works per-hunk
- [ ] All 7 sidebar panels usable
- [ ] Macos notification + badge on agent attention
- [ ] Status bar updates live

## Success Criteria

- [ ] User reviews 5 agent file edits using inline diff + accept/reject; never opens external editor for review (only if they want to modify)
- [ ] Open `README.md` → looks at least as nice as GitHub render
- [ ] Hidden vmux gets system notification + workspace badge when agent needs input

## Risk Assessment

- **WKWebView bundle sizes**: markdown ~1.5MB + code-viewer ~1.5MB on disk. Mitigation: lazy load language packs + mermaid/katex.
- **Diff review correctness**: hunk boundary edge cases (overlapping edits, binary files). Mitigation: aggressive tests; binary files skip review (auto-apply with warning).
- **Notification spam**: poor attention heuristic. Mitigation: per-pattern debounce, user toggle per agent type.
- **Tree-sitter wasm size**: each grammar ~200KB-1MB. Mitigation: lazy load on first file of language.

## Security Considerations

- WKWebViews CSP-locked: `default-src 'none'; script-src 'self'; style-src 'self' 'unsafe-inline'`. No network.
- Markdown raw HTML disabled by default.
- Diff review pending files (`<file>.vmux-staged`) stored under workspace `.vmux/` (gitignored by default).
- Agent cannot auto-accept its own hunks — only user can (UI + sidecar enforcement).
