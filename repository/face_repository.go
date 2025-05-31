package repository

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/camden-git/mediasysbackend/models"
	"gorm.io/gorm"
)

// FaceRepository handles database operations for Face entities
type FaceRepository struct {
	DB *gorm.DB
}

// NewFaceRepository creates a new instance of FaceRepository
func NewFaceRepository(db *gorm.DB) *FaceRepository {
	return &FaceRepository{DB: db}
}

// Create creates a new face record in the database
func (r *FaceRepository) Create(face *models.Face) error {
	now := time.Now().Unix()
	if face.CreatedAt == 0 {
		face.CreatedAt = now
	}
	face.UpdatedAt = now
	face.ImagePath = filepath.ToSlash(face.ImagePath)

	err := r.DB.Create(face).Error
	if err != nil {
		return fmt.Errorf("failed to create face for image %s: %w", face.ImagePath, err)
	}
	return nil
}

// GetByID retrieves a face by its ID, preloading the associated Person
func (r *FaceRepository) GetByID(id uint) (*models.Face, error) {
	var face models.Face
	// preload Person to get PersonName if PersonID is not null
	err := r.DB.Preload("Person").First(&face, id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, err
		}
		return nil, fmt.Errorf("failed to get face by ID %d: %w", id, err)
	}
	return &face, nil
}

// ListByImagePath retrieves all faces for a given image path, preloading associated Person
func (r *FaceRepository) ListByImagePath(imagePath string) ([]models.Face, error) {
	cleanPath := filepath.ToSlash(imagePath)
	var faces []models.Face
	err := r.DB.Preload("Person").Where("image_path = ?", cleanPath).Order("id ASC").Find(&faces).Error
	if err != nil {
		return nil, fmt.Errorf("failed to list faces for image %s: %w", cleanPath, err)
	}
	return faces, nil
}

// Update updates an existing face's details (coordinates, PersonID)
// Pass a pointer to uint for PersonID to explicitly set it to NULL if needed.
// For coordinates, pass pointers to int; if a pointer is nil, that field won't be updated.
func (r *FaceRepository) Update(faceID uint, personID *uint, x1, y1, x2, y2 *int) error {
	updates := make(map[string]interface{})
	hasUpdates := false

	if personID != nil {
		if *personID == 0 {
			updates["person_id"] = gorm.Expr("NULL")
		} else {
			updates["person_id"] = *personID
		}
		hasUpdates = true
	}

	if x1 != nil {
		updates["x1"] = *x1
		hasUpdates = true
	}
	if y1 != nil {
		updates["y1"] = *y1
		hasUpdates = true
	}
	if x2 != nil {
		updates["x2"] = *x2
		hasUpdates = true
	}
	if y2 != nil {
		updates["y2"] = *y2
		hasUpdates = true
	}

	if !hasUpdates {
		return nil
	}

	updates["updated_at"] = time.Now().Unix()

	result := r.DB.Model(&models.Face{}).Where("id = ?", faceID).Updates(updates)
	if result.Error != nil {
		return fmt.Errorf("failed to update face ID %d: %w", faceID, result.Error)
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// Delete removes a face by its ID
func (r *FaceRepository) Delete(id uint) error {
	result := r.DB.Delete(&models.Face{}, id)
	if result.Error != nil {
		return fmt.Errorf("failed to delete face ID %d: %w", id, result.Error)
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// DeleteUntaggedByImagePath deletes all faces for a given image path that do not have a PersonID
// Returns the number of faces deleted
func (r *FaceRepository) DeleteUntaggedByImagePath(imagePath string) (int64, error) {
	cleanPath := filepath.ToSlash(imagePath)
	result := r.DB.Where("image_path = ? AND person_id IS NULL", cleanPath).Delete(&models.Face{})
	if result.Error != nil {
		return 0, fmt.Errorf("failed to delete untagged faces for image %s: %w", cleanPath, result.Error)
	}
	return result.RowsAffected, nil
}

// TagFace assigns a PersonID to an existing face
func (r *FaceRepository) TagFace(faceID uint, personID uint) error {
	updates := map[string]interface{}{
		"person_id":  personID,
		"updated_at": time.Now().Unix(),
	}
	result := r.DB.Model(&models.Face{}).Where("id = ?", faceID).Updates(updates)
	if result.Error != nil {
		return fmt.Errorf("failed to tag face ID %d with person ID %d: %w", faceID, personID, result.Error)
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// UntagFace sets the PersonID of an existing face to NULL.
func (r *FaceRepository) UntagFace(faceID uint) error {
	updates := map[string]interface{}{
		"person_id":  gorm.Expr("NULL"),
		"updated_at": time.Now().Unix(),
	}
	result := r.DB.Model(&models.Face{}).Where("id = ?", faceID).Updates(updates)
	if result.Error != nil {
		return fmt.Errorf("failed to untag face ID %d: %w", faceID, result.Error)
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
