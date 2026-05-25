// Package rpc implements a minimal, line-delimited JSON-RPC 2.0 server over a
// stream connection (the vmux Unix socket). Each message is a single JSON
// object terminated by '\n'. It supports client→server requests/responses and
// server→client notifications (used for streaming PTY data and git events).
package rpc

import "encoding/json"

const version = "2.0"

// Request is an incoming JSON-RPC call. A nil ID marks a notification (no reply).
type Request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      *json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// Response is a reply to a Request. Exactly one of Result/Error is set.
type Response struct {
	JSONRPC string           `json:"jsonrpc"`
	ID      *json.RawMessage `json:"id,omitempty"`
	Result  json.RawMessage  `json:"result,omitempty"`
	Error   *Error           `json:"error,omitempty"`
}

// Notification is a server-initiated message with no reply expected.
type Notification struct {
	JSONRPC string `json:"jsonrpc"`
	Method  string `json:"method"`
	Params  any    `json:"params,omitempty"`
}

// Error is a JSON-RPC 2.0 error object.
type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

func (e *Error) Error() string { return e.Message }

// Standard JSON-RPC 2.0 error codes.
const (
	CodeParse          = -32700
	CodeInvalidRequest = -32600
	CodeMethodNotFound = -32601
	CodeInvalidParams  = -32602
	CodeInternal       = -32603
)

func newNotification(method string, params any) Notification {
	return Notification{JSONRPC: version, Method: method, Params: params}
}

func resultResponse(id *json.RawMessage, result json.RawMessage) Response {
	return Response{JSONRPC: version, ID: id, Result: result}
}

func errorResponse(id *json.RawMessage, code int, msg string) Response {
	return Response{JSONRPC: version, ID: id, Error: &Error{Code: code, Message: msg}}
}
