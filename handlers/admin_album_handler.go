package handlers

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/camden-git/mediasysbackend/config"
	"github.com/camden-git/mediasysbackend/media"
	"github.com/camden-git/mediasysbackend/models"
	"github.com/camden-git/mediasysbackend/realtime"
	"github.com/camden-git/mediasysbackend/repository"
	"github.com/camden-git/mediasysbackend/workers"
	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"
)

type AdminAlbumHandler struct {
	AlbumRepo repository.AlbumRepositoryInterface
	ImageRepo repository.ImageRepositoryInterface
	UserRepo  repository.UserRepository
	RoleRepo  repository.RoleRepository
	Cfg       config.Config
	ImgProc   *workers.ImageProcessor
	Hub       *realtime.Hub
}

func NewAdminAlbumHandler(
	albumRepo repository.AlbumRepositoryInterface,
	imageRepo repository.ImageRepositoryInterface,
	userRepo repository.UserRepository,
	roleRepo repository.RoleRepository,
	cfg config.Config,
	imgProc *workers.ImageProcessor,
	hub *realtime.Hub,
) *AdminAlbumHandler {
	return &AdminAlbumHandler{
		AlbumRepo: albumRepo,
		ImageRepo: imageRepo,
		UserRepo:  userRepo,
		RoleRepo:  roleRepo,
		Cfg:       cfg,
		ImgProc:   imgProc,
		Hub:       hub,
	}
}

// UploadImages handles multipart folder or multiple file uploads into the album's folder and queues processing
func (h *AdminAlbumHandler) UploadImages(w http.ResponseWriter, r *http.Request) {
	albumIDStr := chi.URLParam(r, "id")
	albumID, err := strconv.ParseUint(albumIDStr, 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid album ID"})
		return
	}

	album, err := h.AlbumRepo.GetByID(uint(albumID))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "Album not found"})
		} else {
			log.Printf("Error fetching album %d for upload: %v", albumID, err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to fetch album"})
		}
		return
	}

	if h.ImgProc == nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Image processor not configured"})
		return
	}

	reader, err := r.MultipartReader()
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid multipart form: " + err.Error()})
		return
	}

	albumBase := filepath.Join(h.Cfg.RootDirectory, album.FolderPath)
	if err := os.MkdirAll(albumBase, 0755); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to ensure album folder"})
		return
	}

	var relPathsQueue []string
	saved := 0
	for {
		part, err := reader.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Printf("UploadImages: error reading part: %v", err)
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Malformed upload data"})
			return
		}

		field := part.FormName()
		if field == "relative_path" {
			data, _ := io.ReadAll(part)
			rp := strings.TrimSpace(string(data))
			rp = filepath.Clean(rp)
			rp = filepath.ToSlash(rp)
			// prevent path escape
			rp = strings.TrimPrefix(rp, "./")
			rp = strings.TrimPrefix(rp, "/")
			relPathsQueue = append(relPathsQueue, rp)
			continue
		}

		if field != "files" {
			// ignore unknown fields
			continue
		}

		filename := part.FileName()
		rel := filename
		if len(relPathsQueue) > 0 {
			rel = relPathsQueue[0]
			relPathsQueue = relPathsQueue[1:]
		}
		if rel == "" {
			rel = filename
		}
		rel = filepath.Clean(rel)
		rel = filepath.ToSlash(rel)
		rel = strings.TrimPrefix(rel, "./")
		rel = strings.TrimPrefix(rel, "/")
		// strip top-level folder (e.g., `todo/`) from webkitRelativePath so files land at album root
		if idx := strings.Index(rel, "/"); idx >= 0 {
			rel = rel[idx+1:]
		}

		destPath := filepath.Join(albumBase, rel)
		// security: ensure inside albumBase
		if !strings.HasPrefix(filepath.Clean(destPath), filepath.Clean(albumBase)) {
			log.Printf("UploadImages: blocked path traversal: %s", destPath)
			continue
		}
		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			log.Printf("UploadImages: mkdir error for %s: %v", destPath, err)
			continue
		}

		out, err := os.Create(destPath)
		if err != nil {
			log.Printf("UploadImages: create error for %s: %v", destPath, err)
			// broadcast error
			if h.Hub != nil {
				relFromRoot, _ := filepath.Rel(h.Cfg.RootDirectory, destPath)
				h.Hub.Broadcast(realtime.Event{Type: "upload", Path: filepath.ToSlash(relFromRoot), Status: "error", Error: err.Error(), Timestamp: time.Now().Unix()})
			}
			continue
		}
		// compute db key before copy for consistent events
		relFromRoot, err := filepath.Rel(h.Cfg.RootDirectory, destPath)
		if err == nil && h.Hub != nil {
			h.Hub.Broadcast(realtime.Event{Type: "upload", Path: filepath.ToSlash(relFromRoot), Status: "uploading", Timestamp: time.Now().Unix()})
		}

		if _, err := io.Copy(out, part); err != nil {
			log.Printf("UploadImages: write error for %s: %v", destPath, err)
			out.Close()
			if h.Hub != nil && relFromRoot != "" {
				h.Hub.Broadcast(realtime.Event{Type: "upload", Path: filepath.ToSlash(relFromRoot), Status: "error", Error: err.Error(), Timestamp: time.Now().Unix()})
			}
			continue
		}
		out.Close()

		// Ensure DB record and queue processing if raster image
		// Compute DB key relative to root
		relFromRoot, err = filepath.Rel(h.Cfg.RootDirectory, destPath)
		if err != nil {
			log.Printf("UploadImages: failed to compute relative path for %s: %v", destPath, err)
			continue
		}
		relDBKey := filepath.ToSlash(relFromRoot)

		if h.Hub != nil {
			h.Hub.Broadcast(realtime.Event{Type: "upload", Path: relDBKey, Status: "uploaded", Timestamp: time.Now().Unix()})
		}

		info, err := os.Stat(destPath)
		if err != nil {
			log.Printf("UploadImages: stat error for %s: %v", destPath, err)
			continue
		}

		// Only queue tasks for raster images
		if media.IsRasterImage(destPath) {
			var uploadedBy *uint
			if user, ok := r.Context().Value(UserContextKey).(*models.User); ok && user != nil {
				uploadedBy = &user.ID
			}
			if _, err := h.ImageRepo.EnsureExistsWithUploader(relDBKey, info.ModTime().Unix(), uploadedBy); err != nil {
				log.Printf("UploadImages: EnsureExists error for %s: %v", relDBKey, err)
			}
			baseJob := workers.ImageJob{OriginalImagePath: destPath, OriginalRelativePath: relDBKey, ModTimeUnix: info.ModTime().Unix()}
			// Queue tasks
			for _, task := range []string{workers.TaskThumbnail, workers.TaskMetadata, workers.TaskDetection} {
				job := baseJob
				job.TaskType = task
				h.ImgProc.QueueJob(job)
			}
		}

		saved++
	}

	writeJSON(w, http.StatusCreated, map[string]any{"uploaded": saved})
}

