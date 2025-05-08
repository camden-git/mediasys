package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/camden-git/mediasysbackend/media"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/camden-git/mediasysbackend/config"
	"github.com/camden-git/mediasysbackend/database"
	"github.com/camden-git/mediasysbackend/workers"
	"github.com/go-chi/chi/v5"
)

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if data != nil {
		if err := json.NewEncoder(w).Encode(data); err != nil {
			log.Printf("Error encoding JSON response: %v", err)
		}
	}
}

type AlbumHandler struct {
	DB             *sql.DB
	Cfg            config.Config
	ThumbGen       *workers.ImageProcessor
	MediaProcessor *media.Processor
}

func (ah *AlbumHandler) getAlbumByIdentifier(identifier string) (database.Album, error) {
	// try parsing as ID
	if albumID, err := strconv.ParseInt(identifier, 10, 64); err == nil {
		album, err := database.GetAlbumByID(ah.DB, albumID)
		if err == nil {
			return album, nil // found by ID
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return database.Album{}, fmt.Errorf("error fetching album by ID %d: %w", albumID, err)
		}
	}

	// not a valid ID or not found by ID, try fetching by slug
	album, err := database.GetAlbumBySlug(ah.DB, identifier)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return database.Album{}, sql.ErrNoRows // not found by slug either
		}
		return database.Album{}, fmt.Errorf("error fetching album by slug '%s': %w", identifier, err)
	}
	return album, nil
}

func (ah *AlbumHandler) CreateAlbum(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name        string `json:"name"`
		Slug        string `json:"slug"`
		FolderPath  string `json:"folder_path"`
		Description string `json:"description"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request body: " + err.Error()})
		return
	}

	if req.Name == "" || req.FolderPath == "" || req.Slug == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Missing required fields: name, slug, and folder_path"})
		return
	}

	if strings.ContainsAny(req.Slug, " /\\?%*:|\"<>") || strings.TrimSpace(req.Slug) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid slug format. Use URL-safe characters without spaces."})
		return
	}

	cleanRelativePath := filepath.Clean(req.FolderPath)
	if filepath.IsAbs(cleanRelativePath) || strings.HasPrefix(cleanRelativePath, "..") {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "folder_path must be relative and cannot use '..'"})
		return
	}
	folderPathForDB := filepath.ToSlash(cleanRelativePath)
	fullPath := filepath.Join(ah.Cfg.RootDirectory, folderPathForDB)
	stat, err := os.Stat(fullPath)
	if os.IsNotExist(err) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "folder_path does not exist: " + folderPathForDB})
		return
	}
	if err != nil {
		log.Printf("Error stating folder path %s during album creation: %v", fullPath, err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Could not verify folder_path"})
		return
	}
	if !stat.IsDir() {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "folder_path is not a directory: " + folderPathForDB})
		return
	}

	albumID, err := database.CreateAlbum(ah.DB, req.Name, req.Slug, req.Description, folderPathForDB)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			writeJSON(w, http.StatusConflict, map[string]string{"error": "Album name, slug, or folder path already exists"})
		} else {
			log.Printf("Error creating album '%s' (slug '%s'): %v", req.Name, req.Slug, err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to create album"})
		}
		return
	}

	newAlbum, err := database.GetAlbumByID(ah.DB, albumID)
	if err != nil {
		log.Printf("Error fetching newly created album %d: %v", albumID, err)
		writeJSON(w, http.StatusCreated, map[string]interface{}{"message": "Album created successfully", "id": albumID})
		return
	}
	writeJSON(w, http.StatusCreated, newAlbum)
}

func (ah *AlbumHandler) ListAlbums(w http.ResponseWriter, r *http.Request) {
	albums, err := database.ListAlbums(ah.DB)
	if err != nil {
		log.Printf("Error listing albums: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to retrieve albums"})
		return
	}
	if albums == nil {
		albums = []database.Album{}
	}
	writeJSON(w, http.StatusOK, albums)
}

func (ah *AlbumHandler) GetAlbum(w http.ResponseWriter, r *http.Request) {
	identifier := chi.URLParam(r, "album_identifier")

	album, err := ah.getAlbumByIdentifier(identifier)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "Album not found"})
		} else {
			log.Printf("Error getting album by identifier '%s': %v", identifier, err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to retrieve album"})
		}
		return
	}
	writeJSON(w, http.StatusOK, album)
}

func (ah *AlbumHandler) GetAlbumContents(w http.ResponseWriter, r *http.Request) {
	identifier := chi.URLParam(r, "album_identifier")

	album, err := ah.getAlbumByIdentifier(identifier)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
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

	fileInfos, err := listDirectoryContents(albumFullPath, "/"+album.FolderPath, ah.Cfg, ah.DB, ah.ThumbGen, album.SortOrder)
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
		Path:  "/" + album.FolderPath,
		Files: fileInfos,
		// Parent: "/api/albums",
	}
	writeJSON(w, http.StatusOK, listing)
}

func (ah *AlbumHandler) UpdateAlbumSortOrder(w http.ResponseWriter, r *http.Request) {
	identifier := chi.URLParam(r, "album_identifier")

	album, err := ah.getAlbumByIdentifier(identifier)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "Album not found"})
		} else {
			log.Printf("Error finding album '%s' for sort update: %v", identifier, err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to find album"})
		}
		return
	}

	var req struct {
		SortOrder string `json:"sort_order"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request body: " + err.Error()})
		return
	}

	if !database.IsValidSortOrder(req.SortOrder) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid sort_order value provided"})
		return
	}

	err = database.UpdateAlbumSortOrder(ah.DB, album.ID, req.SortOrder)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "Album not found during update"})
		} else {
			log.Printf("Error updating sort order for album %d: %v", album.ID, err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to update sort order"})
		}
		return
	}

	updatedAlbum, err := database.GetAlbumByID(ah.DB, album.ID)
	if err != nil {
		log.Printf("Error fetching updated album %d after sort update: %v", album.ID, err)
		writeJSON(w, http.StatusOK, map[string]string{"message": "Sort order updated successfully"})
		return
	}
	writeJSON(w, http.StatusOK, updatedAlbum)
}

