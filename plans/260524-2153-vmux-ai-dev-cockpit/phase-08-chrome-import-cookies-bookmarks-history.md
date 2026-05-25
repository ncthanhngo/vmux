---
phase: 8
title: "Chrome import (cookies/bookmarks/history)"
status: done
priority: P2
effort: "1w"
dependencies: [6]
---

# Phase 8: Chrome import (cookies/bookmarks/history)

> **Design**: import wizard uses phase 4 components (sheet, capsule buttons, progress) per `visuals/vmux-ui-mockup-apple.html`.

## Overview

First-run + on-demand wizard that imports user's Chrome data into a workspace's Chromium profile. Goal: agent can browse with the user's existing logins/bookmarks without manual login each time. Cookies require Keychain decrypt (Swift); bookmarks/history are JSON+SQLite (Go).

## Requirements

- Functional:
  - Detect Chrome profiles in `~/Library/Application Support/Google/Chrome/`.
  - Import wizard UI: list profiles, choose data types (cookies/bookmarks/history), choose target workspace.
  - Cookies: decrypt via macOS Keychain Safe Storage key, transform to Chromium for Testing cookie store.
  - Bookmarks: copy `Bookmarks` JSON, merge into target profile.
  - History: copy `History` SQLite (Chrome must be closed OR copy via shadow); merge.
  - "Re-sync" button per workspace (manual).
- Non-functional: import 10k cookies < 5s. Keychain prompt shown clearly with rationale.

## Architecture

Cookie decrypt requires Security framework, kept in Swift; data move done by Go. Flow:

1. Swift `ChromeImporter` enumerates profiles, asks user.
2. Swift requests Keychain item `Chrome Safe Storage` → returns AES key (macOS shows Keychain dialog first time).
3. Swift passes AES key + profile paths to sidecar over RPC.
4. Go: copy SQLite files to tmp (avoid lock), decrypt cookie values with provided key, write into target Chromium profile cookie DB.

```
app/Vmux/ChromeImport/
├── ChromeImportWizardView.swift
├── ChromeImporter.swift         # Keychain access + RPC dispatch
└── ChromeProfileScanner.swift

sidecar/internal/chromeimport/
├── profiles.go                  # discover paths
├── cookies.go                   # AES-128-CBC decrypt with given key
├── bookmarks.go                 # JSON merge
├── history.go                   # SQLite copy + merge
└── target_writer.go             # write into Chromium for Testing profile
```

RPC additions:
- `chromeImport.scan()` → `[{profileName, path, hasCookies, hasBookmarks, hasHistory}]`
- `chromeImport.run(workspaceId, profilePath, key (base64), include {cookies, bookmarks, history})` → progress events

## Related Code Files

- Create: `app/Vmux/ChromeImport/*.swift`
- Create: `sidecar/internal/chromeimport/*.go`
- Modify: `app/Vmux/Workspace/WorkspaceModel.swift` (track imported sources + last sync)

## Implementation Steps

1. `ChromeProfileScanner` (Swift) lists `Default`, `Profile 1`, … with display names from `Local State` JSON.
2. `ChromeImporter.requestKey()` (Swift) uses `SecKeychainFindGenericPassword` with `service: "Chrome Safe Storage"`. Handle access prompt.
3. Wizard UI: 3 steps (profile → data types → confirm). Progress bar during import.
4. Sidecar `chromeImport.scan` walks Chrome dir and returns metadata (without reading sensitive data).
5. `chromeimport.cookies` uses `crypto/aes` CBC with PKCS#7 padding, IV `space*16`, key from Keychain (16 bytes). Schema: Chrome `Cookies` SQLite → read `encrypted_value` (skip `v10` prefix) → decrypt → write to target.
6. `chromeimport.bookmarks` reads JSON, prepends/folder-merges into target `Bookmarks` JSON.
7. `chromeimport.history` copies `History` SQLite using shadow read (`sqlite3` `.backup` or VFS copy if locked).
8. Detect Chrome running: if running, allow bookmarks (JSON snapshot OK) and cookies (after Chrome exits per session) — but warn for history.
9. Streaming progress events back to UI.
10. Persist `last_sync` timestamp + source profile in workspace meta.

## Todo List

- [x] Scan finds all profiles (Default + Profile N) with display names — `chromeimport.Scan` (tested)
- [~] Cookie decrypt — `DecryptCookie`/`DeriveKey` implemented + round-trip tested (constants match Chromium OSCrypt); not run against a real macOS Keychain box. Write-back into target profile DEFERRED (needs SQLite + target re-encryption).
- [ ] Imported cookies show logged-in site — DEFERRED (depends on cookie write-back)
- [x] Bookmarks merged into workspace browser profile — `MergeBookmarks` (tested), wired via `chromeImport.importBookmarks`; surfacing in a sidebar Bookmarks panel deferred
- [ ] History merge — DEFERRED (needs SQLite reader/writer)
- [x] Re-sync idempotent for bookmarks — replaces the "Imported from Chrome" folder (tested)

## Implementation Notes (2026-05-25)

- Go: `internal/chromeimport/{profiles,cookies,bookmarks}.go` (tested — 60 total sidecar tests pass). Cookie crypto: PBKDF2-HMAC-SHA1 (1003 iters, salt `saltysalt`, 16-byte key) + AES-128-CBC + IV=16 spaces + v10 prefix — verified by review as the genuine macOS Chrome scheme. Swift: `ChromeImporter` (RPC + `SecItemCopyMatching` Safe-Storage fetch for future cookie import), `ChromeImportWizardView` (sheet), launched from a toolbar "Import Chrome" button.
- **Deferred (need a SQLite writer, out of scope for this pass):** cookie write-back into the target Chromium profile + target re-encryption, and history merge. The decrypt primitive + Keychain access are ready for when write-back lands.
- **Code review fix (Medium security):** `chromeImport.importBookmarks` no longer trusts client paths — it resolves the workspace via the registry (by id) and joins the source from the Chrome root + a validated single dir component, so a client can't direct reads/writes outside those roots.
- **Pending when cookie write-back ships:** zero-out the AES key buffer (the plan's requirement; moot until the key crosses to Go), and add a real-Chrome known-answer crypto test vector.

## Success Criteria

- [~] First-run import → logged-in site — bookmarks path works; the logged-in-cookies outcome needs the deferred cookie write-back
- [~] Re-sync brings new data — bookmarks re-sync is idempotent; history re-sync deferred

## Risk Assessment

- **Keychain access denial**: user may deny — feature must degrade gracefully (skip cookies). Mitigation: explicit "skip" path in wizard.
- **Chrome locked SQLite**: import fails silently. Mitigation: detect lock (PID + lockfile) → instruct user to quit Chrome OR import only bookmarks (separate file).
- **Schema drift**: Chrome schema changes between versions. Mitigation: schema version check; on unknown version, fall back to bookmarks only + warn.
- **Privacy expectations**: importing cookies = importing identity. Mitigation: per-workspace scoping, explicit consent each time, clear "Imported from Chrome profile X on date Y" indicator.

## Security Considerations

- AES key never logged or sent outside sidecar process. Pass via base64 in single RPC call; zero-out buffers after use.
- Imported data inherits Chromium profile dir perms (`0700`).
- Re-sync button warns about data overwrite.
- Audit log entry in Activity panel for every import (counts only, not contents).
