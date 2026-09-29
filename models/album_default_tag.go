package models

import "time"

// AlbumDefaultTag represents a default key-value tag configured on an album.
// When a new image is added to the album, these tags are applied to the image
// with source="album_default".
type AlbumDefaultTag struct {
	ID        uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	AlbumID   uint      `gorm:"not null;uniqueIndex:idx_album_default_tags_unique" json:"album_id"`
	TagKey    string    `gorm:"not null;uniqueIndex:idx_album_default_tags_unique" json:"tag_key"`
	TagValue  string    `gorm:"not null;uniqueIndex:idx_album_default_tags_unique" json:"tag_value"`
	CreatedAt time.Time `gorm:"not null" json:"created_at"`
}

// TableName explicitly sets the table name for GORM.
func (AlbumDefaultTag) TableName() string {
	return "album_default_tags"
}
