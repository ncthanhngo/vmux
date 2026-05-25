package mcp

import (
	"context"
	"encoding/json"
	"net"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/vmux/sidecar/internal/workspace"
)

type captureActivity struct {
	mu   sync.Mutex
	recs []ToolCallRecord
}

func (c *captureActivity) LogToolCall(r ToolCallRecord) {
	c.mu.Lock()
	c.recs = append(c.recs, r)
	c.mu.Unlock()
}

func (c *captureActivity) records() []ToolCallRecord {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]ToolCallRecord(nil), c.recs...)
}

// setupClient wires a client Peer to a proxy handler over an in-memory pipe.
func setupClient(t *testing.T, p *Proxy) (*Peer, context.Context) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	cConn, sConn := net.Pipe()
	server := NewPeer(sConn, p.handler(ctx))
	go server.Run(ctx)
	client := NewPeer(cConn, nil)
	go client.Run(ctx)
	t.Cleanup(func() { client.Close() })

	if _, err := client.Call(ctx, "initialize", InitializeParams{ProtocolVersion: ProtocolVersion}); err != nil {
		t.Fatalf("initialize: %v", err)
	}
	_ = client.Notify("notifications/initialized", struct{}{})
	return client, ctx
}

func callTool(t *testing.T, c *Peer, ctx context.Context, name string, args string) CallToolResult {
	t.Helper()
	cctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	raw, err := c.Call(cctx, "tools/call", CallToolParams{Name: name, Arguments: json.RawMessage(args)})
	if err != nil {
		t.Fatalf("tools/call %s: %v", name, err)
	}
	var res CallToolResult
	if err := json.Unmarshal(raw, &res); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	return res
}

func TestProxyNativeAndUpstream(t *testing.T) {
	// Workspace with a readable file for the native file_read tool.
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "hello.txt"), []byte("HI"), 0o644); err != nil {
		t.Fatal(err)
	}
	reg, err := workspace.NewRegistry(nil, filepath.Join(t.TempDir(), "ws.json"))
	if err != nil {
		t.Fatal(err)
	}
	defer reg.Shutdown()
	ws, err := reg.Open(dir)
	if err != nil {
		t.Fatal(err)
	}

	act := &captureActivity{}
	proxy := NewProxy(nil, act, NewNativeTools(reg, act))

	// Spawn the fake upstream MCP server (real subprocess via the helper).
	u, err := StartUpstream(context.Background(), UpstreamConfig{
		Name:    "fake",
		Command: os.Args[0],
		Args:    []string{"-test.run=TestHelperMCPServer"},
		Env:     append(os.Environ(), "VMUX_FAKE_MCP=1"),
	}, nil)
	if err != nil {
		t.Fatalf("start upstream: %v", err)
	}
	defer u.Close()
	proxy.AddUpstream(u)

	client, ctx := setupClient(t, proxy)

	// tools/list aggregates native + upstream tools.
	raw, err := client.Call(ctx, "tools/list", struct{}{})
	if err != nil {
		t.Fatalf("tools/list: %v", err)
	}
	var lt ListToolsResult
	json.Unmarshal(raw, &lt)
	names := map[string]bool{}
	for _, tl := range lt.Tools {
		names[tl.Name] = true
	}
	for _, want := range []string{"file_read", "workspace_list", "fake_echo"} {
		if !names[want] {
			t.Errorf("tools/list missing %q (got %v)", want, names)
		}
	}

	// Native tool: read a workspace file.
	res := callTool(t, client, ctx, "file_read", `{"workspaceId":"`+ws.ID+`","path":"hello.txt"}`)
	if len(res.Content) == 0 || res.Content[0].Text != "HI" {
		t.Errorf("file_read = %+v, want text HI", res.Content)
	}

	// Native traversal guard rejects escaping paths.
	res = callTool(t, client, ctx, "file_read", `{"workspaceId":"`+ws.ID+`","path":"../../etc/passwd"}`)
	if !res.IsError {
		t.Errorf("expected traversal guard error, got %+v", res)
	}

	// Upstream tool routes to the fake server and echoes.
	res = callTool(t, client, ctx, "fake_echo", `{"x":1}`)
	if len(res.Content) == 0 || res.Content[0].Text != `echo:{"x":1}` {
		t.Errorf("fake_echo = %+v", res.Content)
	}

	// Activity logged the calls, including the upstream attribution.
	var sawUpstream bool
	for _, r := range act.records() {
		if r.Tool == "fake_echo" && r.Upstream == "fake" {
			sawUpstream = true
		}
	}
	if !sawUpstream {
		t.Errorf("activity did not record upstream call: %+v", act.records())
	}
}
