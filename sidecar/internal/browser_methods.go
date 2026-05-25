package internal

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os/exec"

	"github.com/vmux/sidecar/internal/rpc"
)

// browserList reports the browser sessions that currently hold screenshots,
// with their latest shot id.
func (s *Service) browserList(_ context.Context, _ json.RawMessage) (any, error) {
	type sessionInfo struct {
		SessionID  string `json:"sessionId"`
		ShotCount  int    `json:"shotCount"`
		LatestShot string `json:"latestShot,omitempty"`
	}
	var out []sessionInfo
	for _, sid := range s.Shots.Sessions() {
		info := sessionInfo{SessionID: sid, ShotCount: len(s.Shots.List(sid))}
		if shot, ok := s.Shots.Latest(sid); ok {
			info.LatestShot = shot.ID
		}
		out = append(out, info)
	}
	return map[string]any{"sessions": out}, nil
}

type latestShotParams struct {
	SessionID string `json:"sessionId"`
}

// browserLatestShot returns the most recent screenshot for a session as base64 PNG.
func (s *Service) browserLatestShot(_ context.Context, raw json.RawMessage) (any, error) {
	var p latestShotParams
	if err := decode(raw, &p); err != nil {
		return nil, err
	}
	shot, ok := s.Shots.Latest(p.SessionID)
	if !ok {
		return nil, &rpc.Error{Code: rpc.CodeInvalidParams, Message: "no screenshots for session"}
	}
	data, err := s.Shots.ReadFull(p.SessionID, shot.ID)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"shotId": shot.ID,
		"width":  shot.Width,
		"height": shot.Height,
		"png":    base64.StdEncoding.EncodeToString(data),
	}, nil
}

type browserSessionParams struct {
	SessionID string `json:"sessionId"`
}

// browserClose clears a session's screenshots (and stops its Chrome instance if
// one is tracked under that workspace id).
func (s *Service) browserClose(_ context.Context, raw json.RawMessage) (any, error) {
	var p browserSessionParams
	if err := decode(raw, &p); err != nil {
		return nil, err
	}
	s.Shots.Clear(p.SessionID)
	if s.Browser != nil {
		s.Browser.Stop(p.SessionID)
	}
	return struct{}{}, nil
}

// browserFocus brings the external browser window to the front. This is a
// UI-only action — agents cannot invoke it (defense against focus stealing is
// enforced by it not being an MCP tool).
func (s *Service) browserFocus(_ context.Context, _ json.RawMessage) (any, error) {
	if s.Browser == nil {
		return nil, &rpc.Error{Code: rpc.CodeInternal, Message: "no browser detected"}
	}
	name := s.Browser.Browser().Name
	if name == "" {
		return nil, &rpc.Error{Code: rpc.CodeInternal, Message: "no browser detected"}
	}
	script := fmt.Sprintf("tell application %q to activate", name)
	if err := exec.Command("osascript", "-e", script).Run(); err != nil {
		return nil, err
	}
	return struct{}{}, nil
}
