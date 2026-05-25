package browsersession

import (
	"reflect"
	"testing"
)

func TestScanDetectsCommonDevServerOutput(t *testing.T) {
	cases := map[string]int{
		"  ➜  Local:   http://localhost:5173/":             5173, // Vite
		"ready - started server on 0.0.0.0:3000":           3000, // Next.js
		"Server listening at http://127.0.0.1:8080":        8080, // Fastify
		"Example app listening on port http://localhost:4000": 4000, // Express
	}
	for line, want := range cases {
		d := NewPortDetector()
		got := d.Scan("s1", line)
		if len(got) != 1 || got[0] != want {
			t.Errorf("Scan(%q) = %v, want [%d]", line, got, want)
		}
	}
}

func TestScanDedupesPerSession(t *testing.T) {
	d := NewPortDetector()
	if got := d.Scan("s1", "listening on localhost:3000"); !reflect.DeepEqual(got, []int{3000}) {
		t.Fatalf("first scan = %v, want [3000]", got)
	}
	if got := d.Scan("s1", "still on localhost:3000"); got != nil {
		t.Errorf("duplicate port should be suppressed, got %v", got)
	}
	// A different session reports independently.
	if got := d.Scan("s2", "localhost:3000"); !reflect.DeepEqual(got, []int{3000}) {
		t.Errorf("second session should report, got %v", got)
	}
}

func TestScanResetReSurfaces(t *testing.T) {
	d := NewPortDetector()
	d.Scan("s1", "localhost:3000")
	d.Reset("s1")
	if got := d.Scan("s1", "localhost:3000"); !reflect.DeepEqual(got, []int{3000}) {
		t.Errorf("after reset port should re-surface, got %v", got)
	}
}

func TestScanIgnoresNonLocal(t *testing.T) {
	d := NewPortDetector()
	if got := d.Scan("s1", "fetching https://api.example.com:443/v1"); got != nil {
		t.Errorf("remote host should not match, got %v", got)
	}
}
