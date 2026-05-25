package mcp

import (
	_ "embed"
	"encoding/json"
	"fmt"
)

// registryJSON is the embedded copy of the known-MCP-server catalog. The
// human-facing canonical lives at shared/mcp-server-registry.json; keep the two
// in sync (this copy is embedded so the sidecar works inside the app bundle).
//
//go:embed registry_data.json
var registryJSON []byte

// ServerEntry is one known MCP server and how to launch it.
type ServerEntry struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	Capabilities []string `json:"capabilities"`
	Runtime      string   `json:"runtime"` // "node" or "python"
	Package      string   `json:"package"`
	Version      string   `json:"version"`
	Command      string   `json:"command"`
	Args         []string `json:"args"`
	Recommended  bool     `json:"recommended"`
}

// Registry is the catalog of installable MCP servers.
type Registry struct {
	Version int           `json:"version"`
	Servers []ServerEntry `json:"servers"`
}

// LoadRegistry parses the embedded server catalog.
func LoadRegistry() (*Registry, error) {
	var r Registry
	if err := json.Unmarshal(registryJSON, &r); err != nil {
		return nil, fmt.Errorf("parse mcp registry: %w", err)
	}
	return &r, nil
}

// Find returns the entry with the given id.
func (r *Registry) Find(id string) (ServerEntry, bool) {
	for _, e := range r.Servers {
		if e.ID == id {
			return e, true
		}
	}
	return ServerEntry{}, false
}

// Recommended returns the entries flagged as recommended defaults.
func (r *Registry) Recommended() []ServerEntry {
	var out []ServerEntry
	for _, e := range r.Servers {
		if e.Recommended {
			out = append(out, e)
		}
	}
	return out
}
