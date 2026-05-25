---
phase: 6
title: "Browser session orchestration + screenshot preview [v0.2]"
status: done
priority: P1
effort: "4-5d"
dependencies: [3, 4, 5]
---

# Phase 6: Browser session orchestration + screenshot preview [v0.2]

> **Design**: screenshot preview pane + sessions panel use phase 4 `CardPane`, toolbar, lightbox styling per `visuals/vmux-ui-mockup-apple.html`.

## Overview

**Reframed (validation session 2):** vmux does NOT embed a browser pane in the UI. Instead, vmux **orchestrates external Chrome sessions** (spawned by Chrome DevTools MCP from phase 3) and surfaces a lightweight **screenshot preview pane** + browser session controls. Real browser interaction (if needed) happens in the standalone Chrome window. App stays ~40MB, no Chromium bundle, full DevTools-level automation via MCP.

The "embedded browser pane" idea from earlier plan iteration is discarded.

## Requirements

- Functional:
  - **Browser session manager UI**: list active Chrome sessions per workspace, status (running/idle/closed), URL, last screenshot.
  - **Screenshot preview pane** (new tab type, `⌘⇧L`): renders latest screenshot returned by browser MCP. Auto-updates when agent captures new screenshots. Click → full-size lightbox.
  - **Quick browser actions** from vmux UI (route through MCP):
    - "Open URL in this workspace's Chrome" (helpful when user wants to peek)
    - "Reveal Chrome window" (bring external Chrome to front)
    - "Close session" (terminate Chrome process)
  - **Port auto-detect**: parse PTY for `localhost:<port>` → toast "Open in browser session?" → 1-click invokes MCP `browser_navigate`.
  - **Headless mode toggle** per workspace: default headless (true autonomy), toggle to visible when user wants to watch.
- Non-functional: screenshot preview update latency < 500ms after agent captures. Lightbox open < 100ms.

## Architecture

```
sidecar/internal/browser_session/
├── manager.go            # spawned-by-MCP Chrome process tracking (already in phase 3)
├── screenshot_store.go   # per-workspace ring buffer of recent screenshots (PNG bytes + meta)
├── port_detector.go      # PTY output regex → port discovery
└── rpc.go                # browserSession.list, .open, .focus, .close, .latestShot, .subscribe

app/Vmux/BrowserSession/
├── BrowserSessionsView.swift     # right-sidebar panel: list of active sessions
├── ScreenshotPreviewPane.swift   # tab content: shows latest screenshot
├── ScreenshotLightboxView.swift  # full-size modal viewer
├── PortDetectorBanner.swift      # in-tab toast "Open in browser?"
└── BrowserSessionModel.swift
```

RPC additions:
- `browserSession.list(workspaceId)` → `[{id, url, title, lastShot, headless, status}]`
- `browserSession.open(workspaceId, url, headless?)` → MCP-routed; returns session id
- `browserSession.focus(sessionId)` → bring external Chrome window to front (uses `osascript` / NSWorkspace)
- `browserSession.close(sessionId)`
- `browserSession.latestShot(sessionId)` → PNG bytes
- Server→client: `browserSession.shotCaptured(sessionId, shotId, thumbnailB64)`, `browserSession.navigated(sessionId, url)`

Screenshot store: ring buffer 50 shots/session, on-disk under `<ws>/.vmux/shots/`, GC on workspace close.

## Related Code Files

- Create: `sidecar/internal/browser_session/screenshot_store.go`, `port_detector.go`, `rpc.go`
- Create: `app/Vmux/BrowserSession/*.swift`
- Modify: `sidecar/internal/mcp/proxy.go` — intercept browser MCP tool responses; persist screenshots; emit events
- Modify: `app/Vmux/Layout/TabBarView.swift` — add "Screenshot Preview" tab type
- Modify: `app/Vmux/Sidebar/PanelSwitcher.swift` — add "Browser Sessions" panel (phase 8 may shuffle)

## Implementation Steps

