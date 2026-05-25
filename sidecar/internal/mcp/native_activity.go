package mcp

import (
	"context"
	"encoding/json"
	"time"
)

func (n *NativeTools) activityAnnotateTool() registeredTool {
	return registeredTool{
		tool: Tool{
			Name:        "activity_annotate",
			Description: "Add a note to the vmux Activity stream (e.g. what the agent is about to do).",
			InputSchema: json.RawMessage(`{"type":"object","properties":{"note":{"type":"string"}},"required":["note"]}`),
		},
		handler: func(_ context.Context, args json.RawMessage) (CallToolResult, error) {
			var p struct {
				Note string `json:"note"`
			}
			if err := json.Unmarshal(args, &p); err != nil {
				return ErrorResult("invalid arguments: " + err.Error()), nil
			}
			if n.activity != nil {
				n.activity.LogToolCall(ToolCallRecord{
					Time:    time.Now().UTC(),
					Tool:    "activity_annotate",
					Args:    jsonString(p),
					Summary: p.Note,
				})
			}
			return TextResult("annotated"), nil
		},
	}
}
