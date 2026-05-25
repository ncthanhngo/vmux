package approval

import (
	"sync"

	"github.com/vmux/sidecar/internal/id"
)

// PendingItem is an action awaiting a human decision.
type PendingItem struct {
	ID       string
	Action   Action
	Rule     Rule
	decision chan bool
}

// PendingView is the UI-facing snapshot of a pending item (no channel).
type PendingView struct {
	ID        string   `json:"id"`
	Tool      string   `json:"tool"`
	Command   string   `json:"command"`
	Args      []string `json:"args"`
	Workspace string   `json:"workspace"`
	RuleID    string   `json:"ruleId"`
	Reason    string   `json:"reason"`
}

// Queue holds pending approvals and notifies the UI when the set changes.
type Queue struct {
	mu       sync.Mutex
	items    map[string]*PendingItem
	onChange func()
}

func NewQueue() *Queue {
	return &Queue{items: make(map[string]*PendingItem)}
}

// SetOnChange registers a callback fired whenever the pending set changes.
func (q *Queue) SetOnChange(fn func()) { q.onChange = fn }

// Add enqueues an action and returns the item (whose decision channel the
// caller waits on).
func (q *Queue) Add(a Action, r Rule) *PendingItem {
	item := &PendingItem{ID: id.New(), Action: a, Rule: r, decision: make(chan bool, 1)}
	q.mu.Lock()
	q.items[item.ID] = item
	q.mu.Unlock()
	q.notify()
	return item
}

// Decide resolves a pending item; returns false if the id is unknown.
func (q *Queue) Decide(itemID string, allow bool) bool {
	q.mu.Lock()
	item := q.items[itemID]
	delete(q.items, itemID)
	q.mu.Unlock()
	if item == nil {
		return false
	}
	item.decision <- allow
	q.notify()
	return true
}

// remove drops an item without a decision (e.g. caller cancelled).
func (q *Queue) remove(itemID string) {
	q.mu.Lock()
	delete(q.items, itemID)
	q.mu.Unlock()
	q.notify()
}

// List returns UI-facing snapshots of pending items.
func (q *Queue) List() []PendingView {
	q.mu.Lock()
	defer q.mu.Unlock()
	out := make([]PendingView, 0, len(q.items))
	for _, it := range q.items {
		out = append(out, PendingView{
			ID: it.ID, Tool: it.Action.Tool, Command: it.Action.Command,
			Args: it.Action.Args, Workspace: it.Action.Workspace,
			RuleID: it.Rule.ID, Reason: it.Rule.Reason,
		})
	}
	return out
}

func (q *Queue) notify() {
	if q.onChange != nil {
		q.onChange()
	}
}
