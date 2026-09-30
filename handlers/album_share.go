package handlers

import (
	"errors"
	"log"
	"net/http"

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
			writeShareNotFound(w)
		} else {
			log.Printf("Error getting album for share '%s': %v", identifier, err)
			WriteAPIError(w, http.StatusInternalServerError, "AlbumFetchError", "Internal Server Error")
		}
		return
	}

	absolute := absoluteURLFunc(r, ah.PublicURL)

	pageURL := absolute("/album/" + album.Slug)

	var imageURL string
	banners, err := ah.AlbumRepo.GetBanners(album.ID)
	if err != nil {
		log.Printf("Error loading banners for share page of album %d: %v", album.ID, err)
	} else if len(banners) > 0 {
		imageURL = absolute("/api/" + banners[0].ImagePath)
	}

	title := album.Name
	desc := ""
	if album.Description != nil {
		desc = *album.Description
	}

	writeSharePage(w, sharePage{Title: title, Desc: desc, PageURL: pageURL, ImageURL: imageURL, LinkText: "Open album"})
}
