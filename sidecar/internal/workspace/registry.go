package workspace

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/vmux/sidecar/internal/id"
)

// Notifier delivers server→client notifications (satisfied by *rpc.Server).
type Notifier interface {
	Notify(method string, params any)
}

// Workspace is an opened project directory and its current git state.
type Workspace struct {
	ID   string    `json:"id"`
	Path string    `json:"path"`
	Meta Meta      `json:"meta"`
	Git  GitStatus `json:"git"`

	watcher *watcher
}

type gitChangedParams struct {
	WorkspaceID string    `json:"workspaceId"`
	Status      GitStatus `json:"status"`
}

// Registry tracks opened workspaces and persists the id→path mapping so they
// reopen across sidecar restarts.
type Registry struct {
	notifier  Notifier
	storePath string

	mu     sync.Mutex
	byID   map[string]*Workspace
	byPath map[string]string
}

// NewRegistry loads the persisted registry and reopens its workspaces.
func NewRegistry(notifier Notifier, storePath string) (*Registry, error) {
	r := &Registry{
		notifier:  notifier,
		storePath: storePath,
		byID:      make(map[string]*Workspace),
		byPath:    make(map[string]string),
	}
	r.load()
	return r, nil
}

// Open registers path (or returns the existing workspace for an already-open
// path) and starts watching it for git changes.
func (r *Registry) Open(path string) (*Workspace, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	fi, err := os.Stat(abs)
	if err != nil {
		return nil, err
	}
	if !fi.IsDir() {
		return nil, fmt.Errorf("workspace: %q is not a directory", abs)
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	if existingID, ok := r.byPath[abs]; ok {
		return r.byID[existingID], nil
	}
	ws, err := r.openLocked(abs, "")
	if err != nil {
		return nil, err
	}
	r.persistLocked()
	return ws, nil
}

// openLocked does the real open work. Caller holds r.mu. fixedID reuses a
// persisted id; empty generates a new one.
func (r *Registry) openLocked(abs, fixedID string) (*Workspace, error) {
	meta, err := readOrCreateMeta(abs)
	if err != nil {
		return nil, err
	}
	st, err := ReadGitStatus(abs)
	if err != nil {
		return nil, err
	}
	wsID := fixedID
	if wsID == "" {
		wsID = id.New()
	}
	ws := &Workspace{ID: wsID, Path: abs, Meta: meta, Git: st}

	w, err := startWatcher(wsID, abs, r.onGitChange)
	if err != nil {
		return nil, err
	}
	ws.watcher = w

	r.byID[wsID] = ws
	r.byPath[abs] = wsID
	return ws, nil
}

// List returns a snapshot copy of the open workspaces.
func (r *Registry) List() []Workspace {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]Workspace, 0, len(r.byID))
	for _, ws := range r.byID {
		out = append(out, *ws)
	}
	return out
}

// Close stops watching and forgets the workspace.
func (r *Registry) Close(wsID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	ws, ok := r.byID[wsID]
	if !ok {
		return fmt.Errorf("workspace: no workspace %q", wsID)
	}
	ws.watcher.stop()
	delete(r.byID, wsID)
	delete(r.byPath, ws.Path)
	r.persistLocked()
	return nil
}

// Shutdown stops all watchers without altering persisted state.
func (r *Registry) Shutdown() {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, ws := range r.byID {
		ws.watcher.stop()
	}
}

func (r *Registry) onGitChange(wsID string, st GitStatus) {
	r.mu.Lock()
	if ws, ok := r.byID[wsID]; ok {
		ws.Git = st
	}
	r.mu.Unlock()
	if r.notifier != nil {
		r.notifier.Notify("workspace.gitChanged", gitChangedParams{WorkspaceID: wsID, Status: st})
	}
}

type persistedEntry struct {
	ID   string `json:"id"`
	Path string `json:"path"`
}

func (r *Registry) load() {
	data, err := os.ReadFile(r.storePath)
	if err != nil {
		return // no persisted registry yet
	}
	var entries []persistedEntry
	if json.Unmarshal(data, &entries) != nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, e := range entries {
		if _, err := os.Stat(e.Path); err != nil {
			continue // workspace dir gone since last run
		}
		if _, dup := r.byPath[e.Path]; dup {
			continue
		}
		_, _ = r.openLocked(e.Path, e.ID)
	}
}

// persistLocked writes the id→path registry. Caller holds r.mu.
func (r *Registry) persistLocked() {
	entries := make([]persistedEntry, 0, len(r.byID))
	for _, ws := range r.byID {
		entries = append(entries, persistedEntry{ID: ws.ID, Path: ws.Path})
	}
	out, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return
	}
	if err := os.MkdirAll(filepath.Dir(r.storePath), 0o700); err != nil {
		return
	}
	_ = os.WriteFile(r.storePath, out, 0o600)
}
