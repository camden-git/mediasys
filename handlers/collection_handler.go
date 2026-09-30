package handlers

import (
	"errors"
	"log"
	"net/http"

	"github.com/camden-git/mediasysbackend/database"
	"github.com/camden-git/mediasysbackend/models"
	"github.com/camden-git/mediasysbackend/repository"
	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"
)

// CollectionHandler serves public collection endpoints.
type CollectionHandler struct {
	CollectionRepo repository.CollectionRepositoryInterface
	PublicURL      string // externally visible base URL for share pages
}

// collectionPublicResponse is the public view of a collection with banners as paths.
type collectionPublicResponse struct {
	ID                       uint                         `json:"id"`
	Name                     string                       `json:"name"`
	Slug                     string                       `json:"slug"`
	Description              *string                      `json:"description,omitempty"`
	IsPublic                 bool                         `json:"is_public"`
	FilterMatch              string                       `json:"filter_match"`
	SortOrder                string                       `json:"sort_order"`
	InheritBannersFromAlbums bool                         `json:"inherit_banners_from_albums"`
	Banners                  []string                     `json:"banners"`
	Filters                  []models.CollectionTagFilter `json:"filters,omitempty"`
	CreatedAt                int64                        `json:"created_at"`
	UpdatedAt                int64                        `json:"updated_at"`
}

// buildPublicResponse builds the public view of a collection. preloaded, when
// non-nil, holds the collection's own banners (batch-loaded by the caller).
func (h *CollectionHandler) buildPublicResponse(c *models.Collection, preloaded []models.CollectionBanner) *collectionPublicResponse {
	var bannerPaths []string
	if c.InheritBannersFromAlbums {
		inherited, err := h.CollectionRepo.GetInheritedBannerPaths(c.ID)
		if err != nil {
			log.Printf("Error loading inherited banners for collection %d: %v", c.ID, err)
		}
		bannerPaths = inherited
	} else {
		banners := preloaded
		if banners == nil {
			var err error
			if banners, err = h.CollectionRepo.GetBanners(c.ID); err != nil {
				log.Printf("Error loading banners for collection %d: %v", c.ID, err)
			}
		}
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
		SortOrder:                c.SortOrder,
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
	ids := make([]uint, len(collections))
	for i := range collections {
		ids[i] = collections[i].ID
	}
	bannersByID, err := h.CollectionRepo.GetBannersByCollectionIDs(ids)
	if err != nil {
		log.Printf("Error loading banners for public collections: %v", err)
		WriteAPIError(w, http.StatusInternalServerError, "CollectionListError", "Failed to retrieve collections")
		return
	}
	result := make([]*collectionPublicResponse, len(collections))
	for i := range collections {
		banners := bannersByID[collections[i].ID]
		if banners == nil {
			banners = []models.CollectionBanner{}
		}
		result[i] = h.buildPublicResponse(&collections[i], banners)
	}
	setCacheHeaders(w, 300)
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
	setCacheHeaders(w, 300)
	WriteAPIResponse(w, http.StatusOK, h.buildPublicResponse(c, nil))
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
	offset, limit := parseListWindow(q)

	sortOrder := c.SortOrder
	if !database.IsValidSortOrder(sortOrder) {
		sortOrder = database.DefaultSortOrder
	}

	images, total, err := h.CollectionRepo.ListImages(c.ID, sortOrder, offset, limit)
	if err != nil {
		log.Printf("Error querying collection photos for '%s': %v", slug, err)
		WriteAPIError(w, http.StatusInternalServerError, "CollectionPhotosError", "Failed to retrieve collection photos")
		return
	}

	files := make([]FileInfo, 0, len(images))
	for i := range images {
		files = append(files, imageToFileInfo(&images[i]))
	}
	listing := DirectoryListing{
		Path:    "/collections/" + slug + "/photos",
		Files:   files,
		Total:   total,
		Offset:  offset,
		Limit:   limit,
		HasMore: offset+len(files) < total,
	}
	setCacheHeaders(w, 300)
	WriteAPIResponse(w, http.StatusOK, listing)
}
