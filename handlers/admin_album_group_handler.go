package handlers

import (
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/camden-git/mediasysbackend/media"
	"github.com/camden-git/mediasysbackend/models"
	"github.com/camden-git/mediasysbackend/repository"
	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"
)

// AdminAlbumGroupHandler handles admin CRUD for album groups.
type AdminAlbumGroupHandler struct {
	GroupRepo      repository.AlbumGroupRepositoryInterface
	MediaProcessor *media.Processor
	Store          *media.Store
}

// NewAdminAlbumGroupHandler creates a new AdminAlbumGroupHandler.
func NewAdminAlbumGroupHandler(
	groupRepo repository.AlbumGroupRepositoryInterface,
	mediaProcessor *media.Processor,
	store *media.Store,
) *AdminAlbumGroupHandler {
	return &AdminAlbumGroupHandler{
		GroupRepo:      groupRepo,
		MediaProcessor: mediaProcessor,
		Store:          store,
	}
}

// ListGroups lists all album groups (including hidden).
func (h *AdminAlbumGroupHandler) ListGroups(w http.ResponseWriter, r *http.Request) {
	groups, err := h.GroupRepo.ListAllAdmin()
	if err != nil {
		log.Printf("Error listing album groups for admin: %v", err)
		WriteAPIError(w, http.StatusInternalServerError, "GroupListError", "Failed to retrieve album groups")
		return
	}
	WriteAPIResponse(w, http.StatusOK, groups)
}

// GetGroup returns a single album group by ID.
func (h *AdminAlbumGroupHandler) GetGroup(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		WriteAPIError(w, http.StatusBadRequest, "InvalidID", "Invalid group ID")
		return
	}

	group, err := h.GroupRepo.GetByID(uint(id))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			WriteAPIError(w, http.StatusNotFound, "GroupNotFound", "Album group not found")
		} else {
			log.Printf("Error getting album group %d for admin: %v", id, err)
			WriteAPIError(w, http.StatusInternalServerError, "GroupFetchError", "Failed to retrieve album group")
		}
		return
	}
	WriteAPIResponse(w, http.StatusOK, group)
}

// CreateGroup creates a new album group.
func (h *AdminAlbumGroupHandler) CreateGroup(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name        string  `json:"name"`
		Slug        string  `json:"slug"`
		Description *string `json:"description"`
		IsHidden    *bool   `json:"is_hidden"`
	}
	if err := decodeJSONBody(w, r, &req); err != nil {
		if isBodyTooLarge(err) {
			WriteAPIError(w, http.StatusRequestEntityTooLarge, "PayloadTooLarge", "Request body is too large")
			return
		}
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

	group := &models.AlbumGroup{
		Name:        req.Name,
		Slug:        req.Slug,
		Description: req.Description,
	}
	if req.IsHidden != nil {
		group.IsHidden = *req.IsHidden
	}

	if err := h.GroupRepo.Create(group); err != nil {
		if isUniqueViolation(err) {
			WriteAPIError(w, http.StatusConflict, "GroupConflict", "Album group name or slug already exists")
		} else {
			log.Printf("Error creating album group '%s': %v", req.Name, err)
			WriteAPIError(w, http.StatusInternalServerError, "GroupCreateError", "Failed to create album group")
		}
		return
	}

	WriteAPIResponse(w, http.StatusCreated, group)
}

// UpdateGroup updates an existing album group.
func (h *AdminAlbumGroupHandler) UpdateGroup(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		WriteAPIError(w, http.StatusBadRequest, "InvalidID", "Invalid group ID")
		return
	}

	group, err := h.GroupRepo.GetByID(uint(id))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			WriteAPIError(w, http.StatusNotFound, "GroupNotFound", "Album group not found")
		} else {
			log.Printf("Error finding album group %d for update: %v", id, err)
			WriteAPIError(w, http.StatusInternalServerError, "GroupFetchError", "Failed to find album group")
		}
		return
	}

	var req struct {
		Name        *string `json:"name"`
		Slug        *string `json:"slug"`
		Description *string `json:"description"`
		IsHidden    *bool   `json:"is_hidden"`
	}
	if err := decodeJSONBody(w, r, &req); err != nil {
		if isBodyTooLarge(err) {
			WriteAPIError(w, http.StatusRequestEntityTooLarge, "PayloadTooLarge", "Request body is too large")
			return
		}
		WriteAPIError(w, http.StatusBadRequest, "InvalidPayload", "Invalid request body: "+err.Error())
		return
	}

	name := group.Name
	slug := group.Slug
	description := group.Description
	isHidden := group.IsHidden

	if req.Name != nil {
		name = *req.Name
	}
	if req.Slug != nil {
		if strings.ContainsAny(*req.Slug, " /\\?%*:|\"<>") || strings.TrimSpace(*req.Slug) == "" {
			WriteAPIError(w, http.StatusBadRequest, "ValidationError", "Invalid slug format.")
			return
		}
		slug = *req.Slug
	}
	if req.Description != nil {
		description = req.Description
	}
	if req.IsHidden != nil {
		isHidden = *req.IsHidden
	}

	if err := h.GroupRepo.Update(uint(id), name, slug, description, isHidden); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			WriteAPIError(w, http.StatusNotFound, "GroupNotFound", "Album group not found during update")
		} else if isUniqueViolation(err) {
			WriteAPIError(w, http.StatusConflict, "GroupConflict", "Album group name or slug already exists")
		} else {
			log.Printf("Error updating album group %d: %v", id, err)
			WriteAPIError(w, http.StatusInternalServerError, "GroupUpdateError", "Failed to update album group")
		}
		return
	}

	updated, err := h.GroupRepo.GetByID(uint(id))
	if err != nil {
		WriteAPIResponse(w, http.StatusOK, map[string]string{"message": "Album group updated"})
		return
	}
	WriteAPIResponse(w, http.StatusOK, updated)
}

