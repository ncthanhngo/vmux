// Package mcp implements the Model Context Protocol pieces vmux needs: a
// bidirectional JSON-RPC peer (newline-delimited, per the MCP stdio transport),
// an upstream client that drives external MCP servers, and a proxy server that
// aggregates upstream + vmux-native tools for AI agents.
package mcp

import "encoding/json"

// ProtocolVersion is the MCP revision vmux advertises. 2024-11-05 is broadly
// supported by current servers (Chrome DevTools MCP, Playwright MCP).
const ProtocolVersion = "2024-11-05"

// Message is a JSON-RPC 2.0 envelope covering requests, responses, and
// notifications in either direction.
type Message struct {
	JSONRPC string           `json:"jsonrpc"`
	ID      *json.RawMessage `json:"id,omitempty"`
	Method  string           `json:"method,omitempty"`
	Params  json.RawMessage  `json:"params,omitempty"`
	Result  json.RawMessage  `json:"result,omitempty"`
	Error   *RPCError        `json:"error,omitempty"`
}

// RPCError is a JSON-RPC error object.
type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

func (e *RPCError) Error() string { return e.Message }

// --- Lifecycle ---

type Implementation struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// Capabilities is a permissive view: presence of a key signals support.
type Capabilities struct {
	Tools     *struct{} `json:"tools,omitempty"`
	Resources *struct{} `json:"resources,omitempty"`
	Prompts   *struct{} `json:"prompts,omitempty"`
}

type InitializeParams struct {
	ProtocolVersion string         `json:"protocolVersion"`
	Capabilities    Capabilities   `json:"capabilities"`
	ClientInfo      Implementation `json:"clientInfo"`
}

type InitializeResult struct {
	ProtocolVersion string         `json:"protocolVersion"`
	Capabilities    Capabilities   `json:"capabilities"`
	ServerInfo      Implementation `json:"serverInfo"`
}

// --- Tools ---

// Tool describes a callable tool. InputSchema is a raw JSON Schema object.
type Tool struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	InputSchema json.RawMessage `json:"inputSchema,omitempty"`
}

type ListToolsResult struct {
	Tools []Tool `json:"tools"`
}

type CallToolParams struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments,omitempty"`
}

// Content is one item of a tool result. vmux primarily emits text; image and
// other types pass through untouched from upstreams.
type Content struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
	Data string `json:"data,omitempty"`     // base64 for image/audio
	MIME string `json:"mimeType,omitempty"`
}

type CallToolResult struct {
	Content []Content `json:"content"`
	IsError bool      `json:"isError,omitempty"`
}

// TextResult builds a single-text-content tool result.
func TextResult(text string) CallToolResult {
	return CallToolResult{Content: []Content{{Type: "text", Text: text}}}
}

// ErrorResult builds a tool result flagged as an error.
func ErrorResult(text string) CallToolResult {
	return CallToolResult{Content: []Content{{Type: "text", Text: text}}, IsError: true}
}
