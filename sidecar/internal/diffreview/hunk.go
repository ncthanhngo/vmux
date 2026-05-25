// Package diffreview computes line-level diffs between a file's known-clean
// content and an agent's proposed content, groups changes into hunks, and lets
// the user accept or reject each hunk independently before it touches the file.
package diffreview

import "strings"

// Hunk is a contiguous change: old lines [OldStart,OldStart+len(OldLines)) are
// replaced by NewLines. Pure insertions have empty OldLines; pure deletions
// have empty NewLines.
type Hunk struct {
	ID       int      `json:"id"`
	OldStart int      `json:"oldStart"`
	OldLines []string `json:"oldLines"`
	NewLines []string `json:"newLines"`
}

// ComputeHunks diffs old→new line-by-line and returns the change hunks.
func ComputeHunks(oldText, newText string) []Hunk {
	o := splitLines(oldText)
	n := splitLines(newText)

	// Trim common prefix/suffix so the LCS only runs on the changed middle.
	pre := commonPrefix(o, n)
	suf := commonSuffix(o[pre:], n[pre:])
	oMid := o[pre : len(o)-suf]
	nMid := n[pre : len(n)-suf]

	ops := diffMiddle(oMid, nMid)

	// Group consecutive del/ins ops into hunks, offset by the prefix length.
	var hunks []Hunk
	id := 0
	oi := pre // index into old
	i := 0
	for i < len(ops) {
		if ops[i].kind == opEqual {
			oi++
			i++
			continue
		}
		start := oi
		var oldL, newL []string
		for i < len(ops) && ops[i].kind != opEqual {
			switch ops[i].kind {
			case opDel:
				oldL = append(oldL, ops[i].line)
				oi++
			case opIns:
				newL = append(newL, ops[i].line)
			}
			i++
		}
		// Skip no-op hunks (can arise from a trailing-newline-only difference).
		if len(oldL) == 0 && len(newL) == 0 {
			continue
		}
		hunks = append(hunks, Hunk{ID: id, OldStart: start, OldLines: oldL, NewLines: newL})
		id++
	}
	return hunks
}

// ApplyHunks rebuilds the file text: accepted hunks contribute their NewLines,
// rejected hunks keep their OldLines. accepted maps hunk ID → accept.
func ApplyHunks(oldText string, hunks []Hunk, accepted map[int]bool) string {
	o := splitLines(oldText)
	var out []string
	cursor := 0
	for _, h := range hunks {
		// Emit unchanged lines before this hunk.
		for cursor < h.OldStart && cursor < len(o) {
			out = append(out, o[cursor])
			cursor++
		}
		if accepted[h.ID] {
			out = append(out, h.NewLines...)
		} else {
			out = append(out, h.OldLines...)
		}
		cursor += len(h.OldLines)
	}
	for cursor < len(o) {
		out = append(out, o[cursor])
		cursor++
	}
	return strings.Join(out, "\n")
}

// --- line diff internals ---

type opKind int

const (
	opEqual opKind = iota
	opDel
	opIns
)

type op struct {
	kind opKind
	line string
}

// diffMiddle returns edit ops via an LCS table. Inputs are the trimmed middles,
// usually small for a typical agent edit.
func diffMiddle(o, n []string) []op {
	lcs := lcsTable(o, n)
	var ops []op
	i, j := 0, 0
	for i < len(o) && j < len(n) {
		if o[i] == n[j] {
			ops = append(ops, op{opEqual, o[i]})
			i++
			j++
		} else if lcs[i+1][j] >= lcs[i][j+1] {
			ops = append(ops, op{opDel, o[i]})
			i++
		} else {
			ops = append(ops, op{opIns, n[j]})
			j++
		}
	}
	for ; i < len(o); i++ {
		ops = append(ops, op{opDel, o[i]})
	}
	for ; j < len(n); j++ {
		ops = append(ops, op{opIns, n[j]})
	}
	return ops
}

func lcsTable(o, n []string) [][]int {
	t := make([][]int, len(o)+1)
	for i := range t {
		t[i] = make([]int, len(n)+1)
	}
	for i := len(o) - 1; i >= 0; i-- {
		for j := len(n) - 1; j >= 0; j-- {
			if o[i] == n[j] {
				t[i][j] = t[i+1][j+1] + 1
			} else if t[i+1][j] >= t[i][j+1] {
				t[i][j] = t[i+1][j]
			} else {
				t[i][j] = t[i][j+1]
			}
		}
	}
	return t
}

func splitLines(s string) []string {
	if s == "" {
		return nil
	}
	return strings.Split(s, "\n")
}

func commonPrefix(a, b []string) int {
	i := 0
	for i < len(a) && i < len(b) && a[i] == b[i] {
		i++
	}
	return i
}

func commonSuffix(a, b []string) int {
	i := 0
	for i < len(a) && i < len(b) && a[len(a)-1-i] == b[len(b)-1-i] {
		i++
	}
	return i
}
