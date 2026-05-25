package mcp

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"time"
)

// callTool routes a tools/call to a native handler or an owning upstream,
// logging the call (tool, args, summary, duration) to the Activity stream.
func (p *Proxy) callTool(ctx context.Context, params json.RawMessage) (any, error) {
	var call CallToolParams
	if err := json.Unmarshal(params, &call); err != nil {
		return nil, &RPCError{Code: -32602, Message: "invalid tools/call params"}
	}

	start := time.Now()
	rec := ToolCallRecord{Time: start.UTC(), Tool: call.Name, Args: string(call.Arguments)}

	if handler, ok := p.nativeByName[call.Name]; ok {
		res, err := handler(ctx, call.Arguments)
		p.finishLog(rec, res, err, start)
		if err != nil {
			return nil, err
		}
		return res, nil
	}

	if u := p.ownerOf(call.Name); u != nil {
		rec.Upstream = u.Name
		res, err := u.CallTool(ctx, call.Name, call.Arguments)
		p.finishLog(rec, res, err, start)
		if err != nil {
			return nil, err
		}
		p.captureScreenshots(u.Name, res)
		return res, nil
	}

	return nil, &RPCError{Code: -32602, Message: "unknown tool: " + call.Name}
}

// ownerOf finds the upstream advertising the named tool.
func (p *Proxy) ownerOf(toolName string) *Upstream {
	p.mu.RLock()
	defer p.mu.RUnlock()
	for _, u := range p.upstreams {
		for _, t := range u.Tools() {
			if t.Name == toolName {
				return u
			}
		}
	}
	return nil
}

// captureScreenshots forwards any image content in a tool result to the
// OnScreenshot hook (so the screenshot store can persist browser captures).
func (p *Proxy) captureScreenshots(upstream string, res CallToolResult) {
	if p.OnScreenshot == nil {
		return
	}
	for _, c := range res.Content {
		if c.Type == "image" && c.Data != "" {
			if raw, err := base64.StdEncoding.DecodeString(c.Data); err == nil {
				p.OnScreenshot(upstream, raw)
			}
		}
	}
}

func (p *Proxy) finishLog(rec ToolCallRecord, res CallToolResult, err error, start time.Time) {
	rec.DurationM = time.Since(start).Milliseconds()
	if err != nil {
		rec.IsError = true
		rec.Summary = err.Error()
	} else {
		rec.IsError = res.IsError
		rec.Summary = summarize(res)
	}
	p.activity.LogToolCall(rec)
}
