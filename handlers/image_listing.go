package handlers

import (
	"path"

	"github.com/camden-git/mediasysbackend/database"
	"github.com/camden-git/mediasysbackend/models"
)

// FileInfo struct
type FileInfo struct {
	Name            string   `json:"name"`
	Path            string   `json:"path"`
	IsDir           bool     `json:"is_dir"`
	Size            int64    `json:"size"`
	ModTime         int64    `json:"mod_time"`
	ThumbnailPath   *string  `json:"thumbnail_path,omitempty"`
	Width           *int     `json:"width,omitempty"`
	Height          *int     `json:"height,omitempty"`
	Aperture        *float64 `json:"aperture,omitempty"`
	ShutterSpeed    *string  `json:"shutter_speed,omitempty"`
	ISO             *int     `json:"iso,omitempty"`
	FocalLength     *float64 `json:"focal_length,omitempty"`
	LensMake        *string  `json:"lens_make,omitempty"`
	LensModel       *string  `json:"lens_model,omitempty"`
	CameraMake      *string  `json:"camera_make,omitempty"`
	CameraModel     *string  `json:"camera_model,omitempty"`
	TakenAt         *int64   `json:"taken_at,omitempty"`
	Rating          *int     `json:"rating,omitempty"`
	ThumbnailStatus string   `json:"thumbnail_status,omitempty"`
	MetadataStatus  string   `json:"metadata_status,omitempty"`
	DetectionStatus string   `json:"detection_status,omitempty"`
}

type DirectoryListing struct {
	Path    string     `json:"path"`
	Files   []FileInfo `json:"files"`
	Parent  string     `json:"parent,omitempty"`
	Total   int        `json:"total,omitempty"`
	Offset  int        `json:"offset,omitempty"`
	Limit   int        `json:"limit,omitempty"`
	HasMore bool       `json:"has_more,omitempty"`
}

// imageToFileInfo converts an image row to the listing shape the frontend expects.
func imageToFileInfo(img *models.Image) FileInfo {
	fi := FileInfo{
		Name:            path.Base(img.OriginalPath),
		Path:            "/" + img.OriginalPath,
		Size:            img.Size,
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
		// relative to the API base url, e.g. /thumbnails/<uuid>.webp
		u := "/" + *img.ThumbnailPath
		fi.ThumbnailPath = &u
	}
	return fi
}