// AdminAlbumResponse represents the admin view of an album with additional fields
type AdminAlbumResponse struct {
	ID                 uint    `json:"id"`
	Name               string  `json:"name"`
	Slug               string  `json:"slug"`
	Description        *string `json:"description,omitempty"`
	FolderPath         string  `json:"folder_path"`
	BannerImagePath    *string `json:"banner_image_path,omitempty"`
	SortOrder          string  `json:"sort_order"`
	ZipPath            *string `json:"zip_path,omitempty"`
	ZipSize            *int64  `json:"zip_size,omitempty"`
	ZipStatus          string  `json:"zip_status"`
	ZipLastGeneratedAt *int64  `json:"zip_last_generated_at,omitempty"`
	ZipLastRequestedAt *int64  `json:"zip_last_requested_at,omitempty"`
	ZipError           *string `json:"zip_error,omitempty"`
	CreatedAt          int64   `json:"created_at"`
	UpdatedAt          int64   `json:"updated_at"`
	IsHidden           bool    `json:"is_hidden"`
	Location           *string `json:"location,omitempty"`
	Artists            []struct {
		ID        uint   `json:"id"`
		Username  string `json:"username"`
		FirstName string `json:"first_name"`
		LastName  string `json:"last_name"`
	} `json:"artists,omitempty"`
}

// convertAlbumToAdminResponse converts a models.Album to AdminAlbumResponse
func convertAlbumToAdminResponse(album *models.Album) *AdminAlbumResponse {
	return &AdminAlbumResponse{
		ID:                 album.ID,
		Name:               album.Name,
		Slug:               album.Slug,
		Description:        album.Description,
		FolderPath:         album.FolderPath,
		BannerImagePath:    album.BannerImagePath,
		SortOrder:          album.SortOrder,
		ZipPath:            album.ZipPath,
		ZipSize:            album.ZipSize,
		ZipStatus:          album.ZipStatus,
		ZipLastGeneratedAt: album.ZipLastGeneratedAt,
		ZipLastRequestedAt: album.ZipLastRequestedAt,
		ZipError:           album.ZipError,
		CreatedAt:          album.CreatedAt,
		UpdatedAt:          album.UpdatedAt,
		IsHidden:           album.IsHidden,
		Location:           album.Location,
	}
}

