// Package pty spawns and manages PTY-backed child processes (shells, AI agent
// CLIs) and streams their output to clients as JSON-RPC notifications.
package pty

import (
	"encoding/base64"
	"os"
	"os/exec"
	"sync"

	"github.com/creack/pty"
)

// chunkSize bounds a single pty.data payload before base64 encoding. 32 KiB
// keeps base64 + JSON overhead under the RPC line budget.
const chunkSize = 32 << 10

// Emitter delivers server→client notifications (satisfied by *rpc.Server).
type Emitter interface {
	Notify(method string, params any)
}

type dataParams struct {
	SessionID string `json:"sessionId"`
	Chunk     string `json:"chunk"` // base64
}

type exitParams struct {
	SessionID string `json:"sessionId"`
	Code      int    `json:"code"`
}

// Session is a single PTY-backed process.
type Session struct {
	ID  string
	cmd *exec.Cmd
	pty *os.File

	emitter Emitter
	onExit  func(id string)
	once    sync.Once
}

// startSession launches name+args under a PTY in cwd with env, then begins
// pumping its output. onExit fires once after the process exits.
func startSession(id, cwd, name string, args, env []string, em Emitter, onExit func(string)) (*Session, error) {
	cmd := exec.Command(name, args...)
	cmd.Dir = cwd
	cmd.Env = buildEnv(env)

	// creack/pty sets Setsid + controlling tty, making the child a session
	// leader. That means closing the master (e.g. on sidecar death) delivers
	// SIGHUP to the child group, so no orphan survives an ungraceful exit.
	ptmx, err := pty.StartWithSize(cmd, &pty.Winsize{Cols: 80, Rows: 24})
	if err != nil {
		return nil, err
	}

	s := &Session{ID: id, cmd: cmd, pty: ptmx, emitter: em, onExit: onExit}
	go s.readPump()
	return s, nil
}

func (s *Session) readPump() {
	buf := make([]byte, chunkSize)
	for {
		n, err := s.pty.Read(buf)
		if n > 0 {
			s.emitter.Notify("pty.data", dataParams{
				SessionID: s.ID,
				Chunk:     base64.StdEncoding.EncodeToString(buf[:n]),
			})
		}
		if err != nil {
			break // EOF when the child exits or the pty is closed
		}
	}
	code := 0
	if err := s.cmd.Wait(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			code = exitErr.ExitCode()
		} else {
			code = -1
		}
	}
	s.emitter.Notify("pty.exit", exitParams{SessionID: s.ID, Code: code})
	s.onExit(s.ID)
}

func (s *Session) write(data []byte) error {
	_, err := s.pty.Write(data)
	return err
}

func (s *Session) resize(cols, rows uint16) error {
	return pty.Setsize(s.pty, &pty.Winsize{Cols: cols, Rows: rows})
}

// kill terminates the child's process group and closes the master, which makes
// the readPump observe EOF and emit pty.exit.
func (s *Session) kill() {
	s.once.Do(func() {
		if s.cmd.Process != nil {
			// Negative pid signals the whole process group (session leader),
			// reaping grandchildren the agent may have spawned.
			_ = killGroup(s.cmd.Process.Pid)
		}
		s.pty.Close()
	})
}
