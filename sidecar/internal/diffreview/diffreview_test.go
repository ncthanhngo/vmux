package diffreview

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestComputeHunksSingleEdit(t *testing.T) {
	old := "a\nb\nc\nd"
	neu := "a\nB\nc\nd"
	hunks := ComputeHunks(old, neu)
	if len(hunks) != 1 {
		t.Fatalf("expected 1 hunk, got %d: %+v", len(hunks), hunks)
	}
	h := hunks[0]
	if h.OldStart != 1 || len(h.OldLines) != 1 || h.OldLines[0] != "b" || h.NewLines[0] != "B" {
		t.Errorf("hunk = %+v", h)
	}
}

func TestComputeHunksMultiEdit(t *testing.T) {
	old := "1\n2\n3\n4\n5\n6\n7"
	neu := "1\nX\n3\n4\n5\nY\n7"
	hunks := ComputeHunks(old, neu)
	if len(hunks) != 2 {
		t.Fatalf("expected 2 hunks, got %d: %+v", len(hunks), hunks)
	}
}

func TestComputeHunksInsertAndDelete(t *testing.T) {
	if h := ComputeHunks("a\nc", "a\nb\nc"); len(h) != 1 || len(h[0].OldLines) != 0 || len(h[0].NewLines) != 1 {
		t.Errorf("insert hunk wrong: %+v", h)
	}
	if h := ComputeHunks("a\nb\nc", "a\nc"); len(h) != 1 || len(h[0].OldLines) != 1 || len(h[0].NewLines) != 0 {
		t.Errorf("delete hunk wrong: %+v", h)
	}
}

func TestApplyHunksAcceptSubset(t *testing.T) {
	old := "1\n2\n3\n4\n5\n6\n7"
	neu := "1\nX\n3\n4\n5\nY\n7"
	hunks := ComputeHunks(old, neu)

	// Accept only the first hunk.
	got := ApplyHunks(old, hunks, map[int]bool{0: true, 1: false})
	want := "1\nX\n3\n4\n5\n6\n7"
	if got != want {
		t.Errorf("partial apply = %q, want %q", got, want)
	}
	// Accept both → equals new.
	if got := ApplyHunks(old, hunks, map[int]bool{0: true, 1: true}); got != neu {
		t.Errorf("accept-all = %q, want %q", got, neu)
	}
	// Reject both → equals old.
	if got := ApplyHunks(old, hunks, map[int]bool{}); got != old {
		t.Errorf("reject-all = %q, want %q", got, old)
	}
}

func TestStageAndResolveWritesFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "f.txt")
	os.WriteFile(path, []byte("a\nb\nc"), 0o644)

	store := NewStore()
	changed := 0
	store.SetOnChange(func() { changed++ })

	if !store.Stage(path, "a\nb\nc", "a\nB\nc") {
		t.Fatal("Stage should report a change")
	}
	if len(store.List()) != 1 {
		t.Fatalf("expected 1 pending, got %d", len(store.List()))
	}
	// Accept the only hunk → file is written, pending cleared.
	if err := store.AcceptAll(path); err != nil {
		t.Fatalf("AcceptAll: %v", err)
	}
	if len(store.List()) != 0 {
		t.Error("pending not cleared after resolve")
	}
	data, _ := os.ReadFile(path)
	if strings.TrimSpace(string(data)) != "a\nB\nc" {
		t.Errorf("file = %q, want accepted content", data)
	}
	if changed == 0 {
		t.Error("onChange never fired")
	}
}

func TestRejectAllKeepsOriginal(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "f.txt")
	os.WriteFile(path, []byte("orig"), 0o644)
	store := NewStore()
	store.Stage(path, "orig", "changed")
	if err := store.RejectAll(path); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(path)
	if string(data) != "orig" {
		t.Errorf("reject should keep original, got %q", data)
	}
}
