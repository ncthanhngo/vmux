package activity

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

const memRingSize = 5000

// Store appends events to a per-session JSONL file and keeps a bounded
// in-memory ring of the most recent events for fast UI reads. Secrets are
// redacted before persistence. Subscribers receive every appended event.
type Store struct {
	baseDir string

	mu      sync.Mutex
	seq     int64
	ring    map[string][]Event // sessionId -> recent events
	files   map[string]*os.File
	subs    map[int]chan Event
	nextSub int
}

// NewStore writes session logs under baseDir/<sessionId>/events.jsonl.
func NewStore(baseDir string) *Store {
	return &Store{
		baseDir: baseDir,
		ring:    make(map[string][]Event),
		files:   make(map[string]*os.File),
		subs:    make(map[int]chan Event),
	}
}

// Append redacts, sequences, persists, rings, and publishes an event.
func (s *Store) Append(e Event) Event {
	s.mu.Lock()
	s.seq++
	e.Seq = s.seq
	e.Summary = Redact(e.Summary)
	e.Detail = Redact(e.Detail)

	ring := append(s.ring[e.SessionID], e)
	if len(ring) > memRingSize {
		ring = ring[len(ring)-memRingSize:]
	}
	s.ring[e.SessionID] = ring
	s.persistLocked(e)

	subs := make([]chan Event, 0, len(s.subs))
	for _, ch := range s.subs {
		subs = append(subs, ch)
	}
	s.mu.Unlock()

	for _, ch := range subs {
		select {
		case ch <- e:
		default: // drop for a slow subscriber rather than block the agent
		}
	}
	return e
}

// Recent returns up to n most-recent events for a session (newest last).
func (s *Store) Recent(sessionID string, n int) []Event {
	s.mu.Lock()
	defer s.mu.Unlock()
	ring := s.ring[sessionID]
	if n > 0 && len(ring) > n {
		ring = ring[len(ring)-n:]
	}
	return append([]Event(nil), ring...)
}

// All reads the full session log from disk (for replay of older events beyond
// the in-memory ring).
func (s *Store) All(sessionID string) ([]Event, error) {
	path := s.logPath(sessionID)
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()

	var events []Event
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 8*1024*1024)
	for sc.Scan() {
		var e Event
		if json.Unmarshal(sc.Bytes(), &e) == nil {
			events = append(events, e)
		}
	}
	return events, sc.Err()
}

// Subscribe returns a channel of future events plus an unsubscribe func.
func (s *Store) Subscribe() (<-chan Event, func()) {
	s.mu.Lock()
	defer s.mu.Unlock()
	id := s.nextSub
	s.nextSub++
	ch := make(chan Event, 256)
	s.subs[id] = ch
	return ch, func() {
		s.mu.Lock()
		delete(s.subs, id)
		close(ch)
		s.mu.Unlock()
	}
}

// Close flushes and closes all open session files.
func (s *Store) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, f := range s.files {
		f.Close()
	}
	s.files = make(map[string]*os.File)
}

func (s *Store) logPath(sessionID string) string {
	return filepath.Join(s.baseDir, safeKey(sessionID), "events.jsonl")
}

// persistLocked appends one JSON line. Caller holds s.mu.
func (s *Store) persistLocked(e Event) {
	f := s.files[e.SessionID]
	if f == nil {
		path := s.logPath(e.SessionID)
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			return
		}
		var err error
		f, err = os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
		if err != nil {
			return
		}
		s.files[e.SessionID] = f
	}
	if line, err := json.Marshal(e); err == nil {
		f.Write(append(line, '\n'))
	}
}

func safeKey(s string) string {
	out := make([]rune, 0, len(s))
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_':
			out = append(out, r)
		default:
			out = append(out, '_')
		}
	}
	if len(out) == 0 {
		return "session"
	}
	return string(out)
}
