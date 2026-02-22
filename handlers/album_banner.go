package handlers

import (
	"errors"
	"log"
	"net/http"
	"os"

	"github.com/camden-git/mediasysbackend/media"
	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"
)

func (ah *AlbumHandler) UploadAlbumBanner(w http.ResponseWriter, r *http.Request) {
	identifier := chi.URLParam(r, "id")

	album, err := ah.getAlbumByIdentifier(identifier)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "Album not found"})
		} else {
			log.Printf("Error finding album '%s' for banner upload: %v", identifier, err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to find album"})
		}
		return
	}

	const maxUploadSize = 20 << 20 // 20 MB
	if err := r.ParseMultipartForm(maxUploadSize); err != nil {
		log.Printf("Error parsing multipart form for banner upload: %v", err)
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid form data: " + err.Error()})
		return
	}

	file, handler, err := r.FormFile("banner_image")
	if err != nil {
		if errors.Is(err, http.ErrMissingFile) {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "No file uploaded in 'banner_image' field"})
		} else {
			log.Printf("Error retrieving uploaded file 'banner_image': %v", err)
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Could not retrieve uploaded file"})
		}
		return
	}
	defer file.Close()

	log.Printf("Received banner upload for album %d/%s: %s (Size: %d)", album.ID, album.Slug, handler.Filename, handler.Size)

	if ah.MediaProcessor == nil {
		log.Printf("CRITICAL ERROR: MediaProcessor not configured in AlbumHandler for banner upload.")
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Server configuration error"})
		return
	}

	savedRelPath, procErr := ah.MediaProcessor.ProcessBanner(file)
	if procErr != nil {
		log.Printf("Error processing/saving banner for album %d/%s: %v", album.ID, album.Slug, procErr)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to process banner image"})
		return
	}

	oldBannerRelativePathPtr := album.BannerImagePath
	newBannerRelativePath := savedRelPath
	if oldBannerRelativePathPtr != nil && (*oldBannerRelativePathPtr != newBannerRelativePath) {
		mediaStore, storeErr := media.NewLocalStorage(ah.Cfg.MediaStoragePath, map[media.AssetType]string{})
		if storeErr == nil { // only attempt to delete if store initialized
			oldBannerFullPath, pathErr := mediaStore.GetFullPath(*oldBannerRelativePathPtr)
			if pathErr == nil {
				if removeErr := os.Remove(oldBannerFullPath); removeErr != nil && !os.IsNotExist(removeErr) {
					log.Printf("Warning: Failed to remove old banner file %s: %v", oldBannerFullPath, removeErr)
				} else if removeErr == nil {
					log.Printf("Removed old banner file: %s", oldBannerFullPath)
				}
			} else {
				log.Printf("Warning: Could not resolve full path for old banner %s: %v", *oldBannerRelativePathPtr, pathErr)
			}
		} else {
			log.Printf("Warning: Could not initialize media store to delete old banner: %v", storeErr)
		}
	}

	dbErr := ah.AlbumRepo.UpdateBannerPath(album.ID, &newBannerRelativePath)
	if dbErr != nil {
		mediaStore, storeErr := media.NewLocalStorage(ah.Cfg.MediaStoragePath, map[media.AssetType]string{})
		if storeErr == nil {
			// attempt to delete the newly saved banner if DB update fails
			if delErr := mediaStore.Delete(newBannerRelativePath); delErr != nil {
				log.Printf("Warning: Failed to delete banner %s after DB update failure: %v", newBannerRelativePath, delErr)
			}
		}
		log.Printf("Error updating banner path in DB for album %d/%s: %v", album.ID, album.Slug, dbErr)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to save banner information"})
		return
	}

	updatedAlbum, err := ah.AlbumRepo.GetByID(album.ID)
	if err != nil {
		log.Printf("Error fetching updated album %d after banner upload: %v", album.ID, err)
		// the banner was uploaded and DB updated, so this is a partial success
		writeJSON(w, http.StatusOK, map[string]interface{}{"message": "Banner uploaded successfully", "banner_image_path": newBannerRelativePath})
		return
	}
	writeJSON(w, http.StatusOK, updatedAlbum)
}