// ListAlbums retrieves all albums (including hidden ones) for admin view
func (h *AdminAlbumHandler) ListAlbums(w http.ResponseWriter, r *http.Request) {
	albums, err := h.AlbumRepo.ListAllAdmin()
	if err != nil {
		log.Printf("Error listing albums for admin: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to retrieve albums"})
		return
	}

	adminAlbums := make([]*AdminAlbumResponse, len(albums))
	for i, album := range albums {
		adminAlbums[i] = convertAlbumToAdminResponse(&album)
	}

	writeJSON(w, http.StatusOK, adminAlbums)
}

// GetAlbum retrieves a single album by ID for admin view
func (h *AdminAlbumHandler) GetAlbum(w http.ResponseWriter, r *http.Request) {
	albumIDStr := chi.URLParam(r, "id")
	albumID, err := strconv.ParseUint(albumIDStr, 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid album ID"})
		return
	}

	album, err := h.AlbumRepo.GetByID(uint(albumID))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "Album not found"})
		} else {
			log.Printf("Error getting album %d for admin: %v", albumID, err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to retrieve album"})
		}
		return
	}

	adminAlbum := convertAlbumToAdminResponse(album)
	// populate artists with names
	if ids, err := h.ImageRepo.GetDistinctUploaderIDsByFolderPrefix(album.FolderPath); err == nil && len(ids) > 0 {
		for _, id := range ids {
			if u, err := h.UserRepo.GetByID(id); err == nil && u != nil {
				adminAlbum.Artists = append(adminAlbum.Artists, struct {
					ID        uint   `json:"id"`
					Username  string `json:"username"`
					FirstName string `json:"first_name"`
					LastName  string `json:"last_name"`
				}{ID: u.ID, Username: u.Username, FirstName: u.FirstName, LastName: u.LastName})
			}
		}
	}
	writeJSON(w, http.StatusOK, adminAlbum)
}

// CreateAlbum creates a new album
func (h *AdminAlbumHandler) CreateAlbum(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name        string  `json:"name"`
		Slug        string  `json:"slug"`
		FolderPath  string  `json:"folder_path"`
		Description *string `json:"description"`
		IsHidden    *bool   `json:"is_hidden"`
		Location    *string `json:"location"`
		SortOrder   *string `json:"sort_order"`
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
	fullPath := filepath.Join(h.Cfg.RootDirectory, folderPathForDB)
	stat, err := os.Stat(fullPath)
	if os.IsNotExist(err) {
		// create the directory if it doesn't exist
		err = os.MkdirAll(fullPath, 0755)
		if err != nil {
			log.Printf("Error creating folder path %s during album creation: %v", fullPath, err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Could not create folder_path"})
			return
		}
		log.Printf("Created folder path: %s", fullPath)
	} else if err != nil {
		log.Printf("Error stating folder path %s during album creation: %v", fullPath, err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Could not verify folder_path"})
		return
	} else if !stat.IsDir() {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "folder_path is not a directory: " + folderPathForDB})
		return
	}

	newAlbum := models.Album{
		Name:        req.Name,
		Slug:        req.Slug,
		Description: req.Description,
		FolderPath:  folderPathForDB,
	}
	if req.IsHidden != nil {
		newAlbum.IsHidden = *req.IsHidden
	}
	if req.Location != nil {
		newAlbum.Location = req.Location
	}
	if req.SortOrder != nil {
		newAlbum.SortOrder = *req.SortOrder
	}

	err = h.AlbumRepo.Create(&newAlbum)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			writeJSON(w, http.StatusConflict, map[string]string{"error": "Album name, slug, or folder path already exists"})
		} else {
			log.Printf("Error creating album '%s' (slug '%s'): %v", req.Name, req.Slug, err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to create album"})
		}
		return
	}

	adminAlbum := convertAlbumToAdminResponse(&newAlbum)
	writeJSON(w, http.StatusCreated, adminAlbum)
}

