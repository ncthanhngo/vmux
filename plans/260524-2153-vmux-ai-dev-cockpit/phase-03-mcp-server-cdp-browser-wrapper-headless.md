---
phase: 3
title: "MCP proxy + auto-install external MCP servers (headless)"
status: pending
priority: P1
effort: "1w"
dependencies: [2]
---

# Phase 3: MCP proxy + auto-install external MCP servers (headless)

## Overview

**Reframed (validation session 2):** vmux does NOT build its own CDP wrapper. Instead, vmux is an **MCP proxy/middleware** between AI agents and external MCP servers (Chrome DevTools MCP, Playwright MCP, etc.). vmux's value: auto-install MCP servers, route tool calls, log everything into Activity stream, gate dangerous ops via Approval queue. Headless-testable end-to-end with Claude Code.

This phase de-risks: browser automation works for agents without vmux owning Chromium binary or CDP code.

## Requirements

- Functional:
  - **MCP proxy server** in Go sidecar: accepts MCP protocol from agents, forwards to backing servers, logs every call.
  - **Auto-install wizard** for recommended MCP servers:
    - **Chrome DevTools MCP** (Google official) — primary browser MCP for vmux
    - **Playwright MCP** (Microsoft) — alternative
    - Install via `npx`/`uvx` no global pollution; pin versions
  - **Built-in MCP tools** vmux owns: `workspace.*`, `file.read`, `file.write`, `command.run` (gated), `activity.annotate`
  - **Auto-config helper** (opt-in, with backup): write `~/.config/claude/mcp.json` to route Claude Code through vmux proxy
  - **Tool call logging** to activity stream with full request+response
- Non-functional: proxy latency overhead < 20ms per call. Auto-install one MCP < 30s on first launch.

## Architecture

```
sidecar/internal/
├── mcp/
│   ├── proxy.go               # MCP server (stdio + Unix socket) - exposed to agents
│   ├── upstream.go            # MCP client connections to backing servers
│   ├── installer.go           # npx/uvx orchestration, version pinning
│   ├── registry.go            # known MCP servers + install commands
│   └── tools/
│       ├── workspace.go       # vmux-native tools
│       ├── file.go
│       ├── command.go         # gated via approval (phase 6)
│       └── activity.go        # agents annotate activity stream
├── browser_session/
│   ├── manager.go             # spawn/track Chrome per workspace (via DevTools MCP profile)
│   └── profile.go             # per-workspace user-data-dir
└── bridge/
    └── cmd/vmux-mcp-bridge/main.go  # stdio bridge agents spawn
```

vmux installs MCP servers into `~/Library/Application Support/vmux/mcp-servers/` (isolated). Spawns via `npx --prefix <dir>` or `uvx` so they don't pollute user's global node_modules.

Tool call flow:
```
Claude Code → stdio MCP → vmux-mcp-bridge → sidecar MCP proxy
   → Activity log
   → Approval gate (phase 6)
   → route to backing server (Chrome DevTools MCP, etc.)
   → response → log → return to agent
```

## Related Code Files

- Create: `sidecar/internal/mcp/proxy.go`, `upstream.go`, `installer.go`, `registry.go`, `tools/*.go`
- Create: `sidecar/internal/browser_session/manager.go`, `profile.go`
- Create: `sidecar/cmd/vmux-mcp-bridge/main.go`
- Create: `app/Vmux/Onboarding/McpInstallWizardView.swift`
- Create: `tests/e2e/headless-agent-test.sh`
- Create: `shared/mcp-server-registry.json` (known servers + install commands)

## Implementation Steps

1. Define `shared/mcp-server-registry.json` — entries for Chrome DevTools MCP, Playwright MCP, browser-use, etc. with install command, transport, default args.
2. `mcp.installer`: detect node/npm/uvx → install registry entries into vmux-managed dir → verify launch.
3. `mcp.upstream`: client implementation of MCP stdio + Unix socket. Spawn upstream server, handle restarts.
4. `mcp.proxy`: server exposing MCP protocol to agents. Multiplexes calls to upstream servers based on tool namespace prefix.
5. Implement vmux-native tools (`workspace.*`, `file.*`, `command.run`, `activity.annotate`).
6. `browser_session.Manager`: when agent calls Chrome DevTools MCP, ensure Chrome is running for that workspace with correct profile (spawn Chrome with `--user-data-dir=<ws>/.vmux/chrome-profile --remote-debugging-port=0`). Detect Chrome.app or Chromium-based browser (Brave/Arc/Edge fallbacks).
7. Auto-config helper (**opt-in**): detect `~/.config/claude/mcp.json` → show diff preview → backup `mcp.json.vmux-backup-<ts>` → atomic write entry pointing to vmux-mcp-bridge.
8. Tool call logging: every proxy call writes activity record (ts, tool, args, result-summary, duration) to phase 6's activity store. Screenshot results auto-saved as thumbnails.
9. E2E test: install Chrome DevTools MCP via wizard → spawn Claude Code via bridge → ask "navigate to example.com and list all network requests" → assert response includes XHRs.
10. Onboarding wizard UI shell (full UI built phase 4): list recommended MCPs, install button per row, progress.

## Todo List

- [ ] Registry of recommended MCP servers
- [ ] Auto-install via npx/uvx works on clean machine
- [ ] Proxy routes calls correctly to upstream
- [ ] vmux-native tools functional (workspace/file/command/activity)
- [ ] Chrome auto-spawn with per-workspace profile via DevTools MCP
- [ ] Auto-config diff+backup flow works
- [ ] E2E with Chrome DevTools MCP: agent reads network panel data
- [ ] Test fallback to Brave/Arc/Edge when Chrome absent

## Success Criteria

- [ ] On a clean macOS machine: install vmux → run wizard → Claude Code can autonomously navigate, screenshot, read XHR responses with zero manual setup beyond approving 1 Keychain prompt for Chrome profile
- [ ] Proxy adds <20ms per call (benchmarked)
- [ ] Killing sidecar cleanly terminates upstream MCP processes (no orphans)

## Risk Assessment

- **External MCP server bugs/breakage**: out of our control. Mitigation: pin versions, registry has multiple alternatives per capability, allow user override.
- **MCP spec churn**: protocol still moving. Mitigation: target stable subset (tools+resources), update SDK quarterly.
- **Chrome version drift across users**: user A has Chrome 142, user B has Chrome 138 → behavior differs. Mitigation: log Chrome version in activity; surface in bug reports.
- **No Chrome installed**: rare on dev machine but possible. Mitigation: wizard offers (a) "install Chrome" link, (b) "use Brave/Arc/Edge" auto-detect, (c) "download Chromium for Testing (~150MB)" as last resort.

## Security Considerations

- Approval gate (phase 6) wraps every proxy call; default-deny for `command.run` outside whitelist.
- MCP servers installed under vmux dir, not global; signature/checksum verification on install.
- `file.write` path-traversal guard (workspace-rooted).
- Bridge validates workspace id env var against sidecar before forwarding.
- Auto-config: never write to `~/.config/claude/mcp.json` without explicit user OK + backup.
