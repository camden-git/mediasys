package handlers

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// AssetServer creates a handler to serve static files from a specific base directory.
// it expects the request path to contain the relative path within that base directory.
// example Usage in main.go:
//
//	r.Get("/banners/*", AssetServer(cfg.MediaStoragePath, "album_banners"))
//	r.Get("/archives/*", AssetServer(cfg.MediaStoragePath, "album_archives"))
//
// where the route prefix matches the subDir.
func AssetServer(baseStoragePath, subDir string) http.HandlerFunc {
	fullAssetDirPath := filepath.Join(baseStoragePath, subDir)
	fullAssetDirPath = filepath.Clean(fullAssetDirPath)
	log.Printf("Serving assets for '/%s/*' from directory: %s", subDir, fullAssetDirPath)

	if !strings.HasPrefix(fullAssetDirPath, baseStoragePath) {
		log.Fatalf("FATAL: Asset subdirectory '%s' resolved outside base storage path '%s'. Resolved path: '%s'", subDir, baseStoragePath, fullAssetDirPath)
	}

	return func(w http.ResponseWriter, r *http.Request) {
		// e.g., for route /banners/* and request /banners/image.jpg, extract "image.jpg"
		routePrefix := "/api/" + subDir + "/"
		relativePath := strings.TrimPrefix(r.URL.Path, routePrefix)

		if relativePath == "" || strings.Contains(relativePath, "..") {
			http.Error(w, "Invalid asset path", http.StatusBadRequest)
			return
		}

		requestedAssetPath := filepath.Join(fullAssetDirPath, relativePath)
		cleanedAssetPath := filepath.Clean(requestedAssetPath)

		if !strings.HasPrefix(cleanedAssetPath, fullAssetDirPath) {
			http.Error(w, "Forbidden", http.StatusForbidden)
			log.Printf("SECURITY: Attempted asset access outside designated directory: Request='%s', Resolved='%s', Allowed Base='%s'",
				r.URL.Path, cleanedAssetPath, fullAssetDirPath)
			return
		}

		if _, err := os.Stat(cleanedAssetPath); os.IsNotExist(err) {
			http.NotFound(w, r)
			// log.Printf("Asset not found: %s", cleanedAssetPath) // less verbose logging for 404
			return
		} else if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			log.Printf("Error stating asset file %s: %v", cleanedAssetPath, err)
			return
		}

		cacheDuration := 24 * time.Hour
		w.Header().Set("Cache-Control", fmt.Sprintf("public, max-age=%d", int(cacheDuration.Seconds())))
		w.Header().Set("Expires", time.Now().Add(cacheDuration).Format(http.TimeFormat))

		http.ServeFile(w, r, cleanedAssetPath)
	}
}
