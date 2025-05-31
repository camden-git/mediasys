package models

import "gorm.io/gorm"

// Album represents an album of images in the database using GORM.
// It corresponds to the 'albums' table.
type Album struct {
	ID                 uint           `gorm:"primaryKey;autoIncrement" json:"id"`
	Name               string         `gorm:"not null;unique" json:"name"`
	Slug               string         `gorm:"not null;unique" json:"slug"`
	Description        *string        `gorm:"" json:"description,omitempty"` // Nullable
	FolderPath         string         `gorm:"not null;unique" json:"folder_path"`
	BannerImagePath    *string        `gorm:"" json:"banner_image_path,omitempty"` // Nullable
	SortOrder          string         `gorm:"not null;default:'name_asc'" json:"sort_order"`
	ZipPath            *string        `gorm:"" json:"zip_path,omitempty"` // Nullable
	ZipSize            *int64         `gorm:"" json:"zip_size,omitempty"` // Nullable
	ZipStatus          string         `gorm:"not null;default:notRequired" json:"zip_status"`
	ZipLastGeneratedAt *int64         `gorm:"" json:"zip_last_generated_at,omitempty"` // Nullable, Unix timestamp
	ZipLastRequestedAt *int64         `gorm:"" json:"zip_last_requested_at,omitempty"` // Nullable, Unix timestamp
	ZipError           *string        `gorm:"" json:"zip_error,omitempty"`             // Nullable
	CreatedAt          int64          `gorm:"not null" json:"created_at"`              // Stored as INTEGER in SQLite, Unix timestamp
	UpdatedAt          int64          `gorm:"not null" json:"updated_at"`              // Stored as INTEGER in SQLite, Unix timestamp
	IsHidden           bool           `gorm:"not null;default:false" json:"-"`
	Location           *string        `gorm:"" json:"location,omitempty"`        // Nullable
	DeletedAt          gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"` // For soft deletes
}

// TableName explicitly sets the table name for GORM.
func (Album) TableName() string {
	return "albums"
}
