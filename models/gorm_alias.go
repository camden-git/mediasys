package models

// Alias represents an alternative name for a person in the database using GORM.
// It corresponds to the 'aliases' table.
type Alias struct {
	ID       uint   `gorm:"primaryKey;autoIncrement" json:"id"`
	PersonID uint   `gorm:"not null;uniqueIndex:idx_person_name" json:"person_id"` // Foreign key to people table
	Name     string `gorm:"not null;uniqueIndex:idx_person_name" json:"name"`
}

// TableName explicitly sets the table name for GORM.
func (Alias) TableName() string {
	return "aliases"
}
