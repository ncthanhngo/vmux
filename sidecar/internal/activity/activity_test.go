package activity

import (
	"strings"
	"testing"
	"time"
)

func TestRedact(t *testing.T) {
	cases := []struct {
		in       string
		mustHide string
		mustKeep string
	}{
		{`export AWS_SECRET_ACCESS_KEY=wJalrXUtnFEMIK7MDENGbPxRfiCYEXAMPLEKEY`, "wJalrXUtnFEMIK", "AWS_SECRET"},
		{`Authorization: Bearer abcdef1234567890XYZ`, "abcdef1234567890", "Authorization"},
		{`token=ghp_ABCDEFGHIJKLMNOPQRSTUVWXYZ012345`, "ghp_ABCDEFGHIJ", "token"},
		{`AKIAIOSFODNN7EXAMPLE in config`, "AKIAIOSFODNN7EXAMPLE", "config"},
	}
	for _, c := range cases {
		out := Redact(c.in)
		if strings.Contains(out, c.mustHide) {
			t.Errorf("Redact(%q) leaked %q: %q", c.in, c.mustHide, out)
		}
		if c.mustKeep != "" && !strings.Contains(out, c.mustKeep) {
			t.Errorf("Redact(%q) dropped context %q: %q", c.in, c.mustKeep, out)
		}
	}
}

func TestRedactLeavesCleanText(t *testing.T) {
	clean := "git commit -m 'fix bug in parser'"
	if got := Redact(clean); got != clean {
		t.Errorf("Redact mangled clean text: %q", got)
	}
}

func TestStoreAppendRingAndPersist(t *testing.T) {
	store := NewStore(t.TempDir())
	defer store.Close()

	for i := 0; i < 3; i++ {
		store.Append(Event{Ts: time.Now(), SessionID: "s1", Kind: KindToolCall, Summary: "call"})
	}
	recent := store.Recent("s1", 10)
	if len(recent) != 3 {
		t.Fatalf("Recent = %d, want 3", len(recent))
	}
	if recent[0].Seq != 1 || recent[2].Seq != 3 {
		t.Errorf("sequence numbers wrong: %d..%d", recent[0].Seq, recent[2].Seq)
	}

	// Persisted to disk and reloadable.
	all, err := store.All("s1")
	if err != nil {
		t.Fatalf("All: %v", err)
	}
	if len(all) != 3 {
		t.Errorf("disk reload = %d, want 3", len(all))
	}
}

func TestStorePersistRedacts(t *testing.T) {
	store := NewStore(t.TempDir())
	defer store.Close()
	store.Append(Event{Ts: time.Now(), SessionID: "s1", Kind: KindCommand, Summary: "run token=ghp_ABCDEFGHIJKLMNOPQRSTUV012345"})
	all, _ := store.All("s1")
	if len(all) != 1 || strings.Contains(all[0].Summary, "ghp_ABCDEFGHIJ") {
		t.Errorf("persisted event not redacted: %+v", all)
	}
}

func TestStoreSubscribe(t *testing.T) {
	store := NewStore(t.TempDir())
	defer store.Close()
	ch, unsub := store.Subscribe()
	defer unsub()

	store.Append(Event{Ts: time.Now(), SessionID: "s1", Kind: KindToolCall, Summary: "x"})
	select {
	case e := <-ch:
		if e.SessionID != "s1" {
			t.Errorf("got %+v", e)
		}
	case <-time.After(time.Second):
		t.Fatal("subscriber did not receive event")
	}
}
