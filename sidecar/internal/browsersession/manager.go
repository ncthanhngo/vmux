package browsersession

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"syscall"
)

// Browser is a detected Chromium-family browser executable.
type Browser struct {
	ID       string
	Name     string
	ExecPath string
}

// candidates lists supported browsers in preference order (Chrome first — it is
// the reference target for Chrome DevTools MCP).
var candidates = []Browser{
	{ID: "chrome", Name: "Google Chrome", ExecPath: "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome"},
	{ID: "brave", Name: "Brave Browser", ExecPath: "/Applications/Brave Browser.app/Contents/MacOS/Brave Browser"},
	{ID: "edge", Name: "Microsoft Edge", ExecPath: "/Applications/Microsoft Edge.app/Contents/MacOS/Microsoft Edge"},
	{ID: "arc", Name: "Arc", ExecPath: "/Applications/Arc.app/Contents/MacOS/Arc"},
	{ID: "chromium", Name: "Chromium", ExecPath: "/Applications/Chromium.app/Contents/MacOS/Chromium"},
}

// DetectBrowsers returns every supported browser found installed.
func DetectBrowsers() []Browser {
	var found []Browser
	for _, b := range candidates {
		if _, err := os.Stat(b.ExecPath); err == nil {
			found = append(found, b)
		}
	}
	return found
}

// DetectBrowser returns the most-preferred installed browser.
func DetectBrowser() (Browser, error) {
	if found := DetectBrowsers(); len(found) > 0 {
		return found[0], nil
	}
	return Browser{}, fmt.Errorf("no supported browser found (install Chrome, or Brave/Edge/Arc/Chromium)")
}

// LaunchArgs builds the Chrome command-line for an isolated automation session.
// Port 0 lets Chrome pick a free DevTools port (read back from DevTalkActivePort).
func LaunchArgs(profileDir string) []string {
	return []string{
		"--user-data-dir=" + profileDir,
		"--remote-debugging-port=0",
		"--no-first-run",
		"--no-default-browser-check",
		"--disable-features=Translate",
	}
}

// Manager spawns and tracks one browser instance per workspace.
type Manager struct {
	browser Browser

	mu        sync.Mutex
	instances map[string]*exec.Cmd
}

// NewManager detects a browser and prepares the manager. The error is advisory:
// callers may still construct sessions later once a browser is installed.
func NewManager() (*Manager, error) {
	m := &Manager{instances: make(map[string]*exec.Cmd)}
	b, err := DetectBrowser()
	if err != nil {
		return m, err
	}
	m.browser = b
	return m, nil
}

// Browser returns the detected browser (zero value if none).
func (m *Manager) Browser() Browser { return m.browser }

// EnsureRunning launches the browser for a workspace if not already running.
func (m *Manager) EnsureRunning(workspaceID, workspacePath string) error {
	if m.browser.ExecPath == "" {
		return fmt.Errorf("no browser detected")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if cmd, ok := m.instances[workspaceID]; ok && cmd.ProcessState == nil {
		return nil // already running
	}
	profileDir, err := EnsureProfileDir(workspacePath)
	if err != nil {
		return err
	}
	cmd := exec.Command(m.browser.ExecPath, LaunchArgs(profileDir)...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("launch browser: %w", err)
	}
	m.instances[workspaceID] = cmd
	return nil
}

// Stop terminates the workspace's browser instance.
func (m *Manager) Stop(workspaceID string) {
	m.mu.Lock()
	cmd := m.instances[workspaceID]
	delete(m.instances, workspaceID)
	m.mu.Unlock()
	stop(cmd)
}

// StopAll terminates every tracked browser instance.
func (m *Manager) StopAll() {
	m.mu.Lock()
	all := make([]*exec.Cmd, 0, len(m.instances))
	for _, cmd := range m.instances {
		all = append(all, cmd)
	}
	m.instances = make(map[string]*exec.Cmd)
	m.mu.Unlock()
	for _, cmd := range all {
		stop(cmd)
	}
}

func stop(cmd *exec.Cmd) {
	if cmd == nil || cmd.Process == nil {
		return
	}
	_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGTERM)
	_ = cmd.Wait()
}

// DevToolsPortFile is the file Chrome writes its chosen debugging port into.
func DevToolsPortFile(profileDir string) string {
	return filepath.Join(profileDir, "DevToolsActivePort")
}
