package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
)

// attach connects the local terminal to a PTY session: stdin → pty.write,
// pty.data → stdout, with SIGWINCH-driven resize, until the session exits.
func attach(c *client, sid string) {
	done := make(chan int, 1)

	c.onNotify = func(method string, params json.RawMessage) {
		switch method {
		case "pty.data":
			var p struct {
				SessionID string `json:"sessionId"`
				Chunk     string `json:"chunk"`
			}
			if json.Unmarshal(params, &p) == nil && p.SessionID == sid {
				if data, err := base64.StdEncoding.DecodeString(p.Chunk); err == nil {
					os.Stdout.Write(data)
				}
			}
		case "pty.exit":
			var p struct {
				SessionID string `json:"sessionId"`
				Code      int    `json:"code"`
			}
			if json.Unmarshal(params, &p) == nil && p.SessionID == sid {
				select {
				case done <- p.Code:
				default:
				}
			}
		}
	}

	sendResize(c, sid)

	if restore, err := makeRaw(); err == nil {
		defer restore()
	}

	winch := make(chan os.Signal, 1)
	signal.Notify(winch, syscall.SIGWINCH)
	go func() {
		for range winch {
			sendResize(c, sid)
		}
	}()

	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := os.Stdin.Read(buf)
			if n > 0 {
				_ = c.call("pty.write", map[string]string{
					"sessionId": sid,
					"data":      base64.StdEncoding.EncodeToString(buf[:n]),
				}, nil)
			}
			if err != nil {
				return
			}
		}
	}()

	code := <-done
	fmt.Fprintf(os.Stderr, "\r\n[vmux-cli] session exited (code %d)\r\n", code)
}

func sendResize(c *client, sid string) {
	rows, cols := ttySize()
	_ = c.call("pty.resize", map[string]any{"sessionId": sid, "cols": cols, "rows": rows}, nil)
}

// makeRaw puts the controlling terminal into raw mode via stty and returns a
// restore func. If stdin is not a TTY (piped input) it is a no-op.
func makeRaw() (func(), error) {
	if !stdinIsTTY() {
		return func() {}, nil
	}
	saved, err := sttyCapture("-g")
	if err != nil {
		return nil, err
	}
	if err := sttyApply("raw", "-echo"); err != nil {
		return nil, err
	}
	return func() { _ = sttyApply(strings.TrimSpace(saved)) }, nil
}

func ttySize() (uint16, uint16) {
	out, err := sttyCapture("size")
	if err != nil {
		return 24, 80
	}
	var r, c int
	if _, err := fmt.Sscanf(strings.TrimSpace(out), "%d %d", &r, &c); err != nil || r == 0 {
		return 24, 80
	}
	return uint16(r), uint16(c)
}

func stdinIsTTY() bool {
	fi, err := os.Stdin.Stat()
	return err == nil && fi.Mode()&os.ModeCharDevice != 0
}

func sttyCapture(args ...string) (string, error) {
	cmd := exec.Command("stty", args...)
	cmd.Stdin = os.Stdin
	out, err := cmd.Output()
	return string(out), err
}

func sttyApply(args ...string) error {
	cmd := exec.Command("stty", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	return cmd.Run()
}
