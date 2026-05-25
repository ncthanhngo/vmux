package mcp

import (
	"context"
	"encoding/json"
	"os"
	"testing"
)

// TestHelperMCPServer is not a real test: when VMUX_FAKE_MCP is set the test
// binary re-execs into this function and behaves as a minimal MCP server over
// stdio. Used by TestProxyRoutesUpstream to exercise the real spawn path.
func TestHelperMCPServer(t *testing.T) {
	if os.Getenv("VMUX_FAKE_MCP") == "" {
		t.Skip("helper process; not run directly")
	}
	runFakeMCPServer()
	os.Exit(0) // bypass the test framework's stdout writes (would corrupt MCP)
}

func runFakeMCPServer() {
	handler := func(_ context.Context, method string, params json.RawMessage) (any, error) {
		switch method {
		case "initialize":
			return InitializeResult{
				ProtocolVersion: ProtocolVersion,
				Capabilities:    Capabilities{Tools: &struct{}{}},
				ServerInfo:      Implementation{Name: "fake", Version: "1"},
			}, nil
		case "notifications/initialized":
			return nil, nil
		case "tools/list":
			return ListToolsResult{Tools: []Tool{{
				Name:        "fake_echo",
				Description: "echo arguments back",
				InputSchema: json.RawMessage(`{"type":"object"}`),
			}}}, nil
		case "tools/call":
			var c CallToolParams
			_ = json.Unmarshal(params, &c)
			return TextResult("echo:" + string(c.Arguments)), nil
		default:
			return nil, &RPCError{Code: -32601, Message: "method not found"}
		}
	}
	peer := NewPeer(rwc{r: os.Stdin, w: os.Stdout}, handler)
	_ = peer.Run(context.Background())
}
