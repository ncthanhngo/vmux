package editorintegration

import (
	"path/filepath"
	"reflect"
	"testing"
)

func TestLoadRegistry(t *testing.T) {
	editors, err := LoadRegistry()
	if err != nil {
		t.Fatalf("LoadRegistry: %v", err)
	}
	if len(editors) == 0 {
		t.Fatal("registry empty")
	}
	if _, ok := Find("cursor"); !ok {
		t.Error("cursor entry missing")
	}
	if _, ok := Find("fallback-open"); !ok {
		t.Error("fallback entry missing")
	}
}

func TestBuildArgsSubstitution(t *testing.T) {
	e, _ := Find("vscode")
	got := BuildArgs(e, "/work/src/app.ts", 142, 7)
	want := []string{"-g", "/work/src/app.ts:142:7"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("BuildArgs = %v, want %v", got, want)
	}
}

func TestBuildArgsPathWithSpacesStaysOneArg(t *testing.T) {
	e, _ := Find("vscode")
	got := BuildArgs(e, "/my work/a b.ts", 1, 0)
	// The path (with spaces) must remain a single argument — no shell splitting.
	want := []string{"-g", "/my work/a b.ts:1:0"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("BuildArgs split a spaced path: %v", got)
	}
}

func TestBuildArgsNvimTemplate(t *testing.T) {
	e, _ := Find("nvim-terminal")
	got := BuildArgs(e, "/p/f.go", 50, 0)
	want := []string{"+50", "/p/f.go"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("nvim args = %v, want %v", got, want)
	}
}

func TestDetectRuns(t *testing.T) {
	// Environment-dependent; just ensure it doesn't panic and "open" resolves.
	_ = Detect()
}

func TestPrefsRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "preferences.json")
	p := LoadPrefs(path)
	if err := p.Set("", "cursor"); err != nil { // global
		t.Fatal(err)
	}
	if err := p.Set("ws1", "vscode"); err != nil { // override
		t.Fatal(err)
	}
	// Reload from disk.
	p2 := LoadPrefs(path)
	if p2.EditorFor("ws1") != "vscode" {
		t.Errorf("workspace override not persisted: %q", p2.EditorFor("ws1"))
	}
	if p2.EditorFor("other") != "cursor" {
		t.Errorf("global default not persisted: %q", p2.EditorFor("other"))
	}
}
