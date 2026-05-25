package browsersession

import (
	"bytes"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"sync"
	"time"

	_ "image/jpeg" // register JPEG decoder so we can normalize to PNG

	"github.com/vmux/sidecar/internal/id"
)

// maxShotsPerSession bounds the in-memory + on-disk ring per browser session.
const maxShotsPerSession = 50

// thumbMaxDim is the longest edge of a generated thumbnail, in pixels.
const thumbMaxDim = 320

// Shot is one captured screenshot's metadata. The bytes live on disk.
type Shot struct {
	ID         string    `json:"id"`
	SessionID  string    `json:"sessionId"`
	Width      int       `json:"width"`
	Height     int       `json:"height"`
	CapturedAt time.Time `json:"capturedAt"`

	fullPath  string
	thumbPath string
}

// ScreenshotStore keeps a per-session ring buffer of screenshots, persisted
// under baseDir/<sessionId>/ with 0700 perms (page content may be sensitive).
type ScreenshotStore struct {
	baseDir string

	mu    sync.Mutex
	shots map[string][]Shot
}

func NewScreenshotStore(baseDir string) *ScreenshotStore {
	return &ScreenshotStore{baseDir: baseDir, shots: make(map[string][]Shot)}
}

// Add normalizes raw image bytes to PNG, stores the full image + a thumbnail,
// appends to the session ring (evicting the oldest), and returns the metadata
// plus a base64 PNG thumbnail for the shotCaptured event.
func (s *ScreenshotStore) Add(sessionID string, raw []byte) (Shot, string, error) {
	img, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return Shot{}, "", fmt.Errorf("decode screenshot: %w", err)
	}
	thumb := downscale(img, thumbMaxDim)

	var fullBuf, thumbBuf bytes.Buffer
	if err := png.Encode(&fullBuf, img); err != nil {
		return Shot{}, "", err
	}
	if err := png.Encode(&thumbBuf, thumb); err != nil {
		return Shot{}, "", err
	}

	dir := filepath.Join(s.baseDir, safeKey(sessionID))
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return Shot{}, "", err
	}
	shotID := id.New()
	shot := Shot{
		ID:         shotID,
		SessionID:  sessionID,
		Width:      img.Bounds().Dx(),
		Height:     img.Bounds().Dy(),
		CapturedAt: time.Now().UTC(),
		fullPath:   filepath.Join(dir, shotID+".png"),
		thumbPath:  filepath.Join(dir, shotID+".thumb.png"),
	}
	if err := os.WriteFile(shot.fullPath, fullBuf.Bytes(), 0o600); err != nil {
		return Shot{}, "", err
	}
	if err := os.WriteFile(shot.thumbPath, thumbBuf.Bytes(), 0o600); err != nil {
		os.Remove(shot.fullPath) // don't orphan the full image if the thumb fails
		return Shot{}, "", err
	}

	s.mu.Lock()
	ring := append(s.shots[sessionID], shot)
	for len(ring) > maxShotsPerSession {
		evicted := ring[0]
		ring = ring[1:]
		os.Remove(evicted.fullPath)
		os.Remove(evicted.thumbPath)
	}
	s.shots[sessionID] = ring
	s.mu.Unlock()

	return shot, base64PNG(thumbBuf.Bytes()), nil
}

// Latest returns the most recent shot for a session.
func (s *ScreenshotStore) Latest(sessionID string) (Shot, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	ring := s.shots[sessionID]
	if len(ring) == 0 {
		return Shot{}, false
	}
	return ring[len(ring)-1], true
}

// Sessions returns the ids of sessions that currently hold shots.
func (s *ScreenshotStore) Sessions() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	ids := make([]string, 0, len(s.shots))
	for sid := range s.shots {
		ids = append(ids, sid)
	}
	return ids
}

// List returns the session's shots oldest→newest.
func (s *ScreenshotStore) List(sessionID string) []Shot {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]Shot(nil), s.shots[sessionID]...)
}

// ReadFull returns the PNG bytes of a shot by id within a session.
func (s *ScreenshotStore) ReadFull(sessionID, shotID string) ([]byte, error) {
	s.mu.Lock()
	var path string
	for _, sh := range s.shots[sessionID] {
		if sh.ID == shotID {
			path = sh.fullPath
		}
	}
	s.mu.Unlock()
	if path == "" {
		return nil, fmt.Errorf("shot %q not found", shotID)
	}
	return os.ReadFile(path)
}

// Clear deletes all shots for a session (UI "Clear shots" / workspace close).
func (s *ScreenshotStore) Clear(sessionID string) {
	s.mu.Lock()
	ring := s.shots[sessionID]
	delete(s.shots, sessionID)
	s.mu.Unlock()
	for _, sh := range ring {
		os.Remove(sh.fullPath)
		os.Remove(sh.thumbPath)
	}
	os.RemoveAll(filepath.Join(s.baseDir, safeKey(sessionID)))
}

// safeKey maps a session id to a single path-safe directory component, so a
// session id can never escape baseDir (defense in depth — ids come from the
// trusted registry today, but this keeps the on-disk layout robust).
func safeKey(sessionID string) string {
	safe := make([]rune, 0, len(sessionID))
	for _, r := range sessionID {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_':
			safe = append(safe, r)
		default:
			safe = append(safe, '_')
		}
	}
	if len(safe) == 0 {
		return "session"
	}
	return string(safe)
}
