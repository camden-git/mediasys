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
	PublicURL string // externally visible base URL for share pages
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

// getPublicAlbumByIdentifier is getAlbumByIdentifier for the public (unauthenticated) routes.
// Hidden albums are unlisted but reachable by link, i.e. by slug; a numeric ID resolves to a
// hidden album only for an authenticated user with view access to it, so hidden albums cannot
// be enumerated by ID. private reports that the album was resolved through such access, in
// which case the response must not be cached publicly.
func (ah *AlbumHandler) getPublicAlbumByIdentifier(r *http.Request, identifier string) (album *models.Album, private bool, err error) {
	if albumID, parseErr := strconv.ParseUint(identifier, 10, 64); parseErr == nil {
		byID, err := ah.AlbumRepo.GetByID(uint(albumID))
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, false, fmt.Errorf("error fetching album by ID %d: %w", albumID, err)
		}
		if err == nil {
			if !byID.IsHidden {
				return byID, false, nil
			}
			if canViewHiddenAlbum(optionalRequestUser(ah.UserRepo, r), byID.ID) {
				return byID, true, nil
			}
		}
		// not found (or hidden) by ID, continue to try by slug
	}

	album, err = ah.AlbumRepo.GetBySlug(identifier)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, false, gorm.ErrRecordNotFound
		}
		return nil, false, fmt.Errorf("error fetching album by slug '%s': %w", identifier, err)
	}
	return album, false, nil
}

// canViewHiddenAlbum reports whether user may look up a hidden album by ID, mirroring the
// access check on the admin album routes.
func canViewHiddenAlbum(user *models.User, albumID uint) bool {
	return user != nil && (user.HasGlobalPermission("album.list") || user.HasAlbumPermission(albumID, "album.view.content"))
}

// setAlbumCacheHeaders sets public cache headers, or disables caching when the album was
// resolved through the requester's own access (see getPublicAlbumByIdentifier).
func setAlbumCacheHeaders(w http.ResponseWriter, private bool, maxAge int) {
	if private {
		w.Header().Set("Cache-Control", "private, no-store")
		return
	}
	setCacheHeaders(w, maxAge)
}

func (ah *AlbumHandler) ListAlbums(w http.ResponseWriter, r *http.Request) {
	albums, err := ah.AlbumRepo.ListAll()
	if err != nil {
		log.Printf("Error listing albums: %v", err)
		WriteAPIError(w, http.StatusInternalServerError, "InternalError", "Failed to retrieve albums")
		return
	}
	setCacheHeaders(w, 300)
	WriteAPIResponse(w, http.StatusOK, toPublicAlbums(albums))
}

func (ah *AlbumHandler) GetAlbum(w http.ResponseWriter, r *http.Request) {
	identifier := chi.URLParam(r, "album_identifier")

	album, private, err := ah.getPublicAlbumByIdentifier(r, identifier)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			WriteAPIError(w, http.StatusNotFound, "AlbumNotFound", "Album not found")
		} else {
			log.Printf("Error getting album by identifier '%s': %v", identifier, err)
			WriteAPIError(w, http.StatusInternalServerError, "InternalError", "Failed to retrieve album")
		}
		return
	}

	var artists []map[string]interface{}
	if ah.ImageRepo != nil {
		users, err := ah.ImageRepo.ListUploadersByAlbum(album.ID)
		if err != nil {
			log.Printf("Error loading uploaders for album %d: %v", album.ID, err)
		}
		for _, u := range users {
			artists = append(artists, map[string]interface{}{
				"id":         u.ID,
				"username":   u.Username,
				"first_name": u.FirstName,
				"last_name":  u.LastName,
			})
		}
	}

	banners, err := ah.AlbumRepo.GetBanners(album.ID)
	if err != nil {
		log.Printf("Error loading banners for album %d: %v", album.ID, err)
		WriteAPIError(w, http.StatusInternalServerError, "InternalError", "Failed to retrieve album")
		return
	}
	bannerPaths := make([]string, 0, len(banners))
	for _, b := range banners {
		bannerPaths = append(bannerPaths, b.ImagePath)
	}

	type albumWithArtists struct {
		PublicAlbum
		Artists []map[string]interface{} `json:"artists,omitempty"`
		Banners []string                 `json:"banners"`
	}
	setAlbumCacheHeaders(w, private, 300)
	WriteAPIResponse(w, http.StatusOK, albumWithArtists{PublicAlbum: toPublicAlbum(album), Artists: artists, Banners: bannerPaths})
}
