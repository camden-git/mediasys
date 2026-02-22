package handlers

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/camden-git/mediasysbackend/config"
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
	TagRepo   repository.ImageTagRepositoryInterface
	Cfg       config.Config
	ImgProc   *workers.ImageProcessor
	Hub       *realtime.Hub
}

func NewAdminAlbumHandler(
	albumRepo repository.AlbumRepositoryInterface,
	imageRepo repository.ImageRepositoryInterface,
	userRepo repository.UserRepository,
	roleRepo repository.RoleRepository,
	tagRepo repository.ImageTagRepositoryInterface,
	cfg config.Config,
	imgProc *workers.ImageProcessor,
	hub *realtime.Hub,
) *AdminAlbumHandler {
	return &AdminAlbumHandler{
		AlbumRepo: albumRepo,
		ImageRepo: imageRepo,
		UserRepo:  userRepo,
		RoleRepo:  roleRepo,
		TagRepo:   tagRepo,
		Cfg:       cfg,
		ImgProc:   imgProc,
		Hub:       hub,
	}
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
		WriteAPIError(w, http.StatusInternalServerError, "AlbumListError", "Failed to retrieve albums")
		return
	}

	adminAlbums := make([]*AdminAlbumResponse, len(albums))
	for i, album := range albums {
		adminAlbums[i] = convertAlbumToAdminResponse(&album)
	}

	WriteAPIResponse(w, http.StatusOK, adminAlbums)
}

// GetAlbum retrieves a single album by ID for admin view
func (h *AdminAlbumHandler) GetAlbum(w http.ResponseWriter, r *http.Request) {
	albumIDStr := chi.URLParam(r, "id")
	albumID, err := strconv.ParseUint(albumIDStr, 10, 64)
	if err != nil {
		WriteAPIError(w, http.StatusBadRequest, "InvalidAlbumID", "Invalid album ID")
		return
	}

	album, err := h.AlbumRepo.GetByID(uint(albumID))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			WriteAPIError(w, http.StatusNotFound, "AlbumNotFound", "Album not found")
		} else {
			log.Printf("Error getting album %d for admin: %v", albumID, err)
			WriteAPIError(w, http.StatusInternalServerError, "AlbumFetchError", "Failed to retrieve album")
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
	WriteAPIResponse(w, http.StatusOK, adminAlbum)
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
		WriteAPIError(w, http.StatusBadRequest, "InvalidPayload", "Invalid request body: "+err.Error())
		return
	}

	if req.Name == "" || req.FolderPath == "" || req.Slug == "" {
		WriteAPIError(w, http.StatusBadRequest, "ValidationError", "Missing required fields: name, slug, and folder_path")
		return
	}

	if strings.ContainsAny(req.Slug, " /\\?%*:|\"<>") || strings.TrimSpace(req.Slug) == "" {
		WriteAPIError(w, http.StatusBadRequest, "ValidationError", "Invalid slug format. Use URL-safe characters without spaces.")
		return
	}

	cleanRelativePath := filepath.Clean(req.FolderPath)
	if filepath.IsAbs(cleanRelativePath) || strings.HasPrefix(cleanRelativePath, "..") {
		WriteAPIError(w, http.StatusBadRequest, "ValidationError", "folder_path must be relative and cannot use '..'")
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
			WriteAPIError(w, http.StatusInternalServerError, "FolderCreateError", "Could not create folder_path")
			return
		}
		log.Printf("Created folder path: %s", fullPath)
	} else if err != nil {
		log.Printf("Error stating folder path %s during album creation: %v", fullPath, err)
		WriteAPIError(w, http.StatusInternalServerError, "FolderError", "Could not verify folder_path")
		return
	} else if !stat.IsDir() {
		WriteAPIError(w, http.StatusBadRequest, "ValidationError", "folder_path is not a directory: "+folderPathForDB)
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
			WriteAPIError(w, http.StatusConflict, "AlbumConflict", "Album name, slug, or folder path already exists")
		} else {
			log.Printf("Error creating album '%s' (slug '%s'): %v", req.Name, req.Slug, err)
			WriteAPIError(w, http.StatusInternalServerError, "AlbumCreateError", "Failed to create album")
		}
		return
	}

	WriteAPIResponse(w, http.StatusCreated, convertAlbumToAdminResponse(&newAlbum))
}

