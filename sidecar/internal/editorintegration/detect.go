package editorintegration

import (
	"os"
	"os/exec"
	"path/filepath"
)

// DetectedEditor is a registry editor found installed on this machine.
type DetectedEditor struct {
	Editor
	ResolvedBin string `json:"resolvedBin"`
}

// Detect returns the registry editors whose binary is found on PATH or in a
// known /Applications bundle.
func Detect() []DetectedEditor {
	editors, err := LoadRegistry()
	if err != nil {
		return nil
	}
	var found []DetectedEditor
	for _, e := range editors {
		if bin := resolveBinary(e.Bin); bin != "" {
			found = append(found, DetectedEditor{Editor: e, ResolvedBin: bin})
		}
	}
	return found
}

// resolveBinary finds an editor's launcher on PATH, falling back to common
// app-bundle locations for CLIs that aren't symlinked into PATH (e.g. Cursor).
func resolveBinary(bin string) string {
	if p, err := exec.LookPath(bin); err == nil {
		return p
	}
	for _, candidate := range bundleCandidates[bin] {
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	return ""
}

// bundleCandidates maps a CLI name to app-bundle binary paths to probe when it
// isn't on PATH.
var bundleCandidates = map[string][]string{
	"code":   {"/Applications/Visual Studio Code.app/Contents/Resources/app/bin/code"},
	"cursor": {"/Applications/Cursor.app/Contents/Resources/app/bin/cursor"},
	"zed":    {"/Applications/Zed.app/Contents/MacOS/cli"},
	"subl":   {"/Applications/Sublime Text.app/Contents/SharedSupport/bin/subl"},
	"mvim":   {"/Applications/MacVim.app/Contents/bin/mvim"},
}

func init() {
	// "open" is always available on macOS.
	if _, err := exec.LookPath("open"); err != nil {
		bundleCandidates["open"] = []string{filepath.Join("/usr/bin", "open")}
	}
}
