// Package replay reconstructs an agent session over time from its activity log:
// a timeline (with density "moments" for the scrubber) and at-time-t state
// snapshots (latest screenshot, files touched, commands run, console output).
package replay

import (
	"sort"
	"time"

	"github.com/vmux/sidecar/internal/activity"
)

// EventSource provides a session's full ordered event list (the activity store).
type EventSource interface {
	All(sessionID string) ([]activity.Event, error)
}

// Replay answers timeline/snapshot queries over the activity log.
type Replay struct {
	source EventSource
}

func New(source EventSource) *Replay {
	return &Replay{source: source}
}

// Moment groups events in a short window for the scrubber heatmap.
type Moment struct {
	T        time.Time `json:"t"`
	Count    int       `json:"count"`
	Kind     string    `json:"kind"` // dominant kind in the window
	MaxRisk  string    `json:"maxRisk"`
}

// Timeline returns the session bounds and density moments (bucketed by
// bucketSeconds) for rendering the scrubber.
func (r *Replay) Timeline(sessionID string, bucketSeconds int) (start, end time.Time, moments []Moment, err error) {
	events, err := r.source.All(sessionID)
	if err != nil || len(events) == 0 {
		return time.Time{}, time.Time{}, nil, err
	}
	if bucketSeconds <= 0 {
		bucketSeconds = 5
	}
	start = events[0].Ts
	end = events[len(events)-1].Ts
	bucket := time.Duration(bucketSeconds) * time.Second

	byBucket := map[int64]*Moment{}
	var order []int64
	for _, e := range events {
		key := e.Ts.Sub(start) / bucket
		m := byBucket[int64(key)]
		if m == nil {
			m = &Moment{T: start.Add(time.Duration(key) * bucket)}
			byBucket[int64(key)] = m
			order = append(order, int64(key))
		}
		m.Count++
		m.Kind = string(e.Kind)
		if riskRank(string(e.Risk)) > riskRank(m.MaxRisk) {
			m.MaxRisk = string(e.Risk)
		}
	}
	sort.Slice(order, func(i, j int) bool { return order[i] < order[j] })
	for _, k := range order {
		moments = append(moments, *byBucket[k])
	}
	return start, end, moments, nil
}

// Snapshot is the reconstructed state at time t.
type Snapshot struct {
	T              time.Time         `json:"t"`
	LatestShot     map[string]string `json:"latestShot,omitempty"`     // refs of the most recent screenshot event ≤ t
	LastToolCall   string            `json:"lastToolCall,omitempty"`   // summary
	FilesTouched   []string          `json:"filesTouched,omitempty"`   // within the window
	CommandsRun    []string          `json:"commandsRun,omitempty"`    // within the window
	ConsoleErrors  []string          `json:"consoleErrors,omitempty"`  // within the window
}

// At reconstructs session state at time t: the most recent screenshot at or
// before t, plus files/commands/console within windowSeconds before t.
func (r *Replay) At(sessionID string, t time.Time, windowSeconds int) (Snapshot, error) {
	events, err := r.source.All(sessionID)
	if err != nil {
		return Snapshot{}, err
	}
	if windowSeconds <= 0 {
		windowSeconds = 30
	}
	windowStart := t.Add(-time.Duration(windowSeconds) * time.Second)
	snap := Snapshot{T: t}

	for _, e := range events {
		if e.Ts.After(t) {
			break
		}
		switch e.Kind {
		case activity.KindScreenshot:
			snap.LatestShot = e.Refs
		case activity.KindToolCall:
			snap.LastToolCall = e.Summary
		}
		if e.Ts.Before(windowStart) {
			continue
		}
		switch e.Kind {
		case activity.KindFileEdit:
			snap.FilesTouched = append(snap.FilesTouched, e.Summary)
		case activity.KindCommand:
			snap.CommandsRun = append(snap.CommandsRun, e.Summary)
		case activity.KindConsole:
			snap.ConsoleErrors = append(snap.ConsoleErrors, e.Summary)
		}
	}
	return snap, nil
}

func riskRank(r string) int {
	switch r {
	case "high":
		return 3
	case "medium":
		return 2
	case "low":
		return 1
	default:
		return 0
	}
}
