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
	embeddingRepo       *repository.FaceEmbeddingRepository
	similarityThreshold float32
}

// NewFaceRecognitionService creates a new face recognition service
func NewFaceRecognitionService(
	faceRepo repository.FaceRepositoryInterface,
	embeddingRepo *repository.FaceEmbeddingRepository,
	similarityThreshold float32,
) *FaceRecognitionService {
	return &FaceRecognitionService{
		faceRepo:            faceRepo,
		embeddingRepo:       embeddingRepo,
		similarityThreshold: similarityThreshold,
	}
}

// SimilarFaceResult represents a similar face found during recognition
type SimilarFaceResult struct {
	FaceID     uint    `json:"face_id"`
	PersonID   *uint   `json:"person_id,omitempty"`
	PersonName *string `json:"person_name,omitempty"`
	Confirmed  bool    `json:"confirmed"`
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
	if len(targetVector) == 0 {
		return nil, fmt.Errorf("target face has no valid embedding")
	}

	// threshold, model filter and limit are applied in SQL
	similar, err := s.embeddingRepo.FindSimilarFaces(targetVector, targetEmbedding.EmbeddingModel, faceID, s.similarityThreshold, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to find similar faces: %w", err)
	}

	results := make([]SimilarFaceResult, 0, len(similar))
	for _, sf := range similar {
		results = append(results, SimilarFaceResult{
			FaceID:     sf.FaceID,
			PersonID:   sf.PersonID,
			PersonName: sf.PersonName,
			Confirmed:  sf.Confirmed,
			ImagePath:  sf.ImagePath,
			Similarity: sf.Similarity,
			X1:         sf.X1,
			Y1:         sf.Y1,
			X2:         sf.X2,
			Y2:         sf.Y2,
		})
	}
	return results, nil
}

// personVote is one similar face's contribution to a person suggestion.
type personVote struct {
	PersonID   *uint
	PersonName *string
	Similarity float32
}

// PersonSuggestion is the winning person among a set of similar faces.
type PersonSuggestion struct {
	PersonID   *uint
	PersonName *string
	Count      int     // number of similar faces tagged with the suggested person
	Similarity float32 // highest similarity among those faces
}

// suggestPerson picks the person appearing most often among the votes, breaking
// ties by the highest similarity. Untagged votes are ignored.
// Callers must only pass votes from confirmed faces.
func suggestPerson(votes []personVote) PersonSuggestion {
	counts := make(map[uint]int)
	similarities := make(map[uint]float32)
	names := make(map[uint]*string)
	for _, v := range votes {
		if v.PersonID == nil {
			continue
		}
		pid := *v.PersonID
		counts[pid]++
		if v.Similarity > similarities[pid] {
			similarities[pid] = v.Similarity
		}
		if names[pid] == nil {
			names[pid] = v.PersonName
		}
	}

	var best PersonSuggestion
	for pid, count := range counts {
		sim := similarities[pid]
		if count > best.Count || (count == best.Count && sim > best.Similarity) {
			id := pid
			best = PersonSuggestion{PersonID: &id, PersonName: names[pid], Count: count, Similarity: sim}
		}
	}
	return best
}

// SuggestPerson suggests a person for a face based on its most similar faces
func (s *FaceRecognitionService) SuggestPerson(faceID uint) (PersonSuggestion, error) {
	similarFaces, err := s.FindSimilarFaces(faceID, 10)
	if err != nil {
		return PersonSuggestion{}, err
	}

	// only human-confirmed tags vote; auto-tags must not reinforce themselves
	votes := make([]personVote, 0, len(similarFaces))
	for _, sf := range similarFaces {
		if sf.Confirmed {
			votes = append(votes, personVote{PersonID: sf.PersonID, PersonName: sf.PersonName, Similarity: sf.Similarity})
		}
	}
	return suggestPerson(votes), nil
}

// SuggestPersonForFace suggests a person for an untagged face based on similar faces
func (s *FaceRecognitionService) SuggestPersonForFace(faceID uint) (*uint, *string, float32, error) {
	suggestion, err := s.SuggestPerson(faceID)
	if err != nil {
		return nil, nil, 0, err
	}
	return suggestion.PersonID, suggestion.PersonName, suggestion.Similarity, nil
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

// untaggedNeighbours is how many nearest faces are considered per untagged face.
const untaggedNeighbours = 20

// GetUntaggedFacesWithSuggestions returns untagged faces with person suggestions. The page
// is selected in SQL and all neighbours are fetched in one batched query. A non-nil albumID
// restricts the faces to that album.
func (s *FaceRecognitionService) GetUntaggedFacesWithSuggestions(limit int, filter repository.UntaggedFaceFilter, albumID *uint) ([]map[string]interface{}, error) {
	faces, err := s.embeddingRepo.ListUntagged(filter, albumID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get untagged faces: %w", err)
	}

	ids := make([]uint, len(faces))
	for i, f := range faces {
		ids[i] = f.FaceID
	}
	neighbours, err := s.embeddingRepo.NeighborsForFaces(ids, s.similarityThreshold, untaggedNeighbours)
	if err != nil {
		return nil, err
	}
	bySource := make(map[uint][]repository.SimilarFace, len(faces))
	for _, n := range neighbours {
		bySource[n.SourceFaceID] = append(bySource[n.SourceFaceID], n)
	}

	results := make([]map[string]interface{}, 0, len(faces))
	for _, f := range faces {
		similar := bySource[f.FaceID]
		// only confirmed neighbours vote for a person
		var votes []personVote
		for _, n := range similar {
			if n.Confirmed {
				votes = append(votes, personVote{PersonID: n.PersonID, PersonName: n.PersonName, Similarity: n.Similarity})
			}
		}
		suggestion := suggestPerson(votes)

		item := map[string]interface{}{
			"face_id":               f.FaceID,
			"image_path":            f.ImagePath,
			"x1":                    f.X1,
			"y1":                    f.Y1,
			"x2":                    f.X2,
			"y2":                    f.Y2,
			"detection_confidence":  f.DetectionConfidence,
			"quality_score":         f.QualityScore,
			"similar_faces_count":   len(similar),
			"suggested_person_id":   suggestion.PersonID,
			"suggested_person_name": suggestion.PersonName,
			"suggestion_count":      suggestion.Count,
		}
		// original dimensions let the frontend normalise pixel-space boxes
		if f.ImageWidth != nil && f.ImageHeight != nil {
			item["image_width"] = *f.ImageWidth
			item["image_height"] = *f.ImageHeight
		}
		results = append(results, item)
	}
	return results, nil
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
