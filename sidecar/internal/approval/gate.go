package approval

import (
	"context"
	"fmt"
	"sync"
)

// Gate decides whether an action may proceed, per the workspace's mode and the
// rule set. In gate mode a dangerous action blocks until a human decides.
type Gate struct {
	rules *RuleSet
	Queue *Queue

	mu    sync.RWMutex
	modes map[string]Mode
}

func NewGate() *Gate {
	return &Gate{rules: DefaultRuleSet(), Queue: NewQueue(), modes: make(map[string]Mode)}
}

// Mode returns the workspace's mode (DefaultMode if unset).
func (g *Gate) Mode(workspace string) Mode {
	g.mu.RLock()
	defer g.mu.RUnlock()
	if m, ok := g.modes[workspace]; ok {
		return m
	}
	return DefaultMode
}

// SetMode updates a workspace's mode.
func (g *Gate) SetMode(workspace string, m Mode) error {
	if !m.valid() {
		return fmt.Errorf("invalid mode %q", m)
	}
	g.mu.Lock()
	g.modes[workspace] = m
	g.mu.Unlock()
	return nil
}

// Evaluate returns the decision for an action without blocking.
func (g *Gate) Evaluate(a Action) (Decision, Rule) {
	switch g.Mode(a.Workspace) {
	case ModeSandbox:
		if a.Tool == "command_run" || a.Tool == "file_write" {
			return Deny, Rule{ID: "sandbox", Reason: "sandbox mode blocks writes and commands"}
		}
		return Allow, Rule{}
	case ModeGate:
		if a.Tool == "command_run" {
			if rule, ok := g.rules.FirstMatch(a.Command, a.Args); ok {
				return NeedApproval, rule
			}
		}
		return Allow, Rule{}
	default: // watch
		return Allow, Rule{}
	}
}

// Check evaluates an action and, in gate mode, blocks until the user approves or
// denies (or ctx is cancelled). Returns whether the action may proceed.
func (g *Gate) Check(ctx context.Context, a Action) (bool, string) {
	decision, rule := g.Evaluate(a)
	switch decision {
	case Allow:
		return true, ""
	case Deny:
		return false, rule.Reason
	default: // NeedApproval
		item := g.Queue.Add(a, rule)
		select {
		case allow := <-item.decision:
			if allow {
				return true, ""
			}
			return false, "denied: " + rule.Reason
		case <-ctx.Done():
			g.Queue.remove(item.ID)
			return false, "approval cancelled"
		}
	}
}
