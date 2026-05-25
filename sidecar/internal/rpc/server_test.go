package rpc

import (
	"bufio"
	"context"
	"encoding/json"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// newTestServer starts a server on a short-path Unix socket and returns a
// connected client reader/writer plus the server.
func newTestServer(t *testing.T) (*Server, net.Conn, *bufio.Reader) {
	t.Helper()
	dir, err := os.MkdirTemp("/tmp", "vmuxrpc")
	if err != nil {
		t.Fatalf("mkdtemp: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	sock := filepath.Join(dir, "s.sock")

	ln, err := net.Listen("unix", sock)
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	srv := NewServer(nil)

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	go srv.Serve(ctx, ln)

	conn, err := net.Dial("unix", sock)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	t.Cleanup(func() { conn.Close() })
	return srv, conn, bufio.NewReader(conn)
}

func writeLine(t *testing.T, conn net.Conn, v any) {
	t.Helper()
	b, _ := json.Marshal(v)
	if _, err := conn.Write(append(b, '\n')); err != nil {
		t.Fatalf("write: %v", err)
	}
}

func readResponse(t *testing.T, r *bufio.Reader) Response {
	t.Helper()
	line, err := r.ReadBytes('\n')
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	var resp Response
	if err := json.Unmarshal(line, &resp); err != nil {
		t.Fatalf("unmarshal response: %v (%s)", err, line)
	}
	return resp
}

func TestRequestResponse(t *testing.T) {
	srv, conn, r := newTestServer(t)
	srv.Register("echo", func(_ context.Context, params json.RawMessage) (any, error) {
		var v map[string]int
		json.Unmarshal(params, &v)
		return v, nil
	})

	id := json.RawMessage(`1`)
	writeLine(t, conn, Request{JSONRPC: "2.0", ID: &id, Method: "echo", Params: json.RawMessage(`{"v":42}`)})

	resp := readResponse(t, r)
	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}
	var got map[string]int
	json.Unmarshal(resp.Result, &got)
	if got["v"] != 42 {
		t.Errorf("echo result = %v, want v=42", got)
	}
}

func TestMethodNotFound(t *testing.T) {
	_, conn, r := newTestServer(t)
	id := json.RawMessage(`7`)
	writeLine(t, conn, Request{JSONRPC: "2.0", ID: &id, Method: "nope"})

	resp := readResponse(t, r)
	if resp.Error == nil || resp.Error.Code != CodeMethodNotFound {
		t.Fatalf("want MethodNotFound, got %+v", resp.Error)
	}
}

func TestServerNotification(t *testing.T) {
	srv, _, r := newTestServer(t)
	// Give Serve a moment to register the connection before broadcasting.
	time.Sleep(50 * time.Millisecond)
	srv.Notify("pty.data", map[string]string{"sessionId": "s1", "chunk": "aGk="})

	line, err := r.ReadBytes('\n')
	if err != nil {
		t.Fatalf("read notification: %v", err)
	}
	var n struct {
		Method string            `json:"method"`
		Params map[string]string `json:"params"`
	}
	if err := json.Unmarshal(line, &n); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if n.Method != "pty.data" || n.Params["chunk"] != "aGk=" {
		t.Errorf("unexpected notification: %s", line)
	}
}

func TestParseError(t *testing.T) {
	_, conn, r := newTestServer(t)
	conn.Write([]byte("{not json}\n"))
	resp := readResponse(t, r)
	if resp.Error == nil || resp.Error.Code != CodeParse {
		t.Fatalf("want ParseError, got %+v", resp.Error)
	}
}
