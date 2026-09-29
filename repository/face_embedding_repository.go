package repository

import (
	"errors"
	"fmt"

	"github.com/camden-git/mediasysbackend/models"
	"github.com/pgvector/pgvector-go"
	"gorm.io/gorm"
)

// FaceEmbeddingRepository handles database operations for FaceEmbedding entities
type FaceEmbeddingRepository struct {
	DB *gorm.DB
}

// Ensure FaceEmbeddingRepository implements FaceEmbeddingRepositoryInterface
var _ FaceEmbeddingRepositoryInterface = (*FaceEmbeddingRepository)(nil)

// NewFaceEmbeddingRepository creates a new instance of FaceEmbeddingRepository
func NewFaceEmbeddingRepository(db *gorm.DB) *FaceEmbeddingRepository {
	return &FaceEmbeddingRepository{DB: db}
}

// GetByFaceID retrieves a face embedding by its face ID
func (r *FaceEmbeddingRepository) GetByFaceID(faceID uint) (*models.FaceEmbedding, error) {
	var embedding models.FaceEmbedding
	err := r.DB.Where("face_id = ?", faceID).Preload("Face").First(&embedding).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
		return nil, fmt.Errorf("failed to get face embedding by face ID %d: %w", faceID, err)
	}
	return &embedding, nil
}

// DeleteByFaceID removes a face embedding by its face ID
func (r *FaceEmbeddingRepository) DeleteByFaceID(faceID uint) error {
	result := r.DB.Where("face_id = ?", faceID).Delete(&models.FaceEmbedding{})
	if result.Error != nil {
		return fmt.Errorf("failed to delete face embedding for face ID %d: %w", faceID, result.Error)
	}
	return nil
}

// GetUntaggedEmbeddingsFiltered retrieves untagged face embeddings with optional quality/confidence
// filtering and configurable sort order.
func (r *FaceEmbeddingRepository) GetUntaggedEmbeddingsFiltered(filter UntaggedFaceFilter) ([]models.FaceEmbedding, error) {
	q := r.DB.Joins("JOIN faces ON face_embeddings.face_id = faces.id AND faces.deleted_at IS NULL").
		Where("faces.person_id IS NULL")

	if filter.MinQuality != nil {
		q = q.Where("faces.quality_score >= ?", *filter.MinQuality)
	}
	if filter.MinConfidence != nil {
		q = q.Where("faces.detection_confidence >= ?", *filter.MinConfidence)
	}

	orderDir := "DESC"
	if filter.SortOrder == "asc" {
		orderDir = "ASC"
	}
	switch filter.SortBy {
	case "quality":
		q = q.Order("faces.quality_score " + orderDir)
	case "confidence":
		q = q.Order("faces.detection_confidence " + orderDir)
	default:
		q = q.Order("face_embeddings.created_at " + orderDir)
	}

	var embeddings []models.FaceEmbedding
	if err := q.Preload("Face").Find(&embeddings).Error; err != nil {
		return nil, fmt.Errorf("failed to get filtered untagged embeddings: %w", err)
	}
	return embeddings, nil
}

// FindSimilarFaces finds faces with similar embeddings to a given embedding, ordered by
// cosine similarity (highest first), using pgvector's HNSW index (embedding <=> $1) rather
// than pulling every embedding into memory. threshold is a minimum cosine similarity
// (0.0 disables filtering); it is translated to a max cosine *distance* of (1 - threshold)
// for the SQL WHERE clause, since pgvector's <=> operator returns cosine distance.
func (r *FaceEmbeddingRepository) FindSimilarFaces(targetEmbedding []float32, threshold float32, limit int) ([]models.FaceEmbedding, error) {
	target := pgvector.NewVector(targetEmbedding)

	var embeddings []models.FaceEmbedding
	q := r.DB.Preload("Face").
		Preload("Face.Person").
		Where("embedding IS NOT NULL")

	if threshold > 0 {
		q = q.Where("(embedding <=> ?) <= ?", target, 1-threshold)
	}

	// gorm's Order() has no (query, args) overload, so the vector is rendered inline; it is
	// safe to embed since pgvector.Vector.String() only emits floats, e.g. "[0.1,0.2,...]".
	orderExpr := fmt.Sprintf("embedding <=> '%s'", target.String())
	err := q.Order(orderExpr).Limit(limit).Find(&embeddings).Error
	if err != nil {
		return nil, fmt.Errorf("failed to find similar faces: %w", err)
	}

	return embeddings, nil
}
