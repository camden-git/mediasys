package handlers

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/camden-git/mediasysbackend/database"
	"github.com/camden-git/mediasysbackend/media"
	"github.com/camden-git/mediasysbackend/models"
	"github.com/camden-git/mediasysbackend/repository"
	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"
)

// adminCollectionResponse is the admin view of a collection with its banner objects.
type adminCollectionResponse struct {
	ID                       uint                         `json:"id"`
	Name                     string                       `json:"name"`
	Slug                     string                       `json:"slug"`
	Description              *string                      `json:"description,omitempty"`
	IsPublic                 bool                         `json:"is_public"`
	FilterMatch              string                       `json:"filter_match"`
	SortOrder                string                       `json:"sort_order"`
	InheritBannersFromAlbums bool                         `json:"inherit_banners_from_albums"`
	Banners                  []models.CollectionBanner    `json:"banners"`
	Filters                  []models.CollectionTagFilter `json:"filters,omitempty"`
	CreatedAt                int64                        `json:"created_at"`
	UpdatedAt                int64                        `json:"updated_at"`
}

func buildCollectionAdminResponse(c *models.Collection, banners []models.CollectionBanner) *adminCollectionResponse {
	if banners == nil {
		banners = []models.CollectionBanner{}
	}
	return &adminCollectionResponse{
		ID:                       c.ID,
		Name:                     c.Name,
		Slug:                     c.Slug,
		Description:              c.Description,
		IsPublic:                 c.IsPublic,
		FilterMatch:              c.FilterMatch,
		SortOrder:                c.SortOrder,
		InheritBannersFromAlbums: c.InheritBannersFromAlbums,
		Banners:                  banners,
		Filters:                  c.Filters,
		CreatedAt:                c.CreatedAt,
		UpdatedAt:                c.UpdatedAt,
	}
}

// AdminCollectionHandler handles admin CRUD for collections.
type AdminCollectionHandler struct {
	CollectionRepo repository.CollectionRepositoryInterface
	MediaProcessor *media.Processor
	Store          *media.Store
}

