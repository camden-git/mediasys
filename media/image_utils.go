package media

import (
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
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
