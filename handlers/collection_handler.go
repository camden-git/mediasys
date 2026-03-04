package handlers

import (
	"errors"
	"log"
	"net/http"
	"path/filepath"
	"strconv"

	"github.com/camden-git/mediasysbackend/database"
	"github.com/camden-git/mediasysbackend/models"
	"github.com/camden-git/mediasysbackend/repository"
	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"
)

// CollectionHandler serves public collection endpoints.
type CollectionHandler struct {
	CollectionRepo repository.CollectionRepositoryInterface
	ImageRepo      repository.ImageRepositoryInterface
}

// collectionPublicResponse is the public view of a collection with banners as paths.
type collectionPublicResponse struct {
	ID                       uint                         `json:"id"`
	Name                     string                       `json:"name"`
	Slug                     string                       `json:"slug"`
	Description              *string                      `json:"description,omitempty"`
	IsPublic                 bool                         `json:"is_public"`
	FilterMatch              string                       `json:"filter_match"`
	InheritBannersFromAlbums bool                         `json:"inherit_banners_from_albums"`
	Banners                  []string                     `json:"banners"`
	Filters                  []models.CollectionTagFilter `json:"filters,omitempty"`
	CreatedAt                int64                        `json:"created_at"`
	UpdatedAt                int64                        `json:"updated_at"`
}

func (h *CollectionHandler) buildPublicResponse(c *models.Collection) *collectionPublicResponse {
	var bannerPaths []string
	if c.InheritBannersFromAlbums {
		inherited, _ := h.CollectionRepo.GetInheritedBannerPaths(c.ID)
		bannerPaths = inherited
	} else {
		banners, _ := h.CollectionRepo.GetBanners(c.ID)
		bannerPaths = make([]string, 0, len(banners))
		for _, b := range banners {
			bannerPaths = append(bannerPaths, b.ImagePath)
		}
	}
	if bannerPaths == nil {
		bannerPaths = []string{}
	}
	return &collectionPublicResponse{
		ID:                       c.ID,
		Name:                     c.Name,
		Slug:                     c.Slug,
		Description:              c.Description,
		IsPublic:                 c.IsPublic,
		FilterMatch:              c.FilterMatch,
		InheritBannersFromAlbums: c.InheritBannersFromAlbums,
		Banners:                  bannerPaths,
		Filters:                  c.Filters,
		CreatedAt:                c.CreatedAt,
		UpdatedAt:                c.UpdatedAt,
	}
}

// ListCollections returns all public collections.
func (h *CollectionHandler) ListCollections(w http.ResponseWriter, r *http.Request) {
	collections, err := h.CollectionRepo.ListPublic()
	if err != nil {
		log.Printf("Error listing public collections: %v", err)
		WriteAPIError(w, http.StatusInternalServerError, "CollectionListError", "Failed to retrieve collections")
		return
	}
	result := make([]*collectionPublicResponse, len(collections))
	for i, c := range collections {
		c := c
		result[i] = h.buildPublicResponse(&c)
	}
	setCacheHeaders(w, 60)
	WriteAPIResponse(w, http.StatusOK, result)
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
	setCacheHeaders(w, 60)
	WriteAPIResponse(w, http.StatusOK, h.buildPublicResponse(c))
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
	setCacheHeaders(w, 60)
	WriteAPIResponse(w, http.StatusOK, listing)
}
