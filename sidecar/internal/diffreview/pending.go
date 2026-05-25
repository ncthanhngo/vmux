package diffreview

import (
	"fmt"
	"os"
	"sync"
)

// Pending is a staged file change awaiting per-hunk review.
type Pending struct {
	Path     string
	oldText  string
	mode     os.FileMode
	hunks    []Hunk
	accepted map[int]bool
	decided  map[int]bool
}

// PendingView is the UI-facing snapshot.
type PendingView struct {
	Path  string `json:"path"`
	Hunks []Hunk `json:"hunks"`
}

// Store tracks staged file changes per absolute path and writes the resolved
// result to disk once every hunk is decided.
type Store struct {
	mu       sync.Mutex
	byPath   map[string]*Pending
	onChange func()
}

func NewStore() *Store {
	return &Store{byPath: make(map[string]*Pending)}
}

// SetOnChange registers a callback fired when the pending set changes.
func (s *Store) SetOnChange(fn func()) { s.onChange = fn }

// Stage computes hunks for a proposed change and records it for review. If
// there are no changes it is a no-op returning false.
func (s *Store) Stage(path, oldText, newText string) bool {
	hunks := ComputeHunks(oldText, newText)
	if len(hunks) == 0 {
		return false
	}
	mode := os.FileMode(0o644)
	if fi, err := os.Stat(path); err == nil {
		mode = fi.Mode().Perm()
	}
	s.mu.Lock()
	s.byPath[path] = &Pending{
		Path: path, oldText: oldText, mode: mode, hunks: hunks,
		accepted: map[int]bool{}, decided: map[int]bool{},
	}
	s.mu.Unlock()
	s.notify()
	return true
}

// List returns the pending reviews.
func (s *Store) List() []PendingView {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]PendingView, 0, len(s.byPath))
	for _, p := range s.byPath {
		out = append(out, PendingView{Path: p.Path, Hunks: p.hunks})
	}
	return out
}

// Decide records an accept/reject for one hunk. When all hunks are decided, the
// resolved text is written to disk and the pending entry cleared.
func (s *Store) Decide(path string, hunkID int, accept bool) error {
	return s.decideMany(path, map[int]bool{hunkID: accept}, false)
}

// AcceptAll / RejectAll decide every remaining hunk at once.
func (s *Store) AcceptAll(path string) error { return s.decideMany(path, nil, true) }
func (s *Store) RejectAll(path string) error { return s.decideMany(path, nil, false) }

func (s *Store) decideMany(path string, single map[int]bool, bulkValue bool) error {
	s.mu.Lock()
	p := s.byPath[path]
	if p == nil {
		s.mu.Unlock()
		return fmt.Errorf("diffreview: no pending review for %q", path)
	}
	if single == nil {
		for _, h := range p.hunks {
			if !p.decided[h.ID] {
				p.decided[h.ID] = true
				p.accepted[h.ID] = bulkValue
			}
		}
	} else {
		for id, v := range single {
			p.decided[id] = true
			p.accepted[id] = v
		}
	}

	resolved := len(p.decided) == len(p.hunks)
	var writeErr error
	if resolved {
		// Guard against a lost update: if the file changed on disk since we
		// staged it, our hunks are based on a stale base — abort rather than
		// clobber the newer content. The user can re-trigger the edit.
		if current, err := os.ReadFile(path); err == nil && string(current) != p.oldText {
			delete(s.byPath, path)
			s.mu.Unlock()
			s.notify()
			return fmt.Errorf("diffreview: %q changed on disk since staging; review discarded", path)
		}
		final := ApplyHunks(p.oldText, p.hunks, p.accepted)
		writeErr = os.WriteFile(path, []byte(final), p.mode)
		delete(s.byPath, path)
	}
	s.mu.Unlock()

	s.notify()
	return writeErr
}

func (s *Store) notify() {
	if s.onChange != nil {
		s.onChange()
	}
}
