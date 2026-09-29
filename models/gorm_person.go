package models

// Person represents a person in the database using GORM.
// It corresponds to the 'people' table.
type Person struct {
	ID          uint   `gorm:"primaryKey;autoIncrement" json:"id"`
	PrimaryName string `gorm:"not null" json:"primary_name"`
	CreatedAt   int64  `gorm:"not null" json:"created_at"` // Unix timestamp
	UpdatedAt   int64  `gorm:"not null" json:"updated_at"` // Unix timestamp

	KeyPhotoFaceID *uint `gorm:"index" json:"key_photo_face_id,omitempty"`

	// Relationships
	// omitempty will hide these if they are not preloaded or are empty
	Aliases []Alias `gorm:"foreignKey:PersonID" json:"aliases,omitempty"`
	Faces   []Face  `gorm:"foreignKey:PersonID" json:"faces,omitempty"`
}

// TableName explicitly sets the table name for GORM.
func (Person) TableName() string {
	return "people"
}
