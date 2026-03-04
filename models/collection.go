package models

import "gorm.io/gorm"

// Collection represents a virtual grouping of images based on tag filters.
// Collections query images across all albums matching configured tag criteria.
type Collection struct {
	ID                       uint                  `gorm:"primaryKey;autoIncrement" json:"id"`
	Name                     string                `gorm:"not null;unique" json:"name"`
	Slug                     string                `gorm:"not null;unique" json:"slug"`
	Description              *string               `gorm:"" json:"description,omitempty"`
	InheritBannersFromAlbums bool                  `gorm:"not null;default:false" json:"inherit_banners_from_albums"`
	Banners                  []CollectionBanner    `gorm:"foreignKey:CollectionID" json:"-"`
	IsPublic                 bool                  `gorm:"not null;default:true" json:"is_public"`
	FilterMatch              string                `gorm:"not null;default:'all'" json:"filter_match"`
	SortOrder                string                `gorm:"not null;default:'filename_asc'" json:"sort_order"`
	CreatedAt                int64                 `gorm:"not null" json:"created_at"`
	UpdatedAt                int64                 `gorm:"not null" json:"updated_at"`
	DeletedAt                gorm.DeletedAt        `gorm:"index" json:"deleted_at,omitempty"`
	Filters                  []CollectionTagFilter `gorm:"foreignKey:CollectionID" json:"filters,omitempty"`
}

// TableName explicitly sets the table name for GORM.
func (Collection) TableName() string {
	return "collections"
}

// CollectionTagFilter represents a single tag filter for a collection.
// Negate=false means the image must have this tag; Negate=true means the image must NOT have it.
type CollectionTagFilter struct {
	ID           uint   `gorm:"primaryKey;autoIncrement" json:"id"`
	CollectionID uint   `gorm:"not null;index" json:"collection_id"`
	TagKey       string `gorm:"not null" json:"tag_key"`
	TagValue     string `gorm:"not null" json:"tag_value"`
	Negate       bool   `gorm:"not null;default:false" json:"negate"`
}

// TableName explicitly sets the table name for GORM.
func (CollectionTagFilter) TableName() string {
	return "collection_tag_filters"
}
