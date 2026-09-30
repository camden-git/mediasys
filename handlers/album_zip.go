package handlers

import (
	"errors"
	"fmt"
	"log"
	"net/http"

	"github.com/camden-git/mediasysbackend/database"
	"github.com/camden-git/mediasysbackend/repository"
	"github.com/camden-git/mediasysbackend/workers"
	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"
)

func (ah *AlbumHandler) RequestAlbumZipGeneration(w http.ResponseWriter, r *http.Request) {
	identifier := chi.URLParam(r, "id")

	album, err := ah.getAlbumByIdentifier(identifier)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			WriteAPIError(w, http.StatusNotFound, "AlbumNotFound", "Album not found")
		} else {
			log.Printf("Error finding album '%s' for zip request: %v", identifier, err)
			WriteAPIError(w, http.StatusInternalServerError, "AlbumFetchError", "Failed to find album")
		}
		return
	}

	if album.ZipStatus == database.StatusPending || album.ZipStatus == database.StatusProcessing {
		WriteAPIError(w, http.StatusConflict, "AlbumZipConflict", "Album ZIP generation is already pending or processing.")
		return
	}

	err = ah.AlbumRepo.RequestZip(album.ID)
	if err != nil {
		if errors.Is(err, repository.ErrZipInProgress) {
			WriteAPIError(w, http.StatusConflict, "AlbumZipConflict", "Album ZIP generation is already pending or processing.")
			return
		}
		if errors.Is(err, gorm.ErrRecordNotFound) {
			WriteAPIError(w, http.StatusNotFound, "AlbumNotFound", "Album not found")
			return
		}
		log.Printf("Error marking album zip pending for ID %d: %v", album.ID, err)
		WriteAPIError(w, http.StatusInternalServerError, "AlbumZipRequestError", "Failed to request ZIP generation")
		return
	}

	zipJob := workers.ImageJob{
		AlbumID:  album.ID,
		TaskType: workers.TaskAlbumZip,
	}
	// if the queue is full the dispatcher picks the pending archive up shortly
	ah.ThumbGen.QueueJob(zipJob)

	log.Printf("Album ZIP generation requested and queued for Album ID: %d", album.ID)
	writeJSON(w, http.StatusAccepted, map[string]string{"message": "Album ZIP generation request accepted and queued."})
}

func (ah *AlbumHandler) DownloadAlbumZipByID(w http.ResponseWriter, r *http.Request) {
	identifier := chi.URLParam(r, "id")
	ah.serveAlbumZip(w, r, identifier)
}

func (ah *AlbumHandler) DownloadAlbumZip(w http.ResponseWriter, r *http.Request) {
	identifier := chi.URLParam(r, "album_identifier")
	ah.serveAlbumZip(w, r, identifier)
}

func (ah *AlbumHandler) serveAlbumZip(w http.ResponseWriter, r *http.Request, identifier string) {
	album, err := ah.getAlbumByIdentifier(identifier)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			WriteAPIError(w, http.StatusNotFound, "AlbumNotFound", "Album not found")
		} else {
			log.Printf("Error finding album '%s' for zip download: %v", identifier, err)
			WriteAPIError(w, http.StatusInternalServerError, "AlbumFetchError", "Failed to find album")
		}
		return
	}

	if album.ZipStatus != database.StatusDone || album.ZipPath == nil || *album.ZipPath == "" {
		if album.ZipStatus == database.StatusPending || album.ZipStatus == database.StatusProcessing {
			writeJSON(w, http.StatusAccepted, map[string]string{
				"status":  album.ZipStatus,
				"message": "ZIP archive is currently being generated. Please try again later.",
			})
		} else if album.ZipStatus == database.StatusError && album.ZipError != nil {
			WriteAPIError(w, http.StatusConflict, "AlbumZipError", fmt.Sprintf("ZIP generation failed: %s", *album.ZipError))
		} else {
			WriteAPIError(w, http.StatusNotFound, "AlbumZipNotFound", "ZIP archive not available for this album or not yet generated.")
		}
		return
	}

	serveObject(w, r, ah.Store, *album.ZipPath, "private, max-age=0, must-revalidate",
		contentDisposition("attachment", album.Slug+"_archive.zip"), "")
}
