package mcp

import (
	"context"
	"encoding/json"
	"net"
	"testing"
	"time"
)

func TestPeerCallRoundTrip(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cConn, sConn := net.Pipe()
	server := NewPeer(sConn, func(_ context.Context, method string, params json.RawMessage) (any, error) {
		if method != "add" {
			return nil, &RPCError{Code: -32601, Message: "no"}
		}
		var p []int
		json.Unmarshal(params, &p)
		return map[string]int{"sum": p[0] + p[1]}, nil
	})
	go server.Run(ctx)
	client := NewPeer(cConn, nil)
	go client.Run(ctx)
	defer client.Close()

	raw, err := client.Call(ctx, "add", []int{2, 3})
	if err != nil {
		t.Fatalf("Call: %v", err)
	}
	var res map[string]int
	json.Unmarshal(raw, &res)
	if res["sum"] != 5 {
		t.Errorf("sum = %d, want 5", res["sum"])
	}
}

func TestPeerCallError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cConn, sConn := net.Pipe()
	server := NewPeer(sConn, func(_ context.Context, _ string, _ json.RawMessage) (any, error) {
		return nil, &RPCError{Code: -32000, Message: "boom"}
	})
	go server.Run(ctx)
	client := NewPeer(cConn, nil)
	go client.Run(ctx)
	defer client.Close()

	if _, err := client.Call(ctx, "anything", nil); err == nil {
		t.Fatal("expected error from server handler")
	}
}
