package handlers

import (
	"errors"
	"log"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"
)

func (ah *AlbumHandler) GetAlbumContents(w http.ResponseWriter, r *http.Request) {
	identifier := chi.URLParam(r, "album_identifier")

	album, err := ah.getAlbumByIdentifier(identifier)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			WriteAPIError(w, http.StatusNotFound, "AlbumNotFound", "Album not found")
		} else {
			log.Printf("Error getting album '%s' for contents: %v", identifier, err)
			WriteAPIError(w, http.StatusInternalServerError, "InternalError", "Failed to retrieve album information")
		}
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

	images, err := ah.ImageRepo.ListByAlbum(album.ID, minRating)
	if err != nil {
		log.Printf("Error listing contents for album %d/%s: %v", album.ID, album.Slug, err)
		WriteAPIError(w, http.StatusInternalServerError, "InternalError", "Failed to list album contents")
		return
	}

	listing := paginate("/"+album.FolderPath, imagesToSortedFileInfos(images, album.SortOrder), offset, limit)
	setCacheHeaders(w, 120)
	writeJSON(w, http.StatusOK, listing)
}
