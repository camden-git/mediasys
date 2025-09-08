package repository

import (
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/camden-git/mediasysbackend/models"
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
	result := r.DB.Model(&models.FaceEmbedding{ID: embedding.ID}).Updates(models.FaceEmbedding{
		EmbeddingData:  embedding.EmbeddingData,
		EmbeddingModel: embedding.EmbeddingModel,
		QualityScore:   embedding.QualityScore,
		UpdatedAt:      embedding.UpdatedAt,
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

// FindSimilarFaces finds faces with similar embeddings to a given embedding
func (r *FaceEmbeddingRepository) FindSimilarFaces(targetEmbedding []float32, threshold float32, limit int) ([]models.FaceEmbedding, error) {
	var embeddings []models.FaceEmbedding

	// Get all embeddings to compare against
	err := r.DB.Preload("Face").Find(&embeddings).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get embeddings for similarity search: %w", err)
	}

	// Calculate similarities and create pairs for sorting
	type embeddingWithSimilarity struct {
		embedding  models.FaceEmbedding
		similarity float32
	}

	var embeddingPairs []embeddingWithSimilarity
	for _, embedding := range embeddings {
		embeddingVector := embedding.GetEmbedding()
		if embeddingVector != nil {
			similarity := calculateCosineSimilarity(targetEmbedding, embeddingVector)
			embeddingPairs = append(embeddingPairs, embeddingWithSimilarity{
				embedding:  embedding,
				similarity: similarity,
			})
		}
	}

	// Sort by similarity (highest first)
	for i := 0; i < len(embeddingPairs)-1; i++ {
		for j := i + 1; j < len(embeddingPairs); j++ {
			if embeddingPairs[i].similarity < embeddingPairs[j].similarity {
				embeddingPairs[i], embeddingPairs[j] = embeddingPairs[j], embeddingPairs[i]
			}
		}
	}

	// Return top results (let the service handle threshold filtering)
	var result []models.FaceEmbedding
	for i, pair := range embeddingPairs {
		if i >= limit {
			break
		}
		result = append(result, pair.embedding)
	}

	return result, nil
}

// calculateCosineSimilarity calculates cosine similarity between two embedding vectors
func calculateCosineSimilarity(embedding1, embedding2 []float32) float32 {
	if len(embedding1) != len(embedding2) || len(embedding1) == 0 {
		return 0.0
	}

	var dotProduct float32
	var norm1 float32
	var norm2 float32

	for i := 0; i < len(embedding1); i++ {
		dotProduct += embedding1[i] * embedding2[i]
		norm1 += embedding1[i] * embedding1[i]
		norm2 += embedding2[i] * embedding2[i]
	}

	if norm1 == 0 || norm2 == 0 {
		return 0.0
	}

	// Calculate the square roots of the squared norms to get the actual L2 norms
	norm1Sqrt := float32(math.Sqrt(float64(norm1)))
	norm2Sqrt := float32(math.Sqrt(float64(norm2)))

	return dotProduct / (norm1Sqrt * norm2Sqrt)
}
