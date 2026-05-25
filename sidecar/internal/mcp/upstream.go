package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os/exec"
	"sync"
	"syscall"
	"time"
)

// UpstreamConfig describes how to launch a backing MCP server.
type UpstreamConfig struct {
	Name    string
	Command string
	Args    []string
	Env     []string
}

// Upstream is a running backing MCP server vmux drives as a client.
type Upstream struct {
	Name string

	cmd    *exec.Cmd
	peer   *Peer
	cancel context.CancelFunc

	mu    sync.RWMutex
	tools []Tool
}

type rwc struct {
	r io.ReadCloser
	w io.WriteCloser
}

func (c rwc) Read(p []byte) (int, error)  { return c.r.Read(p) }
func (c rwc) Write(p []byte) (int, error) { return c.w.Write(p) }
func (c rwc) Close() error {
	werr := c.w.Close()
	rerr := c.r.Close()
	if werr != nil {
		return werr
	}
	return rerr
}

// StartUpstream launches the server, performs the MCP initialize handshake, and
// fetches its tool list.
func StartUpstream(ctx context.Context, cfg UpstreamConfig, log *slog.Logger) (*Upstream, error) {
	cmd := exec.Command(cfg.Command, cfg.Args...)
	cmd.Env = cfg.Env
	// Own process group so Close reaps any children the server spawns.
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Stderr = &logWriter{log: log, name: cfg.Name}

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start %s: %w", cfg.Name, err)
	}

	runCtx, cancel := context.WithCancel(context.Background())
	peer := NewPeer(rwc{r: stdout, w: stdin}, nil)
	go peer.Run(runCtx)

	u := &Upstream{Name: cfg.Name, cmd: cmd, peer: peer, cancel: cancel}
	if err := u.handshake(ctx); err != nil {
		u.Close()
		return nil, fmt.Errorf("handshake %s: %w", cfg.Name, err)
	}
	return u, nil
}

func (u *Upstream) handshake(ctx context.Context) error {
	hsCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	_, err := u.peer.Call(hsCtx, "initialize", InitializeParams{
		ProtocolVersion: ProtocolVersion,
		Capabilities:    Capabilities{Tools: &struct{}{}},
		ClientInfo:      Implementation{Name: "vmux", Version: "0.0.0"},
	})
	if err != nil {
		return err
	}
	if err := u.peer.Notify("notifications/initialized", struct{}{}); err != nil {
		return err
	}
	return u.refreshTools(hsCtx)
}

func (u *Upstream) refreshTools(ctx context.Context) error {
	raw, err := u.peer.Call(ctx, "tools/list", struct{}{})
	if err != nil {
		return err
	}
	var res ListToolsResult
	if err := json.Unmarshal(raw, &res); err != nil {
		return err
	}
	u.mu.Lock()
	u.tools = res.Tools
	u.mu.Unlock()
	return nil
}

// Tools returns the upstream's advertised tools.
func (u *Upstream) Tools() []Tool {
	u.mu.RLock()
	defer u.mu.RUnlock()
	out := make([]Tool, len(u.tools))
	copy(out, u.tools)
	return out
}

// CallTool forwards a tools/call to the upstream server.
func (u *Upstream) CallTool(ctx context.Context, name string, args json.RawMessage) (CallToolResult, error) {
	raw, err := u.peer.Call(ctx, "tools/call", CallToolParams{Name: name, Arguments: args})
	if err != nil {
		return CallToolResult{}, err
	}
	var res CallToolResult
	if err := json.Unmarshal(raw, &res); err != nil {
		return CallToolResult{}, err
	}
	return res, nil
}

// Close terminates the upstream process group and its peer.
func (u *Upstream) Close() {
	u.cancel()
	u.peer.Close()
	if u.cmd.Process != nil {
		_ = syscall.Kill(-u.cmd.Process.Pid, syscall.SIGKILL)
	}
	_ = u.cmd.Wait()
}

// logWriter routes an upstream server's stderr into the structured log.
type logWriter struct {
	log  *slog.Logger
	name string
}

func (w *logWriter) Write(p []byte) (int, error) {
	if w.log != nil {
		w.log.Debug("upstream stderr", "server", w.name, "msg", string(p))
	}
	return len(p), nil
}