1. Screenshot interceptor: in MCP proxy (phase 3), when upstream returns `browser_screenshot` or similar, extract image → save to `screenshot_store` → emit `shotCaptured` event with thumbnail (resized to 320px).
2. `screenshot_store`: in-memory ring buffer + disk persistence. Index by sessionId+shotId.
3. `port_detector`: subscribe to PTY data stream, regex scan, dedupe (per-port-per-session, surface once until port closes), emit detection event.
4. Swift `BrowserSessionsView`: subscribes to `browserSession.list` + event stream. Card per session: URL, last shot thumbnail, status, action buttons (Focus / Close).
5. `ScreenshotPreviewPane`: tab content rendering latest shot for selected session. Polls events; on new shot, fades in. Toolbar: session selector, "Open lightbox", "Save to Desktop".
6. `PortDetectorBanner`: when port detected in active terminal, shows toast above terminal: "Open localhost:3000 in browser?" with 1-click action.
7. `browserSession.focus`: use `NSWorkspace.shared.open(url)` to bring app to front, or `osascript -e 'tell application "Google Chrome" to activate'`.
8. Headless toggle: pass through to MCP server config (Chrome DevTools MCP supports headless flag); recreate session if user toggles mid-flight.
9. Memory management: cap total in-memory screenshots per workspace (50 shots × ~200KB = 10MB max); flush oldest to disk.
10. Integration test: agent screenshots → preview pane updates within 500ms; lightbox opens full quality.

## Todo List

- [x] Screenshot interceptor in MCP proxy — `Proxy.OnScreenshot` scans tool results for image content → store
- [x] Screenshot store (ring buffer 50, PNG normalize + thumbnail, disk 0600, path-safe dir) — tested
- [x] Sessions panel lists sessions — `BrowserSessionsView` from `browserSession.list` + events
- [~] Preview pane auto-updates — thumbnails update live via `shotCaptured`; surfaced in the right-sidebar sessions panel (not a separate tab type — see deviations)
- [x] Lightbox + Save to Desktop — `ScreenshotLightboxView`, full image via `browserSession.latestShot`
- [x] Port detector + 1-click open — server-side regex tap → `portDetected`; `PortDetectorBanner` opens `localhost:PORT`
- [x] Focus button — `browserSession.focus` via `osascript ... activate`

## Implementation Notes (2026-05-25)

- Go: `internal/browsersession/{port_detector,screenshot_store,image_util}.go` (tested), `internal/browser_methods.go`, PTY output `tap`, MCP `OnScreenshot` hook. Swift: `app/Vmux/BrowserSession/*` integrated into the right sidebar + a port banner over the terminal.
- **Deviations:** screenshots are keyed by **upstream MCP server name** (one "session" per browser MCP) — per-tab keying deferred (documented limitation). The screenshot preview lives as a **right-sidebar sessions panel + lightbox** rather than a dedicated tab type (avoids a TabContent refactor); the dedicated "Screenshot Preview" tab + panel-switcher can come when phase 7/9 reshuffle the right sidebar. Port-banner "Open" opens the URL in the user's default browser (peek) — routing through MCP `browser_navigate` needs a live upstream (deferred).
- Verified: Go build + 42 tests (-race); app builds, launches (browser-session subscription active), quits clean, no orphan. **Live Chrome screenshot E2E deferred** (needs a running Chrome DevTools MCP upstream + Chrome).
- Code review fixes: path-safe session dir (`safeKey`) hardening the registry-trust invariant; cleanup of the full image if the thumbnail write fails.

## Success Criteria

- [~] Overnight screenshot timeline — store + sessions panel ready; needs live agent + Chrome to exercise
- [x] Click screenshot → lightbox full image — wired (`latestShot` → `ScreenshotLightboxView`)
- [x] Port detection for npm/Vite/Next/Fastify/Express output — covered by `port_detector_test.go`

## Risk Assessment

- **Chrome window management on multi-monitor**: focusing Chrome from vmux may land on wrong display. Mitigation: respect Chrome's last-position; provide "Move to current display" option in v1.1.
- **Screenshot storage growth**: long autonomous sessions can fill disk. Mitigation: per-workspace size cap (default 500MB), GC oldest.
- **MCP screenshot format variance**: different MCP servers return different encodings. Mitigation: normalize to PNG in interceptor.

## Security Considerations

- Screenshots may contain sensitive page content (tokens, PII). Mitigation: workspace-scoped storage with `0700` perms; "Clear shots" action; never upload anywhere.
- Port-detect toast does NOT auto-open (requires explicit click) to avoid exposing internal services unintentionally.
- `browserSession.focus` cannot be invoked by agent (UI-only action) — defense against agent stealing focus.
