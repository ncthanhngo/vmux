package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/vmux/sidecar/internal/workspace"
)

// NativeTools are the tools vmux implements itself (vs. proxied upstreams).
type NativeTools struct {
	ws       *workspace.Registry
	activity ActivityLogger
	// cmdAllowlist gates command_run until the phase-7 approval queue exists.
	cmdAllowlist map[string]bool
}

// NewNativeTools builds the native tool provider.
//
// command_run is deny-by-default with a deliberately narrow allowlist of
// low-risk, read-mostly commands. NOTE: the allowlist only matches the command
// NAME, not its arguments — interpreters/build tools that can execute arbitrary
// code (go, node, npm, and `git -c …`) are intentionally excluded until the
// phase-7 approval queue can prompt the user per-invocation. Until then, treat
// command_run as a convenience for inspection, not a sandbox.
func NewNativeTools(ws *workspace.Registry, activity ActivityLogger) *NativeTools {
	return &NativeTools{
		ws:       ws,
		activity: activity,
		cmdAllowlist: map[string]bool{
			"ls": true, "cat": true, "pwd": true, "echo": true,
			"head": true, "tail": true, "wc": true, "grep": true,
		},
	}
}

func (n *NativeTools) registered() []registeredTool {
	return []registeredTool{
		n.workspaceListTool(),
		n.workspaceOpenTool(),
		n.fileReadTool(),
		n.fileWriteTool(),
		n.commandRunTool(),
		n.activityAnnotateTool(),
	}
}

// resolveWorkspacePath joins a workspace-relative path against its root and
// rejects any path that escapes the root — including via symlinks. It resolves
// symlinks on the deepest existing ancestor of the target and re-checks
// containment, so a symlink inside the workspace cannot point out of it.
func (n *NativeTools) resolveWorkspacePath(workspaceID, rel string) (string, error) {
	ws, ok := n.ws.Get(workspaceID)
	if !ok {
		return "", fmt.Errorf("unknown workspace %q", workspaceID)
	}
	realRoot, err := filepath.EvalSymlinks(ws.Path)
	if err != nil {
		return "", err
	}
	joined := filepath.Join(realRoot, rel)
	real := resolveExisting(joined)
	if !withinRoot(realRoot, real) {
		return "", fmt.Errorf("path %q escapes workspace", rel)
	}
	return joined, nil
}

// resolveExisting resolves symlinks on the longest existing prefix of p, then
// re-appends the non-existent tail (so it works for files about to be created).
func resolveExisting(p string) string {
	suffix := ""
	cur := p
	for {
		if resolved, err := filepath.EvalSymlinks(cur); err == nil {
			return filepath.Join(resolved, suffix)
		}
		parent := filepath.Dir(cur)
		if parent == cur {
			return p // nothing along the path exists
		}
		suffix = filepath.Join(filepath.Base(cur), suffix)
		cur = parent
	}
}

func withinRoot(root, p string) bool {
	rel, err := filepath.Rel(root, p)
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)))
}

func (n *NativeTools) workspaceListTool() registeredTool {
	return registeredTool{
		tool: Tool{
			Name:        "workspace_list",
			Description: "List the workspaces vmux currently tracks.",
			InputSchema: json.RawMessage(`{"type":"object","properties":{}}`),
		},
		handler: func(_ context.Context, _ json.RawMessage) (CallToolResult, error) {
			return TextResult(jsonString(n.ws.List())), nil
		},
	}
}

func (n *NativeTools) workspaceOpenTool() registeredTool {
	return registeredTool{
		tool: Tool{
			Name:        "workspace_open",
			Description: "Open (register) a directory as a vmux workspace.",
			InputSchema: json.RawMessage(`{"type":"object","properties":{"path":{"type":"string"}},"required":["path"]}`),
		},
		handler: func(_ context.Context, args json.RawMessage) (CallToolResult, error) {
			var p struct {
				Path string `json:"path"`
			}
			if err := json.Unmarshal(args, &p); err != nil {
				return ErrorResult("invalid arguments: " + err.Error()), nil
			}
			ws, err := n.ws.Open(p.Path)
			if err != nil {
				return ErrorResult(err.Error()), nil
			}
			return TextResult(jsonString(ws)), nil
		},
	}
}
