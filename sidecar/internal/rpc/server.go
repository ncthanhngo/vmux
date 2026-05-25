package rpc

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net"
	"sync"

	"github.com/vmux/sidecar/internal/peercred"
)

// maxLine caps a single inbound JSON message (the socket is local + trusted, so
// this only guards against a runaway allocation, not an adversary).
const maxLine = 16 << 20 // 16 MiB

// sendBuffer is the per-connection outbound queue depth. When a client cannot
// keep up, the queue fills and emitters block, which propagates backpressure to
// the PTY read pumps (the kernel then throttles the child). Output is never
// dropped for a connected client and memory stays bounded — but note this
// backpressure is shared: a client that stops draining will pause every PTY
// session feeding it, not just the noisy one. Per-session isolation (separate
// ring buffers) is deferred until multi-pane throughput demands it.
const sendBuffer = 256

// Server is a line-delimited JSON-RPC 2.0 server. Register methods before
// Serve; Notify may be called concurrently once serving.
type Server struct {
	router *router
	log    *slog.Logger

	mu    sync.Mutex
	conns map[*conn]struct{}
}

// NewServer creates a server with no registered methods.
func NewServer(log *slog.Logger) *Server {
	if log == nil {
		log = slog.Default()
	}
	return &Server{router: newRouter(), log: log, conns: make(map[*conn]struct{})}
}

// Register binds a handler to a method name. Not safe to call after Serve.
func (s *Server) Register(method string, h Handler) { s.router.register(method, h) }

// Notify broadcasts a server-initiated notification to every connected client.
// It blocks per-connection if a client's send queue is full (backpressure).
// With multiple clients the per-connection sends run concurrently so a slow
// client cannot delay delivery to a fast one.
func (s *Server) Notify(method string, params any) {
	n := newNotification(method, params)
	line, err := json.Marshal(n)
	if err != nil {
		s.log.Error("marshal notification", "method", method, "err", err)
		return
	}
	conns := s.snapshotConns()
	if len(conns) <= 1 {
		for _, c := range conns {
			c.send(line)
		}
		return
	}
	var wg sync.WaitGroup
	for _, c := range conns {
		wg.Add(1)
		go func(c *conn) {
			defer wg.Done()
			c.send(line)
		}(c)
	}
	wg.Wait()
}

func (s *Server) snapshotConns() []*conn {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]*conn, 0, len(s.conns))
	for c := range s.conns {
		out = append(out, c)
	}
	return out
}

// Serve accepts connections until ctx is cancelled or the listener closes.
func (s *Server) Serve(ctx context.Context, ln net.Listener) error {
	go func() {
		<-ctx.Done()
		ln.Close() // unblock Accept
	}()
	for {
		nc, err := ln.Accept()
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return err
		}
		if err := peercred.Check(nc); err != nil {
			s.log.Warn("rejecting connection", "err", err)
			nc.Close()
			continue
		}
		c := newConn(nc)
		s.addConn(c)
		go s.handleConn(ctx, c)
	}
}

func (s *Server) addConn(c *conn) {
	s.mu.Lock()
	s.conns[c] = struct{}{}
	s.mu.Unlock()
}

func (s *Server) removeConn(c *conn) {
	s.mu.Lock()
	delete(s.conns, c)
	s.mu.Unlock()
}

func (s *Server) handleConn(ctx context.Context, c *conn) {
	defer func() {
		s.removeConn(c)
		c.close()
	}()
	go c.writeLoop(s.log)

	reader := bufio.NewReaderSize(c.netConn, 64<<10)
	for {
		line, err := readLine(reader)
		if err != nil {
			if !errors.Is(err, io.EOF) && ctx.Err() == nil {
				s.log.Debug("conn read ended", "err", err)
			}
			return
		}
		if len(line) == 0 {
			continue
		}
		go s.dispatch(ctx, c, line)
	}
}

func (s *Server) dispatch(ctx context.Context, c *conn, line []byte) {
	var req Request
	if err := json.Unmarshal(line, &req); err != nil {
		c.sendValue(s.log, errorResponse(nil, CodeParse, "parse error"))
		return
	}
	// A request without an id is a notification from the client: run it, no reply.
	isNotification := req.ID == nil

	h, ok := s.router.lookup(req.Method)
	if !ok {
		if !isNotification {
			c.sendValue(s.log, errorResponse(req.ID, CodeMethodNotFound, "method not found: "+req.Method))
		}
		return
	}

	result, err := h(ctx, req.Params)
	if isNotification {
		return
	}
	if err != nil {
		var rpcErr *Error
		if errors.As(err, &rpcErr) {
			c.sendValue(s.log, Response{JSONRPC: version, ID: req.ID, Error: rpcErr})
		} else {
			c.sendValue(s.log, errorResponse(req.ID, CodeInternal, err.Error()))
		}
		return
	}
	raw, err := json.Marshal(result)
	if err != nil {
		c.sendValue(s.log, errorResponse(req.ID, CodeInternal, "marshal result: "+err.Error()))
		return
	}
	c.sendValue(s.log, resultResponse(req.ID, raw))
}

// readLine reads one newline-delimited message, rejecting oversize frames.
func readLine(r *bufio.Reader) ([]byte, error) {
	line, err := r.ReadBytes('\n')
	if len(line) > maxLine {
		return nil, errors.New("rpc: message exceeds size limit")
	}
	// Trim trailing newline (and CR) without allocating.
	for len(line) > 0 && (line[len(line)-1] == '\n' || line[len(line)-1] == '\r') {
		line = line[:len(line)-1]
	}
	return line, err
}
