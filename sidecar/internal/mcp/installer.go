package mcp

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// Runtimes reports the resolved paths of the runtimes used to launch MCP
// servers. An empty string means the runtime was not found on PATH.
type Runtimes struct {
	Node string
	Npx  string
	Uvx  string
}

// DetectRuntimes locates node/npx/uvx on PATH.
func DetectRuntimes() Runtimes {
	return Runtimes{
		Node: lookPath("node"),
		Npx:  lookPath("npx"),
		Uvx:  lookPath("uvx"),
	}
}

func lookPath(name string) string {
	p, err := exec.LookPath(name)
	if err != nil {
		return ""
	}
	return p
}

// Installer prepares launch configs for registry servers, isolating their
// package caches under a vmux-managed directory (no global pollution).
type Installer struct {
	serversDir string
	runtimes   Runtimes
}

// NewInstaller manages servers under serversDir (e.g. AppSupport/vmux/mcp-servers).
func NewInstaller(serversDir string) *Installer {
	return &Installer{serversDir: serversDir, runtimes: DetectRuntimes()}
}

// LaunchConfig builds the UpstreamConfig for a registry entry, resolving the
// runtime to an absolute path and isolating its cache directory.
func (i *Installer) LaunchConfig(e ServerEntry) (UpstreamConfig, error) {
	switch e.Command {
	case "npx":
		if i.runtimes.Npx == "" {
			return UpstreamConfig{}, fmt.Errorf("npx not found; install Node.js to use %q", e.Name)
		}
		cacheDir := filepath.Join(i.serversDir, "npm-cache")
		return UpstreamConfig{
			Name:    e.ID,
			Command: i.runtimes.Npx,
			Args:    e.Args,
			Env:     append(os.Environ(), "npm_config_cache="+cacheDir),
		}, nil
	case "uvx":
		if i.runtimes.Uvx == "" {
			return UpstreamConfig{}, fmt.Errorf("uvx not found; install uv to use %q", e.Name)
		}
		cacheDir := filepath.Join(i.serversDir, "uv-cache")
		return UpstreamConfig{
			Name:    e.ID,
			Command: i.runtimes.Uvx,
			Args:    e.Args,
			Env:     append(os.Environ(), "UV_CACHE_DIR="+cacheDir),
		}, nil
	default:
		return UpstreamConfig{}, fmt.Errorf("unsupported command %q for %q", e.Command, e.Name)
	}
}

// Prewarm fetches a server's package into the isolated cache so the first real
// launch is fast. Requires network; returns an error if the fetch fails.
func (i *Installer) Prewarm(ctx context.Context, e ServerEntry) error {
	cfg, err := i.LaunchConfig(e)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(i.serversDir, 0o700); err != nil {
		return err
	}
	warmCtx, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()

	// `npx -y <pkg> --help` downloads + caches the package without starting a
	// long-lived server. Best-effort: some servers ignore --help.
	args := append(append([]string{}, cfg.Args...), "--help")
	cmd := exec.CommandContext(warmCtx, cfg.Command, args...)
	cmd.Env = cfg.Env
	_ = cmd.Run() // exit code is unreliable across servers; cache fill is the goal
	return warmCtx.Err()
}
