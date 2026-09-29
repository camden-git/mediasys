package handlers

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/camden-git/mediasysbackend/media"
	"github.com/camden-git/mediasysbackend/models"
	"github.com/camden-git/mediasysbackend/repository"
	"github.com/camden-git/mediasysbackend/workers"
	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"
)

type AlbumHandler struct {
	AlbumRepo repository.AlbumRepositoryInterface
	ImageRepo repository.ImageRepositoryInterface
	UserRepo  repository.UserRepository
	ThumbGen  *workers.ImageProcessor
	Store     *media.Store
}

func (ah *AlbumHandler) getAlbumByIdentifier(identifier string) (*models.Album, error) {
	// try parsing as ID
	if albumID, err := strconv.ParseUint(identifier, 10, 64); err == nil {
		album, err := ah.AlbumRepo.GetByID(uint(albumID))
		if err == nil {
			return album, nil // found by ID
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("error fetching album by ID %d: %w", albumID, err)
		}
		// If not found by ID, continue to try by slug
	}

	// not a valid ID or not found by ID, try fetching by slug
	album, err := ah.AlbumRepo.GetBySlug(identifier)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, gorm.ErrRecordNotFound // not found by slug either
		}
		return nil, fmt.Errorf("error fetching album by slug '%s': %w", identifier, err)
	}
	return album, nil
}

func (ah *AlbumHandler) ListAlbums(w http.ResponseWriter, r *http.Request) {
	albums, err := ah.AlbumRepo.ListAll()
	if err != nil {
		log.Printf("Error listing albums: %v", err)
		WriteAPIError(w, http.StatusInternalServerError, "InternalError", "Failed to retrieve albums")
		return
	}
	if albums == nil {
		albums = []models.Album{} // ensure an empty array instead of null for JSON
	}
	setCacheHeaders(w, 300)
	writeJSON(w, http.StatusOK, albums)
}

func (ah *AlbumHandler) GetAlbum(w http.ResponseWriter, r *http.Request) {
	identifier := chi.URLParam(r, "album_identifier")

	album, err := ah.getAlbumByIdentifier(identifier)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			WriteAPIError(w, http.StatusNotFound, "AlbumNotFound", "Album not found")
		} else {
			log.Printf("Error getting album by identifier '%s': %v", identifier, err)
			WriteAPIError(w, http.StatusInternalServerError, "InternalError", "Failed to retrieve album")
		}
		return
	}

	// Build artists list from uploaders
	var artists []map[string]interface{}
	if ah.ImageRepo != nil && ah.UserRepo != nil {
		if ids, err := ah.ImageRepo.GetDistinctUploaderIDsByAlbum(album.ID); err == nil {
			for _, id := range ids {
				if u, err := ah.UserRepo.GetByID(id); err == nil && u != nil {
					artists = append(artists, map[string]interface{}{
						"id":         u.ID,
						"username":   u.Username,
						"first_name": u.FirstName,
						"last_name":  u.LastName,
					})
				}
			}
		}
	}

	banners, _ := ah.AlbumRepo.GetBanners(album.ID)
	bannerPaths := make([]string, 0, len(banners))
	for _, b := range banners {
		bannerPaths = append(bannerPaths, b.ImagePath)
	}

	type albumWithArtists struct {
		*models.Album
		Artists []map[string]interface{} `json:"artists,omitempty"`
		Banners []string                 `json:"banners"`
	}
	setCacheHeaders(w, 300)
	writeJSON(w, http.StatusOK, albumWithArtists{Album: album, Artists: artists, Banners: bannerPaths})
}
