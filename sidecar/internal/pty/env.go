package pty

import (
	"os"
	"strings"
)

// buildEnv prepares the child environment. A terminal/agent generally needs the
// user's real environment (PATH, HOME, API keys, …), so when the caller passes
// none we inherit the sidecar's. We always guarantee TERM so curses apps render.
func buildEnv(requested []string) []string {
	env := requested
	if len(env) == 0 {
		env = os.Environ()
	}
	if !hasKey(env, "TERM") {
		env = append(env, "TERM=xterm-256color")
	}
	return env
}

func hasKey(env []string, key string) bool {
	prefix := key + "="
	for _, kv := range env {
		if strings.HasPrefix(kv, prefix) {
			return true
		}
	}
	return false
}
