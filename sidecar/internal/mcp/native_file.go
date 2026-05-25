package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// maxFileRead caps file_read output to avoid flooding an agent's context.
const maxFileRead = 1 << 20 // 1 MiB

func (n *NativeTools) fileReadTool() registeredTool {
	return registeredTool{
		tool: Tool{
			Name:        "file_read",
			Description: "Read a UTF-8 text file within a workspace.",
			InputSchema: json.RawMessage(`{"type":"object","properties":{"workspaceId":{"type":"string"},"path":{"type":"string"}},"required":["workspaceId","path"]}`),
		},
		handler: func(_ context.Context, args json.RawMessage) (CallToolResult, error) {
			var p struct {
				WorkspaceID string `json:"workspaceId"`
				Path        string `json:"path"`
			}
			if err := json.Unmarshal(args, &p); err != nil {
				return ErrorResult("invalid arguments: " + err.Error()), nil
			}
			abs, err := n.resolveWorkspacePath(p.WorkspaceID, p.Path)
			if err != nil {
				return ErrorResult(err.Error()), nil
			}
			data, err := os.ReadFile(abs)
			if err != nil {
				return ErrorResult(err.Error()), nil
			}
			if len(data) > maxFileRead {
				return ErrorResult(fmt.Sprintf("file too large (%d bytes > %d)", len(data), maxFileRead)), nil
			}
			return TextResult(string(data)), nil
		},
	}
}

func (n *NativeTools) fileWriteTool() registeredTool {
	return registeredTool{
		tool: Tool{
			Name:        "file_write",
			Description: "Write a UTF-8 text file within a workspace (creates parent dirs).",
			InputSchema: json.RawMessage(`{"type":"object","properties":{"workspaceId":{"type":"string"},"path":{"type":"string"},"content":{"type":"string"}},"required":["workspaceId","path","content"]}`),
		},
		handler: func(_ context.Context, args json.RawMessage) (CallToolResult, error) {
			var p struct {
				WorkspaceID string `json:"workspaceId"`
				Path        string `json:"path"`
				Content     string `json:"content"`
			}
			if err := json.Unmarshal(args, &p); err != nil {
				return ErrorResult("invalid arguments: " + err.Error()), nil
			}
			abs, err := n.resolveWorkspacePath(p.WorkspaceID, p.Path)
			if err != nil {
				return ErrorResult(err.Error()), nil
			}
			if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
				return ErrorResult(err.Error()), nil
			}
			if err := os.WriteFile(abs, []byte(p.Content), 0o644); err != nil {
				return ErrorResult(err.Error()), nil
			}
			return TextResult(fmt.Sprintf("wrote %d bytes to %s", len(p.Content), p.Path)), nil
		},
	}
}
