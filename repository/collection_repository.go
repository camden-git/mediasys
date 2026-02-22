package repository

import (
	"fmt"
	"time"

	"github.com/camden-git/mediasysbackend/models"
	"gorm.io/gorm"
)

type GormCollectionRepository struct {
	db *gorm.DB
}

func NewCollectionRepository(db *gorm.DB) *GormCollectionRepository {
	return &GormCollectionRepository{db: db}
}

func (r *GormCollectionRepository) Create(collection *models.Collection) error {
	now := time.Now().Unix()
	collection.CreatedAt = now
	collection.UpdatedAt = now
	return r.db.Create(collection).Error
}

func (r *GormCollectionRepository) GetByID(id uint) (*models.Collection, error) {
	var c models.Collection
	err := r.db.Preload("Filters").First(&c, id).Error
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *GormCollectionRepository) GetBySlug(slug string) (*models.Collection, error) {
	var c models.Collection
	err := r.db.Preload("Filters").Where("slug = ?", slug).First(&c).Error
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *GormCollectionRepository) ListPublic() ([]models.Collection, error) {
	var collections []models.Collection
	err := r.db.Preload("Filters").Where("is_public = ?", true).Order("name").Find(&collections).Error
	return collections, err
}

func (r *GormCollectionRepository) ListAll() ([]models.Collection, error) {
	var collections []models.Collection
	err := r.db.Preload("Filters").Order("name").Find(&collections).Error
	return collections, err
}

func (r *GormCollectionRepository) Update(collectionID uint, name, slug string, description *string, isPublic bool) error {
	updates := map[string]interface{}{
		"name":        name,
		"slug":        slug,
		"description": description,
		"is_public":   isPublic,
		"updated_at":  time.Now().Unix(),
	}
	return r.db.Model(&models.Collection{}).Where("id = ?", collectionID).Updates(updates).Error
}

func (r *GormCollectionRepository) Delete(id uint) error {
	return r.db.Delete(&models.Collection{}, id).Error
}

func (r *GormCollectionRepository) SetBannerPath(collectionID uint, bannerPath *string) error {
	return r.db.Model(&models.Collection{}).Where("id = ?", collectionID).Updates(map[string]interface{}{
		"banner_image_path": bannerPath,
		"updated_at":        time.Now().Unix(),
	}).Error
}

// SetFilters replaces all tag filters for a collection in a transaction.
func (r *GormCollectionRepository) SetFilters(collectionID uint, filters []models.CollectionTagFilter) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("collection_id = ?", collectionID).Delete(&models.CollectionTagFilter{}).Error; err != nil {
			return err
		}
		if len(filters) == 0 {
			return nil
		}
		for i := range filters {
			filters[i].ID = 0
			filters[i].CollectionID = collectionID
		}
		return tx.Create(&filters).Error
	})
}

// GetImagePathsMatchingFilters returns paginated image paths matching the collection's tag filters.
// Logic: AND across distinct tag_key values, OR within same tag_key.
func (r *GormCollectionRepository) GetImagePathsMatchingFilters(collectionID uint, offset, limit int) ([]string, int, error) {
	// Load filters
	var filters []models.CollectionTagFilter
	if err := r.db.Where("collection_id = ?", collectionID).Find(&filters).Error; err != nil {
		return nil, 0, err
	}
	if len(filters) == 0 {
		return []string{}, 0, nil
	}

	// Count distinct keys
	keySet := make(map[string]struct{})
	for _, f := range filters {
		keySet[f.TagKey] = struct{}{}
	}
	distinctKeyCount := len(keySet)

	// Build the filter clause:
	// SELECT image_path FROM image_tags
	// WHERE (tag_key='k1' AND tag_value IN ('v1','v2')) OR (tag_key='k2' AND tag_value IN ('v3'))
	// GROUP BY image_path
	// HAVING COUNT(DISTINCT tag_key) = N
	type kv struct {
		key    string
		values []string
	}
	kvMap := make(map[string][]string)
	for _, f := range filters {
		kvMap[f.TagKey] = append(kvMap[f.TagKey], f.TagValue)
	}

	whereArgs := []interface{}{}
	whereParts := ""
	first := true
	for k, vals := range kvMap {
		if !first {
			whereParts += " OR "
		}
		first = false
		placeholders := ""
		for i, v := range vals {
			if i > 0 {
				placeholders += ","
			}
			placeholders += "?"
			whereArgs = append(whereArgs, v)
		}
		whereParts += fmt.Sprintf("(tag_key = ? AND tag_value IN (%s))", placeholders)
		whereArgs = append([]interface{}{k}, whereArgs...)
		// reorder: k needs to go before its values in whereArgs
		// rebuild correctly
	}

	// Rebuild args in correct order: for each key, push key then its values
	whereArgs = []interface{}{}
	whereParts = ""
	first = true
	for k, vals := range kvMap {
		if !first {
			whereParts += " OR "
		}
		first = false
		placeholders := ""
		for i := range vals {
			if i > 0 {
				placeholders += ","
			}
			placeholders += "?"
		}
		whereParts += fmt.Sprintf("(tag_key = ? AND tag_value IN (%s))", placeholders)
		whereArgs = append(whereArgs, k)
		for _, v := range vals {
			whereArgs = append(whereArgs, v)
		}
	}

	baseSQL := fmt.Sprintf(`SELECT image_path FROM image_tags WHERE %s GROUP BY image_path HAVING COUNT(DISTINCT tag_key) = ?`, whereParts)
	countArgs := append(whereArgs, distinctKeyCount)

	// Count total
	var countRows []struct{ ImagePath string }
	if err := r.db.Raw(baseSQL, countArgs...).Scan(&countRows).Error; err != nil {
		return nil, 0, err
	}
	total := len(countRows)

	// Fetch paginated paths with ORDER BY for stable results
	paginatedSQL := fmt.Sprintf(`SELECT image_path FROM image_tags WHERE %s GROUP BY image_path HAVING COUNT(DISTINCT tag_key) = ? ORDER BY image_path LIMIT ? OFFSET ?`, whereParts)
	paginatedArgs := append(countArgs, limit, offset)

	var rows []struct{ ImagePath string }
	if err := r.db.Raw(paginatedSQL, paginatedArgs...).Scan(&rows).Error; err != nil {
		return nil, 0, err
	}

	paths := make([]string, 0, len(rows))
	for _, row := range rows {
		paths = append(paths, row.ImagePath)
	}
	return paths, total, nil
}
