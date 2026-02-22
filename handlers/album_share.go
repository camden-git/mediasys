package handlers

import (
	"errors"
	"log"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"
)

// ShareAlbumHTML serves minimal HTML with Open Graph/Twitter meta tags so link unfurlers
// (Messages, Slack, Discord, etc.) render album-specific previews.
// Route: GET /share/albums/{album_identifier}
func (ah *AlbumHandler) ShareAlbumHTML(w http.ResponseWriter, r *http.Request) {
	identifier := chi.URLParam(r, "album_identifier")

	album, err := ah.getAlbumByIdentifier(identifier)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			http.NotFound(w, r)
		} else {
			log.Printf("Error getting album for share '%s': %v", identifier, err)
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

	pageURL := absolute("/album/" + album.Slug)

	var imageURL string
	if album.BannerImagePath != nil && *album.BannerImagePath != "" {
		bannersSubDir := filepath.Base(ah.Cfg.BannersPath)
		filename := filepath.Base(*album.BannerImagePath)
		imageURL = absolute("/api/" + bannersSubDir + "/" + filename)
	}

	title := album.Name
	desc := ""
	if album.Description != nil {
		desc = *album.Description
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
		"</head><body><a href=\"" + htmlAttr(pageURL) + "\">Open album</a></body></html>"

	_, _ = w.Write([]byte(html))
}

// htmlEscape escapes text node content.
func htmlEscape(s string) string {
	return strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
	).Replace(s)
}

// htmlAttr escapes attribute values.
func htmlAttr(s string) string {
	return strings.NewReplacer(
		"&", "&amp;",
		"\"", "&quot;",
		"<", "&lt;",
		">", "&gt;",
	).Replace(s)
}
