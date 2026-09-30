package media

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"io"
	"math"
	"os"
	"strconv"

	"github.com/chai2010/webp"
	"github.com/disintegration/imaging"
	"github.com/google/uuid"
)

const (
	BannerTargetWidth   = 2000
	BannerQuality       = 80
	BannerFileExtension = ".webp"

	ThumbnailQuality       = 90
	ThumbnailFileExtension = ".webp"

	PreviewQuality          = 85
	PreviewFileExtension    = ".webp"
	PreviewLandscapeMaxLong = 3400
	PreviewPortraitMaxLong  = 2200

	// webpMaxDimension is the largest width or height WebP can encode.
	webpMaxDimension = 16383

	// DefaultMaxImagePixels caps the decoded size of an image (width x height)
	// to guard against decompression bombs. Override with MEDIA_MAX_PIXELS.
	DefaultMaxImagePixels = 100_000_000

	// maxBannerUploadBytes bounds how much of an uploaded banner is buffered.
	maxBannerUploadBytes = 64 << 20
)

// ErrImageTooLarge is returned for images whose pixel count exceeds the cap.
var ErrImageTooLarge = errors.New("image dimensions exceed the maximum allowed")

// MaxImagePixels returns the configured decoded-pixel cap.
func MaxImagePixels() int64 {
	if v := os.Getenv("MEDIA_MAX_PIXELS"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil && n > 0 {
			return n
		}
	}
	return DefaultMaxImagePixels
}

// checkDimensions reads only the image header from r and rejects images larger
// than the pixel cap before any pixel data is decoded.
func checkDimensions(r io.Reader) error {
	cfg, _, err := image.DecodeConfig(r)
	if err != nil {
		return fmt.Errorf("failed to read image header: %w", err)
	}
	if cfg.Width <= 0 || cfg.Height <= 0 {
		return fmt.Errorf("invalid image dimensions: %dx%d", cfg.Width, cfg.Height)
	}
	if px := int64(cfg.Width) * int64(cfg.Height); px > MaxImagePixels() {
		return fmt.Errorf("%w: %dx%d (%d pixels)", ErrImageTooLarge, cfg.Width, cfg.Height, px)
	}
	return nil
}

// OpenImage decodes the image at path with its EXIF orientation applied,
// refusing images over the pixel cap.
func OpenImage(path string) (image.Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	if err := checkDimensions(f); err != nil {
		return nil, err
	}
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}
	return imaging.Decode(f, imaging.AutoOrientation(true))
}

// Processor handles media transformations like thumbnailing and resizing and
// writes the results to the object store.
type Processor struct {
	store *Store
}

func NewProcessor(store *Store) *Processor {
	return &Processor{store: store}
}

// saveWebP encodes img as WebP and stores it under prefix with a random name.
func (p *Processor) saveWebP(ctx context.Context, img image.Image, quality float32, prefix string) (string, error) {
	var buf bytes.Buffer
	if err := webp.Encode(&buf, img, &webp.Options{Quality: quality}); err != nil {
		return "", fmt.Errorf("webp encoding failed: %w", err)
	}
	key := prefix + uuid.NewString() + ".webp"
	if _, err := p.store.Put(ctx, key, &buf, int64(buf.Len()), "image/webp"); err != nil {
		return "", err
	}
	return key, nil
}

// GenerateThumbnail creates a thumbnail where the longest side matches maxSize.
// returns the object key of the saved thumbnail.
func (p *Processor) GenerateThumbnail(originalImg image.Image, originalRelPath string, maxSize int) (string, error) {
	origBounds := originalImg.Bounds()
	origWidth := origBounds.Dx()
	origHeight := origBounds.Dy()
	if origWidth <= 0 || origHeight <= 0 {
		return "", fmt.Errorf("invalid original image dimensions: %dx%d", origWidth, origHeight)
	}

	var newWidth, newHeight int
	if origWidth > origHeight {
		if origWidth <= maxSize {
			newWidth, newHeight = origWidth, origHeight
		} else {
			newWidth = maxSize
			newHeight = int(math.Round(float64(origHeight) * (float64(maxSize) / float64(origWidth))))
		}
	} else {
		if origHeight <= maxSize {
			newWidth, newHeight = origWidth, origHeight
		} else {
			newHeight = maxSize
			newWidth = int(math.Round(float64(origWidth) * (float64(maxSize) / float64(origHeight))))
		}
	}
	newWidth = max(1, newWidth)
	newHeight = max(1, newHeight)

	thumb := imaging.Resize(originalImg, newWidth, newHeight, imaging.Lanczos)

	key, err := p.saveWebP(context.Background(), thumb, ThumbnailQuality, PrefixThumbnails)
	if err != nil {
		return "", fmt.Errorf("failed to save thumbnail: %w", err)
	}
	return key, nil
}

// GeneratePreview creates an orientation-aware preview (landscape: ≤3400px on long side,
// portrait: ≤2200px on long side), stores it and returns the object key.
func (p *Processor) GeneratePreview(src image.Image, originalRelPath string) (string, error) {
	w, h := src.Bounds().Dx(), src.Bounds().Dy()
	if w <= 0 || h <= 0 {
		return "", fmt.Errorf("invalid image dimensions: %dx%d", w, h)
	}

	var resized image.Image
	if w >= h {
		// landscape (or square)
		if w > PreviewLandscapeMaxLong {
			resized = imaging.Fit(src, PreviewLandscapeMaxLong, 99999, imaging.Lanczos)
		} else {
			resized = src
		}
	} else {
		// portrait
		if h > PreviewPortraitMaxLong {
			resized = imaging.Fit(src, 99999, PreviewPortraitMaxLong, imaging.Lanczos)
		} else {
			resized = src
		}
	}

	key, err := p.saveWebP(context.Background(), resized, PreviewQuality, PrefixPreviews)
	if err != nil {
		return "", fmt.Errorf("failed to save preview: %w", err)
	}
	return key, nil
}

// ProcessBanner resizes an uploaded banner, stores it and returns the object key.
// Banners are never upscaled and stay within WebP's dimension limit.
func (p *Processor) ProcessBanner(fileData io.Reader) (string, error) {
	data, err := io.ReadAll(io.LimitReader(fileData, maxBannerUploadBytes+1))
	if err != nil {
		return "", fmt.Errorf("failed to read uploaded banner: %w", err)
	}
	if len(data) > maxBannerUploadBytes {
		return "", errors.New("uploaded banner is too large")
	}
	if err := checkDimensions(bytes.NewReader(data)); err != nil {
		return "", fmt.Errorf("invalid uploaded banner: %w", err)
	}
	img, err := imaging.Decode(bytes.NewReader(data), imaging.AutoOrientation(true))
	if err != nil {
		return "", fmt.Errorf("failed to decode uploaded banner image: %w", err)
	}

	processedImg := imaging.Fit(img, BannerTargetWidth, webpMaxDimension, imaging.Lanczos)

	key, err := p.saveWebP(context.Background(), processedImg, BannerQuality, PrefixBanners)
	if err != nil {
		return "", fmt.Errorf("failed to save banner: %w", err)
	}
	return key, nil
}
