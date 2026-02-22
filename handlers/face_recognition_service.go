package handlers

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
	FaceID     uint    `json:"face_id"`
	PersonID   *uint   `json:"person_id,omitempty"`
	PersonName *string `json:"person_name,omitempty"`
	ImagePath  string  `json:"image_path"`
	Similarity float32 `json:"similarity"`
	X1         int     `json:"x1"`
	Y1         int     `json:"y1"`
	X2         int     `json:"x2"`
	Y2         int     `json:"y2"`
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

// TagFaceWithPerson tags a face with a person and updates related faces.
// confirmed should be true when a human explicitly approved the assignment; auto-tagged similar faces are always unconfirmed.
func (s *FaceRecognitionService) TagFaceWithPerson(faceID uint, personID uint, confirmed bool) error {
	// Tag the target face with the requested confirmation state
	err := s.faceRepo.TagFace(faceID, personID, confirmed)
	if err != nil {
		return fmt.Errorf("failed to tag face %d with person %d: %w", faceID, personID, err)
	}

	// Find similar faces and suggest tagging them too
	similarFaces, err := s.FindSimilarFaces(faceID, 20)
	if err != nil {
		log.Printf("Warning: Failed to find similar faces for auto-tagging: %v", err)
		return nil // Don't fail the main operation
	}

	// Auto-tag faces with high similarity that are untagged; these are never confirmed
	for _, similarFace := range similarFaces {
		if similarFace.PersonID == nil && similarFace.Similarity > 0.8 {
			err := s.faceRepo.TagFace(similarFace.FaceID, personID, false)
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
func (s *FaceRecognitionService) GetUntaggedFacesWithSuggestions(limit int, filter repository.UntaggedFaceFilter) ([]map[string]interface{}, error) {
	// Query 1: untagged embeddings with Face preloaded (filtered + sorted at DB level)
	untaggedEmbeddings, err := s.embeddingRepo.GetUntaggedEmbeddingsFiltered(filter)
	if err != nil {
		return nil, fmt.Errorf("failed to get untagged embeddings: %w", err)
	}

	// Group-by-image: keep only the first (best) embedding per image path.
	// The embeddings are already sorted by the requested criterion, so the first
	// occurrence per image_path is the "best" one.
	if filter.GroupByImage {
		seen := make(map[string]bool, len(untaggedEmbeddings))
		filtered := untaggedEmbeddings[:0]
		for _, e := range untaggedEmbeddings {
			if e.Face == nil {
				continue
			}
			if !seen[e.Face.ImagePath] {
				seen[e.Face.ImagePath] = true
				filtered = append(filtered, e)
			}
		}
		untaggedEmbeddings = filtered
	}

	// Query 2: ALL embeddings with Face + Person preloaded (GORM issues batched preload queries)
	allEmbeddings, err := s.embeddingRepo.GetAllEmbeddings()
	if err != nil {
		return nil, fmt.Errorf("failed to get all embeddings: %w", err)
	}

	// Build in-memory index: faceID -> vector + person info — O(M) once
	type embeddingInfo struct {
		vector     []float32
		personID   *uint
		personName *string
	}
	embeddingIndex := make(map[uint]embeddingInfo, len(allEmbeddings))
	for _, e := range allEmbeddings {
		vec := e.GetEmbedding()
		if vec == nil {
			continue
		}
		info := embeddingInfo{vector: vec}
		if e.Face != nil {
			info.personID = e.Face.PersonID
			if e.Face.Person != nil {
				name := e.Face.Person.PrimaryName
				info.personName = &name
			}
		}
		embeddingIndex[e.FaceID] = info
	}

	var results []map[string]interface{}
	for i, embedding := range untaggedEmbeddings {
		if i >= limit {
			break
		}
		targetVec := embedding.GetEmbedding()
		if targetVec == nil || embedding.Face == nil {
			continue
		}

		// Find similar faces entirely in memory — no DB calls
		type simResult struct {
			personID   *uint
			personName *string
			similarity float32
		}
		var similar []simResult
		for otherFaceID, info := range embeddingIndex {
			if otherFaceID == embedding.FaceID {
				continue
			}
			sim := s.CalculateSimilarity(targetVec, info.vector)
			if sim >= s.similarityThreshold {
				similar = append(similar, simResult{
					personID:   info.personID,
					personName: info.personName,
					similarity: sim,
				})
			}
		}

		// Vote for suggested person by count, breaking ties by similarity
		personCounts := make(map[uint]int)
		personSimilarities := make(map[uint]float32)
		personNames := make(map[uint]*string)
		for _, sf := range similar {
			if sf.personID != nil {
				pid := *sf.personID
				personCounts[pid]++
				if sf.similarity > personSimilarities[pid] {
					personSimilarities[pid] = sf.similarity
				}
				if personNames[pid] == nil {
					personNames[pid] = sf.personName
				}
			}
		}

		var suggestedPersonID *uint
		var suggestedPersonName *string
		maxCount := 0
		var bestSim float32
		for pid, count := range personCounts {
			sim := personSimilarities[pid]
			if count > maxCount || (count == maxCount && sim > bestSim) {
				maxCount = count
				bestSim = sim
				pid2 := pid
				suggestedPersonID = &pid2
				suggestedPersonName = personNames[pid]
			}
		}

		results = append(results, map[string]interface{}{
			"face_id":               embedding.FaceID,
			"image_path":            embedding.Face.ImagePath,
			"x1":                    embedding.Face.X1,
			"y1":                    embedding.Face.Y1,
			"x2":                    embedding.Face.X2,
			"y2":                    embedding.Face.Y2,
			"detection_confidence":  embedding.Face.DetectionConfidence,
			"quality_score":         embedding.Face.QualityScore,
			"similar_faces_count":   len(similar),
			"suggested_person_id":   suggestedPersonID,
			"suggested_person_name": suggestedPersonName,
			"suggestion_count":      maxCount,
		})
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
