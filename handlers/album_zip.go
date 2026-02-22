package handlers

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

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
		AlbumID:     int64(album.ID),
		TaskType:    workers.TaskAlbumZip,
		ModTimeUnix: time.Now().Unix(),
	}
	queued := ah.ThumbGen.QueueJob(zipJob)
	if !queued {
		log.Printf("Failed to queue album ZIP job for Album ID %d (queue full or already pending).", album.ID)
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "Failed to queue ZIP generation: processing queue is full."})
		return
	}

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

	fullZipPath := filepath.Join(ah.Cfg.MediaStoragePath, *album.ZipPath)
	fullZipPath = filepath.Clean(fullZipPath)

	if !strings.HasPrefix(fullZipPath, ah.Cfg.MediaStoragePath) {
		log.Printf("SECURITY: Attempt to download ZIP outside media storage: %s (resolved from %s)", fullZipPath, *album.ZipPath)
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	file, err := os.Open(fullZipPath)
	if os.IsNotExist(err) {
		log.Printf("ZIP file %s (from DB path %s) not found on disk. Inconsistency.", fullZipPath, *album.ZipPath)
		http.Error(w, "ZIP archive file not found on server.", http.StatusInternalServerError)
		return
	} else if err != nil {
		log.Printf("Error opening ZIP file %s: %v", fullZipPath, err)
		http.Error(w, "Failed to access ZIP archive.", http.StatusInternalServerError)
		return
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		log.Printf("Error stating ZIP file %s: %v", fullZipPath, err)
		http.Error(w, "Failed to get ZIP archive info.", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s_archive.zip\"", album.Slug))
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Length", strconv.FormatInt(fileInfo.Size(), 10))

	if modTime := fileInfo.ModTime(); !modTime.IsZero() {
		w.Header().Set("Last-Modified", modTime.UTC().Format(http.TimeFormat))
	}

	if _, copyErr := io.Copy(w, file); copyErr != nil {
		log.Printf("Error streaming ZIP file %s to client: %v", fullZipPath, copyErr)
	}
}
