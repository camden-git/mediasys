package models

// AlbumBanner represents a single banner image associated with an album.
type AlbumBanner struct {
	ID        uint   `gorm:"primaryKey;autoIncrement" json:"id"`
	AlbumID   uint   `gorm:"not null;index" json:"album_id"`
	ImagePath string `gorm:"not null" json:"image_path"`
	SortOrder int    `gorm:"not null;default:0" json:"sort_order"`
	CreatedAt int64  `gorm:"not null" json:"created_at"`
}

func (AlbumBanner) TableName() string { return "album_banners" }
