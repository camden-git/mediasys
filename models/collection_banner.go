package models

// CollectionBanner represents a single banner image associated with a collection.
type CollectionBanner struct {
	ID           uint   `gorm:"primaryKey;autoIncrement" json:"id"`
	CollectionID uint   `gorm:"not null;index" json:"collection_id"`
	ImagePath    string `gorm:"not null" json:"image_path"`
	SortOrder    int    `gorm:"not null;default:0" json:"sort_order"`
	CreatedAt    int64  `gorm:"not null" json:"created_at"`
}

func (CollectionBanner) TableName() string { return "collection_banners" }
