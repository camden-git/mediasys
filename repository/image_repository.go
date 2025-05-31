package repository

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/camden-git/mediasysbackend/database"
	"github.com/camden-git/mediasysbackend/media"
	"github.com/camden-git/mediasysbackend/models"
	"gorm.io/gorm"
)

// ImageRepository handles database operations for Image entities
type ImageRepository struct {
	DB *gorm.DB
}

// NewImageRepository creates a new instance of ImageRepository
func NewImageRepository(db *gorm.DB) *ImageRepository {
	return &ImageRepository{DB: db}
}

// GetByPath retrieves full image info by its original path
func (r *ImageRepository) GetByPath(originalPath string) (*models.Image, error) {
	var image models.Image
	// GORM automatically respects soft deletes if DeletedAt is on the model
	err := r.DB.Where("original_path = ?", originalPath).First(&image).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, err
		}
		return nil, fmt.Errorf("failed to get image by path %s: %w", originalPath, err)
	}
	return &image, nil
}

// EnsureExists creates a basic image record if it doesn't exist, setting tasks to pending
// returns true if a new record was created, false otherwise
func (r *ImageRepository) EnsureExists(originalPath string, modTime int64) (bool, error) {
	cleanPath := filepath.ToSlash(originalPath)
	image := models.Image{
		OriginalPath:    cleanPath,
		LastModified:    modTime,
		MetadataStatus:  database.StatusPending,
		ThumbnailStatus: database.StatusPending,
		DetectionStatus: database.StatusPending,
	}

	result := r.DB.Where(models.Image{OriginalPath: cleanPath}).FirstOrCreate(&image)

	if result.Error != nil {
		return false, fmt.Errorf("failed to ensure image record for %s: %w", cleanPath, result.Error)
	}

	return result.RowsAffected > 0, nil
}

// MarkTaskProcessing updates a specific task's status to 'processing' and clears its error
func (r *ImageRepository) MarkTaskProcessing(originalPath, taskStatusColumn string) error {
	cleanPath := filepath.ToSlash(originalPath)
	validStatusColumns := map[string]string{
		"metadata_status":  "metadata_error",
		"thumbnail_status": "thumbnail_error",
		"detection_status": "detection_error",
	}

	errorColumn, isValid := validStatusColumns[taskStatusColumn]
	if !isValid {
		return fmt.Errorf("invalid task status column name: %s", taskStatusColumn)
	}

	updates := map[string]interface{}{
		taskStatusColumn: database.StatusProcessing,
		errorColumn:      gorm.Expr("NULL"),
	}

	result := r.DB.Model(&models.Image{}).Where("original_path = ?", cleanPath).Updates(updates)
	if result.Error != nil {
		return fmt.Errorf("failed to mark task %s processing for %s: %w", taskStatusColumn, cleanPath, result.Error)
	}
	if result.RowsAffected == 0 {
		// this could mean the record doesn't exist
	}
	return nil
}

// UpdateThumbnailResult updates the image record with thumbnail generation results
func (r *ImageRepository) UpdateThumbnailResult(originalPath string, thumbPath *string, modTime int64, taskErr error) error {
	cleanPath := filepath.ToSlash(originalPath)
	now := time.Now().Unix()
	status := database.StatusDone
	var errStr *string

	if taskErr != nil {
		status = database.StatusError
		s := taskErr.Error()
		errStr = &s
	}

	updates := models.Image{
		ThumbnailPath:        thumbPath,
		LastModified:         modTime,
		ThumbnailStatus:      status,
		ThumbnailProcessedAt: &now,
		ThumbnailError:       errStr,
	}

	result := r.DB.Model(&models.Image{}).Where("original_path = ?", cleanPath).Updates(updates)
	if result.Error != nil {
		return fmt.Errorf("failed to update thumbnail result for %s: %w", cleanPath, result.Error)
	}
	return nil
}

