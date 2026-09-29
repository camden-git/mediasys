package repository

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
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

// ListAllAdmin retrieves all albums (including hidden ones) for admin view, ordered by name
func (r *AlbumRepository) ListAllAdmin() ([]models.Album, error) {
	var albums []models.Album

	err := r.DB.Order("name ASC").Find(&albums).Error
	if err != nil {
		return nil, fmt.Errorf("failed to list albums for admin: %w", err)
	}
	return albums, nil
}

// GetByID retrieves an album by its ID
func (r *AlbumRepository) GetByID(id uint) (*models.Album, error) {
	var album models.Album
	err := r.DB.First(&album, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
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
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
		return nil, fmt.Errorf("failed to get album by slug %s: %w", slug, err)
	}
	return &album, nil
}

// FolderPathConflicts reports whether another album's folder equals, contains
// or is nested inside the given folder. Image paths are "<folder>/<file>", so
// overlapping folders would let two albums claim the same image path.
func (r *AlbumRepository) FolderPathConflicts(folder string) (bool, error) {
	folder = strings.Trim(filepath.ToSlash(folder), "/")
	var count int64
	err := r.DB.Model(&models.Album{}).
		Where("folder_path = ? OR starts_with(folder_path, ?) OR starts_with(?, folder_path || '/')", folder, folder+"/", folder).
		Count(&count).Error
	if err != nil {
		return false, fmt.Errorf("failed to check folder path conflicts for %s: %w", folder, err)
	}
	return count > 0, nil
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

// InvalidateZip discards a finished (or failed) archive because the album's
// images changed, and returns the old archive's object key so the caller can
// delete it. Archives that are still pending or processing are left alone: they
// read the album's images when they run.
func (r *AlbumRepository) InvalidateZip(albumID uint) (*string, error) {
	var oldPath *string
	err := r.DB.Transaction(func(tx *gorm.DB) error {
		var album models.Album
		if err := tx.Select("id", "zip_status", "zip_path").First(&album, albumID).Error; err != nil {
			return err
		}
		if album.ZipStatus != database.StatusDone && album.ZipStatus != database.StatusError {
			return nil
		}
		oldPath = album.ZipPath
		return tx.Model(&models.Album{}).Where("id = ?", albumID).Updates(map[string]interface{}{
			"zip_status": database.StatusNotRequired,
			"zip_path":   gorm.Expr("NULL"),
			"zip_size":   gorm.Expr("NULL"),
			"zip_error":  gorm.Expr("NULL"),
			"updated_at": time.Now().Unix(),
		}).Error
	})
	if err != nil {
		return nil, fmt.Errorf("failed to invalidate zip for album ID %d: %w", albumID, err)
	}
	return oldPath, nil
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

// GetBanners returns all banners for an album ordered by sort_order.
func (r *AlbumRepository) GetBanners(albumID uint) ([]models.AlbumBanner, error) {
	var banners []models.AlbumBanner
	err := r.DB.Where("album_id = ?", albumID).Order("sort_order ASC, id ASC").Find(&banners).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get banners for album ID %d: %w", albumID, err)
	}
	return banners, nil
}

// AddBanner inserts a new banner record for an album.
func (r *AlbumRepository) AddBanner(banner *models.AlbumBanner) error {
	banner.CreatedAt = time.Now().Unix()
	return r.DB.Create(banner).Error
}

// DeleteBanner removes a banner by ID, scoped to albumID for safety.
func (r *AlbumRepository) DeleteBanner(bannerID uint, albumID uint) error {
	result := r.DB.Where("id = ? AND album_id = ?", bannerID, albumID).Delete(&models.AlbumBanner{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// ReorderBanners updates sort_order for each banner in the given order, scoped to albumID.
func (r *AlbumRepository) ReorderBanners(albumID uint, orderedIDs []uint) error {
	return r.DB.Transaction(func(tx *gorm.DB) error {
		for i, id := range orderedIDs {
			if err := tx.Model(&models.AlbumBanner{}).
				Where("id = ? AND album_id = ?", id, albumID).
				Update("sort_order", i).Error; err != nil {
				return err
			}
		}
		return nil
	})
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

// DeleteCascade removes an album and its related banners, default tags, and
// user/role album permissions in a single transaction, then hard-deletes the
// album itself.
func (r *AlbumRepository) DeleteCascade(albumID uint) error {
	return r.DB.Transaction(func(tx *gorm.DB) error {
		for _, m := range []any{&models.AlbumBanner{}, &models.AlbumDefaultTag{}, &models.UserAlbumPermission{}, &models.RoleAlbumPermission{}} {
			if err := tx.Where("album_id = ?", albumID).Delete(m).Error; err != nil {
				return err
			}
		}
		return tx.Unscoped().Delete(&models.Album{}, albumID).Error
	})
}

// ListPendingZips returns albums whose archive was requested but never finished
// (e.g. interrupted by a restart).
func (r *AlbumRepository) ListPendingZips() ([]models.Album, error) {
	var albums []models.Album
	err := r.DB.Where("zip_status IN ?", []string{database.StatusPending, database.StatusProcessing}).Find(&albums).Error
	return albums, err
}
