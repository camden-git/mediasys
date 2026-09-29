package handlers

import (
	"path"
	"strings"
)

// normalizeFolderPath validates an album's storage folder. Albums no longer map
// to directories on disk; the folder is just the key prefix for their images.
// An empty folder defaults to the slug.
func normalizeFolderPath(folder, slug string) (string, bool) {
	folder = strings.TrimSpace(strings.ReplaceAll(folder, "\\", "/"))
	if folder == "" {
		folder = slug
	}
	clean := strings.Trim(path.Clean("/"+folder), "/")
	if clean == "" || clean == "." || strings.Contains(clean, "..") {
		return "", false
	}
	return clean, true
}
