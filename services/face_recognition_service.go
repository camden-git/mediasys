package services

import (
	"fmt"
	"log"
	"math"

	"github.com/camden-git/mediasysbackend/repository"
)

// FaceRecognitionService provides high-level face recognition operations
type FaceRecognitionService struct {
	faceRepo            repository.FaceRepositoryInterface
	personRepo          repository.PersonRepositoryInterface
	embeddingRepo       *repository.FaceEmbeddingRepository
	similarityThreshold float32
}

// NewFaceRecognitionService creates a new face recognition service
func NewFaceRecognitionService(
	faceRepo repository.FaceRepositoryInterface,
	personRepo repository.PersonRepositoryInterface,
	embeddingRepo *repository.FaceEmbeddingRepository,
	similarityThreshold float32,
) *FaceRecognitionService {
	return &FaceRecognitionService{
		faceRepo:            faceRepo,
		personRepo:          personRepo,
		embeddingRepo:       embeddingRepo,
		similarityThreshold: similarityThreshold,
	}
}

// SimilarFaceResult represents a similar face found during recognition
type SimilarFaceResult struct {
	FaceID         uint    `json:"face_id"`
	PersonID       *uint   `json:"person_id,omitempty"`
	PersonName     *string `json:"person_name,omitempty"`
	ImagePath      string  `json:"image_path"`
	Similarity     float32 `json:"similarity"`
	X1, Y1, X2, Y2 int     `json:"x1, y1, x2, y2"`
}

// FindSimilarFaces finds faces similar to a given face ID
func (s *FaceRecognitionService) FindSimilarFaces(faceID uint, limit int) ([]SimilarFaceResult, error) {
	// Get the target face embedding
	targetEmbedding, err := s.embeddingRepo.GetByFaceID(faceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get target face embedding: %w", err)
	}

	targetVector := targetEmbedding.GetEmbedding()
	if targetVector == nil {
		return nil, fmt.Errorf("target face has no valid embedding")
	}

	// Find similar embeddings (without threshold filtering, we'll do that in the service)
	similarEmbeddings, err := s.embeddingRepo.FindSimilarFaces(targetVector, 0.0, limit*2) // Get more candidates
	if err != nil {
		return nil, fmt.Errorf("failed to find similar faces: %w", err)
	}

	// Convert to results and apply threshold filtering
	var results []SimilarFaceResult
	for _, embedding := range similarEmbeddings {
		if embedding.FaceID == faceID {
			continue // Skip the target face itself
		}

		embeddingVector := embedding.GetEmbedding()
		if embeddingVector == nil {
			continue
		}

		// Calculate similarity
		similarity := s.CalculateSimilarity(targetVector, embeddingVector)

		// Apply threshold filtering
		if similarity < s.similarityThreshold {
			continue
		}

		// Check if Face is loaded
		if embedding.Face == nil {
			log.Printf("Warning: Face data not loaded for embedding %d, skipping", embedding.FaceID)
			continue
		}

		result := SimilarFaceResult{
			FaceID:     embedding.FaceID,
			ImagePath:  embedding.Face.ImagePath,
			Similarity: similarity,
			X1:         embedding.Face.X1,
			Y1:         embedding.Face.Y1,
			X2:         embedding.Face.X2,
			Y2:         embedding.Face.Y2,
		}

		// Add person information if available
		if embedding.Face.PersonID != nil {
			result.PersonID = embedding.Face.PersonID
			if embedding.Face.Person != nil {
				result.PersonName = &embedding.Face.Person.PrimaryName
			}
		}

		results = append(results, result)

		// Limit results
		if len(results) >= limit {
			break
		}
	}

	return results, nil
}

// SuggestPersonForFace suggests a person for an untagged face based on similar faces
func (s *FaceRecognitionService) SuggestPersonForFace(faceID uint) (*uint, *string, float32, error) {
	// Get similar faces
	similarFaces, err := s.FindSimilarFaces(faceID, 10)
	if err != nil {
		return nil, nil, 0, err
	}

	// Count person occurrences
	personCounts := make(map[uint]int)
	personSimilarities := make(map[uint]float32)

	for _, similarFace := range similarFaces {
		if similarFace.PersonID != nil {
			personCounts[*similarFace.PersonID]++
			if similarFace.Similarity > personSimilarities[*similarFace.PersonID] {
				personSimilarities[*similarFace.PersonID] = similarFace.Similarity
			}
		}
	}

	// Find the most common person with highest similarity
	var bestPersonID *uint
	var bestPersonName *string
	var bestSimilarity float32
	maxCount := 0

	for personID, count := range personCounts {
		similarity := personSimilarities[personID]
		if count > maxCount || (count == maxCount && similarity > bestSimilarity) {
			maxCount = count
			bestSimilarity = similarity
			bestPersonID = &personID

			// Get person name
			person, err := s.personRepo.GetByID(personID)
			if err == nil {
				bestPersonName = &person.PrimaryName
			}
		}
	}

	return bestPersonID, bestPersonName, bestSimilarity, nil
}

