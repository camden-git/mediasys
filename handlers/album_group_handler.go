package handlers

import (
	"errors"
	"log"
	"net/http"
	"strconv"

	"github.com/camden-git/mediasysbackend/repository"
	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"
)

// AlbumGroupHandler serves public album group endpoints.
type AlbumGroupHandler struct {
	GroupRepo repository.AlbumGroupRepositoryInterface
	ImageRepo repository.ImageRepositoryInterface
}

// ListGroups returns all non-hidden album groups and their non-hidden albums.
func (h *AlbumGroupHandler) ListGroups(w http.ResponseWriter, r *http.Request) {
	groups, err := h.GroupRepo.ListAll()
	if err != nil {
		log.Printf("Error listing album groups: %v", err)
		WriteAPIError(w, http.StatusInternalServerError, "GroupListError", "Failed to retrieve album groups")
		return
	}
	out := make([]PublicAlbumGroup, len(groups))
	for i := range groups {
		out[i] = toPublicAlbumGroup(&groups[i])
	}
	setCacheHeaders(w, 300)
	WriteAPIResponse(w, http.StatusOK, out)
}

// GetGroup returns a single non-hidden album group by slug.
func (h *AlbumGroupHandler) GetGroup(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	group, err := h.GroupRepo.GetBySlug(slug)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			WriteAPIError(w, http.StatusNotFound, "GroupNotFound", "Album group not found")
		} else {
			log.Printf("Error getting album group '%s': %v", slug, err)
			WriteAPIError(w, http.StatusInternalServerError, "GroupFetchError", "Failed to retrieve album group")
		}
		return
	}
	setCacheHeaders(w, 300)
	WriteAPIResponse(w, http.StatusOK, toPublicAlbumGroup(group))
}

// GetGroupPhotos returns paginated images across all albums in a group, with optional min_rating filter.
func (h *AlbumGroupHandler) GetGroupPhotos(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	group, err := h.GroupRepo.GetBySlug(slug)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			WriteAPIError(w, http.StatusNotFound, "GroupNotFound", "Album group not found")
		} else {
			log.Printf("Error getting album group '%s' for photos: %v", slug, err)
			WriteAPIError(w, http.StatusInternalServerError, "GroupFetchError", "Failed to retrieve album group")
		}
		return
	}

	q := r.URL.Query()
	offset, limit := parseListWindow(q)
	var minRating *int
	if mr := q.Get("min_rating"); mr != "" {
		if v, convErr := strconv.Atoi(mr); convErr == nil && v >= 1 {
			minRating = &v
		}
	}

	// Collect non-hidden albums in this group
	var albumIDs []uint
	for _, album := range group.Albums {
		if !album.IsHidden {
			albumIDs = append(albumIDs, album.ID)
		}
	}

	if len(albumIDs) == 0 {
		WriteAPIResponse(w, http.StatusOK, DirectoryListing{
			Path:    "/groups/" + slug + "/photos",
			Files:   []FileInfo{},
			Total:   0,
			Offset:  offset,
			Limit:   limit,
			HasMore: false,
		})
		return
	}

	images, total, err := h.ImageRepo.GetImagesByAlbumIDs(albumIDs, minRating, offset, limit)
	if err != nil {
		log.Printf("Error querying group photos for '%s': %v", slug, err)
		WriteAPIError(w, http.StatusInternalServerError, "GroupPhotosError", "Failed to retrieve group photos")
		return
	}

	files := make([]FileInfo, 0, len(images))
	for i := range images {
		files = append(files, imageToFileInfo(&images[i]))
	}

	listing := DirectoryListing{
		Path:    "/groups/" + slug + "/photos",
		Files:   files,
		Total:   total,
		Offset:  offset,
		Limit:   limit,
		HasMore: offset+len(files) < total,
	}
	setCacheHeaders(w, 300)
	WriteAPIResponse(w, http.StatusOK, listing)
}
