package handlers

import (
	"github.com/camden-git/mediasysbackend/database"
	"github.com/camden-git/mediasysbackend/models"
)

// PublicAlbum is the public view of an album. It never exposes the archive's
// object key, its error text or soft-delete state; clients use zip_ready to
// decide whether the download link works.
type PublicAlbum struct {
	ID                 uint    `json:"id"`
	Name               string  `json:"name"`
	Slug               string  `json:"slug"`
	Description        *string `json:"description,omitempty"`
	FolderPath         string  `json:"folder_path"`
	SortOrder          string  `json:"sort_order"`
	ZipSize            *int64  `json:"zip_size,omitempty"`
	ZipStatus          string  `json:"zip_status"`
	ZipReady           bool    `json:"zip_ready"`
	ZipLastGeneratedAt *int64  `json:"zip_last_generated_at,omitempty"`
	ZipLastRequestedAt *int64  `json:"zip_last_requested_at,omitempty"`
	CreatedAt          int64   `json:"created_at"`
	UpdatedAt          int64   `json:"updated_at"`
	Location           *string `json:"location,omitempty"`
	GroupID            *uint   `json:"group_id,omitempty"`
}

func toPublicAlbum(a *models.Album) PublicAlbum {
	return PublicAlbum{
		ID:                 a.ID,
		Name:               a.Name,
		Slug:               a.Slug,
		Description:        a.Description,
		FolderPath:         a.FolderPath,
		SortOrder:          a.SortOrder,
		ZipSize:            a.ZipSize,
		ZipStatus:          a.ZipStatus,
		ZipReady:           a.ZipStatus == database.StatusDone && a.ZipPath != nil && *a.ZipPath != "",
		ZipLastGeneratedAt: a.ZipLastGeneratedAt,
		ZipLastRequestedAt: a.ZipLastRequestedAt,
		CreatedAt:          a.CreatedAt,
		UpdatedAt:          a.UpdatedAt,
		Location:           a.Location,
		GroupID:            a.GroupID,
	}
}

func toPublicAlbums(albums []models.Album) []PublicAlbum {
	out := make([]PublicAlbum, len(albums))
	for i := range albums {
		out[i] = toPublicAlbum(&albums[i])
	}
	return out
}

// PublicAlbumGroup is the public view of an album group and its albums.
type PublicAlbumGroup struct {
	ID              uint          `json:"id"`
	Name            string        `json:"name"`
	Slug            string        `json:"slug"`
	Description     *string       `json:"description,omitempty"`
	BannerImagePath *string       `json:"banner_image_path,omitempty"`
	IsHidden        bool          `json:"is_hidden"`
	CreatedAt       int64         `json:"created_at"`
	UpdatedAt       int64         `json:"updated_at"`
	Albums          []PublicAlbum `json:"albums,omitempty"`
}

func toPublicAlbumGroup(g *models.AlbumGroup) PublicAlbumGroup {
	return PublicAlbumGroup{
		ID:              g.ID,
		Name:            g.Name,
		Slug:            g.Slug,
		Description:     g.Description,
		BannerImagePath: g.BannerImagePath,
		IsHidden:        g.IsHidden,
		CreatedAt:       g.CreatedAt,
		UpdatedAt:       g.UpdatedAt,
		Albums:          toPublicAlbums(g.Albums),
	}
}
