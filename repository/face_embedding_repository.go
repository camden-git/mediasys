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
	err := r.DB.Where("face_id = ?", faceID).First(&embedding).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
		return nil, fmt.Errorf("failed to get face embedding by face ID %d: %w", faceID, err)
	}
	return &embedding, nil
}

// DeleteByFaceID hard-deletes (soft-deleted rows stay in the HNSW index) a face embedding by its face ID
func (r *FaceEmbeddingRepository) DeleteByFaceID(faceID uint) error {
	result := r.DB.Unscoped().Where("face_id = ?", faceID).Delete(&models.FaceEmbedding{})
	if result.Error != nil {
		return fmt.Errorf("failed to delete face embedding for face ID %d: %w", faceID, result.Error)
	}
	return nil
}

const (
	// hnswDefaultEfSearch is pgvector's default hnsw.ef_search; a search can never return
	// more rows than ef_search candidates, so larger limits need it raised.
	hnswDefaultEfSearch = 40
	hnswMaxEfSearch     = 1000
)

// UntaggedFace is an untagged face row for the admin tagging queue. It carries no vectors.
type UntaggedFace struct {
	FaceID              uint
	ImagePath           string
	X1, Y1, X2, Y2      int
	DetectionConfidence float32
	QualityScore        *float32
	ImageWidth          *int
	ImageHeight         *int
}

// SimilarFace is a neighbour of a face, scored by cosine similarity (1 = identical).
type SimilarFace struct {
	SourceFaceID   uint // set by NeighborsForFaces: the face this neighbour was found for
	FaceID         uint
	PersonID       *uint
	PersonName     *string
	Confirmed      bool
	ImagePath      string
	X1, Y1, X2, Y2 int
	Similarity     float32
}

// ListUntagged returns up to limit untagged faces that have an embedding, filtered and
// sorted in SQL. With GroupByImage only the best face per image is kept (DISTINCT ON).
// A non-nil albumID restricts results to that album's images.
func (r *FaceEmbeddingRepository) ListUntagged(filter UntaggedFaceFilter, albumID *uint, limit int) ([]UntaggedFace, error) {
	orderDir := "DESC"
	if filter.SortOrder == "asc" {
		orderDir = "ASC"
	}
	sortExpr := "fe.created_at"
	switch filter.SortBy {
	case "quality":
		sortExpr = "f.quality_score"
	case "confidence":
		sortExpr = "f.detection_confidence"
	}
	order := sortExpr + " " + orderDir + " NULLS LAST, f.id " + orderDir

	inner := r.DB.Table("face_embeddings AS fe").
		Joins("JOIN faces f ON f.id = fe.face_id AND f.deleted_at IS NULL").
		Joins("JOIN images i ON i.original_path = f.image_path AND i.deleted_at IS NULL").
		Where("fe.deleted_at IS NULL AND f.person_id IS NULL")
	if albumID != nil {
		inner = inner.Where("i.album_id = ?", *albumID)
	}
	if filter.MinQuality != nil {
		inner = inner.Where("f.quality_score >= ?", *filter.MinQuality)
	}
	if filter.MinConfidence != nil {
		inner = inner.Where("f.detection_confidence >= ?", *filter.MinConfidence)
	}

	cols := "f.id AS face_id, f.image_path, f.x1, f.y1, f.x2, f.y2, f.detection_confidence, f.quality_score, " +
		"i.width AS image_width, i.height AS image_height, fe.created_at AS sort_created_at, " +
		"f.quality_score AS sort_quality, f.detection_confidence AS sort_confidence"

	var rows []UntaggedFace
	var err error
	if filter.GroupByImage {
		// DISTINCT ON needs the image path leading the ORDER BY, so pick the best face per
		// image first, then order and limit the survivors.
		sub := inner.Select("DISTINCT ON (f.image_path) " + cols).Order("f.image_path, " + order)
		outerSort := map[string]string{"quality": "sort_quality", "confidence": "sort_confidence"}[filter.SortBy]
		if outerSort == "" {
			outerSort = "sort_created_at"
		}
		err = r.DB.Table("(?) AS best", sub).
			Order(outerSort + " " + orderDir + " NULLS LAST, face_id " + orderDir).
			Limit(limit).Scan(&rows).Error
	} else {
		err = inner.Select(cols).Order(order).Limit(limit).Scan(&rows).Error
	}
	if err != nil {
		return nil, fmt.Errorf("failed to list untagged faces: %w", err)
	}
	return rows, nil
}

