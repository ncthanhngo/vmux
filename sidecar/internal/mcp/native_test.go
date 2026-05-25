package mcp

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vmux/sidecar/internal/workspace"
)

func newTestWorkspace(t *testing.T) (*NativeTools, string) {
	t.Helper()
	dir := t.TempDir()
	reg, err := workspace.NewRegistry(nil, filepath.Join(t.TempDir(), "ws.json"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(reg.Shutdown)
	ws, err := reg.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	return NewNativeTools(reg, nil, nil, nil), ws.ID
}

// handlerFor returns the handler for a named native tool.
func handlerFor(t *testing.T, n *NativeTools, name string) ToolHandler {
	t.Helper()
	for _, rt := range n.registered() {
		if rt.tool.Name == name {
			return rt.handler
		}
	}
	t.Fatalf("no native tool %q", name)
	return nil
}

func TestFileWriteAndTraversalGuard(t *testing.T) {
	n, wsID := newTestWorkspace(t)
	write := handlerFor(t, n, "file_write")

	res, _ := write(context.Background(), json.RawMessage(`{"workspaceId":"`+wsID+`","path":"sub/out.txt","content":"hello"}`))
	if res.IsError {
		t.Fatalf("file_write failed: %+v", res.Content)
	}
	// Round-trips via file_read.
	read := handlerFor(t, n, "file_read")
	rr, _ := read(context.Background(), json.RawMessage(`{"workspaceId":"`+wsID+`","path":"sub/out.txt"}`))
	if rr.IsError || rr.Content[0].Text != "hello" {
		t.Errorf("file_read after write = %+v", rr.Content)
	}

	// Traversal escape is rejected and nothing is written outside the root.
	esc, _ := write(context.Background(), json.RawMessage(`{"workspaceId":"`+wsID+`","path":"../escape.txt","content":"x"}`))
	if !esc.IsError {
		t.Error("file_write should reject path escaping workspace")
	}
}

// TestSymlinkEscapeRejected ensures a symlink inside the workspace pointing
// outside it cannot be used to read/write beyond the root.
func TestSymlinkEscapeRejected(t *testing.T) {
	n, wsID := newTestWorkspace(t)
	ws, _ := n.ws.Get(wsID)

	outside := t.TempDir()
	if err := os.WriteFile(filepath.Join(outside, "secret.txt"), []byte("SECRET"), 0o644); err != nil {
		t.Fatal(err)
	}
	// link inside the workspace → the external directory.
	if err := os.Symlink(outside, filepath.Join(ws.Path, "link")); err != nil {
		t.Fatal(err)
	}

	read := handlerFor(t, n, "file_read")
	res, _ := read(context.Background(), json.RawMessage(`{"workspaceId":"`+wsID+`","path":"link/secret.txt"}`))
	if !res.IsError {
		t.Errorf("symlink escape should be rejected, got %+v", res.Content)
	}
}

func TestCommandRunAllowlist(t *testing.T) {
	n, wsID := newTestWorkspace(t)
	run := handlerFor(t, n, "command_run")

	ok, _ := run(context.Background(), json.RawMessage(`{"workspaceId":"`+wsID+`","command":"echo","args":["hi"]}`))
	if ok.IsError || !strings.Contains(ok.Content[0].Text, "hi") {
		t.Errorf("allowlisted echo failed: %+v", ok.Content)
	}

	denied, _ := run(context.Background(), json.RawMessage(`{"workspaceId":"`+wsID+`","command":"rm","args":["-rf","/"]}`))
	if !denied.IsError || !strings.Contains(denied.Content[0].Text, "not allowed") {
		t.Errorf("non-allowlisted command should be denied: %+v", denied.Content)
	}
}

func TestWorkspaceOpenInvalid(t *testing.T) {
	n, _ := newTestWorkspace(t)
	open := handlerFor(t, n, "workspace_open")
	res, _ := open(context.Background(), json.RawMessage(`{"path":"/no/such/dir/vmux-test"}`))
	if !res.IsError {
		t.Error("workspace_open should error on a missing path")
	}
}

func TestCloseUpstreamsIdempotent(t *testing.T) {
	reg, _ := workspace.NewRegistry(nil, filepath.Join(t.TempDir(), "ws.json"))
	defer reg.Shutdown()
	p := NewProxy(nil, nil, NewNativeTools(reg, nil, nil, nil))
	p.CloseUpstreams() // no upstreams — must not panic
	p.CloseUpstreams()
}
