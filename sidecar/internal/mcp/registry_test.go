package mcp

import (
	"strings"
	"testing"
)

func TestLoadRegistry(t *testing.T) {
	r, err := LoadRegistry()
	if err != nil {
		t.Fatalf("LoadRegistry: %v", err)
	}
	if len(r.Servers) == 0 {
		t.Fatal("registry is empty")
	}
	if _, ok := r.Find("chrome-devtools"); !ok {
		t.Error("chrome-devtools entry missing")
	}
	rec := r.Recommended()
	if len(rec) == 0 {
		t.Error("expected at least one recommended server")
	}
}

func TestLaunchConfigNpx(t *testing.T) {
	r, _ := LoadRegistry()
	entry, _ := r.Find("chrome-devtools")

	inst := NewInstaller("/tmp/vmux-mcp-servers-test")
	cfg, err := inst.LaunchConfig(entry)
	if err != nil {
		// npx not installed in this environment — skip rather than fail.
		t.Skipf("npx unavailable: %v", err)
	}
	if !strings.HasSuffix(cfg.Command, "npx") {
		t.Errorf("command = %q, want npx", cfg.Command)
	}
	if len(cfg.Args) == 0 || cfg.Args[len(cfg.Args)-1] != "chrome-devtools-mcp@0.6.0" {
		t.Errorf("args = %v, want pinned package", cfg.Args)
	}
	var hasCache bool
	for _, e := range cfg.Env {
		if strings.HasPrefix(e, "npm_config_cache=") {
			hasCache = true
		}
	}
	if !hasCache {
		t.Error("launch env missing isolated npm_config_cache")
	}
}

func TestLaunchConfigUnsupportedCommand(t *testing.T) {
	inst := NewInstaller("/tmp/x")
	_, err := inst.LaunchConfig(ServerEntry{Name: "weird", Command: "deno"})
	if err == nil {
		t.Error("expected error for unsupported command")
	}
}

func TestDetectRuntimesRuns(t *testing.T) {
	// Just ensure detection returns without panicking; values are env-dependent.
	_ = DetectRuntimes()
}
