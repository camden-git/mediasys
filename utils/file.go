package utils

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/disintegration/imaging"
	"github.com/google/uuid"
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

// IsRasterImage checks if the filename has a common raster image extension
func IsRasterImage(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	return supportedImageExtensions[ext]
}

// GenerateThumbnail creates a thumbnail with a UUID filename
// returns the full path where the thumbnail was saved
func GenerateThumbnail(originalImagePath, thumbnailDir string, maxWidth, maxHeight int) (string, error) { // Now takes thumbnailDir, returns save path + error
	if err := os.MkdirAll(thumbnailDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create thumbnail directory %s: %w", thumbnailDir, err)
	}

	img, err := imaging.Open(originalImagePath)
	if err != nil {
		return "", fmt.Errorf("failed to open image %s: %w", originalImagePath, err)
	}

	thumb := imaging.Fit(img, maxWidth, maxHeight, imaging.Lanczos)

	thumbUUID, err := uuid.NewRandom()
	if err != nil {
		return "", fmt.Errorf("failed to generate UUID for thumbnail: %w", err)
	}
	thumbFilename := thumbUUID.String() + ".jpg" // Save all as jpg with UUID name
	thumbnailSavePath := filepath.Join(thumbnailDir, thumbFilename)

	err = imaging.Save(thumb, thumbnailSavePath, imaging.JPEGQuality(80)) // Adjust quality as needed
	if err != nil {
		return "", fmt.Errorf("failed to save thumbnail to %s: %w", thumbnailSavePath, err)
	}

	log.Printf("generated thumbnail (UUID: %s) for %s at %s", thumbUUID.String(), originalImagePath, thumbnailSavePath)
	return thumbnailSavePath, nil
}
