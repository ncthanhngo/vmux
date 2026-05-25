package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"sync"
	"sync/atomic"
)

// maxMessage caps a single MCP frame (defensive; the transport is local/trusted).
const maxMessage = 32 << 20 // 32 MiB

// RequestHandler handles an inbound request or notification. For notifications
// (no id) the return values are ignored.
type RequestHandler func(ctx context.Context, method string, params json.RawMessage) (any, error)

// Peer is a bidirectional JSON-RPC peer over a newline-delimited stream. It can
// issue Calls (client role) and serve inbound requests via a handler (server
// role) on the same connection — MCP uses both directions.
type Peer struct {
	rwc     io.ReadWriteCloser
	handler RequestHandler

	writeMu sync.Mutex
	nextID  int64

	mu      sync.Mutex
	pending map[int64]chan *Message
	closed  bool
}

// NewPeer wraps rwc. handler may be nil for a pure client.
func NewPeer(rwc io.ReadWriteCloser, handler RequestHandler) *Peer {
	return &Peer{rwc: rwc, handler: handler, pending: make(map[int64]chan *Message)}
}

// Run reads and routes messages until EOF or ctx cancellation. Returns the read
// error (nil on clean EOF).
func (p *Peer) Run(ctx context.Context) error {
	go func() {
		<-ctx.Done()
		p.Close()
	}()
	reader := bufio.NewReaderSize(p.rwc, 64<<10)
	for {
		line, err := reader.ReadBytes('\n')
		if len(line) > maxMessage {
			return fmt.Errorf("mcp: message exceeds size limit")
		}
		if len(trim(line)) > 0 {
			var msg Message
			if json.Unmarshal(line, &msg) == nil {
				p.route(ctx, &msg)
			}
		}
		if err != nil {
			p.failPending()
			if ctx.Err() != nil || err == io.EOF {
				return nil
			}
			return err
		}
	}
}

func (p *Peer) route(ctx context.Context, msg *Message) {
	switch {
	case msg.Method != "" && msg.ID != nil: // inbound request
		go p.serveRequest(ctx, msg)
	case msg.Method != "": // inbound notification
		if p.handler != nil {
			method, params := msg.Method, msg.Params
			go func() {
				defer recoverHandler(method)
				p.handler(ctx, method, params)
			}()
		}
	default: // response to one of our Calls
		p.deliver(msg)
	}
}

// recoverHandler stops a panicking tool/notification handler from crashing the
// whole sidecar; one bad request must not take down every session.
func recoverHandler(method string) {
	if r := recover(); r != nil {
		slog.Error("mcp handler panic", "method", method, "panic", r)
	}
}

func (p *Peer) serveRequest(ctx context.Context, req *Message) {
	defer func() {
		if r := recover(); r != nil {
			slog.Error("mcp handler panic", "method", req.Method, "panic", r)
			p.writeMessage(&Message{JSONRPC: "2.0", ID: req.ID, Error: &RPCError{Code: -32603, Message: "internal handler error"}})
		}
	}()
	if p.handler == nil {
		p.writeMessage(&Message{JSONRPC: "2.0", ID: req.ID, Error: &RPCError{Code: -32601, Message: "no handler"}})
		return
	}
	result, err := p.handler(ctx, req.Method, req.Params)
	resp := &Message{JSONRPC: "2.0", ID: req.ID}
	if err != nil {
		resp.Error = toRPCError(err)
	} else {
		raw, mErr := json.Marshal(result)
		if mErr != nil {
			resp.Error = &RPCError{Code: -32603, Message: mErr.Error()}
		} else {
			resp.Result = raw
		}
	}
	p.writeMessage(resp)
}

// Call issues a request and waits for the response.
func (p *Peer) Call(ctx context.Context, method string, params any) (json.RawMessage, error) {
	raw, err := marshalParams(params)
	if err != nil {
		return nil, err
	}
	id := atomic.AddInt64(&p.nextID, 1)
	ch := make(chan *Message, 1)

	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return nil, fmt.Errorf("mcp: peer closed")
	}
	p.pending[id] = ch
	p.mu.Unlock()

	idRaw := json.RawMessage(fmt.Sprintf("%d", id))
	if err := p.writeMessage(&Message{JSONRPC: "2.0", ID: &idRaw, Method: method, Params: raw}); err != nil {
		p.dropPending(id)
		return nil, err
	}

	select {
	case <-ctx.Done():
		p.dropPending(id)
		return nil, ctx.Err()
	case resp, ok := <-ch:
		if !ok {
			return nil, fmt.Errorf("mcp: connection closed awaiting %s", method)
		}
		if resp.Error != nil {
			return nil, resp.Error
		}
		return resp.Result, nil
	}
}

// Notify sends a notification (no response expected).
func (p *Peer) Notify(method string, params any) error {
	raw, err := marshalParams(params)
	if err != nil {
		return err
	}
	return p.writeMessage(&Message{JSONRPC: "2.0", Method: method, Params: raw})
}

func (p *Peer) Close() error {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return nil
	}
	p.closed = true
	p.mu.Unlock()
	return p.rwc.Close()
}

func (p *Peer) writeMessage(msg *Message) error {
	b, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	b = append(b, '\n')
	p.writeMu.Lock()
	defer p.writeMu.Unlock()
	_, err = p.rwc.Write(b)
	return err
}

func (p *Peer) deliver(msg *Message) {
	var idNum int64
	if msg.ID == nil || json.Unmarshal(*msg.ID, &idNum) != nil {
		return
	}
	p.mu.Lock()
	ch := p.pending[idNum]
	delete(p.pending, idNum)
	p.mu.Unlock()
	if ch != nil {
		ch <- msg
	}
}

func (p *Peer) dropPending(id int64) {
	p.mu.Lock()
	delete(p.pending, id)
	p.mu.Unlock()
}

func (p *Peer) failPending() {
	p.mu.Lock()
	defer p.mu.Unlock()
	for id, ch := range p.pending {
		close(ch)
		delete(p.pending, id)
	}
}

func marshalParams(params any) (json.RawMessage, error) {
	if params == nil {
		return nil, nil
	}
	if raw, ok := params.(json.RawMessage); ok {
		return raw, nil
	}
	return json.Marshal(params)
}

func toRPCError(err error) *RPCError {
	if rpcErr, ok := err.(*RPCError); ok {
		return rpcErr
	}
	return &RPCError{Code: -32603, Message: err.Error()}
}

func trim(b []byte) []byte {
	for len(b) > 0 && (b[len(b)-1] == '\n' || b[len(b)-1] == '\r' || b[len(b)-1] == ' ') {
		b = b[:len(b)-1]
	}
	return b
}