// UpdateAlbum updates an existing album's settings (name, description, hidden status, location, sort order)
func (h *AdminAlbumHandler) UpdateAlbum(w http.ResponseWriter, r *http.Request) {
	albumIDStr := chi.URLParam(r, "id")
	albumID, err := strconv.ParseUint(albumIDStr, 10, 64)
	if err != nil {
		WriteAPIError(w, http.StatusBadRequest, "InvalidAlbumID", "Invalid album ID")
		return
	}

	album, err := h.AlbumRepo.GetByID(uint(albumID))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			WriteAPIError(w, http.StatusNotFound, "AlbumNotFound", "Album not found")
		} else {
			log.Printf("Error finding album %d for update: %v", albumID, err)
			WriteAPIError(w, http.StatusInternalServerError, "AlbumFetchError", "Failed to find album for update")
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
		WriteAPIError(w, http.StatusBadRequest, "InvalidPayload", "Invalid request body: "+err.Error())
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
				WriteAPIError(w, http.StatusNotFound, "AlbumNotFound", "Album not found during update")
			} else if strings.Contains(strings.ToLower(err.Error()), "unique") {
				WriteAPIError(w, http.StatusConflict, "AlbumConflict", "Album name already exists")
			} else {
				log.Printf("Error updating album %d/%s: %v", album.ID, album.Slug, err)
				WriteAPIError(w, http.StatusInternalServerError, "AlbumUpdateError", "Failed to update album")
			}
			return
		}
	}

	if req.SortOrder != nil {
		err = h.AlbumRepo.UpdateSortOrder(album.ID, *req.SortOrder)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				WriteAPIError(w, http.StatusNotFound, "AlbumNotFound", "Album not found during sort order update")
			} else {
				log.Printf("Error updating sort order for album %d/%s: %v", album.ID, album.Slug, err)
				WriteAPIError(w, http.StatusInternalServerError, "AlbumUpdateError", "Failed to update sort order")
			}
			return
		}
	}

	updatedAlbum, err := h.AlbumRepo.GetByID(album.ID)
	if err != nil {
		log.Printf("Error fetching updated album %d/%s: %v", album.ID, album.Slug, err)
		WriteAPIResponse(w, http.StatusOK, map[string]string{"message": "Album updated successfully"})
		return
	}

	WriteAPIResponse(w, http.StatusOK, convertAlbumToAdminResponse(updatedAlbum))
}

