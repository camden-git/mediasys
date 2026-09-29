package models

import (
	"github.com/pgvector/pgvector-go"
	"gorm.io/gorm"
)

// FaceEmbeddingDimension is the length of the embedding vector produced by
// media.FaceRecognitionModel for the "arcface" model (see media/face_recognition.go).
const FaceEmbeddingDimension = 512

// FaceEmbedding represents a face embedding vector used for face recognition
// It corresponds to the 'face_embeddings' table.
type FaceEmbedding struct {
	ID             uint            `gorm:"primaryKey;autoIncrement" json:"id"`
	FaceID         uint            `gorm:"uniqueIndex;not null" json:"face_id"`                                      // Foreign key to faces table
	Embedding      pgvector.Vector `gorm:"not null;column:embedding;type:vector(512)" json:"-"`                      // face embedding vector, stored as pgvector for similarity search
	EmbeddingModel string          `gorm:"not null;column:embedding_model;default:'arcface'" json:"embedding_model"` // Name of the model used for embedding
	QualityScore   *float32        `gorm:"column:quality_score" json:"quality_score,omitempty"`                      // Quality score of the embedding
	CreatedAt      int64           `gorm:"not null" json:"created_at"`                                               // Unix timestamp
	UpdatedAt      int64           `gorm:"not null" json:"updated_at"`                                               // Unix timestamp
	DeletedAt      gorm.DeletedAt  `gorm:"index" json:"deleted_at,omitempty"`                                        // For soft deletes

	Face *Face `gorm:"foreignKey:FaceID" json:"face,omitempty"` // Belongs to Face
}

// TableName explicitly sets the table name for GORM.
func (FaceEmbedding) TableName() string {
	return "face_embeddings"
}

// GetEmbedding returns the embedding as a []float32.
func (fe *FaceEmbedding) GetEmbedding() []float32 {
	return fe.Embedding.Slice()
}

// SetEmbedding sets the embedding from a []float32.
func (fe *FaceEmbedding) SetEmbedding(embedding []float32) {
	fe.Embedding = pgvector.NewVector(embedding)
}
