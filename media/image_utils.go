package media

import (
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"mime"
	"path/filepath"
	"strings"
)

var supportedImageExtensions = map[string]string{
	".jpg": "image/jpeg", ".jpeg": "image/jpeg", ".png": "image/png", ".gif": "image/gif",
	".bmp": "image/bmp", ".tif": "image/tiff", ".tiff": "image/tiff", ".webp": "image/webp",
}

// IsRasterImage checks if the filename has a common raster image extension
func IsRasterImage(filename string) bool {
	_, ok := supportedImageExtensions[strings.ToLower(filepath.Ext(filename))]
	return ok
}

// ContentTypeFor guesses a MIME type from a filename.
func ContentTypeFor(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	if ct, ok := supportedImageExtensions[ext]; ok {
		return ct
	}
	if ct := mime.TypeByExtension(ext); ct != "" {
		return ct
	}
	return "application/octet-stream"
}