func NewAdminCollectionHandler(
	collectionRepo repository.CollectionRepositoryInterface,
	mediaProcessor *media.Processor,
	store *media.Store,
) *AdminCollectionHandler {
	return &AdminCollectionHandler{
		CollectionRepo: collectionRepo,
		MediaProcessor: mediaProcessor,
		Store:          store,
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
	result := make([]*adminCollectionResponse, len(collections))
	for i, c := range collections {
		banners, _ := h.CollectionRepo.GetBanners(c.ID)
		c := c
		result[i] = buildCollectionAdminResponse(&c, banners)
	}
	WriteAPIResponse(w, http.StatusOK, result)
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
	banners, _ := h.CollectionRepo.GetBanners(c.ID)
	WriteAPIResponse(w, http.StatusOK, buildCollectionAdminResponse(c, banners))
}

// CreateCollection creates a new collection.
func (h *AdminCollectionHandler) CreateCollection(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name        string  `json:"name"`
		Slug        string  `json:"slug"`
		Description *string `json:"description"`
		IsPublic    *bool   `json:"is_public"`
		FilterMatch *string `json:"filter_match"`
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
	filterMatch := "all"
	if req.FilterMatch != nil && (*req.FilterMatch == "all" || *req.FilterMatch == "any") {
		filterMatch = *req.FilterMatch
	}
	c := &models.Collection{
		Name:        req.Name,
		Slug:        req.Slug,
		Description: req.Description,
		IsPublic:    isPublic,
		FilterMatch: filterMatch,
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
	WriteAPIResponse(w, http.StatusCreated, buildCollectionAdminResponse(c, nil))
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
		FilterMatch *string `json:"filter_match"`
		SortOrder   *string `json:"sort_order"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteAPIError(w, http.StatusBadRequest, "InvalidPayload", "Invalid request body: "+err.Error())
		return
	}

	name := c.Name
	slug := c.Slug
	description := c.Description
	isPublic := c.IsPublic
	filterMatch := c.FilterMatch
	if filterMatch == "" {
		filterMatch = "all"
	}
	sortOrder := c.SortOrder
	if sortOrder == "" {
		sortOrder = "filename_asc"
	}

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
	if req.FilterMatch != nil && (*req.FilterMatch == "all" || *req.FilterMatch == "any") {
		filterMatch = *req.FilterMatch
	}
	if req.SortOrder != nil {
		if !database.IsValidSortOrder(*req.SortOrder) {
			WriteAPIError(w, http.StatusBadRequest, "ValidationError", "Invalid sort order")
			return
		}
		sortOrder = *req.SortOrder
	}

	if err := h.CollectionRepo.Update(id, name, slug, description, isPublic, filterMatch, sortOrder); err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			WriteAPIError(w, http.StatusConflict, "CollectionConflict", "Collection name or slug already exists")
		} else {
			log.Printf("Error updating collection %d: %v", id, err)
			WriteAPIError(w, http.StatusInternalServerError, "CollectionUpdateError", "Failed to update collection")
		}
		return
	}
	updated, _ := h.CollectionRepo.GetByID(id)
	banners, _ := h.CollectionRepo.GetBanners(id)
	WriteAPIResponse(w, http.StatusOK, buildCollectionAdminResponse(updated, banners))
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

// AddCollectionBanner uploads a new banner image for a collection.
// POST /api/admin/collections/{id}/banners
func (h *AdminCollectionHandler) AddCollectionBanner(w http.ResponseWriter, r *http.Request) {
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

	banner := &models.CollectionBanner{
		CollectionID: id,
		ImagePath:    savedRelPath,
		SortOrder:    0,
	}
	if err := h.CollectionRepo.AddBanner(banner); err != nil {
		_ = h.Store.Delete(r.Context(), savedRelPath)
		log.Printf("Error saving banner for collection %d: %v", id, err)
		WriteAPIError(w, http.StatusInternalServerError, "BannerSaveError", "Failed to save banner")
		return
	}

	WriteAPIResponse(w, http.StatusCreated, banner)
}

// DeleteCollectionBanner deletes a specific banner from a collection.
// DELETE /api/admin/collections/{id}/banners/{bannerId}
func (h *AdminCollectionHandler) DeleteCollectionBanner(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "id")
	if err != nil {
		WriteAPIError(w, http.StatusBadRequest, "InvalidID", "Invalid collection ID")
		return
	}
	bannerID, err := parseID(r, "bannerId")
	if err != nil {
		WriteAPIError(w, http.StatusBadRequest, "InvalidBannerID", "Invalid banner ID")
		return
	}

	banners, err := h.CollectionRepo.GetBanners(id)
	if err != nil {
		log.Printf("Error fetching banners for collection %d: %v", id, err)
		WriteAPIError(w, http.StatusInternalServerError, "BannerFetchError", "Failed to fetch banners")
		return
	}

	var bannerPath string
	for _, b := range banners {
		if b.ID == bannerID {
			bannerPath = b.ImagePath
			break
		}
	}

	if err := h.CollectionRepo.DeleteCollectionBanner(bannerID, id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			WriteAPIError(w, http.StatusNotFound, "BannerNotFound", "Banner not found")
		} else {
			log.Printf("Error deleting banner %d from collection %d: %v", bannerID, id, err)
			WriteAPIError(w, http.StatusInternalServerError, "BannerDeleteError", "Failed to delete banner")
		}
		return
	}

	if bannerPath != "" {
		_ = h.Store.Delete(r.Context(), bannerPath)
	}

	w.WriteHeader(http.StatusNoContent)
}

// ReorderCollectionBanners updates the sort order of collection banners.
// PUT /api/admin/collections/{id}/banners/order
func (h *AdminCollectionHandler) ReorderCollectionBanners(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "id")
	if err != nil {
		WriteAPIError(w, http.StatusBadRequest, "InvalidID", "Invalid collection ID")
		return
	}

	var req struct {
		BannerIDs []uint `json:"banner_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteAPIError(w, http.StatusBadRequest, "InvalidPayload", "Invalid request body: "+err.Error())
		return
	}

	if err := h.CollectionRepo.ReorderCollectionBanners(id, req.BannerIDs); err != nil {
		log.Printf("Error reordering banners for collection %d: %v", id, err)
		WriteAPIError(w, http.StatusInternalServerError, "BannerReorderError", "Failed to reorder banners")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// SetCollectionInheritBanners sets the inherit_banners_from_albums flag for a collection.
// PUT /api/admin/collections/{id}/banners/inherit
func (h *AdminCollectionHandler) SetCollectionInheritBanners(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "id")
	if err != nil {
		WriteAPIError(w, http.StatusBadRequest, "InvalidID", "Invalid collection ID")
		return
	}

	var req struct {
		Inherit bool `json:"inherit"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteAPIError(w, http.StatusBadRequest, "InvalidPayload", "Invalid request body: "+err.Error())
		return
	}

	if err := h.CollectionRepo.SetInheritBanners(id, req.Inherit); err != nil {
		log.Printf("Error setting inherit_banners for collection %d: %v", id, err)
		WriteAPIError(w, http.StatusInternalServerError, "InheritSetError", "Failed to update inherit setting")
		return
	}

	updated, _ := h.CollectionRepo.GetByID(id)
	banners, _ := h.CollectionRepo.GetBanners(id)
	WriteAPIResponse(w, http.StatusOK, buildCollectionAdminResponse(updated, banners))
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
			Negate   bool   `json:"negate"`
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
			Negate:   f.Negate,
		})
	}

	if err := h.CollectionRepo.SetFilters(id, filters); err != nil {
		log.Printf("Error setting filters for collection %d: %v", id, err)
		WriteAPIError(w, http.StatusInternalServerError, "FilterSetError", "Failed to set collection filters")
		return
	}
	updated, _ := h.CollectionRepo.GetByID(id)
	banners, _ := h.CollectionRepo.GetBanners(id)
	WriteAPIResponse(w, http.StatusOK, buildCollectionAdminResponse(updated, banners))
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
