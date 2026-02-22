package handlers

import (
	"errors"
	"log"
	"net/http"
	"path/filepath"
	"strconv"

	"github.com/camden-git/mediasysbackend/database"
	"github.com/camden-git/mediasysbackend/repository"
	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"
)

// CollectionHandler serves public collection endpoints.
type CollectionHandler struct {
	CollectionRepo repository.CollectionRepositoryInterface
	ImageRepo      repository.ImageRepositoryInterface
}

// ListCollections returns all public collections.
func (h *CollectionHandler) ListCollections(w http.ResponseWriter, r *http.Request) {
	collections, err := h.CollectionRepo.ListPublic()
	if err != nil {
		log.Printf("Error listing public collections: %v", err)
		WriteAPIError(w, http.StatusInternalServerError, "CollectionListError", "Failed to retrieve collections")
		return
	}
	WriteAPIResponse(w, http.StatusOK, collections)
}

// GetCollection returns a single public collection by slug.
func (h *CollectionHandler) GetCollection(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	c, err := h.CollectionRepo.GetBySlug(slug)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			WriteAPIError(w, http.StatusNotFound, "CollectionNotFound", "Collection not found")
		} else {
			log.Printf("Error getting collection '%s': %v", slug, err)
			WriteAPIError(w, http.StatusInternalServerError, "CollectionFetchError", "Failed to retrieve collection")
		}
		return
	}
	if !c.IsPublic {
		WriteAPIError(w, http.StatusNotFound, "CollectionNotFound", "Collection not found")
		return
	}
	WriteAPIResponse(w, http.StatusOK, c)
}

// GetCollectionPhotos returns paginated images matching a collection's tag filters.
func (h *CollectionHandler) GetCollectionPhotos(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	c, err := h.CollectionRepo.GetBySlug(slug)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			WriteAPIError(w, http.StatusNotFound, "CollectionNotFound", "Collection not found")
		} else {
			log.Printf("Error getting collection '%s' for photos: %v", slug, err)
			WriteAPIError(w, http.StatusInternalServerError, "CollectionFetchError", "Failed to retrieve collection")
		}
		return
	}
	if !c.IsPublic {
		WriteAPIError(w, http.StatusNotFound, "CollectionNotFound", "Collection not found")
		return
	}

	q := r.URL.Query()
	offset := 0
	limit := 120
	if o := q.Get("offset"); o != "" {
		if v, convErr := strconv.Atoi(o); convErr == nil && v >= 0 {
			offset = v
		}
	}
	if l := q.Get("limit"); l != "" {
		if v, convErr := strconv.Atoi(l); convErr == nil && v > 0 {
			limit = v
		}
	}

	paths, total, err := h.CollectionRepo.GetImagePathsMatchingFilters(c.ID, offset, limit)
	if err != nil {
		log.Printf("Error querying collection photos for '%s': %v", slug, err)
		WriteAPIError(w, http.StatusInternalServerError, "CollectionPhotosError", "Failed to retrieve collection photos")
		return
	}

	if len(paths) == 0 {
		WriteAPIResponse(w, http.StatusOK, DirectoryListing{
			Path:    "/collections/" + slug + "/photos",
			Files:   []FileInfo{},
			Total:   total,
			Offset:  offset,
			Limit:   limit,
			HasMore: false,
		})
		return
	}

	images, err := h.ImageRepo.GetImagesByPaths(paths)
	if err != nil {
		log.Printf("Error fetching images for collection '%s': %v", slug, err)
		WriteAPIError(w, http.StatusInternalServerError, "CollectionPhotosError", "Failed to retrieve collection photos")
		return
	}

	files := make([]FileInfo, 0, len(images))
	for _, img := range images {
		fi := FileInfo{
			Name:            filepath.Base(img.OriginalPath),
			Path:            "/" + img.OriginalPath,
			IsDir:           false,
			ModTime:         img.LastModified,
			Width:           img.Width,
			Height:          img.Height,
			Aperture:        img.Aperture,
			ShutterSpeed:    img.ShutterSpeed,
			ISO:             img.ISO,
			FocalLength:     img.FocalLength,
			LensMake:        img.LensMake,
			LensModel:       img.LensModel,
			CameraMake:      img.CameraMake,
			CameraModel:     img.CameraModel,
			TakenAt:         img.TakenAt,
			Rating:          img.Rating,
			ThumbnailStatus: img.ThumbnailStatus,
			MetadataStatus:  img.MetadataStatus,
			DetectionStatus: img.DetectionStatus,
		}
		if img.ThumbnailPath != nil && img.ThumbnailStatus == database.StatusDone {
			thumbFilename := filepath.Base(*img.ThumbnailPath)
			fullThumbURL := "/api" + thumbnailApiPrefix + thumbFilename
			fi.ThumbnailPath = &fullThumbURL
		}
		files = append(files, fi)
	}

	listing := DirectoryListing{
		Path:    "/collections/" + slug + "/photos",
		Files:   files,
		Total:   total,
		Offset:  offset,
		Limit:   limit,
		HasMore: offset+len(files) < total,
	}
	WriteAPIResponse(w, http.StatusOK, listing)
}
