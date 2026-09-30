package repository

import (
	"time"

	"github.com/camden-git/mediasysbackend/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type GormImageTagRepository struct {
	db *gorm.DB
}

func NewImageTagRepository(db *gorm.DB) *GormImageTagRepository {
	return &GormImageTagRepository{db: db}
}

func (r *GormImageTagRepository) GetTagsByImagePath(imagePath string) ([]models.ImageTag, error) {
	var tags []models.ImageTag
	err := r.db.Where("image_path = ?", imagePath).Order("tag_key, tag_value, source").Find(&tags).Error
	return tags, err
}

// SetXMPTags deletes all source="xmp" tags for the image and inserts the new set atomically.
func (r *GormImageTagRepository) SetXMPTags(imagePath string, tags []models.ImageTag) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("image_path = ? AND source = ?", imagePath, "xmp").Delete(&models.ImageTag{}).Error; err != nil {
			return err
		}
		if len(tags) == 0 {
			return nil
		}
		return tx.Create(&tags).Error
	})
}

func (r *GormImageTagRepository) AddManualTag(imagePath, key, value string) error {
	tag := models.ImageTag{
		ImagePath: imagePath,
		TagKey:    key,
		TagValue:  value,
		Source:    "manual",
		CreatedAt: time.Now(),
	}
	return r.db.Clauses(clause.OnConflict{DoNothing: true}).Create(&tag).Error
}

func (r *GormImageTagRepository) RemoveManualTag(imagePath, key, value string) error {
	return r.db.Where("image_path = ? AND tag_key = ? AND tag_value = ? AND source = ?",
		imagePath, key, value, "manual").Delete(&models.ImageTag{}).Error
}

// ApplyAlbumDefaultTags copies album_default_tags for an album to image_tags with source="album_default",
// skipping duplicates, in a single ON CONFLICT DO NOTHING insert.
func (r *GormImageTagRepository) ApplyAlbumDefaultTags(imagePath string, albumID uint) error {
	defaults, err := r.GetAlbumDefaultTags(albumID)
	if err != nil {
		return err
	}
	if len(defaults) == 0 {
		return nil
	}
	now := time.Now()
	tags := make([]models.ImageTag, len(defaults))
	for i, d := range defaults {
		tags[i] = models.ImageTag{
			ImagePath: imagePath,
			TagKey:    d.TagKey,
			TagValue:  d.TagValue,
			Source:    "album_default",
			CreatedAt: now,
		}
	}
	return r.db.Clauses(clause.OnConflict{DoNothing: true}).Create(&tags).Error
}

func (r *GormImageTagRepository) GetAlbumDefaultTags(albumID uint) ([]models.AlbumDefaultTag, error) {
	var tags []models.AlbumDefaultTag
	err := r.db.Where("album_id = ?", albumID).Order("tag_key, tag_value").Find(&tags).Error
	return tags, err
}

// SetAlbumDefaultTags replaces all default tags for an album in a transaction.
func (r *GormImageTagRepository) SetAlbumDefaultTags(albumID uint, tags []models.AlbumDefaultTag) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("album_id = ?", albumID).Delete(&models.AlbumDefaultTag{}).Error; err != nil {
			return err
		}
		if len(tags) == 0 {
			return nil
		}
		now := time.Now()
		for i := range tags {
			tags[i].AlbumID = albumID
			tags[i].ID = 0
			tags[i].CreatedAt = now
		}
		return tx.Create(&tags).Error
	})
}
