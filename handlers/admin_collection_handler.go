package handlers

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/camden-git/mediasysbackend/config"
	"github.com/camden-git/mediasysbackend/media"
	"github.com/camden-git/mediasysbackend/models"
	"github.com/camden-git/mediasysbackend/repository"
	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"
)

// AdminCollectionHandler handles admin CRUD for collections.
type AdminCollectionHandler struct {
	CollectionRepo repository.CollectionRepositoryInterface
	Cfg            config.Config
	MediaProcessor *media.Processor
}

func NewAdminCollectionHandler(
	collectionRepo repository.CollectionRepositoryInterface,
	cfg config.Config,
	mediaProcessor *media.Processor,
) *AdminCollectionHandler {
	return &AdminCollectionHandler{
		CollectionRepo: collectionRepo,
		Cfg:            cfg,
		MediaProcessor: mediaProcessor,
	}
}

// ListCollections lists all collections (admin: includes private ones).
func (h *AdminCollectionHandler) ListCollections(w http.ResponseWriter, r *http.Request) {
	collections, err := h.CollectionRepo.ListAll()
	if err != nil {
		log.Printf("Error listing collections for admin: %v", err)
		WriteAPIError(w, http.StatusInternalServerError, "CollectionListError", "Failed to retrieve collections")
		return
	}
	WriteAPIResponse(w, http.StatusOK, collections)
}

// GetCollection returns a single collection by ID.
func (h *AdminCollectionHandler) GetCollection(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "id")
	if err != nil {
		WriteAPIError(w, http.StatusBadRequest, "InvalidID", "Invalid collection ID")
		return
	}
	c, err := h.CollectionRepo.GetByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			WriteAPIError(w, http.StatusNotFound, "CollectionNotFound", "Collection not found")
		} else {
			log.Printf("Error getting collection %d: %v", id, err)
			WriteAPIError(w, http.StatusInternalServerError, "CollectionFetchError", "Failed to retrieve collection")
		}
		return
	}
	WriteAPIResponse(w, http.StatusOK, c)
}

// CreateCollection creates a new collection.
func (h *AdminCollectionHandler) CreateCollection(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name        string  `json:"name"`
		Slug        string  `json:"slug"`
		Description *string `json:"description"`
		IsPublic    *bool   `json:"is_public"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteAPIError(w, http.StatusBadRequest, "InvalidPayload", "Invalid request body: "+err.Error())
		return
	}
	if req.Name == "" || req.Slug == "" {
		WriteAPIError(w, http.StatusBadRequest, "ValidationError", "name and slug are required")
		return
	}
	if strings.ContainsAny(req.Slug, " /\\?%*:|\"<>") || strings.TrimSpace(req.Slug) == "" {
		WriteAPIError(w, http.StatusBadRequest, "ValidationError", "Invalid slug format")
		return
	}

	isPublic := true
	if req.IsPublic != nil {
		isPublic = *req.IsPublic
	}
	c := &models.Collection{
		Name:        req.Name,
		Slug:        req.Slug,
		Description: req.Description,
		IsPublic:    isPublic,
	}
	if err := h.CollectionRepo.Create(c); err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			WriteAPIError(w, http.StatusConflict, "CollectionConflict", "Collection name or slug already exists")
		} else {
			log.Printf("Error creating collection '%s': %v", req.Name, err)
			WriteAPIError(w, http.StatusInternalServerError, "CollectionCreateError", "Failed to create collection")
		}
		return
	}
	WriteAPIResponse(w, http.StatusCreated, c)
}

// UpdateCollection updates an existing collection.
func (h *AdminCollectionHandler) UpdateCollection(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "id")
	if err != nil {
		WriteAPIError(w, http.StatusBadRequest, "InvalidID", "Invalid collection ID")
		return
	}
	c, err := h.CollectionRepo.GetByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			WriteAPIError(w, http.StatusNotFound, "CollectionNotFound", "Collection not found")
		} else {
			WriteAPIError(w, http.StatusInternalServerError, "CollectionFetchError", "Failed to retrieve collection")
		}
		return
	}

	var req struct {
		Name        *string `json:"name"`
		Slug        *string `json:"slug"`
		Description *string `json:"description"`
		IsPublic    *bool   `json:"is_public"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteAPIError(w, http.StatusBadRequest, "InvalidPayload", "Invalid request body: "+err.Error())
		return
	}

	name := c.Name
	slug := c.Slug
	description := c.Description
	isPublic := c.IsPublic

	if req.Name != nil {
		name = *req.Name
	}
	if req.Slug != nil {
		if strings.ContainsAny(*req.Slug, " /\\?%*:|\"<>") || strings.TrimSpace(*req.Slug) == "" {
			WriteAPIError(w, http.StatusBadRequest, "ValidationError", "Invalid slug format")
			return
		}
		slug = *req.Slug
	}
	if req.Description != nil {
		description = req.Description
	}
	if req.IsPublic != nil {
		isPublic = *req.IsPublic
	}

	if err := h.CollectionRepo.Update(id, name, slug, description, isPublic); err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			WriteAPIError(w, http.StatusConflict, "CollectionConflict", "Collection name or slug already exists")
		} else {
			log.Printf("Error updating collection %d: %v", id, err)
			WriteAPIError(w, http.StatusInternalServerError, "CollectionUpdateError", "Failed to update collection")
		}
		return
	}
	updated, _ := h.CollectionRepo.GetByID(id)
	WriteAPIResponse(w, http.StatusOK, updated)
}

