package repository

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/camden-git/mediasysbackend/database"
	"github.com/camden-git/mediasysbackend/media"
	"github.com/camden-git/mediasysbackend/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var _ ImageRepositoryInterface = (*ImageRepository)(nil)

// ImageRepository handles database operations for Image entities
type ImageRepository struct {
	DB *gorm.DB
}

// NewImageRepository creates a new instance of ImageRepository
func NewImageRepository(db *gorm.DB) *ImageRepository {
	return &ImageRepository{DB: db}
}

var taskColumns = map[string]string{
	"metadata_status":  "metadata_error",
	"thumbnail_status": "thumbnail_error",
	"preview_status":   "preview_error",
	"detection_status": "detection_error",
}

// GetByPath retrieves full image info by its path
func (r *ImageRepository) GetByPath(originalPath string) (*models.Image, error) {
	var image models.Image
	err := r.DB.Where("original_path = ?", originalPath).First(&image).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
		return nil, fmt.Errorf("failed to get image by path %s: %w", originalPath, err)
	}
	return &image, nil
}

// Upsert inserts a freshly uploaded image, or resets an existing row with the
// same path so it gets reprocessed. The previous row (if any) is returned so the
// caller can clean up its generated assets.
func (r *ImageRepository) Upsert(img *models.Image) (*models.Image, error) {
	var previous *models.Image
	err := r.DB.Transaction(func(tx *gorm.DB) error {
		var existing models.Image
		err := tx.Unscoped().Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("original_path = ?", img.OriginalPath).First(&existing).Error
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		if err == nil {
			previous = &existing
			if err := deleteImageRelations(tx, []string{img.OriginalPath}, false); err != nil {
				return err
			}
			if err := tx.Unscoped().Where("original_path = ?", img.OriginalPath).Delete(&models.Image{}).Error; err != nil {
				return err
			}
		}
		return tx.Create(img).Error
	})
	if err != nil {
		return nil, fmt.Errorf("failed to upsert image %s: %w", img.OriginalPath, err)
	}
	return previous, nil
}

