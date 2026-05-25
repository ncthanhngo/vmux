// Package watchdog terminates the sidecar when its launching parent (the Swift
// app) dies, preventing orphaned processes if the app crashes.
package watchdog

import (
	"context"
	"os"
	"time"
)

// WatchParent polls the parent PID and invokes onParentExit once the parent is
// gone. On macOS an orphaned child is reparented to launchd (PID 1), so a ppid
// of 1 — or any change away from the original launcher — means the app exited.
func WatchParent(ctx context.Context, onParentExit func()) {
	initialPPID := os.Getppid()
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// On macOS an orphan reparents to launchd (PID 1); the
			// ppid != initialPPID arm is belt-and-suspenders for any
			// platform that reparents elsewhere.
			ppid := os.Getppid()
			if ppid == 1 || ppid != initialPPID {
				onParentExit()
				return
			}
		}
	}
}
