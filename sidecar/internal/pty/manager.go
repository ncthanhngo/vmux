package pty

import (
	"fmt"
	"sort"
	"sync"

	"github.com/vmux/sidecar/internal/id"
)

// Manager owns the set of live PTY sessions.
type Manager struct {
	emitter Emitter

	mu       sync.Mutex
	sessions map[string]*Session
	tap      func(sessionID string, data []byte)
}

// NewManager returns a Manager that emits pty.* notifications via em.
func NewManager(em Emitter) *Manager {
	return &Manager{emitter: em, sessions: make(map[string]*Session)}
}

// SetOutputTap installs a callback that observes raw output of every session
// (used for server-side port detection). Set once at startup.
func (m *Manager) SetOutputTap(fn func(sessionID string, data []byte)) {
	m.tap = fn
}

// Spawn launches name+args under a PTY in cwd and returns the session id.
func (m *Manager) Spawn(cwd, name string, args, env []string) (string, error) {
	if name == "" {
		return "", fmt.Errorf("pty: empty command")
	}
	sid := id.New()
	s, err := startSession(sid, cwd, name, args, env, m.emitter, m.remove, m.tap)
	if err != nil {
		return "", err
	}
	m.mu.Lock()
	m.sessions[sid] = s
	m.mu.Unlock()
	return sid, nil
}

func (m *Manager) Write(sid string, data []byte) error {
	s, err := m.get(sid)
	if err != nil {
		return err
	}
	return s.write(data)
}

func (m *Manager) Resize(sid string, cols, rows uint16) error {
	s, err := m.get(sid)
	if err != nil {
		return err
	}
	return s.resize(cols, rows)
}

func (m *Manager) Kill(sid string) error {
	s, err := m.get(sid)
	if err != nil {
		return err
	}
	s.kill()
	return nil
}

// KillAll terminates every session; used during sidecar shutdown.
func (m *Manager) KillAll() {
	m.mu.Lock()
	all := make([]*Session, 0, len(m.sessions))
	for _, s := range m.sessions {
		all = append(all, s)
	}
	m.mu.Unlock()
	for _, s := range all {
		s.kill()
	}
}

// List returns the live session ids, sorted for stable output.
func (m *Manager) List() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	ids := make([]string, 0, len(m.sessions))
	for sid := range m.sessions {
		ids = append(ids, sid)
	}
	sort.Strings(ids)
	return ids
}

func (m *Manager) get(sid string) (*Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	s, ok := m.sessions[sid]
	if !ok {
		return nil, fmt.Errorf("pty: no session %q", sid)
	}
	return s, nil
}

func (m *Manager) remove(sid string) {
	m.mu.Lock()
	delete(m.sessions, sid)
	m.mu.Unlock()
}
