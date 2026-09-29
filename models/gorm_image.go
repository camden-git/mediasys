package models

import "gorm.io/gorm"

// Image represents an image record in the database using GORM.
// It corresponds to the 'images' table. The original file lives in object
// storage under ObjectKey; OriginalPath is the logical "<album folder>/<file>"
// identifier used throughout the API.
type Image struct {
	OriginalPath string `gorm:"primaryKey" json:"original_path"`
	AlbumID      uint   `gorm:"not null;index" json:"album_id"`
	ObjectKey    string `gorm:"not null;uniqueIndex" json:"-"`
	Size         int64  `gorm:"not null;default:0" json:"size"`
	ContentType  string `gorm:"not null;default:''" json:"content_type"`
	LastModified int64  `gorm:"not null" json:"last_modified"`
	CreatedAt    int64  `gorm:"not null" json:"created_at"`

	UploadedByUserID *uint `gorm:"index" json:"uploaded_by_user_id,omitempty"`

	Width        *int     `gorm:"" json:"width,omitempty"`         // Nullable
	Height       *int     `gorm:"" json:"height,omitempty"`        // Nullable
	TakenAt      *int64   `gorm:"index" json:"taken_at,omitempty"` // Nullable, Unix timestamp
	CameraMake   *string  `gorm:"" json:"camera_make,omitempty"`   // Nullable
	CameraModel  *string  `gorm:"" json:"camera_model,omitempty"`  // Nullable
	LensMake     *string  `gorm:"" json:"lens_make,omitempty"`     // Nullable
	LensModel    *string  `gorm:"" json:"lens_model,omitempty"`    // Nullable
	FocalLength  *float64 `gorm:"" json:"focal_length,omitempty"`  // Nullable, mm
	Aperture     *float64 `gorm:"" json:"aperture,omitempty"`      // Nullable, F-number
	ShutterSpeed *string  `gorm:"" json:"shutter_speed,omitempty"` // Nullable, e.g., "1/125s"
	ISO          *int     `gorm:"" json:"iso,omitempty"`           // Nullable
	Rating       *int     `gorm:"" json:"rating,omitempty"`        // Nullable, Lightroom XMP star rating (1-5)

	ThumbnailPath *string `gorm:"" json:"thumbnail_path,omitempty"` // object key of the thumbnail
	PreviewPath   *string `gorm:"" json:"preview_path,omitempty"`   // object key of the preview

	MetadataStatus  string `gorm:"not null;default:'pending'" json:"metadata_status"`
	ThumbnailStatus string `gorm:"not null;default:'pending'" json:"thumbnail_status"`
	PreviewStatus   string `gorm:"not null;default:'pending'" json:"preview_status"`
	DetectionStatus string `gorm:"not null;default:'pending'" json:"detection_status"`

	MetadataProcessedAt  *int64 `gorm:"" json:"metadata_processed_at,omitempty"`
	ThumbnailProcessedAt *int64 `gorm:"" json:"thumbnail_processed_at,omitempty"`
	PreviewProcessedAt   *int64 `gorm:"" json:"preview_processed_at,omitempty"`
	DetectionProcessedAt *int64 `gorm:"" json:"detection_processed_at,omitempty"`

	MetadataError  *string `gorm:"" json:"metadata_error,omitempty"`
	ThumbnailError *string `gorm:"" json:"thumbnail_error,omitempty"`
	PreviewError   *string `gorm:"" json:"preview_error,omitempty"`
	DetectionError *string `gorm:"" json:"detection_error,omitempty"`

	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"` // For soft deletes

	// Relationships
	Faces []Face     `gorm:"foreignKey:ImagePath;references:OriginalPath" json:"faces,omitempty"`
	Tags  []ImageTag `gorm:"foreignKey:ImagePath;references:OriginalPath" json:"tags,omitempty"`
}

// TableName explicitly sets the table name for GORM.
func (Image) TableName() string {
	return "images"
}
