package models

import "gorm.io/gorm"

// Face represents a detected face in an image, linked to a person, using GORM.
// It corresponds to the 'faces' table.
type Face struct {
	ID        uint           `gorm:"primaryKey;autoIncrement" json:"id"`
	PersonID  *uint          `gorm:"index" json:"person_id,omitempty"` // Nullable foreign key to people table
	ImagePath string         `gorm:"not null;index" json:"image_path"`
	X1        int            `gorm:"not null" json:"x1"`
	Y1        int            `gorm:"not null" json:"y1"`
	X2        int            `gorm:"not null" json:"x2"`
	Y2        int            `gorm:"not null" json:"y2"`
	CreatedAt int64          `gorm:"not null" json:"created_at"`        // Stored as INTEGER in SQLite, Unix timestamp
	UpdatedAt int64          `gorm:"not null" json:"updated_at"`        // Stored as INTEGER in SQLite, Unix timestamp
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"` // For soft deletes

	Person *Person `gorm:"foreignKey:PersonID" json:"person,omitempty"` // Belongs to Person
}

// TableName explicitly sets the table name for GORM.
func (Face) TableName() string {
	return "faces"
}
