package models

import (
	"gorm.io/gorm"
	"math"
)

// FaceEmbedding represents a face embedding vector used for face recognition
// It corresponds to the 'face_embeddings' table.
type FaceEmbedding struct {
	ID             uint           `gorm:"primaryKey;autoIncrement" json:"id"`
	FaceID         uint           `gorm:"uniqueIndex;not null" json:"face_id"`                                      // Foreign key to faces table
	EmbeddingData  []byte         `gorm:"not null;column:embedding_data" json:"embedding_data"`                     // 128-dimensional face embedding vector as BLOB
	EmbeddingModel string         `gorm:"not null;column:embedding_model;default:'arcface'" json:"embedding_model"` // Name of the model used for embedding
	QualityScore   *float32       `gorm:"column:quality_score" json:"quality_score,omitempty"`                      // Quality score of the embedding
	CreatedAt      int64          `gorm:"not null" json:"created_at"`                                               // Stored as INTEGER in SQLite, Unix timestamp
	UpdatedAt      int64          `gorm:"not null" json:"updated_at"`                                               // Stored as INTEGER in SQLite, Unix timestamp
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`                                        // For soft deletes

	Face *Face `gorm:"foreignKey:FaceID" json:"face,omitempty"` // Belongs to Face
}

// TableName explicitly sets the table name for GORM.
func (FaceEmbedding) TableName() string {
	return "face_embeddings"
}

// GetEmbedding converts the BLOB data to []float32
func (fe *FaceEmbedding) GetEmbedding() []float32 {
	if len(fe.EmbeddingData) == 0 {
		return nil
	}

	// Convert []byte to []float32
	embedding := make([]float32, len(fe.EmbeddingData)/4) // 4 bytes per float32
	for i := 0; i < len(embedding); i++ {
		offset := i * 4
		bits := uint32(fe.EmbeddingData[offset]) |
			uint32(fe.EmbeddingData[offset+1])<<8 |
			uint32(fe.EmbeddingData[offset+2])<<16 |
			uint32(fe.EmbeddingData[offset+3])<<24
		embedding[i] = math.Float32frombits(bits)
	}
	return embedding
}

// SetEmbedding converts []float32 to BLOB data
func (fe *FaceEmbedding) SetEmbedding(embedding []float32) {
	if len(embedding) == 0 {
		fe.EmbeddingData = nil
		return
	}

	// Convert []float32 to []byte
	fe.EmbeddingData = make([]byte, len(embedding)*4) // 4 bytes per float32
	for i, val := range embedding {
		offset := i * 4
		bits := math.Float32bits(val)
		fe.EmbeddingData[offset] = byte(bits)
		fe.EmbeddingData[offset+1] = byte(bits >> 8)
		fe.EmbeddingData[offset+2] = byte(bits >> 16)
		fe.EmbeddingData[offset+3] = byte(bits >> 24)
	}
}
