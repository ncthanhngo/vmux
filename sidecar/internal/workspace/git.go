package workspace

import (
	"bufio"
	"bytes"
	"context"
	"os/exec"
	"strings"
	"time"
)

// GitChange is one entry from `git status --porcelain`.
type GitChange struct {
	Status string `json:"status"` // two-char XY code, e.g. " M", "??"
	Path   string `json:"path"`
}

// GitStatus summarizes a workspace's git state.
type GitStatus struct {
	IsRepo  bool        `json:"isRepo"`
	Branch  string      `json:"branch"`
	Dirty   bool        `json:"dirty"`
	Changes []GitChange `json:"changes"`
}

// ReadGitStatus runs `git status` in dir. A non-repo returns {IsRepo:false}
// without error; only execution failures return an error.
func ReadGitStatus(dir string) (GitStatus, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", "status", "--porcelain=v1", "--branch")
	cmd.Dir = dir
	var out, stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		// Outside a work tree git exits non-zero; treat as "not a repo".
		if strings.Contains(stderr.String(), "not a git repository") {
			return GitStatus{IsRepo: false}, nil
		}
		return GitStatus{}, err
	}
	return parseStatus(out.Bytes()), nil
}

func parseStatus(out []byte) GitStatus {
	st := GitStatus{IsRepo: true, Changes: []GitChange{}}
	sc := bufio.NewScanner(bytes.NewReader(out))
	for sc.Scan() {
		line := sc.Text()
		if strings.HasPrefix(line, "## ") {
			st.Branch = parseBranch(line[3:])
			continue
		}
		if len(line) < 4 {
			continue
		}
		st.Changes = append(st.Changes, GitChange{
			Status: line[:2],
			Path:   line[3:],
		})
	}
	st.Dirty = len(st.Changes) > 0
	return st
}

// parseBranch extracts the local branch from a porcelain header like
// "main...origin/main [ahead 1]" or "No commits yet on main".
func parseBranch(header string) string {
	if i := strings.Index(header, "..."); i >= 0 {
		return header[:i]
	}
	if i := strings.Index(header, " "); i >= 0 {
		return header[:i]
	}
	return header
}