// DeleteCollection soft-deletes a collection.
func (h *AdminCollectionHandler) DeleteCollection(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "id")
	if err != nil {
		WriteAPIError(w, http.StatusBadRequest, "InvalidID", "Invalid collection ID")
		return
	}
	if err := h.CollectionRepo.Delete(id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			WriteAPIError(w, http.StatusNotFound, "CollectionNotFound", "Collection not found")
		} else {
			log.Printf("Error deleting collection %d: %v", id, err)
			WriteAPIError(w, http.StatusInternalServerError, "CollectionDeleteError", "Failed to delete collection")
		}
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// UploadCollectionBanner uploads and sets a banner image for a collection.
func (h *AdminCollectionHandler) UploadCollectionBanner(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "id")
	if err != nil {
		WriteAPIError(w, http.StatusBadRequest, "InvalidID", "Invalid collection ID")
		return
	}
	c, err := h.CollectionRepo.GetByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			WriteAPIError(w, http.StatusNotFound, "CollectionNotFound", "Collection not found")
		} else {
			WriteAPIError(w, http.StatusInternalServerError, "CollectionFetchError", "Failed to find collection")
		}
		return
	}

	const maxUploadSize = 20 << 20
	if err := r.ParseMultipartForm(maxUploadSize); err != nil {
		WriteAPIError(w, http.StatusBadRequest, "InvalidForm", "Invalid form data: "+err.Error())
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
	log.Printf("Received banner upload for collection %d: %s (size: %d)", id, handler.Filename, handler.Size)

	if h.MediaProcessor == nil {
		WriteAPIError(w, http.StatusInternalServerError, "ConfigError", "Media processor not configured")
		return
	}
	savedRelPath, procErr := h.MediaProcessor.ProcessBanner(file)
	if procErr != nil {
		log.Printf("Error processing banner for collection %d: %v", id, procErr)
		WriteAPIError(w, http.StatusInternalServerError, "BannerProcessError", "Failed to process banner image")
		return
	}

	if c.BannerImagePath != nil && *c.BannerImagePath != savedRelPath {
		mediaStore, storeErr := media.NewLocalStorage(h.Cfg.MediaStoragePath, map[media.AssetType]string{})
		if storeErr == nil {
			if oldPath, pathErr := mediaStore.GetFullPath(*c.BannerImagePath); pathErr == nil {
				if removeErr := os.Remove(oldPath); removeErr != nil && !os.IsNotExist(removeErr) {
					log.Printf("Warning: failed to remove old collection banner %s: %v", oldPath, removeErr)
				}
			}
		}
	}

	if dbErr := h.CollectionRepo.SetBannerPath(id, &savedRelPath); dbErr != nil {
		log.Printf("Error saving banner path for collection %d: %v", id, dbErr)
		WriteAPIError(w, http.StatusInternalServerError, "BannerSaveError", "Failed to save banner information")
		return
	}
	updated, _ := h.CollectionRepo.GetByID(id)
	WriteAPIResponse(w, http.StatusOK, updated)
}

// SetCollectionFilters replaces all tag filters for a collection.
// PUT /api/admin/collections/{id}/filters
func (h *AdminCollectionHandler) SetCollectionFilters(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "id")
	if err != nil {
		WriteAPIError(w, http.StatusBadRequest, "InvalidID", "Invalid collection ID")
		return
	}
	if _, err := h.CollectionRepo.GetByID(id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			WriteAPIError(w, http.StatusNotFound, "CollectionNotFound", "Collection not found")
		} else {
			WriteAPIError(w, http.StatusInternalServerError, "CollectionFetchError", "Failed to find collection")
		}
		return
	}

	var req struct {
		Filters []struct {
			TagKey   string `json:"tag_key"`
			TagValue string `json:"tag_value"`
		} `json:"filters"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteAPIError(w, http.StatusBadRequest, "InvalidPayload", "Invalid request body: "+err.Error())
		return
	}

	filters := make([]models.CollectionTagFilter, 0, len(req.Filters))
	for _, f := range req.Filters {
		if f.TagKey == "" || f.TagValue == "" {
			WriteAPIError(w, http.StatusBadRequest, "ValidationError", "Each filter must have non-empty tag_key and tag_value")
			return
		}
		filters = append(filters, models.CollectionTagFilter{
			TagKey:   f.TagKey,
			TagValue: f.TagValue,
		})
	}

	if err := h.CollectionRepo.SetFilters(id, filters); err != nil {
		log.Printf("Error setting filters for collection %d: %v", id, err)
		WriteAPIError(w, http.StatusInternalServerError, "FilterSetError", "Failed to set collection filters")
		return
	}
	updated, _ := h.CollectionRepo.GetByID(id)
	WriteAPIResponse(w, http.StatusOK, updated)
}

// parseID is a helper to parse a uint URL param by name.
func parseID(r *http.Request, param string) (uint, error) {
	idStr := chi.URLParam(r, param)
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		return 0, err
	}
	return uint(id), nil
}