func (ah *AlbumHandler) UpdateAlbum(w http.ResponseWriter, r *http.Request) {
	identifier := chi.URLParam(r, "album_identifier")

	album, err := ah.getAlbumByIdentifier(identifier)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "Album not found"})
		} else {
			log.Printf("Error finding album '%s' for update: %v", identifier, err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to find album for update"})
		}
		return
	}

	var req struct {
		Name        *string `json:"name"`
		Description *string `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request body: " + err.Error()})
		return
	}

	nameUpdate := album.Name
	descUpdate := album.Description
	updateRequested := false
	if req.Name != nil {
		nameUpdate = *req.Name
		updateRequested = true
	}
	if req.Description != nil {
		descUpdate = *req.Description
		updateRequested = true
	}
	if !updateRequested {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "No fields provided for update"})
		return
	}

	err = database.UpdateAlbum(ah.DB, album.ID, nameUpdate, descUpdate)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// should not happen if we found it above, but handle defensively
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "Album not found during update"})
		} else if strings.Contains(err.Error(), "album name conflict") {
			writeJSON(w, http.StatusConflict, map[string]string{"error": "Album name already exists"})
		} else {
			log.Printf("Error updating album %d/%s: %v", album.ID, album.Slug, err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to update album"})
		}
		return
	}

	updatedAlbum, err := database.GetAlbumByID(ah.DB, album.ID)
	if err != nil {
		log.Printf("Error fetching updated album %d/%s: %v", album.ID, album.Slug, err)
		writeJSON(w, http.StatusOK, map[string]string{"message": "Album updated successfully"})
		return
	}
	writeJSON(w, http.StatusOK, updatedAlbum)
}

func (ah *AlbumHandler) UploadAlbumBanner(w http.ResponseWriter, r *http.Request) {
	identifier := chi.URLParam(r, "album_identifier")

	album, err := ah.getAlbumByIdentifier(identifier)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
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
		if storeErr == nil { // only attempt delete if store initialized
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

	dbErr := database.UpdateAlbumBannerPath(ah.DB, album.ID, &newBannerRelativePath)
	if dbErr != nil {
		mediaStore, storeErr := media.NewLocalStorage(ah.Cfg.MediaStoragePath, map[media.AssetType]string{})
		if storeErr == nil {
			_ = mediaStore.Delete(newBannerRelativePath)
		}
		log.Printf("Error updating banner path in DB for album %d/%s: %v", album.ID, album.Slug, dbErr)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to save banner information"})
		return
	}

	updatedAlbum, err := database.GetAlbumByID(ah.DB, album.ID)
	if err != nil {
		log.Printf("Error fetching updated album %d after banner upload: %v", album.ID, err)
		writeJSON(w, http.StatusOK, map[string]string{"message": "Banner uploaded successfully", "banner_image_path": newBannerRelativePath})
		return
	}
	writeJSON(w, http.StatusOK, updatedAlbum)
}

func (ah *AlbumHandler) RequestAlbumZipGeneration(w http.ResponseWriter, r *http.Request) {
	identifier := chi.URLParam(r, "album_identifier")

	album, err := ah.getAlbumByIdentifier(identifier)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
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

	err = database.RequestAlbumZip(ah.DB, album.ID)
	if err != nil {
		log.Printf("Error marking album zip pending for ID %d: %v", album.ID, err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to request ZIP generation"})
		return
	}

	zipJob := workers.ImageJob{
		AlbumID:     album.ID,
		TaskType:    workers.TaskAlbumZip,
		ModTimeUnix: time.Now().Unix(),
	}
	queued := ah.ThumbGen.QueueJob(zipJob) // ThumbGen is ImageProcessor
	if !queued {
		log.Printf("Failed to queue album ZIP job for Album ID %d (queue full or already pending).", album.ID)
		// TODO: for now, just inform client
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "Failed to queue ZIP generation: processing queue is full."})
		return
	}

	log.Printf("Album ZIP generation requested and queued for Album ID: %d", album.ID)
	writeJSON(w, http.StatusAccepted, map[string]string{"message": "Album ZIP generation request accepted and queued."})
}

func (ah *AlbumHandler) DownloadAlbumZip(w http.ResponseWriter, r *http.Request) {
	identifier := chi.URLParam(r, "album_identifier")

	album, err := ah.getAlbumByIdentifier(identifier)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.NotFound(w, r)
		} else {
			log.Printf("Error finding album '%s' for zip download: %v", identifier, err)
			http.Error(w, "Failed to find album", http.StatusInternalServerError)
		}
		return
	}

	if album.ZipStatus != database.StatusDone || album.ZipPath == nil || *album.ZipPath == "" {
		if album.ZipStatus == database.StatusPending || album.ZipStatus == database.StatusProcessing {
			http.Error(w, "ZIP archive is currently being generated. Please try again later.", http.StatusAccepted) // 202 Accepted
		} else if album.ZipStatus == database.StatusError && album.ZipError != nil {
			http.Error(w, fmt.Sprintf("ZIP generation failed: %s", *album.ZipError), http.StatusConflict) // 409 Conflict
		} else {
			http.Error(w, "ZIP archive not available for this album or not yet generated.", http.StatusNotFound)
		}
		return
	}

	// construct full path to the zip file
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

	modTime := fileInfo.ModTime()
	if !modTime.IsZero() {
		w.Header().Set("Last-Modified", modTime.UTC().Format(http.TimeFormat))
	}

	_, copyErr := io.Copy(w, file)
	if copyErr != nil {
		log.Printf("Error streaming ZIP file %s to client: %v", fullZipPath, copyErr)
	}
}

func (ah *AlbumHandler) DeleteAlbum(w http.ResponseWriter, r *http.Request) {
	identifier := chi.URLParam(r, "album_identifier")

	album, err := ah.getAlbumByIdentifier(identifier)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "Album not found"})
		} else {
			log.Printf("Error finding album '%s' for delete: %v", identifier, err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to find album for delete"})
		}
		return
	}

	err = database.DeleteAlbum(ah.DB, album.ID)
	if err != nil {
		log.Printf("Error deleting album %d/%s: %v", album.ID, album.Slug, err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to delete album"})
		return
	}

	// successful deletes return no content instead of a message
	writeJSON(w, http.StatusNoContent, nil)
}
