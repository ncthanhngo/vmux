package browsersession

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestProfileDir(t *testing.T) {
	got := ProfileDir("/tmp/ws")
	want := filepath.Join("/tmp/ws", ".vmux", "chrome-profile")
	if got != want {
		t.Errorf("ProfileDir = %q, want %q", got, want)
	}
}

func TestLaunchArgs(t *testing.T) {
	args := LaunchArgs("/tmp/ws/.vmux/chrome-profile")
	joined := strings.Join(args, " ")
	for _, want := range []string{
		"--user-data-dir=/tmp/ws/.vmux/chrome-profile",
		"--remote-debugging-port=0",
		"--no-first-run",
	} {
		if !strings.Contains(joined, want) {
			t.Errorf("LaunchArgs missing %q (got %v)", want, args)
		}
	}
}

func TestDetectBrowsersDoesNotPanic(t *testing.T) {
	// Result depends on the machine; just ensure detection runs cleanly.
	_ = DetectBrowsers()
	if _, err := NewManager(); err != nil {
		t.Logf("no browser installed (acceptable): %v", err)
	}
}
