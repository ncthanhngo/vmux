// Package internal wires the RPC server to the PTY and workspace subsystems and
// registers the vmux method namespace.
package internal

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"log/slog"

	"github.com/vmux/sidecar/internal/activity"
	"github.com/vmux/sidecar/internal/approval"
	"github.com/vmux/sidecar/internal/browsersession"
	"github.com/vmux/sidecar/internal/mcp"
	"github.com/vmux/sidecar/internal/pty"
	"github.com/vmux/sidecar/internal/replay"
	"github.com/vmux/sidecar/internal/rpc"
	"github.com/vmux/sidecar/internal/workspace"
)

// Service bundles the RPC server with its backing subsystems.
type Service struct {
	RPC        *rpc.Server
	PTY        *pty.Manager
	Workspaces *workspace.Registry
	MCP        *mcp.Proxy
	Browser    *browsersession.Manager
	Shots      *browsersession.ScreenshotStore
	Ports      *browsersession.PortDetector
	Activity   *activity.Store
	Gate       *approval.Gate
	Replay     *replay.Replay
}

// NewService builds the server, subsystems, and registers all methods.
func NewService(log *slog.Logger, workspaceStore, shotsDir, sessionsDir string) (*Service, error) {
	srv := rpc.NewServer(log)
	ptyMgr := pty.NewManager(srv)
	wsReg, err := workspace.NewRegistry(srv, workspaceStore)
	if err != nil {
		return nil, err
	}

	actStore := activity.NewStore(sessionsDir)
	logger := &activityBridge{store: actStore, rpc: srv}
	gate := approval.NewGate()
	gate.Queue.SetOnChange(func() {
		srv.Notify("approval.changed", map[string]any{"pending": redactPending(gate.Queue.List())})
	})

	native := mcp.NewNativeTools(wsReg, logger, gate)
	proxy := mcp.NewProxy(log, logger, native)

	// A missing browser is non-fatal: the wizard surfaces install options and
	// the rest of vmux works without browser automation.
	browser, berr := browsersession.NewManager()
	if berr != nil {
		log.Info("no browser detected for automation", "err", berr)
	}

	s := &Service{
		RPC: srv, PTY: ptyMgr, Workspaces: wsReg, MCP: proxy, Browser: browser,
		Shots:    browsersession.NewScreenshotStore(shotsDir),
		Ports:    browsersession.NewPortDetector(),
		Activity: actStore,
		Gate:     gate,
		Replay:   replay.New(actStore),
	}

	// Server-side port detection: surface localhost ports printed by dev servers.
	ptyMgr.SetOutputTap(func(sessionID string, data []byte) {
		for _, port := range s.Ports.Scan(sessionID, string(data)) {
			srv.Notify("browserSession.portDetected", map[string]any{"sessionId": sessionID, "port": port})
		}
	})

	// Persist browser screenshots returned via MCP, notify the UI, and record
	// an activity event so Session Replay can show what the agent saw.
	proxy.OnScreenshot = func(upstream string, png []byte) {
		shot, thumb, err := s.Shots.Add(upstream, png)
		if err != nil {
			log.Error("store screenshot", "upstream", upstream, "err", err)
			return
		}
		srv.Notify("browserSession.shotCaptured", map[string]any{
			"sessionId": upstream, "shotId": shot.ID, "thumbnail": thumb,
		})
		actStore.Append(activity.Event{
			Ts: shot.CapturedAt, SessionID: upstream, Kind: activity.KindScreenshot,
			Actor: upstream, Summary: "screenshot captured",
			Refs: map[string]string{"shotId": shot.ID, "sessionId": upstream},
		})
	}

	s.registerMethods()
	return s, nil
}

// Shutdown tears down subsystems (kill PTYs, stop watchers, close upstream MCP
// servers and browser instances).
func (s *Service) Shutdown() {
	s.PTY.KillAll()
	s.Workspaces.Shutdown()
	s.MCP.CloseUpstreams()
	if s.Browser != nil {
		s.Browser.StopAll()
	}
	s.Activity.Close()
}

// activityBridge adapts the MCP proxy's ActivityLogger to the activity store +
// live UI notifications.
type activityBridge struct {
	store *activity.Store
	rpc   *rpc.Server
}

func (b *activityBridge) LogToolCall(rec mcp.ToolCallRecord) {
	session := rec.SessionID
	if session == "" {
		session = "default"
	}
	summary := rec.Tool
	if rec.Summary != "" {
		summary += " — " + rec.Summary
	}
	risk := activity.RiskLow
	if rec.IsError {
		risk = activity.RiskMedium
	}
	stored := b.store.Append(activity.Event{
		Ts: rec.Time, SessionID: session, Kind: activity.KindToolCall,
		Actor: rec.Upstream, Summary: summary, Detail: rec.Args, Risk: risk,
	})
	b.rpc.Notify("activity.event", stored)
}

