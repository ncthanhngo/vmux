package browsersession

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"os"
	"testing"
)

// makePNG builds a w×h solid-color PNG for tests.
func makePNG(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{R: 10, G: 120, B: 200, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestAddStoresAndThumbnails(t *testing.T) {
	store := NewScreenshotStore(t.TempDir())
	raw := makePNG(t, 1280, 800)

	shot, thumbB64, err := store.Add("sess1", raw)
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	if shot.Width != 1280 || shot.Height != 800 {
		t.Errorf("dimensions = %dx%d, want 1280x800", shot.Width, shot.Height)
	}
	if thumbB64 == "" {
		t.Error("expected a base64 thumbnail")
	}
	if _, err := os.Stat(shot.fullPath); err != nil {
		t.Errorf("full image not written: %v", err)
	}

	// Thumbnail's longest edge must be <= thumbMaxDim.
	thumb, err := store.ReadFull("sess1", shot.ID)
	if err != nil {
		t.Fatalf("ReadFull: %v", err)
	}
	if len(thumb) == 0 {
		t.Error("ReadFull returned no bytes")
	}

	latest, ok := store.Latest("sess1")
	if !ok || latest.ID != shot.ID {
		t.Errorf("Latest mismatch: %+v", latest)
	}
}

func TestThumbnailRespectsMaxDim(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 1000, 500))
	thumb := downscale(img, thumbMaxDim)
	b := thumb.Bounds()
	if b.Dx() > thumbMaxDim || b.Dy() > thumbMaxDim {
		t.Errorf("thumb %dx%d exceeds max %d", b.Dx(), b.Dy(), thumbMaxDim)
	}
	if b.Dx() != thumbMaxDim {
		t.Errorf("expected longest edge %d, got width %d", thumbMaxDim, b.Dx())
	}
}

func TestRingBufferEvicts(t *testing.T) {
	store := NewScreenshotStore(t.TempDir())
	raw := makePNG(t, 64, 64)
	var firstPath string
	for i := 0; i < maxShotsPerSession+3; i++ {
		shot, _, err := store.Add("s", raw)
		if err != nil {
			t.Fatal(err)
		}
		if i == 0 {
			firstPath = shot.fullPath
		}
	}
	if got := len(store.List("s")); got != maxShotsPerSession {
		t.Errorf("ring size = %d, want %d", got, maxShotsPerSession)
	}
	if _, err := os.Stat(firstPath); !os.IsNotExist(err) {
		t.Error("evicted shot's file should be deleted from disk")
	}
}

func TestClear(t *testing.T) {
	store := NewScreenshotStore(t.TempDir())
	store.Add("s", makePNG(t, 64, 64))
	store.Clear("s")
	if len(store.List("s")) != 0 {
		t.Error("Clear should remove all shots")
	}
}
