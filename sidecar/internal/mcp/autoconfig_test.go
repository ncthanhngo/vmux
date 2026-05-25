package mcp

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAutoConfigAppliesAndBacksUp(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "claude.json")
	existing := `{"mcpServers":{"other":{"command":"foo"}},"theme":"dark"}`
	if err := os.WriteFile(cfgPath, []byte(existing), 0o600); err != nil {
		t.Fatal(err)
	}

	ac := &AutoConfig{ConfigPath: cfgPath, BridgePath: "/usr/local/bin/vmux-mcp-bridge"}

	before, after, err := ac.DiffPreview()
	if err != nil {
		t.Fatalf("DiffPreview: %v", err)
	}
	if strings.Contains(before, "vmux-mcp-bridge") {
		t.Error("before-preview should not yet contain vmux entry")
	}
	if !strings.Contains(after, "vmux-mcp-bridge") {
		t.Error("after-preview should contain vmux entry")
	}

	backup, err := ac.Apply()
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if backup == "" {
		t.Fatal("expected a backup path for an existing config")
	}
	if got, _ := os.ReadFile(backup); string(got) != existing {
		t.Errorf("backup content mismatch: %s", got)
	}

	var cfg map[string]any
	data, _ := os.ReadFile(cfgPath)
	json.Unmarshal(data, &cfg)
	servers := cfg["mcpServers"].(map[string]any)
	if _, ok := servers["vmux"]; !ok {
		t.Error("vmux entry not written")
	}
	if _, ok := servers["other"]; !ok {
		t.Error("existing entry was clobbered")
	}
	if cfg["theme"] != "dark" {
		t.Error("unrelated keys were lost")
	}
}

func TestAutoConfigNoExistingFile(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "nested", "claude.json")
	ac := &AutoConfig{ConfigPath: cfgPath, BridgePath: "/bin/bridge"}

	backup, err := ac.Apply()
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if backup != "" {
		t.Errorf("expected no backup for a fresh config, got %q", backup)
	}
	if _, err := os.Stat(cfgPath); err != nil {
		t.Errorf("config not created: %v", err)
	}
}
