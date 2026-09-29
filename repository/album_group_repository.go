package repository

import (
	"errors"
	"fmt"
	"time"

	"github.com/camden-git/mediasysbackend/models"
	"gorm.io/gorm"
)

// AlbumGroupRepository handles database operations for AlbumGroup entities
type AlbumGroupRepository struct {
	DB *gorm.DB
}

// NewAlbumGroupRepository creates a new instance of AlbumGroupRepository
func NewAlbumGroupRepository(db *gorm.DB) *AlbumGroupRepository {
	return &AlbumGroupRepository{DB: db}
}

// Create creates a new album group record
func (r *AlbumGroupRepository) Create(group *models.AlbumGroup) error {
	now := time.Now().Unix()
	if group.CreatedAt == 0 {
		group.CreatedAt = now
	}
	if group.UpdatedAt == 0 {
		group.UpdatedAt = now
	}
	if err := r.DB.Create(group).Error; err != nil {
		return fmt.Errorf("failed to create album group %s: %w", group.Name, err)
	}
	return nil
}

// GetByID retrieves an album group by its ID
func (r *AlbumGroupRepository) GetByID(id uint) (*models.AlbumGroup, error) {
	var group models.AlbumGroup
	err := r.DB.Preload("Albums").First(&group, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
		return nil, fmt.Errorf("failed to get album group by ID %d: %w", id, err)
	}
	return &group, nil
}

// GetBySlug retrieves an album group by its slug with its non-hidden albums
func (r *AlbumGroupRepository) GetBySlug(slug string) (*models.AlbumGroup, error) {
	var group models.AlbumGroup
	err := r.DB.Where("slug = ?", slug).Preload("Albums", "is_hidden = ?", false).First(&group).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
		return nil, fmt.Errorf("failed to get album group by slug %s: %w", slug, err)
	}
	return &group, nil
}

// ListAll returns all non-hidden album groups with their non-hidden albums
func (r *AlbumGroupRepository) ListAll() ([]models.AlbumGroup, error) {
	var groups []models.AlbumGroup
	err := r.DB.Where("is_hidden = ?", false).
		Preload("Albums", "is_hidden = ?", false).
		Order("name ASC").Find(&groups).Error
	if err != nil {
		return nil, fmt.Errorf("failed to list album groups: %w", err)
	}
	return groups, nil
}

// ListAllAdmin returns all album groups (including hidden) with all their albums
func (r *AlbumGroupRepository) ListAllAdmin() ([]models.AlbumGroup, error) {
	var groups []models.AlbumGroup
	err := r.DB.Preload("Albums").Order("name ASC").Find(&groups).Error
	if err != nil {
		return nil, fmt.Errorf("failed to list album groups for admin: %w", err)
	}
	return groups, nil
}

// Update updates an album group's fields
func (r *AlbumGroupRepository) Update(groupID uint, name, slug string, description *string, isHidden bool) error {
	now := time.Now().Unix()
	result := r.DB.Model(&models.AlbumGroup{}).Where("id = ?", groupID).Updates(map[string]interface{}{
		"name":        name,
		"slug":        slug,
		"description": description,
		"is_hidden":   isHidden,
		"updated_at":  now,
	})
	if result.Error != nil {
		return fmt.Errorf("failed to update album group %d: %w", groupID, result.Error)
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// Delete soft-deletes an album group
func (r *AlbumGroupRepository) Delete(id uint) error {
	result := r.DB.Delete(&models.AlbumGroup{}, id)
	if result.Error != nil {
		return fmt.Errorf("failed to delete album group %d: %w", id, result.Error)
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// SetBannerPath updates the banner image path for an album group
func (r *AlbumGroupRepository) SetBannerPath(groupID uint, bannerPath *string) error {
	now := time.Now().Unix()
	result := r.DB.Model(&models.AlbumGroup{}).Where("id = ?", groupID).Updates(map[string]interface{}{
		"banner_image_path": bannerPath,
		"updated_at":        now,
	})
	if result.Error != nil {
		return fmt.Errorf("failed to set banner path for album group %d: %w", groupID, result.Error)
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// SetAlbumGroup assigns an album to a group (or clears it by passing nil)
func (r *AlbumGroupRepository) SetAlbumGroup(albumID uint, groupID *uint) error {
	now := time.Now().Unix()
	result := r.DB.Model(&models.Album{}).Where("id = ?", albumID).Updates(map[string]interface{}{
		"group_id":   groupID,
		"updated_at": now,
	})
	if result.Error != nil {
		return fmt.Errorf("failed to set group for album %d: %w", albumID, result.Error)
	}
	return nil
}