// DeleteAlbum deletes an album
func (h *AdminAlbumHandler) DeleteAlbum(w http.ResponseWriter, r *http.Request) {
	albumIDStr := chi.URLParam(r, "id")
	albumID, err := strconv.ParseUint(albumIDStr, 10, 64)
	if err != nil {
		WriteAPIError(w, http.StatusBadRequest, "InvalidAlbumID", "Invalid album ID")
		return
	}

	album, err := h.AlbumRepo.GetByID(uint(albumID))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			WriteAPIError(w, http.StatusNotFound, "AlbumNotFound", "Album not found")
		} else {
			log.Printf("Error finding album %d for delete: %v", albumID, err)
			WriteAPIError(w, http.StatusInternalServerError, "AlbumFetchError", "Failed to find album for delete")
		}
		return
	}

	folderPrefix := strings.TrimRight(album.FolderPath, "/") + "/"

	// Cast to concrete type to access raw DB (same pattern as DeleteAlbumImage)
	albumRepo, ok := h.AlbumRepo.(*repository.AlbumRepository)
	if !ok {
		WriteAPIError(w, http.StatusInternalServerError, "InternalError", "repository type assertion failed")
		return
	}
	db := albumRepo.DB

	// Gather generated asset paths before deleting DB rows
	type assetPaths struct{ thumb, preview *string }
	var imagePaths []assetPaths
	var imgs []models.Image
	db.Unscoped().Where("original_path LIKE ?", folderPrefix+"%").
		Select("thumbnail_path, preview_path").Find(&imgs)
	for _, img := range imgs {
		imagePaths = append(imagePaths, assetPaths{img.ThumbnailPath, img.PreviewPath})
	}

	// Single transaction: embeddings → faces → images → album (all hard deletes)
	err = db.Transaction(func(tx *gorm.DB) error {
		// 1. Collect face IDs for images in this album
		var faceIDs []uint
		if err := tx.Unscoped().Model(&models.Face{}).
			Where("image_path LIKE ?", folderPrefix+"%").
			Pluck("id", &faceIDs).Error; err != nil {
			return err
		}

		// 2. Hard-delete face embeddings
		if len(faceIDs) > 0 {
			if err := tx.Unscoped().Where("face_id IN ?", faceIDs).
				Delete(&models.FaceEmbedding{}).Error; err != nil {
				return err
			}
		}

		// 3. Hard-delete faces (person rows are untouched)
		if err := tx.Unscoped().Where("image_path LIKE ?", folderPrefix+"%").
			Delete(&models.Face{}).Error; err != nil {
			return err
		}

		// 4. Hard-delete images
		if err := tx.Unscoped().Where("original_path LIKE ?", folderPrefix+"%").
			Delete(&models.Image{}).Error; err != nil {
			return err
		}

		// 5. Hard-delete the album record itself
		if err := tx.Unscoped().Delete(&models.Album{}, album.ID).Error; err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		log.Printf("Error deleting album %d: %v", album.ID, err)
		WriteAPIError(w, http.StatusInternalServerError, "AlbumDeleteError", "Failed to delete album")
		return
	}

	// Best-effort filesystem cleanup of generated assets
	for _, p := range imagePaths {
		if p.thumb != nil && *p.thumb != "" {
			os.Remove(filepath.Join(h.Cfg.MediaStoragePath, filepath.FromSlash(*p.thumb)))
		}
		if p.preview != nil && *p.preview != "" {
			os.Remove(filepath.Join(h.Cfg.MediaStoragePath, filepath.FromSlash(*p.preview)))
		}
	}
	if album.BannerImagePath != nil && *album.BannerImagePath != "" {
		os.Remove(filepath.Join(h.Cfg.MediaStoragePath, filepath.FromSlash(*album.BannerImagePath)))
	}
	if album.ZipPath != nil && *album.ZipPath != "" {
		os.Remove(filepath.Join(h.Cfg.MediaStoragePath, filepath.FromSlash(*album.ZipPath)))
	}

	w.WriteHeader(http.StatusNoContent)
}

// GetAlbumUploaders returns distinct users who uploaded images within the album's folder
func (h *AdminAlbumHandler) GetAlbumUploaders(w http.ResponseWriter, r *http.Request) {
	albumIDStr := chi.URLParam(r, "id")
	albumID, err := strconv.ParseUint(albumIDStr, 10, 64)
	if err != nil {
		WriteAPIError(w, http.StatusBadRequest, "InvalidAlbumID", "Invalid album ID")
		return
	}

	album, err := h.AlbumRepo.GetByID(uint(albumID))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			WriteAPIError(w, http.StatusNotFound, "AlbumNotFound", "Album not found")
		} else {
			log.Printf("Error fetching album %d for uploaders: %v", albumID, err)
			WriteAPIError(w, http.StatusInternalServerError, "AlbumFetchError", "Failed to fetch album")
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
		WriteAPIError(w, http.StatusInternalServerError, "UploaderFetchError", "Failed to fetch uploaders")
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

	WriteAPIResponse(w, http.StatusOK, users)
}
