package editorintegration

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
)

// BuildArgs renders an editor's argument template into an argument ARRAY (never
// a shell string), substituting {path}/{line}/{col}. Splitting the template
// into tokens first means a path containing spaces stays a single argument —
// no shell, no injection.
func BuildArgs(e Editor, path string, line, col int) []string {
	repl := strings.NewReplacer(
		"{path}", path,
		"{line}", strconv.Itoa(line),
		"{col}", strconv.Itoa(col),
	)
	tokens := strings.Fields(e.Args)
	args := make([]string, 0, len(tokens))
	for _, tok := range tokens {
		args = append(args, repl.Replace(tok))
	}
	return args
}

// Invoke launches the editor at path:line:col without blocking. The editor runs
// in its own process group so it survives a sidecar restart.
func Invoke(e Editor, resolvedBin, path string, line, col int) error {
	if resolvedBin == "" {
		return fmt.Errorf("editor %q not found", e.Name)
	}
	cmd := exec.Command(resolvedBin, BuildArgs(e, path, line, col)...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("launch %s: %w", e.Name, err)
	}
	// Reap asynchronously so the child isn't left a zombie; we don't wait on it.
	go func() { _ = cmd.Wait() }()
	return nil
}
