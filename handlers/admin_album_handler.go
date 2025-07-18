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
	"github.com/camden-git/mediasysbackend/repository"
	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"
)

type AdminAlbumHandler struct {
	AlbumRepo repository.AlbumRepositoryInterface
	ImageRepo repository.ImageRepositoryInterface
	UserRepo  repository.UserRepository
	RoleRepo  repository.RoleRepository
	Cfg       config.Config
}

func NewAdminAlbumHandler(
	albumRepo repository.AlbumRepositoryInterface,
	imageRepo repository.ImageRepositoryInterface,
	userRepo repository.UserRepository,
	roleRepo repository.RoleRepository,
	cfg config.Config,
) *AdminAlbumHandler {
	return &AdminAlbumHandler{
		AlbumRepo: albumRepo,
		ImageRepo: imageRepo,
		UserRepo:  userRepo,
		RoleRepo:  roleRepo,
		Cfg:       cfg,
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
