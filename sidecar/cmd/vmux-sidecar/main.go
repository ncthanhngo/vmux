// Command vmux-sidecar is the Go backend that the vmux macOS app embeds and
// launches. It serves JSON-RPC over a user-only Unix socket — PTY sessions and
// workspace tracking — and self-terminates if the app (its parent) dies.
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/vmux/sidecar/internal"
	"github.com/vmux/sidecar/internal/paths"
	"github.com/vmux/sidecar/internal/socket"
	"github.com/vmux/sidecar/internal/watchdog"
)

// version is overridden at build time via -ldflags "-X main.version=...".
var version = "0.0.0"

func main() {
	showVersion := flag.Bool("version", false, "print version and exit")
	flag.Parse()

	if *showVersion {
		fmt.Printf("vmux-sidecar v%s\n", version)
		return
	}

	// Log to stderr; when launched by the app, stderr is redirected to
	// ~/Library/Logs/vmux/sidecar.log.
	log := slog.New(slog.NewTextHandler(os.Stderr, nil))
	log.Info("vmux-sidecar starting", "version", version, "pid", os.Getpid(), "ppid", os.Getppid())

	ctx, cancel := signalContext()
	defer cancel()

	ln, sockPath, err := socket.Listen()
	if err != nil {
		log.Error("socket listen failed", "err", err)
		os.Exit(1)
	}
	defer ln.Close()
	log.Info("listening", "socket", sockPath)

	store, err := paths.WorkspacesStore()
	if err != nil {
		log.Error("resolve workspace store", "err", err)
		os.Exit(1)
	}
	shotsDir, err := paths.ShotsDir()
	if err != nil {
		log.Error("resolve shots dir", "err", err)
		os.Exit(1)
	}
	sessionsDir, err := paths.SessionsDir()
	if err != nil {
		log.Error("resolve sessions dir", "err", err)
		os.Exit(1)
	}
	svc, err := internal.NewService(log, store, shotsDir, sessionsDir)
	if err != nil {
		log.Error("service init failed", "err", err)
		os.Exit(1)
	}

	// MCP proxy listens on a second socket; agents reach it via vmux-mcp-bridge.
	mcpPath, err := paths.MCPSocketPath()
	if err != nil {
		log.Error("resolve mcp socket", "err", err)
		os.Exit(1)
	}
	mcpLn, err := socket.ListenAt(mcpPath)
	if err != nil {
		log.Error("mcp socket listen failed", "err", err)
		os.Exit(1)
	}
	defer mcpLn.Close()
	log.Info("mcp proxy listening", "socket", mcpPath)

	go watchdog.WatchParent(ctx, func() {
		log.Info("parent process exited; shutting down")
		cancel()
	})

	serveErr := make(chan error, 1)
	go func() { serveErr <- svc.RPC.Serve(ctx, ln) }()
	go func() {
		if err := svc.MCP.Serve(ctx, mcpLn); err != nil {
			log.Error("mcp serve error", "err", err)
		}
	}()

	select {
	case <-ctx.Done():
	case err := <-serveErr:
		if err != nil {
			log.Error("serve error", "err", err)
		}
	}

	log.Info("shutting down")
	svc.Shutdown()
	log.Info("shutdown complete")
}

// signalContext returns a context cancelled on SIGINT/SIGTERM.
func signalContext() (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sig
		cancel()
	}()
	return ctx, cancel
}
