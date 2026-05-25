// Package editorintegration detects the user's installed code editors and
// launches them at a specific file/line — vmux does not embed an editor; it
// hands off to the user's existing one (VS Code, Cursor, Neovim, …).
package editorintegration

import (
	_ "embed"
	"encoding/json"
	"fmt"
)

//go:embed editor-registry.json
var registryJSON []byte

// Editor is a known editor profile: how to launch it and whether it accepts
// a line/column.
type Editor struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Bin     string `json:"bin"`
	Args    string `json:"args"` // template with {path} {line} {col}
	LineCol bool   `json:"lineCol"`
}

// LoadRegistry parses the embedded editor catalog.
func LoadRegistry() ([]Editor, error) {
	var editors []Editor
	if err := json.Unmarshal(registryJSON, &editors); err != nil {
		return nil, fmt.Errorf("parse editor registry: %w", err)
	}
	return editors, nil
}

// Find returns the registry entry with the given id.
func Find(id string) (Editor, bool) {
	editors, err := LoadRegistry()
	if err != nil {
		return Editor{}, false
	}
	for _, e := range editors {
		if e.ID == id {
			return e, true
		}
	}
	return Editor{}, false
}
