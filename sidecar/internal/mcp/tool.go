package mcp

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"
)

// ToolHandler executes a tool call with raw JSON arguments.
type ToolHandler func(ctx context.Context, args json.RawMessage) (CallToolResult, error)

// registeredTool pairs an advertised Tool with its handler.
type registeredTool struct {
	tool    Tool
	handler ToolHandler
}

// ToolCallRecord is one logged proxy tool call (consumed by the Activity store).
type ToolCallRecord struct {
	Time      time.Time `json:"time"`
	Tool      string    `json:"tool"`
	Upstream  string    `json:"upstream,omitempty"` // empty for vmux-native tools
	Args      string    `json:"args"`
	Summary   string    `json:"summary"`
	DurationM int64     `json:"durationMs"`
	IsError   bool      `json:"isError"`
}

// ActivityLogger records tool calls. Phase 7 swaps in the persistent store.
type ActivityLogger interface {
	LogToolCall(rec ToolCallRecord)
}

// NewSlogActivityLogger returns an ActivityLogger that writes tool calls to the
// structured log. Phase 7 replaces this with the persistent Activity store.
func NewSlogActivityLogger(log *slog.Logger) ActivityLogger { return slogActivity{log: log} }

// slogActivity is the default logger used until the Activity store exists.
type slogActivity struct{ log *slog.Logger }

func (s slogActivity) LogToolCall(rec ToolCallRecord) {
	s.log.Info("tool call",
		"tool", rec.Tool, "upstream", rec.Upstream,
		"durationMs", rec.DurationM, "isError", rec.IsError, "summary", rec.Summary)
}

func jsonString(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	return string(b)
}

// summarize trims a result to a short single-line activity summary.
func summarize(res CallToolResult) string {
	for _, c := range res.Content {
		if c.Type == "text" && c.Text != "" {
			s := c.Text
			if len(s) > 200 {
				s = s[:200] + "…"
			}
			return s
		}
	}
	if len(res.Content) > 0 {
		return res.Content[0].Type
	}
	return ""
}
