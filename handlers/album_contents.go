package handlers

import (
	"errors"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"
)

func (ah *AlbumHandler) GetAlbumContents(w http.ResponseWriter, r *http.Request) {
	identifier := chi.URLParam(r, "album_identifier")

	album, err := ah.getAlbumByIdentifier(identifier)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "Album not found"})
		} else {
			log.Printf("Error getting album '%s' for contents: %v", identifier, err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to retrieve album information"})
		}
		return
	}

	albumFullPath := filepath.Join(ah.Cfg.RootDirectory, album.FolderPath)
	albumFullPath = filepath.Clean(albumFullPath)
	if !strings.HasPrefix(albumFullPath, ah.Cfg.RootDirectory) {
		log.Printf("CRITICAL: Album ID %d (slug %s) folder path '%s' resolved outside root directory ('%s'). Aborting.", album.ID, album.Slug, album.FolderPath, albumFullPath)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Album configuration error"})
		return
	}

	defaultLimit := 120
	q := r.URL.Query()
	offset := 0
	limit := defaultLimit
	if o := q.Get("offset"); o != "" {
		if v, convErr := strconv.Atoi(o); convErr == nil && v >= 0 {
			offset = v
		}
	}
	if l := q.Get("limit"); l != "" {
		if v, convErr := strconv.Atoi(l); convErr == nil && v > 0 {
			limit = v
		}
	}
	var minRating *int
	if mr := q.Get("min_rating"); mr != "" {
		if v, convErr := strconv.Atoi(mr); convErr == nil && v >= 1 {
			minRating = &v
		}
	}

	fileInfos, totalCount, err := listDirectoryContents(albumFullPath, "/"+album.FolderPath, ah.Cfg, ah.ImageRepo, ah.ThumbGen, ah.TagRepo, album.ID, album.SortOrder, offset, limit, minRating)
	if err != nil {
		if os.IsNotExist(err) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "Album folder not found on disk: " + album.FolderPath})
		} else if os.IsPermission(err) {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "Permission denied accessing album folder"})
		} else {
			log.Printf("Error listing contents for album %d/%s (path %s): %v", album.ID, album.Slug, albumFullPath, err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to list album contents"})
		}
		return
	}

	listing := DirectoryListing{
		Path:    "/" + album.FolderPath,
		Files:   fileInfos,
		Total:   totalCount,
		Offset:  offset,
		Limit:   limit,
		HasMore: offset+len(fileInfos) < totalCount,
	}
	writeJSON(w, http.StatusOK, listing)
}
