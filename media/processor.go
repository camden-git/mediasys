package media

import (
	"fmt"
	"github.com/chai2010/webp"
	"github.com/disintegration/imaging"
	"github.com/google/uuid"
	"image"
	"io"
	"log"
	"math"
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

// Processor handles media transformations like thumbnailing and resizing. it
// relies on a Store implementation for saving the results.
type Processor struct {
	store Store
}

func NewProcessor(store Store) *Processor {
	return &Processor{store: store}
}

// GenerateThumbnail creates a thumbnail where the longest side matches maxSize.
// saves the result using the Store. returns relative path to saved thumb or error.
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

	reader, writer := io.Pipe()

	go func() {
		defer writer.Close()
		err := webp.Encode(writer, thumb, &webp.Options{Quality: float32(ThumbnailQuality)})
		if err != nil {
			log.Printf("processor: Failed to encode thumbnail: %v", err)
			writer.CloseWithError(fmt.Errorf("thumbnail encoding failed: %w", err))
		}
	}()

	thumbUUID, err := uuid.NewRandom()
	if err != nil {
		reader.Close()
		return "", fmt.Errorf("failed to generate UUID for thumbnail: %w", err)
	}
	targetFilename := thumbUUID.String() + ThumbnailFileExtension

	savedRelPath, err := p.store.Save(AssetTypeThumbnail, "", targetFilename, reader)
	// reader is automatically closed by io.Copy inside Save, or by the encoding goroutine on error

	if err != nil {
		return "", fmt.Errorf("failed to save thumbnail via store: %w", err)
	}

	log.Printf("processor: Generated and saved thumbnail for %s at %s", originalRelPath, savedRelPath)
	return savedRelPath, nil
}

// GeneratePreview creates an orientation-aware preview (landscape: ≤3400px on long side,
// portrait: ≤2200px on long side), saves it via the Store, and returns the relative path.
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

	reader, writer := io.Pipe()
	go func() {
		defer writer.Close()
		err := webp.Encode(writer, resized, &webp.Options{Quality: float32(PreviewQuality)})
		if err != nil {
			log.Printf("processor: Failed to encode preview: %v", err)
			writer.CloseWithError(fmt.Errorf("preview encoding failed: %w", err))
		}
	}()

	previewUUID, err := uuid.NewRandom()
	if err != nil {
		reader.Close()
		return "", fmt.Errorf("failed to generate UUID for preview: %w", err)
	}
	targetFilename := previewUUID.String() + PreviewFileExtension

	savedRelPath, err := p.store.Save(AssetTypePreview, "", targetFilename, reader)
	if err != nil {
		return "", fmt.Errorf("failed to save preview via store: %w", err)
	}

	log.Printf("processor: Generated and saved preview for %s at %s", originalRelPath, savedRelPath)
	return savedRelPath, nil
}

// ProcessBanner resizes an uploaded banner and saves it returns the relative
// path to saved banner or error
func (p *Processor) ProcessBanner(fileData io.Reader) (string, error) {
	img, format, err := image.Decode(fileData)
	if err != nil {
		return "", fmt.Errorf("failed to decode uploaded banner image: %w", err)
	}
	log.Printf("processor: Decoded uploaded banner (format: %s)", format)

	processedImg := imaging.Resize(img, BannerTargetWidth, 0, imaging.Lanczos)

	reader, writer := io.Pipe()
	go func() {
		defer writer.Close()
		err := webp.Encode(writer, processedImg, &webp.Options{Quality: float32(BannerQuality)})
		if err != nil {
			log.Printf("processor: Failed to encode banner: %v", err)
			writer.CloseWithError(fmt.Errorf("banner encoding failed: %w", err))
		}
	}()

	bannerUUID, err := uuid.NewRandom()
	if err != nil {
		reader.Close()
		return "", fmt.Errorf("failed to generate UUID for banner: %w", err)
	}
	targetFilename := bannerUUID.String() + BannerFileExtension

	savedRelPath, err := p.store.Save(AssetTypeBanner, "", targetFilename, reader)
	if err != nil {
		return "", fmt.Errorf("failed to save banner via store: %w", err)
	}

	log.Printf("processor: Processed and saved banner to %s", savedRelPath)
	return savedRelPath, nil
}
