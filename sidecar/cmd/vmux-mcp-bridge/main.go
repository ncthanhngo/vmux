// Command vmux-mcp-bridge is the thin stdio process an AI agent (e.g. Claude
// Code) launches as its MCP server. It forwards the agent's stdio to the vmux
// sidecar's MCP proxy over the Unix socket and copies responses back. All MCP
// logic lives in the sidecar; this bridge is a dumb, robust pipe.
package main

import (
	"fmt"
	"io"
	"net"
	"os"

	"github.com/vmux/sidecar/internal/paths"
)

func main() {
	sockPath, err := paths.MCPSocketPath()
	if err != nil {
		fail(err)
	}
	conn, err := net.Dial("unix", sockPath)
	if err != nil {
		fail(fmt.Errorf("connect vmux MCP proxy at %s: %w (is vmux running?)", sockPath, err))
	}
	defer conn.Close()

	// agent stdin → proxy. On EOF, half-close so the proxy sees end-of-input.
	go func() {
		io.Copy(conn, os.Stdin)
		if uc, ok := conn.(*net.UnixConn); ok {
			uc.CloseWrite()
		}
	}()

	// proxy → agent stdout. Returns when the proxy closes the connection.
	io.Copy(os.Stdout, conn)
}

func fail(err error) {
	fmt.Fprintf(os.Stderr, "vmux-mcp-bridge: %v\n", err)
	os.Exit(1)
}
