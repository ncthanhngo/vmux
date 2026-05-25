package internal

import (
	"context"
	"encoding/json"
	"time"

	"github.com/vmux/sidecar/internal/activity"
	"github.com/vmux/sidecar/internal/approval"
	"github.com/vmux/sidecar/internal/rpc"
)

// --- Activity ---

type activityRecentParams struct {
	SessionID string `json:"sessionId"`
	Limit     int    `json:"limit"`
}

func (s *Service) activityRecent(_ context.Context, raw json.RawMessage) (any, error) {
	var p activityRecentParams
	if err := decode(raw, &p); err != nil {
		return nil, err
	}
	if p.Limit <= 0 {
		p.Limit = 200
	}
	return map[string]any{"events": s.Activity.Recent(p.SessionID, p.Limit)}, nil
}

// --- Approval ---

func (s *Service) approvalList(_ context.Context, _ json.RawMessage) (any, error) {
	return map[string]any{"pending": redactPending(s.Gate.Queue.List())}, nil
}

// redactPending scrubs secrets from the command/args shown to the approver,
// masking only secret substrings so the command shape stays reviewable.
func redactPending(items []approval.PendingView) []approval.PendingView {
	out := make([]approval.PendingView, len(items))
	for i, it := range items {
		it.Command = activity.Redact(it.Command)
		args := make([]string, len(it.Args))
		for j, a := range it.Args {
			args[j] = activity.Redact(a)
		}
		it.Args = args
		out[i] = it
	}
	return out
}

type approvalDecideParams struct {
	ID    string `json:"id"`
	Allow bool   `json:"allow"`
}

func (s *Service) approvalDecide(_ context.Context, raw json.RawMessage) (any, error) {
	var p approvalDecideParams
	if err := decode(raw, &p); err != nil {
		return nil, err
	}
	if !s.Gate.Queue.Decide(p.ID, p.Allow) {
		return nil, &rpc.Error{Code: rpc.CodeInvalidParams, Message: "unknown approval id"}
	}
	return struct{}{}, nil
}

type setModeParams struct {
	WorkspaceID string `json:"workspaceId"`
	Mode        string `json:"mode"`
}

func (s *Service) approvalSetMode(_ context.Context, raw json.RawMessage) (any, error) {
	var p setModeParams
	if err := decode(raw, &p); err != nil {
		return nil, err
	}
	if err := s.Gate.SetMode(p.WorkspaceID, approval.Mode(p.Mode)); err != nil {
		return nil, &rpc.Error{Code: rpc.CodeInvalidParams, Message: err.Error()}
	}
	return struct{}{}, nil
}

type modeParams struct {
	WorkspaceID string `json:"workspaceId"`
}

func (s *Service) approvalMode(_ context.Context, raw json.RawMessage) (any, error) {
	var p modeParams
	if err := decode(raw, &p); err != nil {
		return nil, err
	}
	return map[string]any{"mode": string(s.Gate.Mode(p.WorkspaceID))}, nil
}

// --- Replay ---

type replayTimelineParams struct {
	SessionID     string `json:"sessionId"`
	BucketSeconds int    `json:"bucketSeconds"`
}

func (s *Service) replayTimeline(_ context.Context, raw json.RawMessage) (any, error) {
	var p replayTimelineParams
	if err := decode(raw, &p); err != nil {
		return nil, err
	}
	start, end, moments, err := s.Replay.Timeline(p.SessionID, p.BucketSeconds)
	if err != nil {
		return nil, err
	}
	return map[string]any{"start": start, "end": end, "moments": moments}, nil
}

type replayAtParams struct {
	SessionID     string    `json:"sessionId"`
	T             time.Time `json:"t"`
	WindowSeconds int       `json:"windowSeconds"`
}

func (s *Service) replayAt(_ context.Context, raw json.RawMessage) (any, error) {
	var p replayAtParams
	if err := decode(raw, &p); err != nil {
		return nil, err
	}
	snap, err := s.Replay.At(p.SessionID, p.T, p.WindowSeconds)
	if err != nil {
		return nil, err
	}
	return snap, nil
}
