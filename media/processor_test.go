package media

import (
	"bytes"
	"errors"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func writePNG(t *testing.T, w, h int) string {
	t.Helper()
	var buf bytes.Buffer
	if err := png.Encode(&buf, image.NewRGBA(image.Rect(0, 0, w, h))); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(t.TempDir(), "img.png")
	if err := os.WriteFile(p, buf.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestOpenImagePixelCap(t *testing.T) {
	t.Setenv("MEDIA_MAX_PIXELS", "100")

	if _, err := OpenImage(writePNG(t, 10, 10)); err != nil {
		t.Fatalf("image at the cap rejected: %v", err)
	}
	_, err := OpenImage(writePNG(t, 11, 10))
	if !errors.Is(err, ErrImageTooLarge) {
		t.Fatalf("expected ErrImageTooLarge, got %v", err)
	}
}

func TestMaxImagePixelsDefault(t *testing.T) {
	t.Setenv("MEDIA_MAX_PIXELS", "")
	if got := MaxImagePixels(); got != DefaultMaxImagePixels {
		t.Fatalf("got %d, want %d", got, DefaultMaxImagePixels)
	}
	t.Setenv("MEDIA_MAX_PIXELS", "junk")
	if got := MaxImagePixels(); got != DefaultMaxImagePixels {
		t.Fatalf("got %d, want default for invalid value", got)
	}
}
