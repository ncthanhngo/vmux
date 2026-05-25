// Package paths centralizes the on-disk locations vmux uses under the user's
// Library directory, so every package agrees on where state lives.
package paths

import (
	"fmt"
	"os"
	"path/filepath"
)

// AppSupportDir returns ~/Library/Application Support/vmux.
func AppSupportDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home dir: %w", err)
	}
	return filepath.Join(home, "Library", "Application Support", "vmux"), nil
}

// SocketPath returns the JSON-RPC Unix socket path.
func SocketPath() (string, error) {
	dir, err := AppSupportDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "sidecar.sock"), nil
}

// MCPSocketPath returns the Unix socket the MCP proxy exposes to agents
// (via the vmux-mcp-bridge).
func MCPSocketPath() (string, error) {
	dir, err := AppSupportDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "mcp.sock"), nil
}

// MCPServersDir returns the directory where vmux installs managed MCP servers.
func MCPServersDir() (string, error) {
	dir, err := AppSupportDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "mcp-servers"), nil
}

// WorkspacesStore returns the persisted workspace-registry path.
func WorkspacesStore() (string, error) {
	dir, err := AppSupportDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "workspaces.json"), nil
}
