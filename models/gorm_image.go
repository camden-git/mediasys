package models

import "gorm.io/gorm"

// Image represents an image record in the database using GORM.
// It corresponds to the 'images' table.
type Image struct {
	OriginalPath string `gorm:"primaryKey" json:"original_path"` // path relative to ROOT_DIRECTORY
	LastModified int64  `gorm:"not null" json:"last_modified"`

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

	ThumbnailPath *string `gorm:"" json:"thumbnail_path,omitempty"` // Nullable

	MetadataStatus  string `gorm:"not null;default:pending" json:"metadata_status"`
	ThumbnailStatus string `gorm:"not null;default:pending" json:"thumbnail_status"`
	DetectionStatus string `gorm:"not null;default:pending" json:"detection_status"`

	MetadataProcessedAt  *int64 `gorm:"" json:"metadata_processed_at,omitempty"`  // Nullable, Unix timestamp
	ThumbnailProcessedAt *int64 `gorm:"" json:"thumbnail_processed_at,omitempty"` // Nullable, Unix timestamp
	DetectionProcessedAt *int64 `gorm:"" json:"detection_processed_at,omitempty"` // Nullable, Unix timestamp

	MetadataError  *string `gorm:"" json:"metadata_error,omitempty"`  // Nullable
	ThumbnailError *string `gorm:"" json:"thumbnail_error,omitempty"` // Nullable
	DetectionError *string `gorm:"" json:"detection_error,omitempty"` // Nullable

	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"` // For soft deletes

	// Relationships
	Faces []Face `gorm:"foreignKey:ImagePath;references:OriginalPath" json:"faces,omitempty"`
}

// TableName explicitly sets the table name for GORM.
func (Image) TableName() string {
	return "images"
}
