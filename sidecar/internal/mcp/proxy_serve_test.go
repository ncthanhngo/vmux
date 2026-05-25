package mcp

import (
	"context"
	"encoding/json"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/vmux/sidecar/internal/workspace"
)

// TestProxyServeOverSocket exercises the real production path: Proxy.Serve on a
// Unix socket, a client dialing it (as vmux-mcp-bridge does), and an MCP
// session over that connection.
func TestProxyServeOverSocket(t *testing.T) {
	reg, err := workspace.NewRegistry(nil, filepath.Join(t.TempDir(), "ws.json"))
	if err != nil {
		t.Fatal(err)
	}
	defer reg.Shutdown()
	proxy := NewProxy(nil, nil, NewNativeTools(reg, nil, nil))

	dir, err := os.MkdirTemp("/tmp", "vmuxmcp")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	sock := filepath.Join(dir, "mcp.sock")

	ln, err := net.Listen("unix", sock)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go proxy.Serve(ctx, ln)

	conn, err := net.Dial("unix", sock)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	client := NewPeer(conn, nil)
	go client.Run(ctx)
	defer client.Close()

	cctx, c2 := context.WithTimeout(ctx, 5*time.Second)
	defer c2()
	if _, err := client.Call(cctx, "initialize", InitializeParams{ProtocolVersion: ProtocolVersion}); err != nil {
		t.Fatalf("initialize over socket: %v", err)
	}
	raw, err := client.Call(cctx, "tools/list", struct{}{})
	if err != nil {
		t.Fatalf("tools/list over socket: %v", err)
	}
	var lt ListToolsResult
	json.Unmarshal(raw, &lt)
	if len(lt.Tools) == 0 {
		t.Error("expected native tools over socket")
	}

	// ping is part of the MCP lifecycle.
	if _, err := client.Call(cctx, "ping", struct{}{}); err != nil {
		t.Errorf("ping: %v", err)
	}
	// Unknown method returns a JSON-RPC error.
	if _, err := client.Call(cctx, "bogus/method", struct{}{}); err == nil {
		t.Error("expected error for unknown method")
	}
}
