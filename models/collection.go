package models

import "gorm.io/gorm"

// Collection represents a virtual grouping of images based on tag filters.
// Collections query images across all albums matching configured tag criteria.
type Collection struct {
	ID              uint                  `gorm:"primaryKey;autoIncrement" json:"id"`
	Name            string                `gorm:"not null;unique" json:"name"`
	Slug            string                `gorm:"not null;unique" json:"slug"`
	Description     *string               `gorm:"" json:"description,omitempty"`
	BannerImagePath *string               `gorm:"" json:"banner_image_path,omitempty"`
	IsPublic        bool                  `gorm:"not null;default:true" json:"is_public"`
	CreatedAt       int64                 `gorm:"not null" json:"created_at"`
	UpdatedAt       int64                 `gorm:"not null" json:"updated_at"`
	DeletedAt       gorm.DeletedAt        `gorm:"index" json:"deleted_at,omitempty"`
	Filters         []CollectionTagFilter `gorm:"foreignKey:CollectionID" json:"filters,omitempty"`
}

// TableName explicitly sets the table name for GORM.
func (Collection) TableName() string {
	return "collections"
}

// CollectionTagFilter represents a single tag filter for a collection.
// AND logic across distinct tag_key values; OR logic within the same tag_key.
type CollectionTagFilter struct {
	ID           uint   `gorm:"primaryKey;autoIncrement" json:"id"`
	CollectionID uint   `gorm:"not null;index" json:"collection_id"`
	TagKey       string `gorm:"not null" json:"tag_key"`
	TagValue     string `gorm:"not null" json:"tag_value"`
}

// TableName explicitly sets the table name for GORM.
func (CollectionTagFilter) TableName() string {
	return "collection_tag_filters"
}
