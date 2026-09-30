package handlers

import (
	"errors"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"
)

// ShareCollectionHTML serves minimal HTML with Open Graph/Twitter meta tags so link unfurlers
// (Messages, Slack, Discord, etc.) render collection-specific previews.
// Route: GET /share/collections/{slug}
func (h *CollectionHandler) ShareCollectionHTML(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")

	collection, err := h.CollectionRepo.GetBySlug(slug)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			http.NotFound(w, r)
		} else {
			log.Printf("Error getting collection for share '%s': %v", slug, err)
			WriteAPIError(w, http.StatusInternalServerError, "CollectionFetchError", "Internal Server Error")
		}
		return
	}

	absolute := absoluteURLFunc(r, h.PublicURL)

	pageURL := absolute("/collections/" + collection.Slug)

	var imageURL string
	var firstBannerPath string
	if collection.InheritBannersFromAlbums {
		if paths, err := h.CollectionRepo.GetInheritedBannerPaths(collection.ID); err == nil && len(paths) > 0 {
			firstBannerPath = paths[0]
		}
	} else {
		if banners, err := h.CollectionRepo.GetBanners(collection.ID); err == nil && len(banners) > 0 {
			firstBannerPath = banners[0].ImagePath
		}
	}
	if firstBannerPath != "" {
		imageURL = absolute("/api/" + firstBannerPath)
	}

	title := collection.Name
	desc := ""
	if collection.Description != nil {
		desc = *collection.Description
	}

	writeSharePage(w, sharePage{Title: title, Desc: desc, PageURL: pageURL, ImageURL: imageURL, LinkText: "Open collection"})
}
