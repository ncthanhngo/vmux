// Package browsersession detects an installed Chromium-family browser and
// launches an isolated, per-workspace instance for agent-driven automation
// (via Chrome DevTools MCP). vmux bundles no browser of its own.
package browsersession

import (
	"os"
	"path/filepath"
)

// ProfileDir is the per-workspace Chrome user-data-dir, kept inside the
// workspace so it travels with the project but stays out of git (.vmux is
// git-ignored by convention).
func ProfileDir(workspacePath string) string {
	return filepath.Join(workspacePath, ".vmux", "chrome-profile")
}

// EnsureProfileDir creates the per-workspace profile directory.
func EnsureProfileDir(workspacePath string) (string, error) {
	dir := ProfileDir(workspacePath)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	return dir, nil
}
