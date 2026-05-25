// Command vmux-sidecar is the Go backend that the vmux macOS app embeds and
// launches. It listens on a user-only Unix socket for JSON-RPC traffic and
// self-terminates if the app (its parent process) dies.
//
// Phase 1 scope: version banner, socket listener, parent-PID watchdog, clean
// shutdown. JSON-RPC dispatch, PTY, and the MCP proxy arrive in later phases.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

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

	log.SetPrefix("[vmux-sidecar] ")
	log.SetFlags(log.LstdFlags | log.Lmsgprefix)
	log.Printf("vmux-sidecar v%s starting (pid=%d ppid=%d)", version, os.Getpid(), os.Getppid())

	ctx, cancel := signalContext()
	defer cancel()

	ln, path, err := socket.Listen()
	if err != nil {
		log.Fatalf("socket: %v", err)
	}
	defer ln.Close()
	log.Printf("listening on %s", path)

	go watchdog.WatchParent(ctx, func() {
		log.Printf("parent process exited; shutting down")
		cancel()
	})

	go acceptLoop(ctx, ln)

	<-ctx.Done()
	log.Printf("shutdown complete")
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

// acceptLoop accepts connections until the context is cancelled. Phase 1 only
// drains and closes connections; JSON-RPC dispatch lands in phase 2.
func acceptLoop(ctx context.Context, ln net.Listener) {
	go func() {
		<-ctx.Done()
		ln.Close() // unblock Accept on shutdown
	}()
	for {
		conn, err := ln.Accept()
		if err != nil {
			if ctx.Err() != nil {
				return // expected during shutdown
			}
			log.Printf("accept: %v", err)
			return
		}
		log.Printf("client connected")
		conn.Close()
	}
}
