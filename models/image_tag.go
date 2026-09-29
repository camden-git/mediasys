package models

import "time"

// ImageTag represents a key-value tag attached to an image.
// Tags can come from XMP keywords (source="xmp"), album defaults (source="album_default"),
// or manually added via the admin UI (source="manual").
type ImageTag struct {
	ID        uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	ImagePath string    `gorm:"not null;index;uniqueIndex:idx_image_tags_unique" json:"image_path"`
	TagKey    string    `gorm:"not null;uniqueIndex:idx_image_tags_unique;index:idx_image_tags_key_value" json:"tag_key"`
	TagValue  string    `gorm:"not null;uniqueIndex:idx_image_tags_unique;index:idx_image_tags_key_value" json:"tag_value"`
	Source    string    `gorm:"not null;uniqueIndex:idx_image_tags_unique" json:"source"` // "xmp" | "album_default" | "manual"
	CreatedAt time.Time `gorm:"not null" json:"created_at"`
}

// TableName explicitly sets the table name for GORM.
func (ImageTag) TableName() string {
	return "image_tags"
}
