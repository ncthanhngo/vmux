package internal

import (
	"context"
	"encoding/json"

	"github.com/vmux/sidecar/internal/rpc"
)

func (s *Service) diffReviewList(_ context.Context, _ json.RawMessage) (any, error) {
	return map[string]any{"pending": s.Diff.List()}, nil
}

type diffDecideParams struct {
	Path   string `json:"path"`
	HunkID int    `json:"hunkId"`
	Accept bool   `json:"accept"`
}

func (s *Service) diffReviewDecide(_ context.Context, raw json.RawMessage) (any, error) {
	var p diffDecideParams
	if err := decode(raw, &p); err != nil {
		return nil, err
	}
	if err := s.Diff.Decide(p.Path, p.HunkID, p.Accept); err != nil {
		return nil, &rpc.Error{Code: rpc.CodeInvalidParams, Message: err.Error()}
	}
	return struct{}{}, nil
}

type diffPathParams struct {
	Path string `json:"path"`
}

func (s *Service) diffReviewAcceptAll(_ context.Context, raw json.RawMessage) (any, error) {
	var p diffPathParams
	if err := decode(raw, &p); err != nil {
		return nil, err
	}
	if err := s.Diff.AcceptAll(p.Path); err != nil {
		return nil, &rpc.Error{Code: rpc.CodeInvalidParams, Message: err.Error()}
	}
	return struct{}{}, nil
}

func (s *Service) diffReviewRejectAll(_ context.Context, raw json.RawMessage) (any, error) {
	var p diffPathParams
	if err := decode(raw, &p); err != nil {
		return nil, err
	}
	if err := s.Diff.RejectAll(p.Path); err != nil {
		return nil, &rpc.Error{Code: rpc.CodeInvalidParams, Message: err.Error()}
	}
	return struct{}{}, nil
}
