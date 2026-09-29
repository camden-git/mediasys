package handlers

import (
	"path"
	"sort"
	"strconv"
	"strings"

	"github.com/camden-git/mediasysbackend/database"
	"github.com/camden-git/mediasysbackend/models"
	"github.com/facette/natsort"
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

func parseShutterSeconds(s string) float64 {
	s = strings.TrimSuffix(s, "s")
	if idx := strings.Index(s, "/"); idx >= 0 {
		num, err1 := strconv.ParseFloat(s[:idx], 64)
		den, err2 := strconv.ParseFloat(s[idx+1:], 64)
		if err1 == nil && err2 == nil && den != 0 {
			return num / den
		}
		return 0
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}
	return v
}

func sortCollectionFiles(files []FileInfo, order string) {
	sort.SliceStable(files, func(i, j int) bool {
		return fileInfoOrdering(files[i], files[j], order)
	})
}

func fileInfoOrdering(a, b FileInfo, order string) bool {
	switch order {
	case database.SortFilenameDesc:
		return strings.ToLower(a.Name) > strings.ToLower(b.Name)
	case database.SortFilenameNat:
		return natsort.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name))
	case database.SortDateDesc:
		return collectionCaptureTime(a) > collectionCaptureTime(b)
	case database.SortDateAsc:
		return collectionCaptureTime(a) < collectionCaptureTime(b)
	case database.SortModTimeDesc:
		return a.ModTime > b.ModTime
	case database.SortModTimeAsc:
		return a.ModTime < b.ModTime
	case database.SortFileSizeDesc:
		return a.Size > b.Size
	case database.SortFileSizeAsc:
		return a.Size < b.Size
	case database.SortISODesc:
		return derefInt(a.ISO) > derefInt(b.ISO)
	case database.SortISOAsc:
		return derefInt(a.ISO) < derefInt(b.ISO)
	case database.SortApertureDesc:
		return derefFloat64(a.Aperture) > derefFloat64(b.Aperture)
	case database.SortApertureAsc:
		return derefFloat64(a.Aperture) < derefFloat64(b.Aperture)
	case database.SortFocalLengthDesc:
		return derefFloat64(a.FocalLength) > derefFloat64(b.FocalLength)
	case database.SortFocalLengthAsc:
		return derefFloat64(a.FocalLength) < derefFloat64(b.FocalLength)
	case database.SortShutterSpeedDesc:
		return shutterSpeedSecondsFromPtr(a.ShutterSpeed) > shutterSpeedSecondsFromPtr(b.ShutterSpeed)
	case database.SortShutterSpeedAsc:
		return shutterSpeedSecondsFromPtr(a.ShutterSpeed) < shutterSpeedSecondsFromPtr(b.ShutterSpeed)
	case database.SortCameraAsc:
		return cameraStringFromFileInfo(a) < cameraStringFromFileInfo(b)
	default:
		return strings.ToLower(a.Name) < strings.ToLower(b.Name)
	}
}

func collectionCaptureTime(fi FileInfo) int64 {
	if fi.TakenAt != nil {
		return *fi.TakenAt
	}
	return fi.ModTime
}

func derefInt(p *int) int {
	if p == nil {
		return 0
	}
	return *p
}

func derefFloat64(p *float64) float64 {
	if p == nil {
		return 0
	}
	return *p
}

func shutterSpeedSecondsFromPtr(p *string) float64 {
	if p == nil {
		return 0
	}
	return parseShutterSeconds(*p)
}

func cameraStringFromFileInfo(fi FileInfo) string {
	var parts []string
	if fi.CameraMake != nil {
		parts = append(parts, *fi.CameraMake)
	}
	if fi.CameraModel != nil {
		parts = append(parts, *fi.CameraModel)
	}
	return strings.ToLower(strings.Join(parts, " "))
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

// imagesToSortedFileInfos converts and sorts images using an album/collection sort order.
func imagesToSortedFileInfos(images []models.Image, sortOrder string) []FileInfo {
	files := make([]FileInfo, 0, len(images))
	for i := range images {
		files = append(files, imageToFileInfo(&images[i]))
	}
	sortCollectionFiles(files, sortOrder)
	return files
}

// paginate returns a listing window over files.
func paginate(listingPath string, files []FileInfo, offset, limit int) DirectoryListing {
	total := len(files)
	start := min(max(offset, 0), total)
	end := total
	if limit > 0 {
		end = min(start+limit, total)
	}
	return DirectoryListing{
		Path:    listingPath,
		Files:   files[start:end],
		Total:   total,
		Offset:  start,
		Limit:   limit,
		HasMore: end < total,
	}
}
