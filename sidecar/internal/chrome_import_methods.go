package internal

import (
	"context"
	"encoding/json"
	"path/filepath"
	"strings"

	"github.com/vmux/sidecar/internal/browsersession"
	"github.com/vmux/sidecar/internal/chromeimport"
	"github.com/vmux/sidecar/internal/rpc"
)

// chromeImportScan lists the user's Chrome profiles and which data they hold.
func (s *Service) chromeImportScan(_ context.Context, _ json.RawMessage) (any, error) {
	root, err := chromeimport.ChromeRoot()
	if err != nil {
		return nil, err
	}
	profiles, err := chromeimport.Scan(root)
	if err != nil {
		return nil, &rpc.Error{Code: rpc.CodeInternal, Message: "scan Chrome profiles: " + err.Error()}
	}
	return map[string]any{"profiles": profiles}, nil
}

type importBookmarksParams struct {
	WorkspaceID string `json:"workspaceId"`
	ProfileDir  string `json:"profileDir"` // e.g. "Default" or "Profile 1"
}

// chromeImportBookmarks merges a Chrome profile's bookmarks into the workspace's
// browser profile. Paths are resolved server-side — the target from the trusted
// workspace registry and the source from the Chrome root + a single dir
// component — so a client cannot direct reads/writes outside those roots.
// (Cookie write-back + history merge are deferred — they need a SQLite writer.)
func (s *Service) chromeImportBookmarks(_ context.Context, raw json.RawMessage) (any, error) {
	var p importBookmarksParams
	if err := decode(raw, &p); err != nil {
		return nil, err
	}
	ws, ok := s.Workspaces.Get(p.WorkspaceID)
	if !ok {
		return nil, &rpc.Error{Code: rpc.CodeInvalidParams, Message: "unknown workspace"}
	}
	// ProfileDir must be a single path component (no traversal).
	if p.ProfileDir == "" || strings.ContainsRune(p.ProfileDir, filepath.Separator) || strings.Contains(p.ProfileDir, "..") {
		return nil, &rpc.Error{Code: rpc.CodeInvalidParams, Message: "invalid profile dir"}
	}
	root, err := chromeimport.ChromeRoot()
	if err != nil {
		return nil, err
	}
	sourcePath := filepath.Join(root, p.ProfileDir)
	targetProfile := browsersession.ProfileDir(ws.Path)
	count, err := chromeimport.MergeBookmarks(sourcePath, targetProfile)
	if err != nil {
		return nil, &rpc.Error{Code: rpc.CodeInternal, Message: "merge bookmarks: " + err.Error()}
	}
	return map[string]any{"imported": count}, nil
}
