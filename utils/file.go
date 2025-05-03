package utils

import (
	"fmt"
	"github.com/disintegration/imaging"
	"github.com/google/uuid"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"log"
	"math"
	"os"
	"path/filepath"
	"strings"
)

var supportedImageExtensions = map[string]bool{
	".jpg":  true,
	".jpeg": true,
	".png":  true,
	".gif":  true,
	".bmp":  true,
	".tif":  true,
	".tiff": true,
}

func IsRasterImage(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	return supportedImageExtensions[ext]
}

func GetImageDimensions(filePath string) (width, height int, err error) {
	file, err := os.Open(filePath)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to open image file %s: %w", filePath, err)
	}
	defer file.Close()
	config, format, err := image.DecodeConfig(file)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to decode image config for %s: %w", filePath, err)
	}
	log.Printf("Decoded dimensions for %s (format: %s): %dx%d", filePath, format, config.Width, config.Height)
	return config.Width, config.Height, nil
}

// GenerateThumbnail creates a thumbnail where the longest side matches maxSize, preserving aspect ratio
func GenerateThumbnail(originalImagePath, thumbnailDir string, maxSize int) (string, error) {
	if err := os.MkdirAll(thumbnailDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create thumbnail directory %s: %w", thumbnailDir, err)
	}

	img, err := imaging.Open(originalImagePath, imaging.AutoOrientation(true))
	if err != nil {
		return "", fmt.Errorf("failed to open image %s: %w", originalImagePath, err)
	}

	origBounds := img.Bounds()
	origWidth := origBounds.Dx()
	origHeight := origBounds.Dy()

	if origWidth <= 0 || origHeight <= 0 {
		return "", fmt.Errorf("invalid original image dimensions for %s: %dx%d", originalImagePath, origWidth, origHeight)
	}

	var newWidth, newHeight int
	if origWidth > origHeight {
		if origWidth <= maxSize { // dont scale up
			newWidth = origWidth
			newHeight = origHeight
		} else {
			newWidth = maxSize
			newHeight = int(math.Round(float64(origHeight) * (float64(maxSize) / float64(origWidth))))
		}
	} else {
		if origHeight <= maxSize { // dont scale up
			newWidth = origWidth
			newHeight = origHeight
		} else {
			newHeight = maxSize
			newWidth = int(math.Round(float64(origWidth) * (float64(maxSize) / float64(origHeight))))
		}
	}

	newWidth = max(1, newWidth)
	newHeight = max(1, newHeight)

	thumb := imaging.Resize(img, newWidth, newHeight, imaging.Lanczos)

	thumbUUID, err := uuid.NewRandom()
	if err != nil {
		return "", fmt.Errorf("failed to generate UUID for thumbnail: %w", err)
	}
	thumbFilename := thumbUUID.String() + ".jpg"
	thumbnailSavePath := filepath.Join(thumbnailDir, thumbFilename)

	err = imaging.Save(thumb, thumbnailSavePath, imaging.JPEGQuality(90))
	if err != nil {
		return "", fmt.Errorf("failed to save thumbnail to %s: %w", thumbnailSavePath, err)
	}

	log.Printf("Generated thumbnail (%dx%d) for %s at %s", newWidth, newHeight, originalImagePath, thumbnailSavePath)
	return thumbnailSavePath, nil
}
