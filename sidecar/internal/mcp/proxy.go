package mcp

import (
	"context"
	"encoding/json"
	"log/slog"
	"net"
	"sync"

	"github.com/vmux/sidecar/internal/peercred"
)

// Proxy is an MCP server exposed to agents. It aggregates vmux-native tools and
// the tools of every backing upstream, routing tools/call to the right owner
// and logging each call to the Activity stream.
type Proxy struct {
	log      *slog.Logger
	activity ActivityLogger
	native   *NativeTools

	nativeByName map[string]ToolHandler
	nativeTools  []Tool

	// OnScreenshot, if set, is invoked with decoded PNG bytes whenever an
	// upstream tool result carries an image (browser screenshot capture).
	OnScreenshot func(upstream string, png []byte)

	mu        sync.RWMutex
	upstreams map[string]*Upstream
}

// NewProxy builds a proxy over the given native tools. If activity is nil, a
// slog-backed logger is used.
func NewProxy(log *slog.Logger, activity ActivityLogger, native *NativeTools) *Proxy {
	if log == nil {
		log = slog.Default()
	}
	if activity == nil {
		activity = slogActivity{log: log}
	}
	p := &Proxy{
		log:          log,
		activity:     activity,
		native:       native,
		nativeByName: make(map[string]ToolHandler),
		upstreams:    make(map[string]*Upstream),
	}
	for _, rt := range native.registered() {
		p.nativeByName[rt.tool.Name] = rt.handler
		p.nativeTools = append(p.nativeTools, rt.tool)
	}
	return p
}

// AddUpstream registers a backing server's tools for proxying.
func (p *Proxy) AddUpstream(u *Upstream) {
	p.mu.Lock()
	p.upstreams[u.Name] = u
	p.mu.Unlock()
}

// CloseUpstreams terminates all backing servers (called on shutdown).
func (p *Proxy) CloseUpstreams() {
	p.mu.Lock()
	ups := make([]*Upstream, 0, len(p.upstreams))
	for _, u := range p.upstreams {
		ups = append(ups, u)
	}
	p.upstreams = make(map[string]*Upstream)
	p.mu.Unlock()
	for _, u := range ups {
		u.Close()
	}
}

// Serve accepts agent connections (one MCP session each) until ctx is cancelled
// or the listener closes.
func (p *Proxy) Serve(ctx context.Context, ln net.Listener) error {
	go func() {
		<-ctx.Done()
		ln.Close()
	}()
	for {
		conn, err := ln.Accept()
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return err
		}
		// The MCP proxy exposes file + command tools, so it gets the same
		// same-UID gate as the control socket (defense in depth over 0600).
		if err := peercred.Check(conn); err != nil {
			p.log.Warn("rejecting MCP connection", "err", err)
			conn.Close()
			continue
		}
		peer := NewPeer(conn, p.handler(ctx))
		go peer.Run(ctx)
	}
}

// handler returns the per-connection MCP request handler.
func (p *Proxy) handler(serveCtx context.Context) RequestHandler {
	return func(ctx context.Context, method string, params json.RawMessage) (any, error) {
		switch method {
		case "initialize":
			return InitializeResult{
				ProtocolVersion: ProtocolVersion,
				Capabilities:    Capabilities{Tools: &struct{}{}},
				ServerInfo:      Implementation{Name: "vmux", Version: "0.0.0"},
			}, nil
		case "notifications/initialized":
			return nil, nil
		case "ping":
			return struct{}{}, nil
		case "tools/list":
			return ListToolsResult{Tools: p.allTools()}, nil
		case "tools/call":
			return p.callTool(ctx, params)
		default:
			return nil, &RPCError{Code: -32601, Message: "method not found: " + method}
		}
	}
}

// allTools is the union of native + upstream tools.
func (p *Proxy) allTools() []Tool {
	out := make([]Tool, 0, len(p.nativeTools))
	out = append(out, p.nativeTools...)
	p.mu.RLock()
	for _, u := range p.upstreams {
		out = append(out, u.Tools()...)
	}
	p.mu.RUnlock()
	if out == nil {
		out = []Tool{}
	}
	return out
}
