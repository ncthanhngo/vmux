---
phase: 2
title: "Go sidecar foundation (IPC + PTY + workspace)"
status: pending
priority: P1
effort: "1w"
dependencies: [1]
---

# Phase 2: Go sidecar foundation (IPC + PTY + workspace)

## Overview

Build the Go sidecar core: JSON-RPC server over Unix socket, PTY manager that can spawn arbitrary CLIs (shell, claude, codex) with TTY semantics, and workspace manager that tracks per-project state. Headless-testable via Go tests + a small CLI client. No Swift UI yet.

## Requirements

- Functional:
  - JSON-RPC 2.0 bidirectional (request/response + server-initiated notifications) over Unix socket.
  - Spawn a PTY-backed process with given cwd, env, command. Stream stdout/stderr/exit. Accept stdin + window resize.
  - Workspace CRUD: open(path), list, close. Persist registry to `~/Library/Application Support/vmux/workspaces.json`.
- Non-functional: PTY throughput ≥ 1MB/s, latency keystroke→echo < 10ms. Clean shutdown propagates SIGTERM to children.

## Architecture

```
sidecar/internal/
├── rpc/            # JSON-RPC 2.0 framing (line-delimited), router, notification channel
├── pty/            # creack/pty wrapper, session lifecycle, resize, write/read pumps
├── workspace/      # registry, .vmux/workspace.json read/write, git status reader
├── log/            # structured logging (slog) → ~/Library/Logs/vmux/sidecar.log
└── server.go       # wires everything
```

RPC method namespace:
- `pty.spawn(cwd, cmd, args, env)` → `{sessionId}`
- `pty.write(sessionId, data)` / `pty.resize(sessionId, cols, rows)` / `pty.kill(sessionId)`
- `workspace.open(path)` → `{workspaceId, meta}` / `workspace.list` / `workspace.close(id)`

Server→client notifications:
- `pty.data(sessionId, chunk)` (base64) / `pty.exit(sessionId, code)`
- `workspace.gitChanged(workspaceId, status)`

## Related Code Files

- Create: `sidecar/internal/rpc/server.go`, `router.go`, `codec.go`
- Create: `sidecar/internal/pty/session.go`, `manager.go`
- Create: `sidecar/internal/workspace/registry.go`, `meta.go`, `git.go`
- Create: `sidecar/cmd/vmux-cli/main.go` (test client)
- Modify: `sidecar/cmd/vmux-sidecar/main.go` (wire up server)

## Implementation Steps

1. Dependencies: `github.com/creack/pty`, `github.com/sourcegraph/jsonrpc2` (or hand-rolled framing).
2. Implement `rpc.Server` with line-delimited JSON-RPC over `net.Listen("unix", ...)`. Socket perm `0600`.
3. `pty.Session`: wraps `pty.StartWithSize`, goroutines for stdin/stdout pumps, chan-based event delivery.
4. `pty.Manager`: map[sessionId]*Session, mutex, lifecycle (spawn/kill/list). Generate UUIDs.
5. Register PTY RPC methods + emit `pty.data` notifications with backpressure (drop+coalesce if client slow).
6. `workspace.Registry`: open(path) reads/creates `.vmux/workspace.json`, scans git via `os/exec git status --porcelain`.
7. File watcher with `fsnotify` on workspace root → emit `workspace.gitChanged` (debounced 500ms).
8. `vmux-cli`: dev tool. Subcommands `spawn`, `attach`, `ls`. Used in tests + dev.
9. Tests: PTY echo test, RPC roundtrip test, workspace lifecycle test. Race detector on.
10. Graceful shutdown: SIGTERM → close listener → kill all PTYs → flush logs → exit.

## Todo List

- [ ] JSON-RPC server + dispatch
- [ ] PTY session lifecycle works (spawn `bash`, echo, exit)
- [ ] Resize propagates to TTY (`stty size` reflects)
- [ ] Workspace registry persists across restart
- [ ] Git status streamed on file change
- [ ] `vmux-cli spawn bash` and interactive use works
- [ ] Go test coverage ≥ 70% for these packages

## Success Criteria

- [ ] `vmux-cli spawn -- claude` opens a working Claude Code session over the socket
- [ ] Resize/keystrokes feel instant in raw-mode terminal client
- [ ] Kill -9 sidecar → no orphan PTY processes (verified via `ps`)
- [ ] Concurrent 8 sessions, throughput test passes

## Risk Assessment

- **Backpressure**: slow consumer can OOM if buffered without limit. Mitigation: bounded ring buffer per session + drop policy with marker.
- **PTY ioctl portability**: macOS PTY quirks (window size, controlling tty). Mitigation: `creack/pty` handles, but add integration tests on both arm64 and amd64.
- **JSON-RPC framing**: large PTY chunks → base64 inflation. Mitigation: cap chunk size 32KB; if usage shows overhead, switch to length-prefixed binary frames later.

## Security Considerations

- Socket `0600`. Reject connections from other UIDs.
- PTY env: scrub PATH to known-safe + user PATH; do not leak parent env wholesale.
- No method exposes arbitrary `exec` outside `pty.spawn` (which is intentional).
