// Package socket resolves the vmux IPC socket path and creates a user-only
// (0600) Unix domain listener for JSON-RPC traffic between the Swift app and
// the Go sidecar.
package socket

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"syscall"
)

// Path returns the canonical Unix socket path:
// ~/Library/Application Support/vmux/sidecar.sock
func Path() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home dir: %w", err)
	}
	return filepath.Join(home, "Library", "Application Support", "vmux", "sidecar.sock"), nil
}

// Listen creates a 0600 Unix listener at the canonical Path().
func Listen() (net.Listener, string, error) {
	path, err := Path()
	if err != nil {
		return nil, "", err
	}
	ln, err := ListenAt(path)
	return ln, path, err
}

// ListenAt creates the parent directory, removes any stale socket file, and
// returns a Unix listener bound at the given path with 0600 permissions.
//
// macOS caps the socket path at 104 bytes (sockaddr_un.sun_path); the canonical
// path under the home dir stays well under that.
func ListenAt(path string) (net.Listener, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, fmt.Errorf("create socket dir: %w", err)
	}
	// A leftover socket from a crashed run would make Listen fail with EADDRINUSE.
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("remove stale socket: %w", err)
	}
	// Tighten umask so the socket inode is created 0600 at bind time, closing
	// the brief world-readable window between net.Listen and a follow-up chmod.
	oldMask := syscall.Umask(0o177)
	ln, err := net.Listen("unix", path)
	syscall.Umask(oldMask)
	if err != nil {
		return nil, fmt.Errorf("listen on %s: %w", path, err)
	}
	if err := os.Chmod(path, 0o600); err != nil {
		ln.Close()
		return nil, fmt.Errorf("chmod socket: %w", err)
	}
	return ln, nil
}
