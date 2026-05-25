package pty

import (
	"encoding/base64"
	"strings"
	"sync"
	"testing"
	"time"
)

// captureEmitter collects pty.data chunks and the exit code for assertions.
type captureEmitter struct {
	mu       sync.Mutex
	output   strings.Builder
	exitCode int
	exited   chan struct{}
	once     sync.Once
}

func newCaptureEmitter() *captureEmitter {
	return &captureEmitter{exited: make(chan struct{})}
}

func (e *captureEmitter) Notify(method string, params any) {
	switch method {
	case "pty.data":
		p := params.(dataParams)
		raw, _ := base64.StdEncoding.DecodeString(p.Chunk)
		e.mu.Lock()
		e.output.Write(raw)
		e.mu.Unlock()
	case "pty.exit":
		e.exitCode = params.(exitParams).Code
		e.once.Do(func() { close(e.exited) })
	}
}

func (e *captureEmitter) text() string {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.output.String()
}

func waitExit(t *testing.T, e *captureEmitter) {
	t.Helper()
	select {
	case <-e.exited:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for pty.exit")
	}
}

func TestSpawnEchoAndExit(t *testing.T) {
	em := newCaptureEmitter()
	m := NewManager(em)

	if _, err := m.Spawn("", "echo", []string{"hello-pty"}, nil); err != nil {
		t.Fatalf("spawn: %v", err)
	}
	waitExit(t, em)

	if em.exitCode != 0 {
		t.Errorf("exit code = %d, want 0", em.exitCode)
	}
	if !strings.Contains(em.text(), "hello-pty") {
		t.Errorf("output %q missing echoed text", em.text())
	}
	if len(m.List()) != 0 {
		t.Errorf("session not removed after exit: %v", m.List())
	}
}

func TestResizeReflectedByStty(t *testing.T) {
	em := newCaptureEmitter()
	m := NewManager(em)

	// bash reads a line then reports its TTY size; we resize before writing.
	sid, err := m.Spawn("", "bash", []string{"-c", "read line; stty size; exit"}, nil)
	if err != nil {
		t.Fatalf("spawn: %v", err)
	}
	if err := m.Resize(sid, 120, 40); err != nil {
		t.Fatalf("resize: %v", err)
	}
	if err := m.Write(sid, []byte("\n")); err != nil {
		t.Fatalf("write: %v", err)
	}
	waitExit(t, em)

	// `stty size` prints "rows cols".
	if !strings.Contains(em.text(), "40 120") {
		t.Errorf("stty size output %q does not reflect resize (want 40 120)", em.text())
	}
}

func TestWriteToMissingSession(t *testing.T) {
	m := NewManager(newCaptureEmitter())
	if err := m.Write("does-not-exist", []byte("x")); err == nil {
		t.Error("expected error writing to missing session")
	}
}

func TestKillAll(t *testing.T) {
	em := newCaptureEmitter()
	m := NewManager(em)
	if _, err := m.Spawn("", "sleep", []string{"30"}, nil); err != nil {
		t.Fatalf("spawn: %v", err)
	}
	if len(m.List()) != 1 {
		t.Fatalf("expected 1 session, got %d", len(m.List()))
	}
	m.KillAll()
	waitExit(t, em) // killed sleep still emits pty.exit
	if len(m.List()) != 0 {
		t.Errorf("sessions remain after KillAll: %v", m.List())
	}
}
