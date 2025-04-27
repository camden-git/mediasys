package handlers

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// ThumbnailServer creates a handler to serve thumbnails from the specified directory
func ThumbnailServer(thumbnailDir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		requestedFilename := strings.TrimPrefix(r.URL.Path, thumbnailApiPrefix)
		if requestedFilename == "" || strings.Contains(requestedFilename, "/") || strings.Contains(requestedFilename, "..") {
			http.Error(w, "Invalid thumbnail path", http.StatusBadRequest)
			return
		}

		fullThumbPath := filepath.Join(thumbnailDir, requestedFilename)

		cleanedPath := filepath.Clean(fullThumbPath)

		if !strings.HasPrefix(cleanedPath, thumbnailDir) {
			http.Error(w, "Forbidden", http.StatusForbidden)
			log.Printf("attempted thumbnail access outside thumbnail directory: Request='%s', Resolved='%s', ThumbDir='%s'",
				r.URL.Path, cleanedPath, thumbnailDir)
			return
		}

		if _, err := os.Stat(cleanedPath); os.IsNotExist(err) {
			http.NotFound(w, r)
			log.Printf("thumbnail not found: %s", cleanedPath)
			return
		} else if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			log.Printf("error stating thumbnail %s: %v", cleanedPath, err)
			return
		}

		cacheDuration := 24 * time.Hour
		w.Header().Set("Cache-Control", "public, max-age="+strconv.Itoa(int(cacheDuration.Seconds())))
		w.Header().Set("Expires", time.Now().Add(cacheDuration).Format(http.TimeFormat))

		http.ServeFile(w, r, cleanedPath)
	}
}
