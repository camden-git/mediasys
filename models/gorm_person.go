package models

// Person represents a person in the database using GORM.
// It corresponds to the 'people' table.
type Person struct {
	ID          uint   `gorm:"primaryKey;autoIncrement" json:"id"`
	PrimaryName string `gorm:"not null" json:"primary_name"`
	CreatedAt   int64  `gorm:"not null" json:"created_at"` // Stored as INTEGER in SQLite, Unix timestamp
	UpdatedAt   int64  `gorm:"not null" json:"updated_at"` // Stored as INTEGER in SQLite, Unix timestamp

	// Relationships
	// omitempty will hide these if they are not preloaded or are empty
	Aliases []Alias `gorm:"foreignKey:PersonID;constraint:OnDelete:CASCADE" json:"aliases,omitempty"`
	Faces   []Face  `gorm:"foreignKey:PersonID;constraint:OnDelete:SET NULL" json:"faces,omitempty"`
}

// TableName explicitly sets the table name for GORM.
func (Person) TableName() string {
	return "people"
}
