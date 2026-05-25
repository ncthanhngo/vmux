package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"time"
)

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
	// Deny-by-default: only allowlisted commands run until phase-7 approval
	// can prompt the user for arbitrary commands.
	if !n.cmdAllowlist[p.Command] {
		return ErrorResult(fmt.Sprintf("command %q not allowed (awaiting approval gate)", p.Command)), nil
	}
	ws, ok := n.ws.Get(p.WorkspaceID)
	if !ok {
		return ErrorResult(fmt.Sprintf("unknown workspace %q", p.WorkspaceID)), nil
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
