// Package approval gates dangerous agent actions. Each workspace runs in one of
// three modes; rules decide which actions are dangerous; a queue holds actions
// awaiting a human decision.
package approval

// Mode controls how strictly actions are gated.
type Mode string

const (
	// ModeWatch logs everything, blocks nothing.
	ModeWatch Mode = "watch"
	// ModeGate blocks rule-matched dangerous actions pending approval.
	ModeGate Mode = "gate"
	// ModeSandbox blocks all writes/execs regardless of rules.
	ModeSandbox Mode = "sandbox"
)

// DefaultMode is applied to a workspace until the user changes it.
const DefaultMode = ModeWatch

func (m Mode) valid() bool {
	switch m {
	case ModeWatch, ModeGate, ModeSandbox:
		return true
	}
	return false
}

// Action describes an attempted operation to gate.
type Action struct {
	Tool      string   // e.g. "command_run", "file_write"
	Command   string   // for command_run: the program
	Args      []string // for command_run
	Workspace string
}

// Decision is the outcome of a gate check.
type Decision int

const (
	Allow Decision = iota
	Deny
	NeedApproval
)
