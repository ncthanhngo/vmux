package workspace

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func gitInit(t *testing.T, dir string) {
	t.Helper()
	for _, args := range [][]string{
		{"init"},
		{"config", "user.email", "test@vmux.local"},
		{"config", "user.name", "vmux test"},
	} {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v (%s)", args, err, out)
		}
	}
}

func TestReadGitStatus(t *testing.T) {
	dir := t.TempDir()

	st, err := ReadGitStatus(dir)
	if err != nil {
		t.Fatalf("ReadGitStatus non-repo: %v", err)
	}
	if st.IsRepo {
		t.Error("expected IsRepo=false for a non-repo dir")
	}

	gitInit(t, dir)
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("hi"), 0o644); err != nil {
		t.Fatal(err)
	}
	st, err = ReadGitStatus(dir)
	if err != nil {
		t.Fatalf("ReadGitStatus repo: %v", err)
	}
	if !st.IsRepo || !st.Dirty || len(st.Changes) == 0 {
		t.Errorf("expected dirty repo with changes, got %+v", st)
	}
}

type chanNotifier struct{ ch chan gitChangedParams }

func (n *chanNotifier) Notify(method string, params any) {
	if method == "workspace.gitChanged" {
		select {
		case n.ch <- params.(gitChangedParams):
		default:
		}
	}
}

func TestOpenCreatesMetaAndPersists(t *testing.T) {
	dir := t.TempDir()
	store := filepath.Join(t.TempDir(), "workspaces.json")

	reg, err := NewRegistry(nil, store)
	if err != nil {
		t.Fatalf("NewRegistry: %v", err)
	}
	ws, err := reg.Open(dir)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, ".vmux", "workspace.json")); err != nil {
		t.Errorf("meta file not created: %v", err)
	}
	if ws.Meta.Name != filepath.Base(dir) {
		t.Errorf("meta name = %q, want %q", ws.Meta.Name, filepath.Base(dir))
	}

	// Opening the same path again returns the same workspace (deduped).
	ws2, _ := reg.Open(dir)
	if ws2.ID != ws.ID {
		t.Errorf("re-open created a new id: %s vs %s", ws2.ID, ws.ID)
	}
	reg.Shutdown()

	// A fresh registry reloads the persisted workspace with the same id.
	reg2, err := NewRegistry(nil, store)
	if err != nil {
		t.Fatalf("reload NewRegistry: %v", err)
	}
	defer reg2.Shutdown()
	list := reg2.List()
	if len(list) != 1 || list[0].ID != ws.ID {
		t.Errorf("persisted reload mismatch: %+v (want id %s)", list, ws.ID)
	}
}

func TestGitChangedNotification(t *testing.T) {
	dir := t.TempDir()
	gitInit(t, dir)
	store := filepath.Join(t.TempDir(), "workspaces.json")

	notifier := &chanNotifier{ch: make(chan gitChangedParams, 1)}
	reg, err := NewRegistry(notifier, store)
	if err != nil {
		t.Fatalf("NewRegistry: %v", err)
	}
	defer reg.Shutdown()
	ws, err := reg.Open(dir)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	// Touch a file; the debounced watcher should emit gitChanged.
	if err := os.WriteFile(filepath.Join(dir, "new.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	select {
	case got := <-notifier.ch:
		if got.WorkspaceID != ws.ID {
			t.Errorf("gitChanged for %s, want %s", got.WorkspaceID, ws.ID)
		}
		if !got.Status.Dirty {
			t.Error("expected dirty status in notification")
		}
	case <-time.After(4 * time.Second):
		t.Fatal("timed out waiting for workspace.gitChanged")
	}

	if err := reg.Close(ws.ID); err != nil {
		t.Errorf("Close: %v", err)
	}
	if len(reg.List()) != 0 {
		t.Error("workspace remained after Close")
	}
}
