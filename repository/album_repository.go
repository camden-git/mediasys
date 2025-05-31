package repository

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/camden-git/mediasysbackend/database"
	"github.com/camden-git/mediasysbackend/models"
	"gorm.io/gorm"
)

// AlbumRepository handles database operations for Album entities
type AlbumRepository struct {
	DB *gorm.DB
}

// NewAlbumRepository creates a new instance of AlbumRepository
func NewAlbumRepository(db *gorm.DB) *AlbumRepository {
	return &AlbumRepository{DB: db}
}

// Create creates a new album record in the database
func (r *AlbumRepository) Create(album *models.Album) error {
	now := time.Now().Unix()
	if album.CreatedAt == 0 {
		album.CreatedAt = now
	}
	if album.UpdatedAt == 0 {
		album.UpdatedAt = now
	}
	album.FolderPath = filepath.ToSlash(album.FolderPath)
	if album.SortOrder == "" {
		// who cares i guess???
	}
	if album.ZipStatus == "" {
		album.ZipStatus = database.StatusNotRequired
	}

	err := r.DB.Create(album).Error
	if err != nil {
		return fmt.Errorf("failed to create album %s: %w", album.Name, err)
	}
	return nil
}

// ListAll retrieves all non-hidden albums, ordered by name
func (r *AlbumRepository) ListAll() ([]models.Album, error) {
	var albums []models.Album

	// Filter out hidden albums
	err := r.DB.Where("is_hidden = ?", false).Order("name ASC").Find(&albums).Error
	if err != nil {
		return nil, fmt.Errorf("failed to list albums: %w", err)
	}
	return albums, nil
}

// GetByID retrieves an album by its ID
func (r *AlbumRepository) GetByID(id uint) (*models.Album, error) {
	var album models.Album
	err := r.DB.First(&album, id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, err
		}
		return nil, fmt.Errorf("failed to get album by ID %d: %w", id, err)
	}
	return &album, nil
}

// GetBySlug retrieves an album by its slug
func (r *AlbumRepository) GetBySlug(slug string) (*models.Album, error) {
	var album models.Album
	err := r.DB.Where("slug = ?", slug).First(&album).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, err
		}
		return nil, fmt.Errorf("failed to get album by slug %s: %w", slug, err)
	}
	return &album, nil
}

// Update updates an existing album's name, description, hidden status, and location
// other fields are updated by specific methods
func (r *AlbumRepository) Update(albumID uint, name string, description *string, isHidden *bool, location *string) error {
	now := time.Now().Unix()
	updates := map[string]interface{}{
		"updated_at": now,
	}
	if name != "" {
		updates["name"] = name
	}
	if description != nil {
		updates["description"] = *description
	}
	if isHidden != nil {
		updates["is_hidden"] = *isHidden
	}
	if location != nil {
		if *location == "" { // allow clearing the location
			updates["location"] = gorm.Expr("NULL")
		} else {
			updates["location"] = *location
		}
	}

	// if only updated_at is present, no actual fields were changed
	if len(updates) == 1 {
		return nil
	}

	result := r.DB.Model(&models.Album{}).Where("id = ?", albumID).Updates(updates)
	if result.Error != nil {
		return fmt.Errorf("failed to update album ID %d: %w", albumID, result.Error)
	}
	if result.RowsAffected == 0 {
		var count int64
		r.DB.Model(&models.Album{}).Where("id = ?", albumID).Count(&count)
		if count == 0 {
			return gorm.ErrRecordNotFound
		}
	}
	return nil
}

// RequestZip updates album status to indicate a zip generation is pending
func (r *AlbumRepository) RequestZip(albumID uint) error {
	now := time.Now().Unix()
	updates := map[string]interface{}{
		"zip_status":            database.StatusPending,
		"zip_last_requested_at": now,
		"zip_error":             gorm.Expr("NULL"),
		"updated_at":            now,
	}
	result := r.DB.Model(&models.Album{}).Where("id = ?", albumID).Updates(updates)
	if result.Error != nil {
		return fmt.Errorf("failed to request zip for album ID %d: %w", albumID, result.Error)
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// MarkZipProcessing updates album status to indicate zip generation is in progress
func (r *AlbumRepository) MarkZipProcessing(albumID uint) error {
	now := time.Now().Unix()
	result := r.DB.Model(&models.Album{}).Where("id = ?", albumID).Updates(map[string]interface{}{
		"zip_status": database.StatusProcessing,
		"updated_at": now,
	})
	if result.Error != nil {
		return fmt.Errorf("failed to mark zip processing for album ID %d: %w", albumID, result.Error)
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// SetZipResult updates album with the result of a zip generation task
func (r *AlbumRepository) SetZipResult(albumID uint, zipPath *string, zipSize *int64, taskErr error) error {
	now := time.Now().Unix()
	status := database.StatusDone
	var errStr *string

	if taskErr != nil {
		status = database.StatusError
		s := taskErr.Error()
		errStr = &s
	}

	updates := map[string]interface{}{
		"zip_status": status,
		"zip_error":  errStr,
		"updated_at": now,
	}

	if status == database.StatusDone {
		updates["zip_path"] = zipPath
		updates["zip_size"] = zipSize
		updates["zip_last_generated_at"] = now
	}

	result := r.DB.Model(&models.Album{}).Where("id = ?", albumID).Updates(updates)
	if result.Error != nil {
		return fmt.Errorf("failed to set zip result for album ID %d: %w", albumID, result.Error)
	}

	return nil
}

// UpdateBannerPath updates the banner image path for an album
func (r *AlbumRepository) UpdateBannerPath(albumID uint, bannerPath *string) error {
	now := time.Now().Unix()
	result := r.DB.Model(&models.Album{}).Where("id = ?", albumID).Updates(map[string]interface{}{
		"banner_image_path": bannerPath,
		"updated_at":        now,
	})
	if result.Error != nil {
		return fmt.Errorf("failed to update banner path for album ID %d: %w", albumID, result.Error)
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// UpdateSortOrder updates the sort order for an album
// assumes sortOrder string is validated externally (e.g., by a service layer or IsValidSortOrder)
func (r *AlbumRepository) UpdateSortOrder(albumID uint, sortOrder string) error {
	// TODO: add validation for sortOrder if not handled before this call
	// if !database.IsValidSortOrder(sortOrder) {
	// 	return fmt.Errorf("invalid sort order: %s", sortOrder)
	// }
	now := time.Now().Unix()
	result := r.DB.Model(&models.Album{}).Where("id = ?", albumID).Updates(map[string]interface{}{
		"sort_order": sortOrder,
		"updated_at": now,
	})
	if result.Error != nil {
		return fmt.Errorf("failed to update sort order for album ID %d: %w", albumID, result.Error)
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// Delete removes an album by its ID
// this will perform a soft delete because models.Album has gorm.DeletedAt
func (r *AlbumRepository) Delete(id uint) error {
	result := r.DB.Delete(&models.Album{}, id)
	if result.Error != nil {
		return fmt.Errorf("failed to delete album ID %d: %w", id, result.Error)
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
