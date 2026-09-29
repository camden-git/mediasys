package handlers

import (
	"errors"
	"log"
	"net/http"
	"strings"

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
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
		return
	}

	scheme := "http"
	if r.Header.Get("X-Forwarded-Proto") == "https" || r.TLS != nil {
		scheme = "https"
	}
	host := r.Header.Get("X-Forwarded-Host")
	if host == "" {
		host = r.Host
	}

	absolute := func(path string) string {
		if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
			return path
		}
		if !strings.HasPrefix(path, "/") {
			path = "/" + path
		}
		return scheme + "://" + host + path
	}

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

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	html := "<!doctype html><html lang=\"en\"><head>" +
		"<meta charset=\"utf-8\">" +
		"<meta name=\"viewport\" content=\"width=device-width, initial-scale=1\">" +
		"<title>" + htmlEscape(title) + "</title>" +
		"<meta property=\"og:type\" content=\"website\">" +
		"<meta property=\"og:url\" content=\"" + htmlAttr(pageURL) + "\">" +
		"<meta property=\"og:title\" content=\"" + htmlAttr(title) + "\">" +
		"<meta property=\"og:description\" content=\"" + htmlAttr(desc) + "\">"
	if imageURL != "" {
		html += "<meta property=\"og:image\" content=\"" + htmlAttr(imageURL) + "\">" +
			"<meta property=\"og:image:alt\" content=\"" + htmlAttr(title) + "\">"
	}
	html += "<meta name=\"twitter:card\" content=\"summary_large_image\">" +
		"<meta name=\"twitter:title\" content=\"" + htmlAttr(title) + "\">" +
		"<meta name=\"twitter:description\" content=\"" + htmlAttr(desc) + "\">"
	if imageURL != "" {
		html += "<meta name=\"twitter:image\" content=\"" + htmlAttr(imageURL) + "\">"
	}
	html += "<meta http-equiv=\"refresh\" content=\"0;url=" + htmlAttr(pageURL) + "\">" +
		"</head><body><a href=\"" + htmlAttr(pageURL) + "\">Open collection</a></body></html>"

	_, _ = w.Write([]byte(html))
}