// MarkTaskProcessing updates a specific task's status to 'processing' and clears its error
func (r *ImageRepository) MarkTaskProcessing(originalPath, taskStatusColumn string) error {
	errorColumn, ok := taskColumns[taskStatusColumn]
	if !ok {
		return fmt.Errorf("invalid task status column name: %s", taskStatusColumn)
	}
	result := r.DB.Model(&models.Image{}).Where("original_path = ?", originalPath).Updates(map[string]interface{}{
		taskStatusColumn: database.StatusProcessing,
		errorColumn:      nil,
	})
	if result.Error != nil {
		return fmt.Errorf("failed to mark task %s processing for %s: %w", taskStatusColumn, originalPath, result.Error)
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func taskResult(taskErr error) (string, *string) {
	if taskErr != nil {
		s := taskErr.Error()
		return database.StatusError, &s
	}
	return database.StatusDone, nil
}

// UpdateThumbnailResult records the outcome of thumbnail generation
func (r *ImageRepository) UpdateThumbnailResult(originalPath string, thumbKey *string, taskErr error) error {
	status, errStr := taskResult(taskErr)
	return r.DB.Model(&models.Image{}).Where("original_path = ?", originalPath).Updates(map[string]interface{}{
		"thumbnail_path":         thumbKey,
		"thumbnail_status":       status,
		"thumbnail_processed_at": time.Now().Unix(),
		"thumbnail_error":        errStr,
	}).Error
}

// UpdatePreviewResult records the outcome of preview generation
func (r *ImageRepository) UpdatePreviewResult(originalPath string, previewKey *string, taskErr error) error {
	status, errStr := taskResult(taskErr)
	return r.DB.Model(&models.Image{}).Where("original_path = ?", originalPath).Updates(map[string]interface{}{
		"preview_path":         previewKey,
		"preview_status":       status,
		"preview_processed_at": time.Now().Unix(),
		"preview_error":        errStr,
	}).Error
}

// UpdateMetadataResult records extracted metadata
func (r *ImageRepository) UpdateMetadataResult(originalPath string, meta *media.Metadata, taskErr error) error {
	status, errStr := taskResult(taskErr)
	updateData := map[string]interface{}{
		"metadata_status":       status,
		"metadata_processed_at": time.Now().Unix(),
		"metadata_error":        errStr,
	}
	if meta != nil {
		updateData["width"] = meta.Width
		updateData["height"] = meta.Height
		updateData["aperture"] = meta.Aperture
		updateData["shutter_speed"] = meta.ShutterSpeed
		updateData["iso"] = meta.ISO
		updateData["focal_length"] = meta.FocalLength
		updateData["lens_make"] = meta.LensMake
		updateData["lens_model"] = meta.LensModel
		updateData["camera_make"] = meta.CameraMake
		updateData["camera_model"] = meta.CameraModel
		updateData["taken_at"] = meta.TakenAt
		updateData["rating"] = meta.Rating
	}
	return r.DB.Model(&models.Image{}).Where("original_path = ?", originalPath).Updates(updateData).Error
}

// UpdateDetectionResult replaces untagged faces with fresh detections
func (r *ImageRepository) UpdateDetectionResult(originalPath string, detections []media.DetectionResult, taskErr error) error {
	status, errStr := taskResult(taskErr)
	if taskErr != nil {
		detections = nil
	}

	return r.DB.Transaction(func(tx *gorm.DB) error {
		var oldFaceIDs []uint
		if err := tx.Unscoped().Model(&models.Face{}).
			Where("image_path = ? AND person_id IS NULL", originalPath).Pluck("id", &oldFaceIDs).Error; err != nil {
			return err
		}
		if len(oldFaceIDs) > 0 {
			if err := tx.Unscoped().Where("face_id IN ?", oldFaceIDs).Delete(&models.FaceEmbedding{}).Error; err != nil {
				return err
			}
			if err := tx.Unscoped().Where("id IN ?", oldFaceIDs).Delete(&models.Face{}).Error; err != nil {
				return err
			}
		}

		if len(detections) > 0 {
			now := time.Now().Unix()
			newFaces := make([]models.Face, len(detections))
			for i, det := range detections {
				var landmarksStr *string
				if len(det.Landmarks) > 0 {
					if b, err := json.Marshal(det.Landmarks); err == nil {
						s := string(b)
						landmarksStr = &s
					}
				}
				newFaces[i] = models.Face{
					ImagePath:           originalPath,
					X1:                  det.X,
					Y1:                  det.Y,
					X2:                  det.X + det.W,
					Y2:                  det.Y + det.H,
					DetectionConfidence: det.Confidence,
					QualityScore:        det.QualityScore,
					Landmarks:           landmarksStr,
					PoseYaw:             det.PoseYaw,
					PosePitch:           det.PosePitch,
					PoseRoll:            det.PoseRoll,
					CreatedAt:           now,
					UpdatedAt:           now,
				}
			}
			if err := tx.Create(&newFaces).Error; err != nil {
				return fmt.Errorf("failed to add detected faces for %s: %w", originalPath, err)
			}
			for i, det := range detections {
				if len(det.Embedding) == 0 {
					continue
				}
				embedding := &models.FaceEmbedding{FaceID: newFaces[i].ID, EmbeddingModel: det.ModelName}
				embedding.SetEmbedding(det.Embedding)
				if err := tx.Create(embedding).Error; err != nil {
					return fmt.Errorf("failed to create face embedding for face %d: %w", newFaces[i].ID, err)
				}
			}
			log.Printf("repository: stored %d face(s) for %s", len(newFaces), originalPath)
		}

		return tx.Model(&models.Image{}).Where("original_path = ?", originalPath).Updates(map[string]interface{}{
			"detection_status":       status,
			"detection_processed_at": time.Now().Unix(),
			"detection_error":        errStr,
		}).Error
	})
}

// RequeueTask sets a task back to 'pending' so the dispatcher runs it again.
func (r *ImageRepository) RequeueTask(originalPath, taskStatusColumn string) error {
	errorColumn, ok := taskColumns[taskStatusColumn]
	if !ok {
		return fmt.Errorf("invalid task status column name: %s", taskStatusColumn)
	}
	result := r.DB.Model(&models.Image{}).Where("original_path = ?", originalPath).Updates(map[string]interface{}{
		taskStatusColumn: database.StatusPending,
		errorColumn:      nil,
	})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// ResetInterruptedTasks puts any task left in 'processing' (e.g. after a crash) back to 'pending'.
func (r *ImageRepository) ResetInterruptedTasks() error {
	for col := range taskColumns {
		if err := r.DB.Model(&models.Image{}).Where(col+" = ?", database.StatusProcessing).
			Update(col, database.StatusPending).Error; err != nil {
			return err
		}
	}
	return nil
}

// ListPendingProcessing returns images that still need metadata, thumbnail or preview work.
func (r *ImageRepository) ListPendingProcessing(limit int) ([]models.Image, error) {
	var images []models.Image
	err := r.DB.Where("metadata_status = ? OR thumbnail_status = ? OR preview_status = ?",
		database.StatusPending, database.StatusPending, database.StatusPending).
		Order("created_at ASC").Limit(limit).Find(&images).Error
	return images, err
}

// ListPendingDetection returns images that still need face detection.
func (r *ImageRepository) ListPendingDetection(limit int) ([]models.Image, error) {
	var images []models.Image
	err := r.DB.Where("detection_status = ?", database.StatusPending).
		Order("created_at ASC").Limit(limit).Find(&images).Error
	return images, err
}

// ListByAlbum returns every image in an album, optionally filtered by minimum rating.
func (r *ImageRepository) ListByAlbum(albumID uint, minRating *int) ([]models.Image, error) {
	q := r.DB.Where("album_id = ?", albumID)
	if minRating != nil {
		q = q.Where("rating >= ?", *minRating)
	}
	var images []models.Image
	if err := q.Order("original_path ASC").Find(&images).Error; err != nil {
		return nil, fmt.Errorf("failed to list images for album %d: %w", albumID, err)
	}
	return images, nil
}

// ListByAlbumPaged returns a sorted, paginated page of images in an album, plus the
// total number of matching images, with sorting and paging performed in SQL.
func (r *ImageRepository) ListByAlbumPaged(albumID uint, minRating *int, sortOrder string, offset, limit int) ([]models.Image, int, error) {
	db := r.DB.Model(&models.Image{}).Where("album_id = ?", albumID)
	if minRating != nil {
		db = db.Where("rating >= ?", *minRating)
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count images for album %d: %w", albumID, err)
	}

	var images []models.Image
	if err := db.Order(database.SQLOrderClause(sortOrder)).Offset(offset).Limit(limit).Find(&images).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to list images for album %d: %w", albumID, err)
	}
	return images, int(total), nil
}

// GetImagesByPaths retrieves multiple image records by their paths
func (r *ImageRepository) GetImagesByPaths(originalPaths []string) ([]models.Image, error) {
	if len(originalPaths) == 0 {
		return []models.Image{}, nil
	}
	var images []models.Image
	if err := r.DB.Where("original_path IN ?", originalPaths).Find(&images).Error; err != nil {
		return nil, fmt.Errorf("failed to get images by paths: %w", err)
	}
	return images, nil
}

// GetImagesByAlbumIDs returns paginated images from the given albums, newest first.
func (r *ImageRepository) GetImagesByAlbumIDs(albumIDs []uint, minRating *int, offset, limit int) ([]models.Image, int, error) {
	if len(albumIDs) == 0 {
		return nil, 0, nil
	}
	db := r.DB.Model(&models.Image{}).Where("album_id IN ?", albumIDs)
	if minRating != nil {
		db = db.Where("rating >= ?", *minRating)
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count group photos: %w", err)
	}

	var images []models.Image
	err := db.Order("taken_at DESC NULLS LAST, created_at DESC, original_path ASC").
		Offset(offset).Limit(limit).Find(&images).Error
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query group photos: %w", err)
	}
	return images, int(total), nil
}

// GetDistinctUploaderIDsByAlbum returns distinct uploader user IDs for an album
func (r *ImageRepository) GetDistinctUploaderIDsByAlbum(albumID uint) ([]uint, error) {
	var ids []uint
	err := r.DB.Model(&models.Image{}).
		Where("album_id = ? AND uploaded_by_user_id IS NOT NULL", albumID).
		Distinct().Pluck("uploaded_by_user_id", &ids).Error
	if err != nil {
		return nil, fmt.Errorf("failed to query uploaders for album %d: %w", albumID, err)
	}
	return ids, nil
}

// DeleteImages hard-deletes images plus their faces, embeddings and tags. It
// returns every object key that belonged to them so the caller can remove the
// objects from storage.
func (r *ImageRepository) DeleteImages(paths []string) ([]string, error) {
	if len(paths) == 0 {
		return nil, nil
	}
	var imgs []models.Image
	if err := r.DB.Unscoped().Where("original_path IN ?", paths).Find(&imgs).Error; err != nil {
		return nil, err
	}
	err := r.DB.Transaction(func(tx *gorm.DB) error {
		if err := deleteImageRelations(tx, paths, true); err != nil {
			return err
		}
		return tx.Unscoped().Where("original_path IN ?", paths).Delete(&models.Image{}).Error
	})
	if err != nil {
		return nil, fmt.Errorf("failed to delete images: %w", err)
	}
	return ImageObjectKeys(imgs), nil
}

// DeleteByAlbum hard-deletes every image in an album. See DeleteImages.
func (r *ImageRepository) DeleteByAlbum(albumID uint) ([]string, error) {
	var paths []string
	if err := r.DB.Unscoped().Model(&models.Image{}).Where("album_id = ?", albumID).Pluck("original_path", &paths).Error; err != nil {
		return nil, err
	}
	return r.DeleteImages(paths)
}

// ImageObjectKeys lists the original and generated object keys of the given images.
func ImageObjectKeys(imgs []models.Image) []string {
	keys := make([]string, 0, len(imgs)*3)
	for _, img := range imgs {
		keys = append(keys, img.ObjectKey)
		if img.ThumbnailPath != nil {
			keys = append(keys, *img.ThumbnailPath)
		}
		if img.PreviewPath != nil {
			keys = append(keys, *img.PreviewPath)
		}
	}
	return keys
}

// deleteImageRelations removes faces, embeddings and tags for the given images.
// When includeTagged is false, faces already assigned to a person are kept and
// only non-manual tags are removed (used when an image is re-uploaded).
func deleteImageRelations(tx *gorm.DB, paths []string, includeTagged bool) error {
	faceQuery := tx.Unscoped().Model(&models.Face{}).Where("image_path IN ?", paths)
	if !includeTagged {
		faceQuery = faceQuery.Where("person_id IS NULL")
	}
	var faceIDs []uint
	if err := faceQuery.Pluck("id", &faceIDs).Error; err != nil {
		return err
	}
	if len(faceIDs) > 0 {
		if err := tx.Unscoped().Where("face_id IN ?", faceIDs).Delete(&models.FaceEmbedding{}).Error; err != nil {
			return err
		}
		if err := tx.Unscoped().Where("id IN ?", faceIDs).Delete(&models.Face{}).Error; err != nil {
			return err
		}
	}
	tagQuery := tx.Where("image_path IN ?", paths)
	if !includeTagged {
		tagQuery = tagQuery.Where("source <> ?", "manual")
	}
	return tagQuery.Delete(&models.ImageTag{}).Error
}
