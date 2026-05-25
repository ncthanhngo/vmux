package browsersession

import (
	"regexp"
	"strconv"
	"sync"
)

// portPattern matches the host:port forms dev servers print, e.g.
// "localhost:3000", "127.0.0.1:5173", "http://localhost:8080/". It captures the
// port digits.
var portPattern = regexp.MustCompile(`(?:https?://)?(?:localhost|127\.0\.0\.1|0\.0\.0\.0)[:/]+(\d{2,5})`)

// PortDetector scans PTY output for locally-served ports and reports each port
// once per session until Reset (so a single "listening on :3000" line doesn't
// spam the UI as it scrolls).
type PortDetector struct {
	mu   sync.Mutex
	seen map[string]map[int]bool // sessionId -> set of reported ports
}

func NewPortDetector() *PortDetector {
	return &PortDetector{seen: make(map[string]map[int]bool)}
}

// Scan returns ports newly discovered in line for the given session (empty if
// none new). Ports already reported for the session are suppressed.
func (d *PortDetector) Scan(sessionID, line string) []int {
	matches := portPattern.FindAllStringSubmatch(line, -1)
	if len(matches) == 0 {
		return nil
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	reported := d.seen[sessionID]
	if reported == nil {
		reported = make(map[int]bool)
		d.seen[sessionID] = reported
	}
	var fresh []int
	for _, m := range matches {
		port, err := strconv.Atoi(m[1])
		if err != nil || port < 1 || port > 65535 {
			continue
		}
		if !reported[port] {
			reported[port] = true
			fresh = append(fresh, port)
		}
	}
	return fresh
}

// Reset forgets reported ports for a session (call when its process exits so a
// restart re-surfaces the port).
func (d *PortDetector) Reset(sessionID string) {
	d.mu.Lock()
	delete(d.seen, sessionID)
	d.mu.Unlock()
}
