package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"

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
	AlbumRepo      repository.AlbumRepositoryInterface
	ImageRepo      repository.ImageRepositoryInterface
	UserRepo       repository.UserRepository
	RoleRepo       repository.RoleRepository
	TagRepo        repository.ImageTagRepositoryInterface
	Cfg            config.Config
	ImgProc        *workers.ImageProcessor
	Hub            *realtime.Hub
	MediaProcessor *media.Processor
	Store          *media.Store
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
	ID                 uint                 `json:"id"`
	Name               string               `json:"name"`
	Slug               string               `json:"slug"`
	Description        *string              `json:"description,omitempty"`
	FolderPath         string               `json:"folder_path"`
	Banners            []models.AlbumBanner `json:"banners"`
	SortOrder          string               `json:"sort_order"`
	ZipPath            *string              `json:"zip_path,omitempty"`
	ZipSize            *int64               `json:"zip_size,omitempty"`
	ZipStatus          string               `json:"zip_status"`
	ZipLastGeneratedAt *int64               `json:"zip_last_generated_at,omitempty"`
	ZipLastRequestedAt *int64               `json:"zip_last_requested_at,omitempty"`
	ZipError           *string              `json:"zip_error,omitempty"`
	CreatedAt          int64                `json:"created_at"`
	UpdatedAt          int64                `json:"updated_at"`
	IsHidden           bool                 `json:"is_hidden"`
	Location           *string              `json:"location,omitempty"`
	Artists            []struct {
		ID        uint   `json:"id"`
		Username  string `json:"username"`
		FirstName string `json:"first_name"`
		LastName  string `json:"last_name"`
	} `json:"artists,omitempty"`
}

// convertAlbumToAdminResponse converts a models.Album to AdminAlbumResponse
func convertAlbumToAdminResponse(album *models.Album, banners []models.AlbumBanner) *AdminAlbumResponse {
	if banners == nil {
		banners = []models.AlbumBanner{}
	}
	return &AdminAlbumResponse{
		ID:                 album.ID,
		Name:               album.Name,
		Slug:               album.Slug,
		Description:        album.Description,
		FolderPath:         album.FolderPath,
		Banners:            banners,
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

	// users who don't hold any of the global album permissions (but were allowed through the
	// route middleware via a per-album permission) should only see the albums they actually
	// have album-scoped access to.
	if user, ok := r.Context().Value(UserContextKey).(*models.User); ok && user != nil {
		hasGlobalAccess := false
		for _, p := range []string{"album.list", "album.view", "album.create", "album.edit.general", "album.delete"} {
			if user.HasGlobalPermission(p) {
				hasGlobalAccess = true
				break
			}
		}
		if !hasGlobalAccess {
			filtered := albums[:0]
			for _, album := range albums {
				if len(user.GetAlbumPermissions(album.ID)) > 0 {
					filtered = append(filtered, album)
				}
			}
			albums = filtered
		}
	}

	adminAlbums := make([]*AdminAlbumResponse, len(albums))
	for i, album := range albums {
		adminAlbums[i] = convertAlbumToAdminResponse(&album, nil)
	}

	WriteAPIResponse(w, http.StatusOK, adminAlbums)
}

// GetAlbum retrieves a single album by ID or slug for admin view
func (h *AdminAlbumHandler) GetAlbum(w http.ResponseWriter, r *http.Request) {
	identifier := chi.URLParam(r, "id")

	var album *models.Album
	var err error
	if albumID, parseErr := strconv.ParseUint(identifier, 10, 64); parseErr == nil {
		album, err = h.AlbumRepo.GetByID(uint(albumID))
		if errors.Is(err, gorm.ErrRecordNotFound) {
			album, err = h.AlbumRepo.GetBySlug(identifier)
		}
	} else {
		album, err = h.AlbumRepo.GetBySlug(identifier)
	}

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			WriteAPIError(w, http.StatusNotFound, "AlbumNotFound", "Album not found")
		} else {
			log.Printf("Error getting album %q for admin: %v", identifier, err)
			WriteAPIError(w, http.StatusInternalServerError, "AlbumFetchError", "Failed to retrieve album")
		}
		return
	}

	banners, _ := h.AlbumRepo.GetBanners(album.ID)
	adminAlbum := convertAlbumToAdminResponse(album, banners)
	// populate artists with names
	if ids, err := h.ImageRepo.GetDistinctUploaderIDsByAlbum(album.ID); err == nil && len(ids) > 0 {
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

	if req.Name == "" || req.Slug == "" {
		WriteAPIError(w, http.StatusBadRequest, "ValidationError", "Missing required fields: name and slug")
		return
	}

	if strings.ContainsAny(req.Slug, " /\\?%*:|\"<>") || strings.TrimSpace(req.Slug) == "" {
		WriteAPIError(w, http.StatusBadRequest, "ValidationError", "Invalid slug format. Use URL-safe characters without spaces.")
		return
	}

	folderPathForDB, ok := normalizeFolderPath(req.FolderPath, req.Slug)
	if !ok {
		WriteAPIError(w, http.StatusBadRequest, "ValidationError", "folder_path is invalid")
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

	err := h.AlbumRepo.Create(&newAlbum)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			WriteAPIError(w, http.StatusConflict, "AlbumConflict", "Album name, slug, or folder path already exists")
		} else {
			log.Printf("Error creating album '%s' (slug '%s'): %v", req.Name, req.Slug, err)
			WriteAPIError(w, http.StatusInternalServerError, "AlbumCreateError", "Failed to create album")
		}
		return
	}

	WriteAPIResponse(w, http.StatusCreated, convertAlbumToAdminResponse(&newAlbum, nil))
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

	updatedBanners, _ := h.AlbumRepo.GetBanners(updatedAlbum.ID)
	WriteAPIResponse(w, http.StatusOK, convertAlbumToAdminResponse(updatedAlbum, updatedBanners))
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

	existingBanners, _ := h.AlbumRepo.GetBanners(album.ID)

	imageKeys, err := h.ImageRepo.DeleteByAlbum(album.ID)
	if err != nil {
		log.Printf("Error deleting images of album %d: %v", album.ID, err)
		WriteAPIError(w, http.StatusInternalServerError, "AlbumDeleteError", "Failed to delete album images")
		return
	}

	err = h.AlbumRepo.DeleteCascade(album.ID)
	if err != nil {
		log.Printf("Error deleting album %d: %v", album.ID, err)
		WriteAPIError(w, http.StatusInternalServerError, "AlbumDeleteError", "Failed to delete album")
		return
	}

	// best-effort object cleanup
	keys := imageKeys
	for _, b := range existingBanners {
		keys = append(keys, b.ImagePath)
	}
	if album.ZipPath != nil {
		keys = append(keys, *album.ZipPath)
	}
	go h.Store.DeleteKeys(context.Background(), keys)

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

	dedup, err := h.ImageRepo.GetDistinctUploaderIDsByAlbum(album.ID)
	if err != nil {
		log.Printf("Error querying uploaders for album %d: %v", album.ID, err)
		WriteAPIError(w, http.StatusInternalServerError, "UploaderFetchError", "Failed to fetch uploaders")
		return
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
