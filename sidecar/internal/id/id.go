// Package id generates short, collision-resistant identifiers for sessions and
// workspaces without pulling in a UUID dependency.
package id

import (
	"crypto/rand"
	"encoding/hex"
)

// New returns a 128-bit random identifier as a 32-char hex string.
func New() string {
	var b [16]byte
	// crypto/rand.Read never returns a short read or error on macOS/Linux.
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}
