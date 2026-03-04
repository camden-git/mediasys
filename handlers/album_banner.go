package handlers

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/camden-git/mediasysbackend/models"
	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"
)

// AddAlbumBanner uploads a new banner image and adds it to the album's banner list.
// POST /api/admin/albums/{id}/banners
func (h *AdminAlbumHandler) AddAlbumBanner(w http.ResponseWriter, r *http.Request) {
	albumIDStr := chi.URLParam(r, "id")
	albumID, err := parseID(r, "id")
	if err != nil {
		WriteAPIError(w, http.StatusBadRequest, "InvalidAlbumID", "Invalid album ID")
		return
	}

	album, err := h.AlbumRepo.GetByID(albumID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			WriteAPIError(w, http.StatusNotFound, "AlbumNotFound", "Album not found")
		} else {
			log.Printf("Error finding album %s for banner upload: %v", albumIDStr, err)
			WriteAPIError(w, http.StatusInternalServerError, "AlbumFetchError", "Failed to find album")
		}
		return
	}

	const maxUploadSize = 20 << 20
	if err := r.ParseMultipartForm(maxUploadSize); err != nil {
		WriteAPIError(w, http.StatusBadRequest, "InvalidForm", "Invalid form data: "+err.Error())
		return
	}
	file, handler, err := r.FormFile("banner_image")
	if err != nil {
		if errors.Is(err, http.ErrMissingFile) {
			WriteAPIError(w, http.StatusBadRequest, "MissingFile", "No file in 'banner_image' field")
		} else {
			WriteAPIError(w, http.StatusBadRequest, "FileError", "Could not retrieve uploaded file")
		}
		return
	}
	defer file.Close()
	log.Printf("Received banner upload for album %d: %s (size: %d)", album.ID, handler.Filename, handler.Size)

	if h.MediaProcessor == nil {
		WriteAPIError(w, http.StatusInternalServerError, "ConfigError", "Media processor not configured")
		return
	}
	savedRelPath, procErr := h.MediaProcessor.ProcessBanner(file)
	if procErr != nil {
		log.Printf("Error processing banner for album %d: %v", album.ID, procErr)
		WriteAPIError(w, http.StatusInternalServerError, "BannerProcessError", "Failed to process banner image")
		return
	}

	banner := &models.AlbumBanner{
		AlbumID:   album.ID,
		ImagePath: savedRelPath,
		SortOrder: 0,
	}
	if err := h.AlbumRepo.AddBanner(banner); err != nil {
		os.Remove(filepath.Join(h.Cfg.MediaStoragePath, filepath.FromSlash(savedRelPath)))
		log.Printf("Error saving banner for album %d: %v", album.ID, err)
		WriteAPIError(w, http.StatusInternalServerError, "BannerSaveError", "Failed to save banner")
		return
	}

	WriteAPIResponse(w, http.StatusCreated, banner)
}

// DeleteAlbumBanner deletes a specific banner from an album.
// DELETE /api/admin/albums/{id}/banners/{bannerId}
func (h *AdminAlbumHandler) DeleteAlbumBanner(w http.ResponseWriter, r *http.Request) {
	albumID, err := parseID(r, "id")
	if err != nil {
		WriteAPIError(w, http.StatusBadRequest, "InvalidAlbumID", "Invalid album ID")
		return
	}
	bannerID, err := parseID(r, "bannerId")
	if err != nil {
		WriteAPIError(w, http.StatusBadRequest, "InvalidBannerID", "Invalid banner ID")
		return
	}

	// Load the banner to get the file path before deleting
	banners, err := h.AlbumRepo.GetBanners(albumID)
	if err != nil {
		log.Printf("Error fetching banners for album %d: %v", albumID, err)
		WriteAPIError(w, http.StatusInternalServerError, "BannerFetchError", "Failed to fetch banners")
		return
	}

	var bannerPath string
	for _, b := range banners {
		if b.ID == bannerID {
			bannerPath = b.ImagePath
			break
		}
	}

	if err := h.AlbumRepo.DeleteBanner(bannerID, albumID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			WriteAPIError(w, http.StatusNotFound, "BannerNotFound", "Banner not found")
		} else {
			log.Printf("Error deleting banner %d from album %d: %v", bannerID, albumID, err)
			WriteAPIError(w, http.StatusInternalServerError, "BannerDeleteError", "Failed to delete banner")
		}
		return
	}

	if bannerPath != "" {
		os.Remove(filepath.Join(h.Cfg.MediaStoragePath, filepath.FromSlash(bannerPath)))
	}

	w.WriteHeader(http.StatusNoContent)
}

// ReorderAlbumBanners updates the sort order of album banners.
// PUT /api/admin/albums/{id}/banners/order
func (h *AdminAlbumHandler) ReorderAlbumBanners(w http.ResponseWriter, r *http.Request) {
	albumID, err := parseID(r, "id")
	if err != nil {
		WriteAPIError(w, http.StatusBadRequest, "InvalidAlbumID", "Invalid album ID")
		return
	}

	var req struct {
		BannerIDs []uint `json:"banner_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteAPIError(w, http.StatusBadRequest, "InvalidPayload", "Invalid request body: "+err.Error())
		return
	}

	if err := h.AlbumRepo.ReorderBanners(albumID, req.BannerIDs); err != nil {
		log.Printf("Error reordering banners for album %d: %v", albumID, err)
		WriteAPIError(w, http.StatusInternalServerError, "BannerReorderError", "Failed to reorder banners")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
