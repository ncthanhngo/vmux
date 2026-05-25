// Package activity records what an agent does — MCP tool calls, file edits,
// commands, browser actions, screenshots — into an append-only per-session log
// that powers the Activity panel and Session Replay.
package activity

import "time"

// Kind classifies an activity event.
type Kind string

const (
	KindToolCall   Kind = "tool_call"
	KindFileEdit   Kind = "file_edit"
	KindCommand    Kind = "command_run"
	KindBrowserNav Kind = "browser_nav"
	KindScreenshot Kind = "screenshot"
	KindConsole    Kind = "console"
	KindAttention  Kind = "attention"
	KindCost       Kind = "cost"
)

// Risk is the assessed danger of an event (drives approval + UI emphasis).
type Risk string

const (
	RiskLow    Risk = "low"
	RiskMedium Risk = "medium"
	RiskHigh   Risk = "high"
)

// Event is one recorded agent action. Refs hold opaque pointers (shotId,
// workspaceId, file path, …) the UI can resolve.
type Event struct {
	Seq       int64             `json:"seq"`
	Ts        time.Time         `json:"ts"`
	SessionID string            `json:"sessionId"`
	Kind      Kind              `json:"kind"`
	Actor     string            `json:"actor"`
	Summary   string            `json:"summary"`
	Detail    string            `json:"detail,omitempty"`
	Refs      map[string]string `json:"refs,omitempty"`
	Risk      Risk              `json:"risk,omitempty"`
}
