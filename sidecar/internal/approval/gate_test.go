package approval

import (
	"context"
	"testing"
	"time"
)

func TestRulesFlagDangerousCommands(t *testing.T) {
	rs := DefaultRuleSet()
	danger := [][]string{
		{"rm", "-rf", "node_modules"},
		{"sudo", "rm", "x"},
		{"git", "push", "origin", "main", "--force"},
		{"dd", "if=/dev/zero", "of=/dev/disk2"},
		{"chmod", "-R", "777", "/"},
	}
	for _, d := range danger {
		if _, ok := rs.FirstMatch(d[0], d[1:]); !ok {
			t.Errorf("expected %v to be flagged", d)
		}
	}
}

func TestRulesAllowNormalWorkflows(t *testing.T) {
	rs := DefaultRuleSet()
	safe := [][]string{
		{"npm", "install"},
		{"npm", "run", "build"},
		{"git", "push", "origin", "main"},
		{"git", "commit", "-m", "fix"},
		{"rm", "stale.txt"},
		{"go", "test", "./..."},
		{"ls", "-la"},
	}
	for _, s := range safe {
		if rule, ok := rs.FirstMatch(s[0], s[1:]); ok {
			t.Errorf("false positive on %v (rule %s)", s, rule.ID)
		}
	}
}

func TestGateWatchAllowsEverything(t *testing.T) {
	g := NewGate()
	ok, _ := g.Check(context.Background(), Action{Tool: "command_run", Command: "rm", Args: []string{"-rf", "/"}, Workspace: "w"})
	if !ok {
		t.Error("watch mode must allow (log only)")
	}
}

func TestGateSandboxDeniesWrites(t *testing.T) {
	g := NewGate()
	g.SetMode("w", ModeSandbox)
	if ok, _ := g.Check(context.Background(), Action{Tool: "command_run", Command: "ls", Workspace: "w"}); ok {
		t.Error("sandbox must deny command_run")
	}
	if ok, _ := g.Check(context.Background(), Action{Tool: "file_write", Workspace: "w"}); ok {
		t.Error("sandbox must deny file_write")
	}
}

func TestGateModeBlocksUntilApproval(t *testing.T) {
	g := NewGate()
	g.SetMode("w", ModeGate)

	type result struct {
		ok bool
	}
	done := make(chan result, 1)
	go func() {
		ok, _ := g.Check(context.Background(), Action{Tool: "command_run", Command: "rm", Args: []string{"-rf", "build"}, Workspace: "w"})
		done <- result{ok}
	}()

	// The action should be queued, not resolved yet.
	var itemID string
	for i := 0; i < 50; i++ {
		if list := g.Queue.List(); len(list) == 1 {
			itemID = list[0].ID
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if itemID == "" {
		t.Fatal("action was not queued for approval")
	}
	g.Queue.Decide(itemID, true)

	select {
	case r := <-done:
		if !r.ok {
			t.Error("approved action should proceed")
		}
	case <-time.After(time.Second):
		t.Fatal("Check did not return after approval")
	}
}

func TestGateModeAllowsSafeCommand(t *testing.T) {
	g := NewGate()
	g.SetMode("w", ModeGate)
	ok, _ := g.Check(context.Background(), Action{Tool: "command_run", Command: "npm", Args: []string{"test"}, Workspace: "w"})
	if !ok {
		t.Error("gate mode must allow safe commands without queuing")
	}
}
