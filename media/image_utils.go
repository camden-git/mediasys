package media

import (
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"path/filepath"
	"strings"
)

var supportedImageExtensions = map[string]bool{
	".jpg": true, ".jpeg": true, ".png": true, ".gif": true, ".bmp": true, ".tif": true, ".tiff": true,
}

// IsRasterImage checks if the filename has a common raster image extension
func IsRasterImage(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	return supportedImageExtensions[ext]
}

// GetImageDimensions reads image headers to get dimensions without loading pixels
func GetImageDimensions(filePath string) (width, height int, err error) {
	file, err := os.Open(filePath)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to open image file %s: %w", filePath, err)
	}
	defer file.Close()

	config, _, err := image.DecodeConfig(file)
	if err != nil {
		// Common error: "image: unknown format" if decoder not registered or file corrupted/unsupported
		return 0, 0, fmt.Errorf("failed to decode image config for %s: %w", filePath, err)
	}

	return config.Width, config.Height, nil
}
