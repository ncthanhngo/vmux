package mcp

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// AutoConfig manages an opt-in edit to an agent's MCP config so the agent routes
// through the vmux proxy (via vmux-mcp-bridge). It never writes without an
// explicit Apply call, and always backs up the original first.
type AutoConfig struct {
	ConfigPath string // e.g. ~/.claude.json
	BridgePath string // absolute path to vmux-mcp-bridge
}

const vmuxServerKey = "vmux"

// DiffPreview returns the current and proposed config as pretty JSON, so the UI
// can show the user exactly what would change before they approve.
func (a *AutoConfig) DiffPreview() (before string, after string, err error) {
	cfg, err := a.load()
	if err != nil {
		return "", "", err
	}
	before = prettyJSON(cfg)
	a.applyEntry(cfg)
	after = prettyJSON(cfg)
	return before, after, nil
}

// Apply backs up the existing config (if any) and atomically writes the merged
// config adding the vmux MCP server entry. Returns the backup path (empty if
// there was no prior file).
func (a *AutoConfig) Apply() (backupPath string, err error) {
	cfg, err := a.load()
	if err != nil {
		return "", err
	}
	if _, statErr := os.Stat(a.ConfigPath); statErr == nil {
		backupPath = fmt.Sprintf("%s.vmux-backup-%d", a.ConfigPath, time.Now().Unix())
		orig, readErr := os.ReadFile(a.ConfigPath)
		if readErr != nil {
			return "", readErr
		}
		if err := os.WriteFile(backupPath, orig, 0o600); err != nil {
			return "", err
		}
	}

	a.applyEntry(cfg)
	if err := a.atomicWrite(cfg); err != nil {
		return "", err
	}
	return backupPath, nil
}

// load reads the config as a generic object, or {} if absent.
func (a *AutoConfig) load() (map[string]any, error) {
	data, err := os.ReadFile(a.ConfigPath)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]any{}, nil
		}
		return nil, err
	}
	var cfg map[string]any
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse %s: %w", a.ConfigPath, err)
	}
	if cfg == nil {
		cfg = map[string]any{}
	}
	return cfg, nil
}

// applyEntry inserts/updates the vmux entry under "mcpServers".
func (a *AutoConfig) applyEntry(cfg map[string]any) {
	servers, _ := cfg["mcpServers"].(map[string]any)
	if servers == nil {
		servers = map[string]any{}
		cfg["mcpServers"] = servers
	}
	servers[vmuxServerKey] = map[string]any{
		"command": a.BridgePath,
		"args":    []string{},
	}
}

func (a *AutoConfig) atomicWrite(cfg map[string]any) error {
	out, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(a.ConfigPath), 0o755); err != nil {
		return err
	}
	tmp := a.ConfigPath + ".vmux-tmp"
	if err := os.WriteFile(tmp, out, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, a.ConfigPath) // atomic on the same filesystem
}

func prettyJSON(v any) string {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return ""
	}
	return string(b)
}