// withEfSearch runs fn in a transaction, raising hnsw.ef_search (local to it) so an index
// scan can return at least limit rows after filtering.
func (r *FaceEmbeddingRepository) withEfSearch(limit int, fn func(tx *gorm.DB) error) error {
	return r.DB.Transaction(func(tx *gorm.DB) error {
		if limit > hnswDefaultEfSearch {
			ef := min(limit*2, hnswMaxEfSearch)
			if err := tx.Exec(fmt.Sprintf("SET LOCAL hnsw.ef_search = %d", ef)).Error; err != nil {
				return fmt.Errorf("failed to set hnsw.ef_search: %w", err)
			}
		}
		return fn(tx)
	})
}

func isZeroVector(v []float32) bool {
	for _, x := range v {
		if x != 0 {
			return false
		}
	}
	return true
}

// FindSimilarFaces returns the faces nearest to targetEmbedding (same embedding model,
// excluding excludeFaceID), most similar first, using the HNSW cosine index. threshold is
// a minimum cosine similarity (0 disables filtering). No vectors are transferred.
func (r *FaceEmbeddingRepository) FindSimilarFaces(targetEmbedding []float32, embeddingModel string, excludeFaceID uint, threshold float32, limit int) ([]SimilarFace, error) {
	if limit <= 0 || len(targetEmbedding) == 0 || isZeroVector(targetEmbedding) {
		// cosine distance to a zero vector is NaN, which would sort as the farthest/nearest arbitrarily
		return []SimilarFace{}, nil
	}
	target := pgvector.NewVector(targetEmbedding)

	var results []SimilarFace
	err := r.withEfSearch(limit, func(tx *gorm.DB) error {
		q := tx.Table("face_embeddings AS fe").
			Select("f.id AS face_id, f.person_id, p.primary_name AS person_name, f.confirmed, f.image_path, f.x1, f.y1, f.x2, f.y2, "+
				"1 - (fe.embedding <=> ?) AS similarity", target).
			Joins("JOIN faces f ON f.id = fe.face_id AND f.deleted_at IS NULL").
			Joins("LEFT JOIN people p ON p.id = f.person_id").
			Where("fe.deleted_at IS NULL AND fe.embedding_model = ? AND fe.face_id <> ?", embeddingModel, excludeFaceID)
		if threshold > 0 {
			q = q.Where("(fe.embedding <=> ?) <= ?", target, 1-threshold)
		}
		return q.Order(gorm.Expr("fe.embedding <=> ?", target)).Limit(limit).Scan(&results).Error
	})
	if err != nil {
		return nil, fmt.Errorf("failed to find similar faces: %w", err)
	}
	return results, nil
}

// NeighborsForFaces finds, in one query, the perFace nearest faces (at or above threshold)
// for every given face ID, using a lateral HNSW lookup per source face. Source faces
// without an embedding, or with a zero vector, yield no neighbours.
func (r *FaceEmbeddingRepository) NeighborsForFaces(faceIDs []uint, threshold float32, perFace int) ([]SimilarFace, error) {
	if len(faceIDs) == 0 || perFace <= 0 {
		return []SimilarFace{}, nil
	}
	var results []SimilarFace
	err := r.withEfSearch(perFace, func(tx *gorm.DB) error {
		return tx.Raw(`
SELECT t.face_id AS source_face_id, n.face_id, n.person_id, p.primary_name AS person_name, n.confirmed,
       n.image_path, n.x1, n.y1, n.x2, n.y2, n.similarity
FROM (SELECT face_id, embedding, embedding_model FROM face_embeddings
      WHERE face_id IN ? AND deleted_at IS NULL) t
CROSS JOIN LATERAL (
    SELECT f.id AS face_id, f.person_id, f.confirmed, f.image_path, f.x1, f.y1, f.x2, f.y2,
           1 - (e.embedding <=> t.embedding) AS similarity
    FROM face_embeddings e
    JOIN faces f ON f.id = e.face_id AND f.deleted_at IS NULL
    WHERE e.deleted_at IS NULL AND e.embedding_model = t.embedding_model AND e.face_id <> t.face_id
    ORDER BY e.embedding <=> t.embedding
    LIMIT ?
) n
LEFT JOIN people p ON p.id = n.person_id
WHERE n.similarity <> 'NaN'::float8 AND n.similarity >= ?
ORDER BY t.face_id, n.similarity DESC`, faceIDs, perFace, threshold).Scan(&results).Error
	})
	if err != nil {
		return nil, fmt.Errorf("failed to find neighbours for faces: %w", err)
	}
	return results, nil
}
