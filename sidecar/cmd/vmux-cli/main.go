// Command vmux-cli is a developer/test client for the vmux sidecar. It speaks
// the same JSON-RPC over the Unix socket that the macOS app uses.
//
//	vmux-cli ls                     # list workspaces + sessions
//	vmux-cli spawn -- bash          # spawn a PTY and attach to it
//	vmux-cli attach <sessionId>     # attach to an existing session
//	vmux-cli open <path>            # register a workspace
package main

import (
	"fmt"
	"os"

	"github.com/vmux/sidecar/internal/socket"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	sockPath, err := socket.Path()
	if err != nil {
		fatal(err)
	}
	c, err := dial(sockPath)
	if err != nil {
		fatal(fmt.Errorf("connect %s: %w (is the sidecar running?)", sockPath, err))
	}
	defer c.close()

	switch os.Args[1] {
	case "ls":
		runLs(c)
	case "open":
		if len(os.Args) < 3 {
			fatal(fmt.Errorf("usage: vmux-cli open <path>"))
		}
		runOpen(c, os.Args[2])
	case "spawn":
		runSpawn(c, os.Args[2:])
	case "attach":
		if len(os.Args) < 3 {
			fatal(fmt.Errorf("usage: vmux-cli attach <sessionId>"))
		}
		attach(c, os.Args[2])
	default:
		usage()
		os.Exit(2)
	}
}

func runLs(c *client) {
	var ws struct {
		Workspaces []map[string]any `json:"workspaces"`
	}
	if err := c.call("workspace.list", struct{}{}, &ws); err != nil {
		fatal(err)
	}
	fmt.Println("workspaces:")
	for _, w := range ws.Workspaces {
		fmt.Printf("  %v  %v\n", w["id"], w["path"])
	}

	var ps struct {
		Sessions []string `json:"sessions"`
	}
	if err := c.call("pty.list", struct{}{}, &ps); err != nil {
		fatal(err)
	}
	fmt.Println("sessions:")
	for _, s := range ps.Sessions {
		fmt.Printf("  %s\n", s)
	}
}

func runOpen(c *client, path string) {
	var ws map[string]any
	if err := c.call("workspace.open", map[string]string{"path": path}, &ws); err != nil {
		fatal(err)
	}
	fmt.Printf("opened %v (id %v)\n", ws["path"], ws["id"])
}

// runSpawn spawns a PTY for the command after "--" and attaches to it.
func runSpawn(c *client, args []string) {
	cmd, cmdArgs := parseSpawnArgs(args)
	if cmd == "" {
		fatal(fmt.Errorf("usage: vmux-cli spawn [-- ]<cmd> [args...]"))
	}
	cwd, _ := os.Getwd()
	var res struct {
		SessionID string `json:"sessionId"`
	}
	params := map[string]any{"cwd": cwd, "cmd": cmd, "args": cmdArgs}
	if err := c.call("pty.spawn", params, &res); err != nil {
		fatal(err)
	}
	fmt.Fprintf(os.Stderr, "[vmux-cli] spawned session %s\n", res.SessionID)
	attach(c, res.SessionID)
}

// parseSpawnArgs splits the command from an optional leading "--".
func parseSpawnArgs(args []string) (string, []string) {
	if len(args) > 0 && args[0] == "--" {
		args = args[1:]
	}
	if len(args) == 0 {
		return "", nil
	}
	return args[0], args[1:]
}

func usage() {
	fmt.Fprint(os.Stderr, `vmux-cli — sidecar test client
  vmux-cli ls
  vmux-cli open <path>
  vmux-cli spawn -- <cmd> [args...]
  vmux-cli attach <sessionId>
`)
}

func fatal(err error) {
	fmt.Fprintf(os.Stderr, "vmux-cli: %v\n", err)
	os.Exit(1)
}
