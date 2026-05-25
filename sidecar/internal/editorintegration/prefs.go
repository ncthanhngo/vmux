package editorintegration

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

// Prefs stores the user's editor choice — a global default plus per-workspace
// overrides — persisted to preferences.json.
type Prefs struct {
	path string

	mu           sync.Mutex
	Global       string            `json:"global"`
	PerWorkspace map[string]string `json:"perWorkspace"`
}

// LoadPrefs reads preferences from path (or returns empty prefs if absent).
func LoadPrefs(path string) *Prefs {
	p := &Prefs{path: path, PerWorkspace: map[string]string{}}
	if data, err := os.ReadFile(path); err == nil {
		_ = json.Unmarshal(data, p)
		if p.PerWorkspace == nil {
			p.PerWorkspace = map[string]string{}
		}
	}
	return p
}

// EditorFor returns the preferred editor id for a workspace (per-workspace
// override, else global, else empty).
func (p *Prefs) EditorFor(workspaceID string) string {
	p.mu.Lock()
	defer p.mu.Unlock()
	if id := p.PerWorkspace[workspaceID]; id != "" {
		return id
	}
	return p.Global
}

// Set records a preference: empty workspaceID sets the global default.
func (p *Prefs) Set(workspaceID, editorID string) error {
	p.mu.Lock()
	if workspaceID == "" {
		p.Global = editorID
	} else {
		p.PerWorkspace[workspaceID] = editorID
	}
	p.mu.Unlock()
	return p.save()
}

func (p *Prefs) save() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	out, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p.path), 0o700); err != nil {
		return err
	}
	// Atomic write: a crash mid-write must not truncate preferences.json.
	tmp := p.path + ".tmp"
	if err := os.WriteFile(tmp, out, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, p.path)
}
