package replay

import (
	"testing"
	"time"

	"github.com/vmux/sidecar/internal/activity"
)

// fakeSource serves a fixed event list.
type fakeSource struct{ events []activity.Event }

func (f fakeSource) All(string) ([]activity.Event, error) { return f.events, nil }

func TestTimelineMoments(t *testing.T) {
	base := time.Now()
	src := fakeSource{events: []activity.Event{
		{Ts: base, Kind: activity.KindToolCall},
		{Ts: base.Add(1 * time.Second), Kind: activity.KindCommand, Risk: activity.RiskHigh},
		{Ts: base.Add(20 * time.Second), Kind: activity.KindScreenshot},
	}}
	r := New(src)
	start, end, moments, err := r.Timeline("s", 5)
	if err != nil {
		t.Fatal(err)
	}
	if !start.Equal(base) || !end.Equal(base.Add(20*time.Second)) {
		t.Errorf("bounds wrong: %v..%v", start, end)
	}
	// Bucket 0 has the first two events; a later bucket has the screenshot.
	if len(moments) < 2 {
		t.Fatalf("expected >=2 moments, got %d", len(moments))
	}
	if moments[0].Count != 2 || moments[0].MaxRisk != "high" {
		t.Errorf("first moment = %+v, want count 2 risk high", moments[0])
	}
}

func TestSnapshotAt(t *testing.T) {
	base := time.Now()
	src := fakeSource{events: []activity.Event{
		{Ts: base, Kind: activity.KindScreenshot, Refs: map[string]string{"shotId": "a"}},
		{Ts: base.Add(5 * time.Second), Kind: activity.KindCommand, Summary: "npm test"},
		{Ts: base.Add(8 * time.Second), Kind: activity.KindScreenshot, Refs: map[string]string{"shotId": "b"}},
		{Ts: base.Add(40 * time.Second), Kind: activity.KindToolCall, Summary: "future"},
	}}
	r := New(src)

	// At t=10s: latest shot is "b"; command within 30s window present; future
	// tool call (t=40s) excluded.
	snap, err := r.At("s", base.Add(10*time.Second), 30)
	if err != nil {
		t.Fatal(err)
	}
	if snap.LatestShot["shotId"] != "b" {
		t.Errorf("latest shot = %v, want b", snap.LatestShot)
	}
	if len(snap.CommandsRun) != 1 || snap.CommandsRun[0] != "npm test" {
		t.Errorf("commands = %v", snap.CommandsRun)
	}
	if snap.LastToolCall == "future" {
		t.Error("event after t must be excluded")
	}
}