func (s *Service) registerMethods() {
	s.RPC.Register("pty.spawn", s.ptySpawn)
	s.RPC.Register("pty.write", s.ptyWrite)
	s.RPC.Register("pty.resize", s.ptyResize)
	s.RPC.Register("pty.kill", s.ptyKill)
	s.RPC.Register("pty.list", s.ptyList)
	s.RPC.Register("workspace.open", s.workspaceOpen)
	s.RPC.Register("workspace.list", s.workspaceList)
	s.RPC.Register("workspace.close", s.workspaceClose)
	s.RPC.Register("browserSession.list", s.browserList)
	s.RPC.Register("browserSession.latestShot", s.browserLatestShot)
	s.RPC.Register("browserSession.close", s.browserClose)
	s.RPC.Register("browserSession.focus", s.browserFocus)
	s.RPC.Register("activity.recent", s.activityRecent)
	s.RPC.Register("approval.list", s.approvalList)
	s.RPC.Register("approval.decide", s.approvalDecide)
	s.RPC.Register("approval.setMode", s.approvalSetMode)
	s.RPC.Register("approval.mode", s.approvalMode)
	s.RPC.Register("replay.timeline", s.replayTimeline)
	s.RPC.Register("replay.at", s.replayAt)
}

// --- PTY methods ---

type spawnParams struct {
	Cwd  string   `json:"cwd"`
	Cmd  string   `json:"cmd"`
	Args []string `json:"args"`
	Env  []string `json:"env"`
}

func (s *Service) ptySpawn(_ context.Context, raw json.RawMessage) (any, error) {
	var p spawnParams
	if err := decode(raw, &p); err != nil {
		return nil, err
	}
	sid, err := s.PTY.Spawn(p.Cwd, p.Cmd, p.Args, p.Env)
	if err != nil {
		return nil, err
	}
	return map[string]string{"sessionId": sid}, nil
}

type writeParams struct {
	SessionID string `json:"sessionId"`
	Data      string `json:"data"` // base64
}

func (s *Service) ptyWrite(_ context.Context, raw json.RawMessage) (any, error) {
	var p writeParams
	if err := decode(raw, &p); err != nil {
		return nil, err
	}
	data, err := base64.StdEncoding.DecodeString(p.Data)
	if err != nil {
		return nil, &rpc.Error{Code: rpc.CodeInvalidParams, Message: "data must be base64"}
	}
	if err := s.PTY.Write(p.SessionID, data); err != nil {
		return nil, err
	}
	return struct{}{}, nil
}

type resizeParams struct {
	SessionID string `json:"sessionId"`
	Cols      uint16 `json:"cols"`
	Rows      uint16 `json:"rows"`
}

func (s *Service) ptyResize(_ context.Context, raw json.RawMessage) (any, error) {
	var p resizeParams
	if err := decode(raw, &p); err != nil {
		return nil, err
	}
	if err := s.PTY.Resize(p.SessionID, p.Cols, p.Rows); err != nil {
		return nil, err
	}
	return struct{}{}, nil
}

type sessionIDParams struct {
	SessionID string `json:"sessionId"`
}

func (s *Service) ptyKill(_ context.Context, raw json.RawMessage) (any, error) {
	var p sessionIDParams
	if err := decode(raw, &p); err != nil {
		return nil, err
	}
	if err := s.PTY.Kill(p.SessionID); err != nil {
		return nil, err
	}
	return struct{}{}, nil
}

func (s *Service) ptyList(_ context.Context, _ json.RawMessage) (any, error) {
	return map[string][]string{"sessions": s.PTY.List()}, nil
}

// --- Workspace methods ---

type openParams struct {
	Path string `json:"path"`
}

func (s *Service) workspaceOpen(_ context.Context, raw json.RawMessage) (any, error) {
	var p openParams
	if err := decode(raw, &p); err != nil {
		return nil, err
	}
	ws, err := s.Workspaces.Open(p.Path)
	if err != nil {
		return nil, err
	}
	return ws, nil
}

func (s *Service) workspaceList(_ context.Context, _ json.RawMessage) (any, error) {
	return map[string]any{"workspaces": s.Workspaces.List()}, nil
}

type closeParams struct {
	WorkspaceID string `json:"workspaceId"`
}

func (s *Service) workspaceClose(_ context.Context, raw json.RawMessage) (any, error) {
	var p closeParams
	if err := decode(raw, &p); err != nil {
		return nil, err
	}
	if err := s.Workspaces.Close(p.WorkspaceID); err != nil {
		return nil, err
	}
	return struct{}{}, nil
}

// decode unmarshals params, mapping JSON errors to a JSON-RPC InvalidParams.
func decode(raw json.RawMessage, v any) error {
	if len(raw) == 0 {
		return nil // method takes no params
	}
	if err := json.Unmarshal(raw, v); err != nil {
		return &rpc.Error{Code: rpc.CodeInvalidParams, Message: err.Error()}
	}
	return nil
}
