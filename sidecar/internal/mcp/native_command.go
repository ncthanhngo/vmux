package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"time"

	"github.com/vmux/sidecar/internal/approval"
)

// approvalTimeout bounds how long a gated command waits for a human decision
// before it is denied — prevents an unattended agent from hanging forever on a
// pending approval (the connection/tool call would otherwise block until the
// agent disconnects).
const approvalTimeout = 10 * time.Minute

// commandRunTimeout bounds a single command_run invocation.
const commandRunTimeout = 60 * time.Second

func (n *NativeTools) commandRunTool() registeredTool {
	return registeredTool{
		tool: Tool{
			Name:        "command_run",
			Description: "Run a non-interactive command in a workspace. Restricted to an allowlist until approval gating lands.",
			InputSchema: json.RawMessage(`{"type":"object","properties":{"workspaceId":{"type":"string"},"command":{"type":"string"},"args":{"type":"array","items":{"type":"string"}}},"required":["workspaceId","command"]}`),
		},
		handler: n.runCommand,
	}
}

func (n *NativeTools) runCommand(ctx context.Context, args json.RawMessage) (CallToolResult, error) {
	var p struct {
		WorkspaceID string   `json:"workspaceId"`
		Command     string   `json:"command"`
		Args        []string `json:"args"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return ErrorResult("invalid arguments: " + err.Error()), nil
	}
	ws, ok := n.ws.Get(p.WorkspaceID)
	if !ok {
		return ErrorResult(fmt.Sprintf("unknown workspace %q", p.WorkspaceID)), nil
	}

	// Approval gate: in gate mode a dangerous command blocks for user approval;
	// in sandbox mode commands are denied. In watch mode the gate allows and we
	// fall back to the coarse allowlist as a safety net.
	if n.gate != nil {
		gateCtx, cancelGate := context.WithTimeout(ctx, approvalTimeout)
		allowed, reason := n.gate.Check(gateCtx, approval.Action{
			Tool: "command_run", Command: p.Command, Args: p.Args, Workspace: p.WorkspaceID,
		})
		cancelGate()
		if !allowed {
			return ErrorResult("blocked: " + reason), nil
		}
		if n.gate.Mode(p.WorkspaceID) == approval.ModeWatch && !n.cmdAllowlist[p.Command] {
			return ErrorResult(fmt.Sprintf("command %q not allowed in watch mode (switch to gate mode to approve arbitrary commands)", p.Command)), nil
		}
	} else if !n.cmdAllowlist[p.Command] {
		return ErrorResult(fmt.Sprintf("command %q not allowed", p.Command)), nil
	}

	runCtx, cancel := context.WithTimeout(ctx, commandRunTimeout)
	defer cancel()

	cmd := exec.CommandContext(runCtx, p.Command, p.Args...)
	cmd.Dir = ws.Path
	out, err := cmd.CombinedOutput()
	result := string(out)
	if err != nil {
		return CallToolResult{
			Content: []Content{{Type: "text", Text: fmt.Sprintf("%s\n[exit: %v]", result, err)}},
			IsError: true,
		}, nil
	}
	return TextResult(result), nil
}
