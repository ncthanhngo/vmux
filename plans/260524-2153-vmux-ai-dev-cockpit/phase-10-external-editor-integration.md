---
phase: 10
title: "External editor integration [v1.1]"
status: pending
priority: P2
effort: "3-4d"
dependencies: [9]
---

# Phase 10: External editor integration [v1.1]

## Overview

**Reframed (validation session 2):** vmux does NOT build a full IDE editor. Instead, **integrate seamlessly with the user's existing editor** (VS Code, Cursor, Neovim, JetBrains). 1-click "Open in editor" from any file reference in vmux (file tree, diff review, activity panel, terminal output). Watch external editor saves and reflect in vmux viewer. Detect user's preferred editor automatically.

Phases 9 (full editor + LSP) and 10 (vim + collaborative editing) from the original plan are discarded.

## Requirements

- Functional:
  - **Auto-detect** user's editor preference: check `$EDITOR`, `$VISUAL`, common CLI binaries (`code`, `cursor`, `nvim`, `mvim`, `idea`, `subl`, `mate`), Application bundles (Cursor.app, VSCode.app, JetBrains Toolbox).
  - **"Open in editor" action** wherever a file path appears:
    - File tree right-click + double-click (configurable single-click)
    - Code viewer toolbar
    - Diff review (jump to file at first hunk)
    - Activity panel items (file edited by agent)
    - Terminal output (clickable `file:line` references — basic OSC8 hyperlink support)
  - **Editor invocations** with line/col when supported:
    - VS Code: `code -g <path>:<line>:<col>`
    - Cursor: `cursor -g <path>:<line>:<col>`
    - Neovim (terminal): `nvim +<line> <path>` or `mvim --remote-tab-silent`
    - JetBrains: `idea64 --line <line> <path>` (or generic `tool-launcher`)
    - Sublime: `subl <path>:<line>:<col>`
    - Fallback: `open <path>` (macOS default app)
  - **Per-workspace + global default**: user can override editor per workspace.
  - **Watch external saves**: existing file watcher (phase 2) detects mtime changes from external editor → code viewer reloads + diff review updates.
- Non-functional: editor launch round-trip from click to focused editor < 1s for already-running editor; < 3s cold launch.

## Architecture

```
sidecar/internal/editor_integration/
├── detect.go            # scan PATH + /Applications for known editors, rank by recency
├── invoke.go            # build command-line per editor with line/col args
├── registry.go          # known editor profiles (binary, args template, supports line/col)
└── rpc.go               # editor.detected, .invoke, .preference

app/Vmux/EditorIntegration/
├── EditorPreferenceView.swift     # settings: choose editor, override per workspace
├── OpenInEditorAction.swift       # reusable action invoked from many surfaces
└── FileLineLinkifier.swift        # terminal output regex → clickable
```

RPC additions:
- `editor.detected()` → `[{id, name, binary, supportsLineCol, lastUsed}]`
- `editor.invoke(workspaceId, path, line?, col?)` → exit code
- `editor.preference(workspaceId, editorId)` → save preference

Known editor registry (`shared/editor-registry.json`):
```json
[
  {"id": "vscode", "name": "VS Code", "bin": "code", "args": "-g {path}:{line}:{col}", "lineCol": true},
  {"id": "cursor", "name": "Cursor", "bin": "cursor", "args": "-g {path}:{line}:{col}", "lineCol": true},
  {"id": "nvim-terminal", "name": "nvim (in terminal)", "bin": "nvim", "args": "+{line} {path}", "lineCol": true},
  {"id": "mvim", "name": "MacVim", "bin": "mvim", "args": "--remote-tab-silent +{line} {path}", "lineCol": true},
  {"id": "idea", "name": "IntelliJ IDEA", "bin": "idea64", "args": "--line {line} {path}", "lineCol": true},
  {"id": "subl", "name": "Sublime Text", "bin": "subl", "args": "{path}:{line}:{col}", "lineCol": true},
  {"id": "zed", "name": "Zed", "bin": "zed", "args": "{path}:{line}:{col}", "lineCol": true},
  {"id": "fallback-open", "name": "macOS default", "bin": "open", "args": "{path}", "lineCol": false}
]
```

