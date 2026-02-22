package models

import "gorm.io/gorm"

// AlbumGroup represents a collection of albums under a common event or theme.
type AlbumGroup struct {
	ID              uint           `gorm:"primaryKey;autoIncrement" json:"id"`
	Name            string         `gorm:"not null;unique" json:"name"`
	Slug            string         `gorm:"not null;unique" json:"slug"`
	Description     *string        `gorm:"" json:"description,omitempty"`
	BannerImagePath *string        `gorm:"" json:"banner_image_path,omitempty"`
	IsHidden        bool           `gorm:"not null;default:false" json:"is_hidden"`
	CreatedAt       int64          `gorm:"not null" json:"created_at"`
	UpdatedAt       int64          `gorm:"not null" json:"updated_at"`
	DeletedAt       gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
	Albums          []Album        `gorm:"foreignKey:GroupID" json:"albums,omitempty"`
}

// TableName explicitly sets the table name for GORM.
func (AlbumGroup) TableName() string {
	return "album_groups"
}
