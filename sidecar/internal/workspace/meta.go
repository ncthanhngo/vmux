// Package workspace tracks per-project state: a registry of opened workspaces,
// their .vmux/workspace.json metadata, and git status streamed on change.
package workspace

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// Meta is the git-shareable per-workspace config at <path>/.vmux/workspace.json.
// It must never contain secrets (decision: workspace.json is committed).
type Meta struct {
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"createdAt"`
}

func metaPath(wsPath string) string {
	return filepath.Join(wsPath, ".vmux", "workspace.json")
}

// readOrCreateMeta loads <path>/.vmux/workspace.json, creating a default one
// (named after the directory) if absent.
func readOrCreateMeta(wsPath string) (Meta, error) {
	p := metaPath(wsPath)
	data, err := os.ReadFile(p)
	if err == nil {
		var m Meta
		if jerr := json.Unmarshal(data, &m); jerr != nil {
			return Meta{}, jerr
		}
		return m, nil
	}
	if !os.IsNotExist(err) {
		return Meta{}, err
	}

	m := Meta{Name: filepath.Base(wsPath), CreatedAt: time.Now().UTC()}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return Meta{}, err
	}
	out, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return Meta{}, err
	}
	if err := os.WriteFile(p, out, 0o644); err != nil {
		return Meta{}, err
	}
	return m, nil
}