## Related Code Files

- Create: `sidecar/internal/editor_integration/*.go`
- Create: `app/Vmux/EditorIntegration/*.swift`
- Create: `shared/editor-registry.json`
- Modify: `app/Vmux/FileTree/FileTreeView.swift` — add "Open in editor" context menu + dbl-click action
- Modify: `app/Vmux/CodeViewer/CodeViewerPaneView.swift` — toolbar button
- Modify: `app/Vmux/Activity/ActivityItemView.swift` — file-edit items get "Open in editor" affordance
- Modify: `app/Vmux/Terminal/TerminalPaneView.swift` — linkify `file:line` patterns

## Implementation Steps

1. Define registry JSON; embed into binary via `//go:embed`.
2. `editor_integration.detect`: probe `which <bin>` for each registry entry; check `/Applications/*.app/Contents/MacOS/<bin>`. Rank by recency (lastUsed from preferences).
3. First-run wizard: if no editor preference set, show detected editors → user picks default.
4. `OpenInEditorAction`: SwiftUI button + keyboard shortcut (`⌘E` in code viewer, default `Cmd-click` on file paths in terminal).
5. `FileLineLinkifier`: regex over PTY output (`<path>:<line>(:<col>)?`); validate path exists in workspace; render as underlined link in SwiftTerm.
6. `editor.invoke`: substitute placeholders in args template; `os/exec` with `Setpgid:true` so editor survives sidecar restart; capture stderr on failure for error toast.
7. Per-workspace override: workspace settings panel "Editor: [Use global] / Override → [VS Code | Cursor | …]".
8. Watch saves: existing fsnotify watcher (phase 2) emits events; code viewer subscribes; on mtime change reload + recompute diff.
9. "Open here" terminal command (optional convenience): bundle a `vmux` CLI binary that user can put in PATH; `vmux open <file>` from inside any terminal pane.
10. Settings sync: editor preferences stored in `~/Library/Application Support/vmux/preferences.json`.

## Todo List

- [ ] Detection finds installed editors on test machine (Cursor, VS Code, nvim, JetBrains)
- [ ] First-run wizard
- [ ] Open from file tree, code viewer, activity, diff review
- [ ] File:line links in terminal output
- [ ] External save → code viewer reloads
- [ ] Per-workspace override works
- [ ] Round-trip from click to focused editor < 1s

## Success Criteria

- [ ] Daily flow: user reviews agent's 5 edits in vmux diff viewer; 1 edit needs human change → ⌘E opens Cursor at right file/line in < 1s; saves; vmux viewer reloads with new content automatically
- [ ] User never has to manually navigate to file in editor — always opens at agent's last edit point

## Risk Assessment

- **Editor binaries not in PATH**: Cursor often missing `cursor` CLI (user must "Install 'cursor' command in PATH" once). Mitigation: detect by `.app` bundle; offer "Install CLI helper" link to Cursor docs.
- **Neovim flow**: nvim in a vmux terminal pane vs MacVim window — different UX. Mitigation: separate registry entries (`nvim-terminal` opens nvim in a new vmux tab; `mvim` opens window).
- **JetBrains multiple IDEs**: WebStorm vs IDEA vs PyCharm. Mitigation: detect Toolbox + per-IDE binaries; user picks.
- **Line/col semantics differ slightly**: most accept :line:col; some need flags. Registry handles via template.

## Security Considerations

- Editor invocation runs with user permissions, no special privileges.
- Path passed to editor is sanitized: must be within workspace root or explicitly opened workspace files.
- No shell interpretation of path (use exec arg array, not shell string).
