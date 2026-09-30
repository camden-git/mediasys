package handlers

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/camden-git/mediasysbackend/models"
	"github.com/camden-git/mediasysbackend/repository"
	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"
)

// AdminImageTagHandler handles image tag and album default tag endpoints.
type AdminImageTagHandler struct {
	TagRepo   repository.ImageTagRepositoryInterface
	AlbumRepo repository.AlbumRepositoryInterface
	ImageRepo repository.ImageRepositoryInterface
}

// authorizeImage resolves the image referenced by the "path" query param and
// checks that the caller may edit the image's album: either the global
// album.edit.general permission or album.photo.editmeta on that album. It writes
// the error response itself and returns false on failure.
func (h *AdminImageTagHandler) authorizeImage(w http.ResponseWriter, r *http.Request) (*models.Image, bool) {
	imagePath := strings.TrimPrefix(r.URL.Query().Get("path"), "/")
	if imagePath == "" {
		WriteAPIError(w, http.StatusBadRequest, "MissingPath", "Query param 'path' is required")
		return nil, false
	}
	user, ok := r.Context().Value(UserContextKey).(*models.User)
	if !ok || user == nil {
		WriteAPIError(w, http.StatusInternalServerError, "ContextUserMissing", "User not found in context")
		return nil, false
	}
	img, err := h.ImageRepo.GetByPath(imagePath)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			WriteAPIError(w, http.StatusNotFound, "ImageNotFound", "Image not found")
		} else {
			log.Printf("Error loading image %s for tags: %v", imagePath, err)
			WriteAPIError(w, http.StatusInternalServerError, "ImageFetchError", "Failed to retrieve image")
		}
		return nil, false
	}
	if !user.HasGlobalPermission("album.edit.general") && !user.HasAlbumPermission(img.AlbumID, "album.photo.editmeta") {
		WriteAPIError(w, http.StatusForbidden, "Forbidden", "Forbidden: requires global permission 'album.edit.general' or album permission 'album.photo.editmeta'")
		return nil, false
	}
	return img, true
}

// GetImageTags returns all tags for an image path.
// GET /api/admin/images/tags?path=...
func (h *AdminImageTagHandler) GetImageTags(w http.ResponseWriter, r *http.Request) {
	img, ok := h.authorizeImage(w, r)
	if !ok {
		return
	}
	path := img.OriginalPath
	tags, err := h.TagRepo.GetTagsByImagePath(path)
	if err != nil {
		log.Printf("Error getting tags for %s: %v", path, err)
		WriteAPIError(w, http.StatusInternalServerError, "TagFetchError", "Failed to retrieve image tags")
		return
	}
	WriteAPIResponse(w, http.StatusOK, tags)
}

// AddManualTag adds a manual tag to an image.
// POST /api/admin/images/tags?path=...
func (h *AdminImageTagHandler) AddManualTag(w http.ResponseWriter, r *http.Request) {
	img, ok := h.authorizeImage(w, r)
	if !ok {
		return
	}
	path := img.OriginalPath
	var req struct {
		TagKey   string `json:"tag_key"`
		TagValue string `json:"tag_value"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteAPIError(w, http.StatusBadRequest, "InvalidPayload", "Invalid request body: "+err.Error())
		return
	}
	if req.TagKey == "" || req.TagValue == "" {
		WriteAPIError(w, http.StatusBadRequest, "ValidationError", "tag_key and tag_value are required")
		return
	}
	if err := h.TagRepo.AddManualTag(path, req.TagKey, req.TagValue); err != nil {
		log.Printf("Error adding manual tag to %s: %v", path, err)
		WriteAPIError(w, http.StatusInternalServerError, "TagAddError", "Failed to add tag")
		return
	}
	tags, _ := h.TagRepo.GetTagsByImagePath(path)
	WriteAPIResponse(w, http.StatusCreated, tags)
}

// RemoveManualTag removes a manual tag from an image.
// DELETE /api/admin/images/tags?path=...
func (h *AdminImageTagHandler) RemoveManualTag(w http.ResponseWriter, r *http.Request) {
	img, ok := h.authorizeImage(w, r)
	if !ok {
		return
	}
	path := img.OriginalPath
	var req struct {
		TagKey   string `json:"tag_key"`
		TagValue string `json:"tag_value"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteAPIError(w, http.StatusBadRequest, "InvalidPayload", "Invalid request body: "+err.Error())
		return
	}
	if req.TagKey == "" || req.TagValue == "" {
		WriteAPIError(w, http.StatusBadRequest, "ValidationError", "tag_key and tag_value are required")
		return
	}
	if err := h.TagRepo.RemoveManualTag(path, req.TagKey, req.TagValue); err != nil {
		log.Printf("Error removing manual tag from %s: %v", path, err)
		WriteAPIError(w, http.StatusInternalServerError, "TagRemoveError", "Failed to remove tag")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// GetAlbumDefaultTags returns all default tags configured on an album.
// GET /api/admin/albums/{id}/default-tags
func (h *AdminImageTagHandler) GetAlbumDefaultTags(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		WriteAPIError(w, http.StatusBadRequest, "InvalidID", "Invalid album ID")
		return
	}
	if _, err := h.AlbumRepo.GetByID(uint(id)); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			WriteAPIError(w, http.StatusNotFound, "AlbumNotFound", "Album not found")
		} else {
			WriteAPIError(w, http.StatusInternalServerError, "AlbumFetchError", "Failed to retrieve album")
		}
		return
	}
	tags, err := h.TagRepo.GetAlbumDefaultTags(uint(id))
	if err != nil {
		log.Printf("Error getting default tags for album %d: %v", id, err)
		WriteAPIError(w, http.StatusInternalServerError, "TagFetchError", "Failed to retrieve album default tags")
		return
	}
	WriteAPIResponse(w, http.StatusOK, tags)
}

// SetAlbumDefaultTags replaces all default tags for an album.
// PUT /api/admin/albums/{id}/default-tags
func (h *AdminImageTagHandler) SetAlbumDefaultTags(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		WriteAPIError(w, http.StatusBadRequest, "InvalidID", "Invalid album ID")
		return
	}
	if _, err := h.AlbumRepo.GetByID(uint(id)); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			WriteAPIError(w, http.StatusNotFound, "AlbumNotFound", "Album not found")
		} else {
			WriteAPIError(w, http.StatusInternalServerError, "AlbumFetchError", "Failed to retrieve album")
		}
		return
	}

	var req struct {
		Tags []struct {
			TagKey   string `json:"tag_key"`
			TagValue string `json:"tag_value"`
		} `json:"tags"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteAPIError(w, http.StatusBadRequest, "InvalidPayload", "Invalid request body: "+err.Error())
		return
	}

	tags := make([]models.AlbumDefaultTag, 0, len(req.Tags))
	for _, t := range req.Tags {
		if t.TagKey == "" || t.TagValue == "" {
			WriteAPIError(w, http.StatusBadRequest, "ValidationError", "Each tag must have non-empty tag_key and tag_value")
			return
		}
		tags = append(tags, models.AlbumDefaultTag{
			TagKey:   t.TagKey,
			TagValue: t.TagValue,
		})
	}

	if err := h.TagRepo.SetAlbumDefaultTags(uint(id), tags); err != nil {
		log.Printf("Error setting default tags for album %d: %v", id, err)
		WriteAPIError(w, http.StatusInternalServerError, "TagSetError", "Failed to set album default tags")
		return
	}
	updated, _ := h.TagRepo.GetAlbumDefaultTags(uint(id))
	WriteAPIResponse(w, http.StatusOK, updated)
}