// TagFaceWithPerson tags a face with a person and updates related faces
func (s *FaceRecognitionService) TagFaceWithPerson(faceID uint, personID uint) error {
	// Tag the target face
	err := s.faceRepo.TagFace(faceID, personID)
	if err != nil {
		return fmt.Errorf("failed to tag face %d with person %d: %w", faceID, personID, err)
	}

	// Find similar faces and suggest tagging them too
	similarFaces, err := s.FindSimilarFaces(faceID, 20)
	if err != nil {
		log.Printf("Warning: Failed to find similar faces for auto-tagging: %v", err)
		return nil // Don't fail the main operation
	}

	// Auto-tag faces with high similarity that are untagged
	for _, similarFace := range similarFaces {
		if similarFace.PersonID == nil && similarFace.Similarity > 0.8 {
			err := s.faceRepo.TagFace(similarFace.FaceID, personID)
			if err != nil {
				log.Printf("Warning: Failed to auto-tag similar face %d: %v", similarFace.FaceID, err)
			} else {
				log.Printf("Auto-tagged face %d with person %d (similarity: %.3f)", similarFace.FaceID, personID, similarFace.Similarity)
			}
		}
	}

	return nil
}

// GetUntaggedFacesWithSuggestions returns untagged faces with person suggestions
func (s *FaceRecognitionService) GetUntaggedFacesWithSuggestions(limit int) ([]map[string]interface{}, error) {
	// Get untagged embeddings
	untaggedEmbeddings, err := s.embeddingRepo.GetUntaggedEmbeddings()
	if err != nil {
		return nil, fmt.Errorf("failed to get untagged embeddings: %w", err)
	}

	var results []map[string]interface{}
	for i, embedding := range untaggedEmbeddings {
		if i >= limit {
			break
		}

		// Get similar faces for this untagged face
		similarFaces, err := s.FindSimilarFaces(embedding.FaceID, 5)
		if err != nil {
			log.Printf("Warning: Failed to find similar faces for face %d: %v", embedding.FaceID, err)
			continue
		}

		// Count person suggestions
		personSuggestions := make(map[uint]int)
		for _, similarFace := range similarFaces {
			if similarFace.PersonID != nil {
				personSuggestions[*similarFace.PersonID]++
			}
		}

		// Find most suggested person
		var suggestedPersonID *uint
		var suggestedPersonName *string
		maxSuggestions := 0
		for personID, count := range personSuggestions {
			if count > maxSuggestions {
				maxSuggestions = count
				suggestedPersonID = &personID

				// Get person name
				person, err := s.personRepo.GetByID(personID)
				if err == nil {
					suggestedPersonName = &person.PrimaryName
				}
			}
		}

		result := map[string]interface{}{
			"face_id":               embedding.FaceID,
			"image_path":            embedding.Face.ImagePath,
			"x1":                    embedding.Face.X1,
			"y1":                    embedding.Face.Y1,
			"x2":                    embedding.Face.X2,
			"y2":                    embedding.Face.Y2,
			"detection_confidence":  embedding.Face.DetectionConfidence,
			"quality_score":         embedding.Face.QualityScore,
			"similar_faces_count":   len(similarFaces),
			"suggested_person_id":   suggestedPersonID,
			"suggested_person_name": suggestedPersonName,
			"suggestion_count":      maxSuggestions,
		}

		results = append(results, result)
	}

	return results, nil
}

// GetEmbeddingRepo returns the embedding repository for debugging
func (s *FaceRecognitionService) GetEmbeddingRepo() *repository.FaceEmbeddingRepository {
	return s.embeddingRepo
}

// GetSimilarityThreshold returns the similarity threshold for debugging
func (s *FaceRecognitionService) GetSimilarityThreshold() float32 {
	return s.similarityThreshold
}

// CalculateSimilarity calculates cosine similarity between two embeddings
func (s *FaceRecognitionService) CalculateSimilarity(embedding1, embedding2 []float32) float32 {
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