// UpdateAlbum updates an existing album's settings (name, description, hidden status, location, sort order)
func (h *AdminAlbumHandler) UpdateAlbum(w http.ResponseWriter, r *http.Request) {
	albumIDStr := chi.URLParam(r, "id")
	albumID, err := strconv.ParseUint(albumIDStr, 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid album ID"})
		return
	}

	album, err := h.AlbumRepo.GetByID(uint(albumID))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "Album not found"})
		} else {
			log.Printf("Error finding album %d for update: %v", albumID, err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to find album for update"})
		}
		return
	}

	var req struct {
		Name        *string `json:"name"`
		Description *string `json:"description"`
		IsHidden    *bool   `json:"is_hidden"`
		Location    *string `json:"location"`
		SortOrder   *string `json:"sort_order"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request body: " + err.Error()})
		return
	}

	var nameUpdate string
	var descUpdate *string
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

	if updateRequested {
		err = h.AlbumRepo.Update(album.ID, nameUpdate, descUpdate, isHiddenUpdate, locationUpdate)
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
	}

	if req.SortOrder != nil {
		err = h.AlbumRepo.UpdateSortOrder(album.ID, *req.SortOrder)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				writeJSON(w, http.StatusNotFound, map[string]string{"error": "Album not found during sort order update"})
			} else {
				log.Printf("Error updating sort order for album %d/%s: %v", album.ID, album.Slug, err)
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to update sort order"})
			}
			return
		}
	}

	updatedAlbum, err := h.AlbumRepo.GetByID(album.ID)
	if err != nil {
		log.Printf("Error fetching updated album %d/%s: %v", album.ID, album.Slug, err)
		writeJSON(w, http.StatusOK, map[string]string{"message": "Album updated successfully"})
		return
	}

	adminAlbum := convertAlbumToAdminResponse(updatedAlbum)
	writeJSON(w, http.StatusOK, adminAlbum)
}

// DeleteAlbum deletes an album
func (h *AdminAlbumHandler) DeleteAlbum(w http.ResponseWriter, r *http.Request) {
	albumIDStr := chi.URLParam(r, "id")
	albumID, err := strconv.ParseUint(albumIDStr, 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid album ID"})
		return
	}

	album, err := h.AlbumRepo.GetByID(uint(albumID))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "Album not found"})
		} else {
			log.Printf("Error finding album %d for delete: %v", albumID, err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to find album for delete"})
		}
		return
	}

	err = h.AlbumRepo.Delete(album.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "Album not found or already deleted"})
		} else {
			log.Printf("Error deleting album %d/%s: %v", album.ID, album.Slug, err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to delete album"})
		}
		return
	}

	writeJSON(w, http.StatusNoContent, nil)
}

// GetAlbumUploaders returns distinct users who uploaded images within the album's folder
func (h *AdminAlbumHandler) GetAlbumUploaders(w http.ResponseWriter, r *http.Request) {
	albumIDStr := chi.URLParam(r, "id")
	albumID, err := strconv.ParseUint(albumIDStr, 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid album ID"})
		return
	}

	album, err := h.AlbumRepo.GetByID(uint(albumID))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "Album not found"})
		} else {
			log.Printf("Error fetching album %d for uploaders: %v", albumID, err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to fetch album"})
		}
		return
	}

	// Query distinct uploader IDs from images table where path is under album folder
	type row struct{ UploadedByUserID *uint }
	var rows []row
	likePrefix := album.FolderPath + "/%"
	if err := h.ImageRepo.(*repository.ImageRepository).DB.Model(&models.Image{}).
		Select("uploaded_by_user_id").
		Where("original_path LIKE ? AND uploaded_by_user_id IS NOT NULL", likePrefix).
		Distinct().
		Find(&rows).Error; err != nil {
		log.Printf("Error querying uploaders for album %d: %v", album.ID, err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to fetch uploaders"})
		return
	}

	uploaderIDs := make([]uint, 0, len(rows))
	for _, rrow := range rows {
		if rrow.UploadedByUserID != nil {
			uploaderIDs = append(uploaderIDs, *rrow.UploadedByUserID)
		}
	}
	// Deduplicate (Distinct should already, but ensure)
	idSeen := map[uint]bool{}
	dedup := make([]uint, 0, len(uploaderIDs))
	for _, id := range uploaderIDs {
		if !idSeen[id] {
			idSeen[id] = true
			dedup = append(dedup, id)
		}
	}

	type UserLite struct {
		ID       uint   `json:"id"`
		Username string `json:"username"`
	}

	users := make([]UserLite, 0, len(dedup))
	for _, id := range dedup {
		u, err := h.UserRepo.GetByID(id)
		if err == nil && u != nil {
			users = append(users, UserLite{ID: u.ID, Username: u.Username})
		}
	}

	writeJSON(w, http.StatusOK, users)
}

// ListAlbumImages lists files within the album folder for admin, including metadata and thumbnail info
func (h *AdminAlbumHandler) ListAlbumImages(w http.ResponseWriter, r *http.Request) {
	albumIDStr := chi.URLParam(r, "id")
	albumID, err := strconv.ParseUint(albumIDStr, 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid album ID"})
		return
	}

	album, err := h.AlbumRepo.GetByID(uint(albumID))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "Album not found"})
		} else {
			log.Printf("Error getting album %d for image listing: %v", albumID, err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to retrieve album"})
		}
		return
	}

	albumFullPath := filepath.Join(h.Cfg.RootDirectory, album.FolderPath)
	albumFullPath = filepath.Clean(albumFullPath)
	if !strings.HasPrefix(albumFullPath, h.Cfg.RootDirectory) {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Album configuration error"})
		return
	}

    files, totalCount, err := listDirectoryContents(albumFullPath, "/"+album.FolderPath, h.Cfg, h.ImageRepo, h.ImgProc, album.SortOrder, -1, -1)
	if err != nil {
		if os.IsNotExist(err) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "Album folder not found on disk: " + album.FolderPath})
		} else if os.IsPermission(err) {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "Permission denied accessing album folder"})
		} else {
			log.Printf("Error listing contents for album %d (%s): %v", album.ID, album.FolderPath, err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to list album contents"})
		}
		return
	}

    writeJSON(w, http.StatusOK, DirectoryListing{Path: "/" + album.FolderPath, Files: files, Total: totalCount, Offset: 0, Limit: len(files), HasMore: false})
}

// DeleteAlbumImage deletes a single image file within an album and removes DB records and generated assets
func (h *AdminAlbumHandler) DeleteAlbumImage(w http.ResponseWriter, r *http.Request) {
	albumIDStr := chi.URLParam(r, "id")
	albumID, err := strconv.ParseUint(albumIDStr, 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid album ID"})
		return
	}

	album, err := h.AlbumRepo.GetByID(uint(albumID))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "Album not found"})
		} else {
			log.Printf("Error getting album %d for image delete: %v", albumID, err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to retrieve album"})
		}
		return
	}

	relPath := r.URL.Query().Get("path")
	if relPath == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Missing 'path' query parameter"})
		return
	}
	// Normalize to forward slashes and strip any leading slash
	relPath = filepath.ToSlash(strings.TrimPrefix(relPath, "/"))
	// Security: ensure the path is under the album folder
	if !(relPath == album.FolderPath || strings.HasPrefix(relPath, album.FolderPath+"/")) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "Image path is not within the specified album"})
		return
	}

	// Try to get image DB record to find generated thumbnail path
	var existingThumbPath *string
	if img, err := h.ImageRepo.GetByPath(relPath); err == nil && img != nil {
		existingThumbPath = img.ThumbnailPath
	}

	// Delete the original file from disk
	fullPath := filepath.Join(h.Cfg.RootDirectory, relPath)
	if err := os.Remove(fullPath); err != nil && !os.IsNotExist(err) {
		log.Printf("Error deleting original image '%s': %v", fullPath, err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to delete original image"})
		return
	}

	// Best-effort delete of generated thumbnail asset if known
	if existingThumbPath != nil && *existingThumbPath != "" {
		thumbFull := filepath.Join(h.Cfg.MediaStoragePath, filepath.FromSlash(*existingThumbPath))
		if err := os.Remove(thumbFull); err != nil && !os.IsNotExist(err) {
			log.Printf("Warning: failed to delete thumbnail asset '%s': %v", thumbFull, err)
		}
	}

	// Delete DB records: image row, faces and embeddings for this image using GORM and a transaction
	if repo, ok := h.ImageRepo.(*repository.ImageRepository); ok && repo != nil {
		if err := repo.DB.Transaction(func(tx *gorm.DB) error {
			var faceIDs []uint
			if err := tx.Model(&models.Face{}).Where("image_path = ?", relPath).Pluck("id", &faceIDs).Error; err != nil {
				return err
			}
			if len(faceIDs) > 0 {
				if err := tx.Where("face_id IN ?", faceIDs).Delete(&models.FaceEmbedding{}).Error; err != nil {
					return err
				}
			}
			if err := tx.Where("image_path = ?", relPath).Delete(&models.Face{}).Error; err != nil {
				return err
			}
			if err := tx.Where("original_path = ?", relPath).Delete(&models.Image{}).Error; err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
			return nil
		}); err != nil {
			log.Printf("Error deleting image and related records for %s: %v", relPath, err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to delete image record"})
			return
		}
	} else {
		if err := h.ImageRepo.Delete(relPath); err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			log.Printf("Error deleting image DB record %s: %v", relPath, err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to delete image record"})
			return
		}
	}

	writeJSON(w, http.StatusNoContent, nil)
}
