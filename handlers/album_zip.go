package handlers

import (
	"errors"
	"fmt"
	"log"
	"net/http"

	"github.com/camden-git/mediasysbackend/database"
	"github.com/camden-git/mediasysbackend/workers"
	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"
)

func (ah *AlbumHandler) RequestAlbumZipGeneration(w http.ResponseWriter, r *http.Request) {
	identifier := chi.URLParam(r, "id")

	album, err := ah.getAlbumByIdentifier(identifier)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "Album not found"})
		} else {
			log.Printf("Error finding album '%s' for zip request: %v", identifier, err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to find album"})
		}
		return
	}

	if album.ZipStatus == database.StatusPending || album.ZipStatus == database.StatusProcessing {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "Album ZIP generation is already pending or processing."})
		return
	}

	err = ah.AlbumRepo.RequestZip(album.ID)
	if err != nil {
		log.Printf("Error marking album zip pending for ID %d: %v", album.ID, err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to request ZIP generation"})
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
			http.NotFound(w, r)
		} else {
			log.Printf("Error finding album '%s' for zip download: %v", identifier, err)
			http.Error(w, "Failed to find album", http.StatusInternalServerError)
		}
		return
	}

	if album.ZipStatus != database.StatusDone || album.ZipPath == nil || *album.ZipPath == "" {
		if album.ZipStatus == database.StatusPending || album.ZipStatus == database.StatusProcessing {
			http.Error(w, "ZIP archive is currently being generated. Please try again later.", http.StatusAccepted)
		} else if album.ZipStatus == database.StatusError && album.ZipError != nil {
			http.Error(w, fmt.Sprintf("ZIP generation failed: %s", *album.ZipError), http.StatusConflict)
		} else {
			http.Error(w, "ZIP archive not available for this album or not yet generated.", http.StatusNotFound)
		}
		return
	}

	serveObject(w, r, ah.Store, *album.ZipPath, "private, max-age=0, must-revalidate",
		contentDisposition("attachment", album.Slug+"_archive.zip"))
}