// DeleteGroup soft-deletes an album group.
func (h *AdminAlbumGroupHandler) DeleteGroup(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		WriteAPIError(w, http.StatusBadRequest, "InvalidID", "Invalid group ID")
		return
	}

	bannerPath, err := h.GroupRepo.Delete(uint(id))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			WriteAPIError(w, http.StatusNotFound, "GroupNotFound", "Album group not found")
		} else {
			log.Printf("Error deleting album group %d: %v", id, err)
			WriteAPIError(w, http.StatusInternalServerError, "GroupDeleteError", "Failed to delete album group")
		}
		return
	}
	if bannerPath != nil {
		if err := h.Store.Delete(r.Context(), *bannerPath); err != nil {
			log.Printf("Error deleting banner object of album group %d: %v", id, err)
		}
	}
	w.WriteHeader(http.StatusNoContent)
}

// UploadGroupBanner uploads and sets a banner image for an album group.
func (h *AdminAlbumGroupHandler) UploadGroupBanner(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		WriteAPIError(w, http.StatusBadRequest, "InvalidID", "Invalid group ID")
		return
	}

	group, err := h.GroupRepo.GetByID(uint(id))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			WriteAPIError(w, http.StatusNotFound, "GroupNotFound", "Album group not found")
		} else {
			log.Printf("Error finding album group %d for banner upload: %v", id, err)
			WriteAPIError(w, http.StatusInternalServerError, "GroupFetchError", "Failed to find album group")
		}
		return
	}

	if !parseBannerForm(w, r) {
		return
	}

	file, handler, err := r.FormFile("banner_image")
	if err != nil {
		if errors.Is(err, http.ErrMissingFile) {
			WriteAPIError(w, http.StatusBadRequest, "MissingFile", "No file in 'banner_image' field")
		} else {
			WriteAPIError(w, http.StatusBadRequest, "FileError", "Could not retrieve uploaded file")
		}
		return
	}
	defer file.Close()

	log.Printf("Received banner upload for group %d: %s (size: %d)", id, handler.Filename, handler.Size)

	if h.MediaProcessor == nil {
		WriteAPIError(w, http.StatusInternalServerError, "ConfigError", "Media processor not configured")
		return
	}

	savedRelPath, procErr := h.MediaProcessor.ProcessBanner(file)
	if procErr != nil {
		log.Printf("Error processing banner for group %d: %v", id, procErr)
		WriteAPIError(w, http.StatusInternalServerError, "BannerProcessError", "Failed to process banner image")
		return
	}

	if dbErr := h.GroupRepo.SetBannerPath(uint(id), &savedRelPath); dbErr != nil {
		if err := h.Store.Delete(r.Context(), savedRelPath); err != nil {
			log.Printf("Error deleting orphaned banner object %s: %v", savedRelPath, err)
		}
		if errors.Is(dbErr, gorm.ErrRecordNotFound) {
			WriteAPIError(w, http.StatusNotFound, "GroupNotFound", "Album group not found")
			return
		}
		log.Printf("Error saving banner path for group %d: %v", id, dbErr)
		WriteAPIError(w, http.StatusInternalServerError, "BannerSaveError", "Failed to save banner information")
		return
	}

	// remove the banner this one replaced
	if group.BannerImagePath != nil && *group.BannerImagePath != savedRelPath {
		_ = h.Store.Delete(r.Context(), *group.BannerImagePath)
	}

	updated, err := h.GroupRepo.GetByID(uint(id))
	if err != nil {
		WriteAPIResponse(w, http.StatusOK, map[string]string{"banner_image_path": savedRelPath})
		return
	}
	WriteAPIResponse(w, http.StatusOK, updated)
}

// SetAlbumGroup assigns or removes an album from a group.
// Accepts JSON body: {"group_id": 1} or {"group_id": null} to clear.
func (h *AdminAlbumGroupHandler) SetAlbumGroup(w http.ResponseWriter, r *http.Request) {
	albumIDStr := chi.URLParam(r, "id")
	albumID, err := strconv.ParseUint(albumIDStr, 10, 64)
	if err != nil {
		WriteAPIError(w, http.StatusBadRequest, "InvalidAlbumID", "Invalid album ID")
		return
	}

	var req struct {
		GroupID *uint `json:"group_id"`
	}
	if err := decodeJSONBody(w, r, &req); err != nil {
		if isBodyTooLarge(err) {
			WriteAPIError(w, http.StatusRequestEntityTooLarge, "PayloadTooLarge", "Request body is too large")
			return
		}
		WriteAPIError(w, http.StatusBadRequest, "InvalidPayload", "Invalid request body: "+err.Error())
		return
	}

	if err := h.GroupRepo.SetAlbumGroup(uint(albumID), req.GroupID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			WriteAPIError(w, http.StatusNotFound, "AlbumNotFound", "Album not found")
			return
		}
		if errors.Is(err, repository.ErrGroupNotFound) {
			WriteAPIError(w, http.StatusNotFound, "GroupNotFound", "Album group not found")
			return
		}
		log.Printf("Error setting group for album %d: %v", albumID, err)
		WriteAPIError(w, http.StatusInternalServerError, "SetGroupError", "Failed to update album group assignment")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
