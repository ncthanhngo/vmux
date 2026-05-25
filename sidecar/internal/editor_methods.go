package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/vmux/sidecar/internal/editorintegration"
	"github.com/vmux/sidecar/internal/rpc"
)

func (s *Service) editorDetected(_ context.Context, _ json.RawMessage) (any, error) {
	return map[string]any{"editors": editorintegration.Detect()}, nil
}

type editorInvokeParams struct {
	WorkspaceID string `json:"workspaceId"`
	Path        string `json:"path"` // workspace-relative
	Line        int    `json:"line"`
	Col         int    `json:"col"`
	EditorID    string `json:"editorId,omitempty"` // optional override
}

// editorInvoke opens the chosen editor at a workspace file/line. The path is
// resolved + contained within the workspace root server-side; the editor is
// launched with an argument array (never a shell string).
func (s *Service) editorInvoke(_ context.Context, raw json.RawMessage) (any, error) {
	var p editorInvokeParams
	if err := decode(raw, &p); err != nil {
		return nil, err
	}
	ws, ok := s.Workspaces.Get(p.WorkspaceID)
	if !ok {
		return nil, &rpc.Error{Code: rpc.CodeInvalidParams, Message: "unknown workspace"}
	}
	abs, err := containWorkspacePath(ws.Path, p.Path)
	if err != nil {
		return nil, &rpc.Error{Code: rpc.CodeInvalidParams, Message: err.Error()}
	}

	editorID := p.EditorID
	if editorID == "" {
		editorID = s.EditorPrefs.EditorFor(p.WorkspaceID)
	}

	detected := editorintegration.Detect()
	chosen, bin := pickEditor(detected, editorID)
	if bin == "" {
		return nil, &rpc.Error{Code: rpc.CodeInternal, Message: "no editor available"}
	}
	if err := editorintegration.Invoke(chosen.Editor, bin, abs, p.Line, p.Col); err != nil {
		return nil, &rpc.Error{Code: rpc.CodeInternal, Message: err.Error()}
	}
	return struct{}{}, nil
}

// pickEditor returns the detected editor matching id, else the first detected.
func pickEditor(detected []editorintegration.DetectedEditor, id string) (editorintegration.DetectedEditor, string) {
	for _, d := range detected {
		if d.ID == id {
			return d, d.ResolvedBin
		}
	}
	if len(detected) > 0 {
		return detected[0], detected[0].ResolvedBin
	}
	return editorintegration.DetectedEditor{}, ""
}

type editorPreferenceParams struct {
	WorkspaceID string `json:"workspaceId"` // empty = global default
	EditorID    string `json:"editorId"`
}

func (s *Service) editorPreference(_ context.Context, raw json.RawMessage) (any, error) {
	var p editorPreferenceParams
	if err := decode(raw, &p); err != nil {
		return nil, err
	}
	if err := s.EditorPrefs.Set(p.WorkspaceID, p.EditorID); err != nil {
		return nil, &rpc.Error{Code: rpc.CodeInternal, Message: err.Error()}
	}
	return struct{}{}, nil
}

// containWorkspacePath joins rel under root and rejects escapes.
func containWorkspacePath(root, rel string) (string, error) {
	abs := filepath.Join(root, rel)
	realRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		realRoot = root
	}
	realAbs := abs
	if resolved, rerr := filepath.EvalSymlinks(abs); rerr == nil {
		realAbs = resolved
	}
	r, err := filepath.Rel(realRoot, realAbs)
	if err != nil || r == ".." || strings.HasPrefix(r, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("path escapes workspace")
	}
	return abs, nil
}