// UpdateMetadataResult updates the image record with metadata extraction results
func (r *ImageRepository) UpdateMetadataResult(originalPath string, meta *media.Metadata, modTime int64, taskErr error) error {
	cleanPath := filepath.ToSlash(originalPath)
	now := time.Now().Unix()
	status := database.StatusDone
	var errStr *string

	if taskErr != nil {
		status = database.StatusError
		s := taskErr.Error()
		errStr = &s
	}

	updateData := map[string]interface{}{
		"last_modified":         modTime,
		"metadata_status":       status,
		"metadata_processed_at": &now,
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
	}

	result := r.DB.Model(&models.Image{}).Where("original_path = ?", cleanPath).Updates(updateData)
	if result.Error != nil {
		return fmt.Errorf("failed to update metadata result for %s: %w", cleanPath, result.Error)
	}
	return nil
}

// UpdateDetectionResult updates the image record with face detection results
func (r *ImageRepository) UpdateDetectionResult(originalPath string, detections []media.DetectionResult, modTime int64, taskErr error) error {
	cleanPath := filepath.ToSlash(originalPath)
	now := time.Now().Unix()
	status := database.StatusDone
	var errStr *string

	if taskErr != nil {
		status = database.StatusError
		s := taskErr.Error()
		errStr = &s
		detections = nil // do not process detections if there was an error
	}

	return r.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("image_path = ? AND person_id IS NULL", cleanPath).Delete(&models.Face{}).Error; err != nil {
			return fmt.Errorf("failed to delete old untagged faces for %s: %w", cleanPath, err)
		}

		if taskErr == nil && len(detections) > 0 {
			newFaces := make([]models.Face, len(detections))
			faceCreatedAt := time.Now().Unix() // all faces in this batch get same timestamp
			for i, det := range detections {
				newFaces[i] = models.Face{
					// PersonID is nil for untagged faces
					ImagePath: cleanPath,
					X1:        det.X,
					Y1:        det.Y,
					X2:        det.X + det.W,
					Y2:        det.Y + det.H,
					CreatedAt: faceCreatedAt,
					UpdatedAt: faceCreatedAt,
				}
			}
			if err := tx.Create(&newFaces).Error; err != nil {
				return fmt.Errorf("failed to add new detected faces for %s: %w", cleanPath, err)
			}
		}

		imageUpdates := map[string]interface{}{
			"last_modified":          modTime,
			"detection_status":       status,
			"detection_processed_at": &now,
			"detection_error":        errStr,
		}
		if err := tx.Model(&models.Image{}).Where("original_path = ?", cleanPath).Updates(imageUpdates).Error; err != nil {
			return fmt.Errorf("failed to update image detection result for %s: %w", cleanPath, err)
		}

		return nil
	})
}

// Delete removes an image record by its original path
func (r *ImageRepository) Delete(originalPath string) error {
	cleanPath := filepath.ToSlash(originalPath)
	result := r.DB.Where("original_path = ?", cleanPath).Delete(&models.Image{})
	if result.Error != nil {
		return fmt.Errorf("failed to delete image record for %s: %w", cleanPath, result.Error)
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// GetImagesRequiringProcessing retrieves images that have one or more tasks in 'pending' status
func (r *ImageRepository) GetImagesRequiringProcessing() ([]models.Image, error) {
	var images []models.Image
	err := r.DB.Where("metadata_status = ? OR thumbnail_status = ? OR detection_status = ?",
		database.StatusPending, database.StatusPending, database.StatusPending).
		Find(&images).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get images requiring processing: %w", err)
	}
	return images, nil
}

// GetImagesByPaths retrieves multiple image records by their original paths
func (r *ImageRepository) GetImagesByPaths(originalPaths []string) ([]models.Image, error) {
	if len(originalPaths) == 0 {
		return []models.Image{}, nil
	}

	cleanPaths := make([]string, len(originalPaths))
	for i, p := range originalPaths {
		cleanPaths[i] = filepath.ToSlash(p)
	}

	var images []models.Image
	err := r.DB.Where("original_path IN ?", cleanPaths).Find(&images).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get images by paths: %w", err)
	}
	return images, nil
}
