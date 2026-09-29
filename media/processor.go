package media

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"io"
	"log"
	"math"

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
)

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
	newWidth = maxInt(1, newWidth)
	newHeight = maxInt(1, newHeight)

	thumb := imaging.Resize(originalImg, newWidth, newHeight, imaging.Lanczos)

	key, err := p.saveWebP(context.Background(), thumb, ThumbnailQuality, PrefixThumbnails)
	if err != nil {
		return "", fmt.Errorf("failed to save thumbnail: %w", err)
	}
	log.Printf("processor: Generated thumbnail for %s at %s", originalRelPath, key)
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
	log.Printf("processor: Generated preview for %s at %s", originalRelPath, key)
	return key, nil
}

// ProcessBanner resizes an uploaded banner, stores it and returns the object key.
func (p *Processor) ProcessBanner(fileData io.Reader) (string, error) {
	img, format, err := image.Decode(fileData)
	if err != nil {
		return "", fmt.Errorf("failed to decode uploaded banner image: %w", err)
	}
	log.Printf("processor: Decoded uploaded banner (format: %s)", format)

	processedImg := imaging.Resize(img, BannerTargetWidth, 0, imaging.Lanczos)

	key, err := p.saveWebP(context.Background(), processedImg, BannerQuality, PrefixBanners)
	if err != nil {
		return "", fmt.Errorf("failed to save banner: %w", err)
	}
	log.Printf("processor: Processed and saved banner to %s", key)
	return key, nil
}
