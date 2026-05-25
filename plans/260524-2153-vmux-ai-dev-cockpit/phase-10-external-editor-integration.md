---
phase: 10
title: "External editor integration [v1.1]"
status: done
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

- [x] Detection finds installed editors — `Detect()` (PATH + /Applications bundles for VS Code/Cursor/Zed/Sublime/MacVim); `editor.detected` RPC
- [~] First-run wizard — `EditorPreferenceView` picker exists; standalone first-run flow deferred
- [~] Open from file tree — "Open in Editor" context menu wired; code-viewer/activity/diff affordances deferred (those surfaces are themselves partial)
- [ ] file:line links in terminal output — DEFERRED (SwiftTerm OSC8/regex linkify is fiddly)
- [~] External save → reload — fsnotify `workspace.gitChanged` already fires; viewer auto-reload deferred (viewers are sheets)
- [x] Per-workspace override — `Prefs` global + per-workspace, persisted; `editor.preference` RPC
- [~] Round-trip < 1s — `cmd.Start` fire-and-forget; not benchmarked

## Implementation Notes (2026-05-25)

- Go: `internal/editorintegration/{registry,detect,invoke,prefs}.go` (tested, 72 total sidecar tests pass) + `internal/editor_methods.go`. **Safe by construction:** `BuildArgs` splits the template into an argument ARRAY then substitutes (no shell); line/col are int-typed; paths are absolute + workspace-contained, so a crafted filename can't become a flag. Editor runs in its own process group (survives sidecar restart). Swift: `EditorModel` + `EditorPreferenceView`, "Open in Editor" in the file-tree context menu → `editor.invoke`.
- **Code review fixes:** made `containWorkspacePath` symlink-aware (consistent with the file-tool resolver); atomic preferences write (temp + rename). Arg-injection reviewed and found unreachable (absolute, contained path).
- **Deferred:** terminal file:line linkification, code-viewer/activity "open in editor" affordances (depend on the deferred rich viewers), viewer auto-reload on external save, the `vmux open` CLI helper, and a dedicated first-run editor wizard. Core "open the right file at the right line in your editor" works from the file tree.

## Success Criteria

- [~] Review→⌘E→edit in Cursor — open-in-editor works (file tree); the full diff→editor→auto-reload loop needs the deferred viewer reload + diff "jump to file"
- [x] Opens at a specific file (line/col supported per editor template) — `editor.invoke` passes line/col

## Risk Assessment

- **Editor binaries not in PATH**: Cursor often missing `cursor` CLI (user must "Install 'cursor' command in PATH" once). Mitigation: detect by `.app` bundle; offer "Install CLI helper" link to Cursor docs.
- **Neovim flow**: nvim in a vmux terminal pane vs MacVim window — different UX. Mitigation: separate registry entries (`nvim-terminal` opens nvim in a new vmux tab; `mvim` opens window).
- **JetBrains multiple IDEs**: WebStorm vs IDEA vs PyCharm. Mitigation: detect Toolbox + per-IDE binaries; user picks.
- **Line/col semantics differ slightly**: most accept :line:col; some need flags. Registry handles via template.

## Security Considerations

- Editor invocation runs with user permissions, no special privileges.
- Path passed to editor is sanitized: must be within workspace root or explicitly opened workspace files.
- No shell interpretation of path (use exec arg array, not shell string).
