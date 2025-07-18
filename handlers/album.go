package handlers

import (
	"encoding/json"
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

	"github.com/camden-git/mediasysbackend/config"
	"github.com/camden-git/mediasysbackend/database"
	"github.com/camden-git/mediasysbackend/media"
	"github.com/camden-git/mediasysbackend/models"
	"github.com/camden-git/mediasysbackend/repository"
	"github.com/camden-git/mediasysbackend/workers"
	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"
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
	AlbumRepo      repository.AlbumRepositoryInterface
	ImageRepo      repository.ImageRepositoryInterface
	Cfg            config.Config
	ThumbGen       *workers.ImageProcessor
	MediaProcessor *media.Processor
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

func (ah *AlbumHandler) CreateAlbum(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name        string  `json:"name"`
		Slug        string  `json:"slug"`
		FolderPath  string  `json:"folder_path"`
		Description *string `json:"description"`
		IsHidden    *bool   `json:"is_hidden"`
		Location    *string `json:"location"`
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

	newAlbumGorm := models.Album{
		Name:        req.Name,
		Slug:        req.Slug,
		Description: req.Description,
		FolderPath:  folderPathForDB,
	}
	if req.IsHidden != nil {
		newAlbumGorm.IsHidden = *req.IsHidden
	}
	if req.Location != nil {
		newAlbumGorm.Location = req.Location
	}

	err = ah.AlbumRepo.Create(&newAlbumGorm)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			writeJSON(w, http.StatusConflict, map[string]string{"error": "Album name, slug, or folder path already exists"})
		} else {
			log.Printf("Error creating album '%s' (slug '%s'): %v", req.Name, req.Slug, err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to create album"})
		}
		return
	}

	writeJSON(w, http.StatusCreated, newAlbumGorm)
}

func (ah *AlbumHandler) ListAlbums(w http.ResponseWriter, r *http.Request) {
	albums, err := ah.AlbumRepo.ListAll()
	if err != nil {
		log.Printf("Error listing albums: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to retrieve albums"})
		return
	}
	if albums == nil {
		albums = []models.Album{} // ensure an empty array instead of null for JSON
	}
	writeJSON(w, http.StatusOK, albums)
}

func (ah *AlbumHandler) GetAlbum(w http.ResponseWriter, r *http.Request) {
	identifier := chi.URLParam(r, "album_identifier")

	album, err := ah.getAlbumByIdentifier(identifier)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
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

	// Pass ah.ImageRepo to listDirectoryContents, as it expects an ImageRepositoryInterface
	fileInfos, err := listDirectoryContents(albumFullPath, "/"+album.FolderPath, ah.Cfg, ah.ImageRepo, ah.ThumbGen, album.SortOrder)
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
		if errors.Is(err, gorm.ErrRecordNotFound) {
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

	err = ah.AlbumRepo.UpdateSortOrder(album.ID, req.SortOrder)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "Album not found during update"})
		} else {
			log.Printf("Error updating sort order for album %d: %v", album.ID, err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to update sort order"})
		}
		return
	}

	updatedAlbum, err := ah.AlbumRepo.GetByID(album.ID)
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
		if errors.Is(err, gorm.ErrRecordNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "Album not found"})
		} else {
			log.Printf("Error finding album '%s' for update: %v", identifier, err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to find album for update"})
		}
		return
	}

	var req struct {
		Name        *string `json:"name"` // pointers to distinguish between empty string and not provided
		Description *string `json:"description"`
		IsHidden    *bool   `json:"is_hidden"`
		Location    *string `json:"location"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request body: " + err.Error()})
		return
	}

	var nameUpdate string
	var descUpdate *string // keep as a pointer for repository
	var isHiddenUpdate *bool
	var locationUpdate *string
	updateRequested := false

	if req.Name != nil {
		nameUpdate = *req.Name
		updateRequested = true
	} else {
		nameUpdate = album.Name
	}

	if req.Description != nil {
		descUpdate = req.Description
		updateRequested = true
	} else {
		descUpdate = album.Description
	}

	if req.IsHidden != nil {
		isHiddenUpdate = req.IsHidden
		updateRequested = true
	} else {
		isHiddenUpdate = &album.IsHidden
	}

	if req.Location != nil {
		locationUpdate = req.Location
		updateRequested = true
	} else {
		locationUpdate = album.Location
	}

	if !updateRequested {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "No fields provided for update"})
		return
	}

	err = ah.AlbumRepo.Update(album.ID, nameUpdate, descUpdate, isHiddenUpdate, locationUpdate)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "Album not found during update"})
		} else if strings.Contains(strings.ToLower(err.Error()), "unique") {
			writeJSON(w, http.StatusConflict, map[string]string{"error": "Album name already exists"})
		} else {
			log.Printf("Error updating album %d/%s: %v", album.ID, album.Slug, err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to update album"})
		}
		return
	}

	updatedAlbum, err := ah.AlbumRepo.GetByID(album.ID)
	if err != nil {
		log.Printf("Error fetching updated album %d/%s: %v", album.ID, album.Slug, err)
		writeJSON(w, http.StatusOK, map[string]string{"message": "Album updated successfully"})
		return
	}
	writeJSON(w, http.StatusOK, updatedAlbum)
}

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

	// construct the full path to the zip file
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
		log.Printf("Error copying ZIP file %s to response: %v", fullZipPath, copyErr)
		// don't return error here as the response has already started
	}
}

func (ah *AlbumHandler) DownloadAlbumZip(w http.ResponseWriter, r *http.Request) {
	identifier := chi.URLParam(r, "album_identifier")

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

	// construct the full path to the zip file
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
		if errors.Is(err, gorm.ErrRecordNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "Album not found"})
		} else {
			log.Printf("Error finding album '%s' for delete: %v", identifier, err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to find album for delete"})
		}
		return
	}

	err = ah.AlbumRepo.Delete(album.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) { // if trying to delete already deleted (by another request)
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "Album not found or already deleted"})
		} else {
			log.Printf("Error deleting album %d/%s: %v", album.ID, album.Slug, err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to delete album"})
		}
		return
	}

	writeJSON(w, http.StatusNoContent, nil)
}
