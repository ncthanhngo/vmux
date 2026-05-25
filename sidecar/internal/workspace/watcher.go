package workspace

import (
	"os"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
)

const debounceInterval = 500 * time.Millisecond

// watcher observes a workspace root (and its .git dir) and invokes onChange with
// a fresh git status, debounced so a burst of edits yields a single update.
type watcher struct {
	fsw  *fsnotify.Watcher
	done chan struct{}
}

func startWatcher(wsID, path string, onChange func(id string, st GitStatus)) (*watcher, error) {
	fsw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	if err := fsw.Add(path); err != nil {
		fsw.Close()
		return nil, err
	}
	// Watch .git too so commits/checkouts (index + HEAD changes) are noticed.
	gitDir := filepath.Join(path, ".git")
	if fi, statErr := os.Stat(gitDir); statErr == nil && fi.IsDir() {
		_ = fsw.Add(gitDir)
	}

	w := &watcher{fsw: fsw, done: make(chan struct{})}
	go w.loop(wsID, path, onChange)
	return w, nil
}

func (w *watcher) loop(wsID, path string, onChange func(string, GitStatus)) {
	debounce := time.NewTimer(time.Hour)
	debounce.Stop()
	defer debounce.Stop()

	for {
		select {
		case <-w.done:
			return
		case _, ok := <-w.fsw.Events:
			if !ok {
				return
			}
			if !debounce.Stop() {
				select {
				case <-debounce.C:
				default:
				}
			}
			debounce.Reset(debounceInterval)
		case <-debounce.C:
			if st, err := ReadGitStatus(path); err == nil {
				onChange(wsID, st)
			}
		case <-w.fsw.Errors:
			// Transient watcher errors are non-fatal; keep watching.
		}
	}
}

func (w *watcher) stop() {
	close(w.done)
	w.fsw.Close()
}
