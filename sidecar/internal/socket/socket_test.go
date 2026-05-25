package socket

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPath(t *testing.T) {
	p, err := Path()
	if err != nil {
		t.Fatalf("Path() error: %v", err)
	}
	if !strings.HasSuffix(p, filepath.Join("vmux", "sidecar.sock")) {
		t.Errorf("unexpected socket path: %s", p)
	}
}

// shortSocketPath returns a path under /tmp short enough to fit the macOS
// 104-byte sun_path limit (t.TempDir() under /var/folders is too long).
func shortSocketPath(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("/tmp", "vmux")
	if err != nil {
		t.Fatalf("mkdtemp: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	return filepath.Join(dir, "sidecar.sock")
}

func TestListenAtPermissions(t *testing.T) {
	path := shortSocketPath(t)
	ln, err := ListenAt(path)
	if err != nil {
		t.Fatalf("ListenAt() error: %v", err)
	}
	defer ln.Close()

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat socket: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("socket perms = %o, want 600", perm)
	}
}

func TestListenAtRemovesStaleSocket(t *testing.T) {
	path := shortSocketPath(t)

	// A clean Close() unlinks the socket, but a crashed run leaves the file
	// behind. Simulate that leftover and verify ListenAt reclaims the path.
	if err := os.WriteFile(path, nil, 0o600); err != nil {
		t.Fatalf("seed stale socket file: %v", err)
	}
	ln, err := ListenAt(path)
	if err != nil {
		t.Fatalf("ListenAt() should reclaim stale socket: %v", err)
	}
	ln.Close()
}
