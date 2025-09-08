package models

import "gorm.io/gorm"

// Face represents a detected face in an image, linked to a person, using GORM.
// It corresponds to the 'faces' table.
type Face struct {
	ID        uint   `gorm:"primaryKey;autoIncrement" json:"id"`
	PersonID  *uint  `gorm:"index" json:"person_id,omitempty"` // Nullable foreign key to people table
	ImagePath string `gorm:"not null;index" json:"image_path"`
	X1        int    `gorm:"not null" json:"x1"`
	Y1        int    `gorm:"not null" json:"y1"`
	X2        int    `gorm:"not null" json:"x2"`
	Y2        int    `gorm:"not null" json:"y2"`

	// face recognition and quality fields
	DetectionConfidence   float32  `gorm:"not null;default:0" json:"detection_confidence"` // confidence from face detection
	RecognitionConfidence *float32 `gorm:"" json:"recognition_confidence,omitempty"`       // confidence from face recognition (nullable)
	QualityScore          *float32 `gorm:"" json:"quality_score,omitempty"`                // overall face quality score (nullable)

	// face landmarks for alignment (stored as JSON array of [x,y] coordinates)
	Landmarks *string `gorm:"" json:"landmarks,omitempty"` // JSON array of 5 landmark points

	// face orientation/pose information
	PoseYaw   *float32 `gorm:"" json:"pose_yaw,omitempty"`   // yaw angle in degrees
	PosePitch *float32 `gorm:"" json:"pose_pitch,omitempty"` // pitch angle in degrees
	PoseRoll  *float32 `gorm:"" json:"pose_roll,omitempty"`  // roll angle in degrees

	CreatedAt int64          `gorm:"not null" json:"created_at"`        // Stored as INTEGER in SQLite, Unix timestamp
	UpdatedAt int64          `gorm:"not null" json:"updated_at"`        // Stored as INTEGER in SQLite, Unix timestamp
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"` // For soft deletes

	Person    *Person        `gorm:"foreignKey:PersonID" json:"person,omitempty"`  // Belongs to Person
	Embedding *FaceEmbedding `gorm:"foreignKey:FaceID" json:"embedding,omitempty"` // Has one embedding
}

// TableName explicitly sets the table name for GORM.
func (Face) TableName() string {
	return "faces"
}
