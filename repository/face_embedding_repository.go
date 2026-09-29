package repository

import (
	"errors"
	"fmt"
	"time"

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

// Create creates a new face embedding record in the database
func (r *FaceEmbeddingRepository) Create(embedding *models.FaceEmbedding) error {
	now := time.Now().Unix()
	if embedding.CreatedAt == 0 {
		embedding.CreatedAt = now
	}
	embedding.UpdatedAt = now

	err := r.DB.Create(embedding).Error
	if err != nil {
		return fmt.Errorf("failed to create face embedding for face ID %d: %w", embedding.FaceID, err)
	}
	return nil
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

// GetByID retrieves a face embedding by its ID
func (r *FaceEmbeddingRepository) GetByID(id uint) (*models.FaceEmbedding, error) {
	var embedding models.FaceEmbedding
	err := r.DB.Preload("Face").First(&embedding, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
		return nil, fmt.Errorf("failed to get face embedding by ID %d: %w", id, err)
	}
	return &embedding, nil
}

// Update updates an existing face embedding
func (r *FaceEmbeddingRepository) Update(embedding *models.FaceEmbedding) error {
	embedding.UpdatedAt = time.Now().Unix()
	result := r.DB.Model(&models.FaceEmbedding{ID: embedding.ID}).Updates(map[string]interface{}{
		"embedding":       embedding.Embedding,
		"embedding_model": embedding.EmbeddingModel,
		"quality_score":   embedding.QualityScore,
		"updated_at":      embedding.UpdatedAt,
	})

	if result.Error != nil {
		return fmt.Errorf("failed to update face embedding ID %d: %w", embedding.ID, result.Error)
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// Delete removes a face embedding by its ID
func (r *FaceEmbeddingRepository) Delete(id uint) error {
	result := r.DB.Delete(&models.FaceEmbedding{}, id)
	if result.Error != nil {
		return fmt.Errorf("failed to delete face embedding ID %d: %w", id, result.Error)
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// DeleteByFaceID removes a face embedding by its face ID
func (r *FaceEmbeddingRepository) DeleteByFaceID(faceID uint) error {
	result := r.DB.Where("face_id = ?", faceID).Delete(&models.FaceEmbedding{})
	if result.Error != nil {
		return fmt.Errorf("failed to delete face embedding for face ID %d: %w", faceID, result.Error)
	}
	return nil
}

// GetEmbeddingsByPersonID retrieves all face embeddings for a given person
func (r *FaceEmbeddingRepository) GetEmbeddingsByPersonID(personID uint) ([]models.FaceEmbedding, error) {
	var embeddings []models.FaceEmbedding
	err := r.DB.Joins("JOIN faces ON face_embeddings.face_id = faces.id").
		Where("faces.person_id = ?", personID).
		Preload("Face").
		Find(&embeddings).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get embeddings for person ID %d: %w", personID, err)
	}
	return embeddings, nil
}

// GetUntaggedEmbeddings retrieves all face embeddings for untagged faces
func (r *FaceEmbeddingRepository) GetUntaggedEmbeddings() ([]models.FaceEmbedding, error) {
	var embeddings []models.FaceEmbedding
	err := r.DB.Joins("JOIN faces ON face_embeddings.face_id = faces.id").
		Where("faces.person_id IS NULL").
		Preload("Face").
		Find(&embeddings).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get untagged embeddings: %w", err)
	}
	return embeddings, nil
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

// GetEmbeddingsByImagePath retrieves all face embeddings for a given image
func (r *FaceEmbeddingRepository) GetEmbeddingsByImagePath(imagePath string) ([]models.FaceEmbedding, error) {
	var embeddings []models.FaceEmbedding
	err := r.DB.Joins("JOIN faces ON face_embeddings.face_id = faces.id").
		Where("faces.image_path = ?", imagePath).
		Preload("Face").
		Find(&embeddings).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get embeddings for image %s: %w", imagePath, err)
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
