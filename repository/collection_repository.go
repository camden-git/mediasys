package repository

import (
	"fmt"
	"path"
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

func (r *GormCollectionRepository) Update(collectionID uint, name, slug string, description *string, isPublic bool, filterMatch string) error {
	updates := map[string]interface{}{
		"name":         name,
		"slug":         slug,
		"description":  description,
		"is_public":    isPublic,
		"filter_match": filterMatch,
		"updated_at":   time.Now().Unix(),
	}
	return r.db.Model(&models.Collection{}).Where("id = ?", collectionID).Updates(updates).Error
}

func (r *GormCollectionRepository) Delete(id uint) error {
	return r.db.Delete(&models.Collection{}, id).Error
}

// GetBanners returns all banners for a collection ordered by sort_order.
func (r *GormCollectionRepository) GetBanners(collectionID uint) ([]models.CollectionBanner, error) {
	var banners []models.CollectionBanner
	err := r.db.Where("collection_id = ?", collectionID).Order("sort_order ASC, id ASC").Find(&banners).Error
	return banners, err
}

// AddBanner inserts a new banner for a collection.
func (r *GormCollectionRepository) AddBanner(banner *models.CollectionBanner) error {
	banner.CreatedAt = time.Now().Unix()
	return r.db.Create(banner).Error
}

// DeleteCollectionBanner removes a collection banner by ID, scoped to collectionID.
func (r *GormCollectionRepository) DeleteCollectionBanner(bannerID uint, collectionID uint) error {
	result := r.db.Where("id = ? AND collection_id = ?", bannerID, collectionID).Delete(&models.CollectionBanner{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// ReorderCollectionBanners updates sort_order for each banner in the given order.
func (r *GormCollectionRepository) ReorderCollectionBanners(collectionID uint, orderedIDs []uint) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		for i, id := range orderedIDs {
			if err := tx.Model(&models.CollectionBanner{}).
				Where("id = ? AND collection_id = ?", id, collectionID).
				Update("sort_order", i).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// SetInheritBanners sets the inherit_banners_from_albums flag on a collection.
func (r *GormCollectionRepository) SetInheritBanners(collectionID uint, inherit bool) error {
	return r.db.Model(&models.Collection{}).Where("id = ?", collectionID).Updates(map[string]interface{}{
		"inherit_banners_from_albums": inherit,
		"updated_at":                  time.Now().Unix(),
	}).Error
}

// GetInheritedBannerPaths returns banner image paths from all albums whose folder_path
// matches the directories of images in this collection.
func (r *GormCollectionRepository) GetInheritedBannerPaths(collectionID uint) ([]string, error) {
	paths, _, err := r.GetImagePathsMatchingFilters(collectionID, 0, 500)
	if err != nil {
		return nil, err
	}

	folderSet := make(map[string]struct{})
	for _, p := range paths {
		folderSet[path.Dir(p)] = struct{}{}
	}
	if len(folderSet) == 0 {
		return []string{}, nil
	}

	folderPaths := make([]string, 0, len(folderSet))
	for f := range folderSet {
		folderPaths = append(folderPaths, f)
	}

	var result []string
	err = r.db.Table("album_banners").
		Joins("JOIN albums ON album_banners.album_id = albums.id").
		Where("albums.folder_path IN ? AND albums.deleted_at IS NULL", folderPaths).
		Order("albums.id, album_banners.sort_order").
		Pluck("album_banners.image_path", &result).Error
	if err != nil {
		return nil, err
	}
	return result, nil
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
// Positive filters (Negate=false): image must have the tag.
// Negative filters (Negate=true): image must NOT have the tag.
// FilterMatch="all": image must match every positive inclusion group (AND across keys, OR within key).
// FilterMatch="any": image must match at least one positive inclusion group (OR across groups).
// Negative filters always apply regardless of filter_match mode.
func (r *GormCollectionRepository) GetImagePathsMatchingFilters(collectionID uint, offset, limit int) ([]string, int, error) {
	// Load filters
	var filters []models.CollectionTagFilter
	if err := r.db.Where("collection_id = ?", collectionID).Find(&filters).Error; err != nil {
		return nil, 0, err
	}
	if len(filters) == 0 {
		return []string{}, 0, nil
	}

	// Load filter_match from collection
	var coll struct {
		FilterMatch string
	}
	if err := r.db.Model(&models.Collection{}).Select("filter_match").Where("id = ?", collectionID).Scan(&coll).Error; err != nil {
		return nil, 0, err
	}
	filterMatch := coll.FilterMatch
	if filterMatch != "any" {
		filterMatch = "all"
	}

	// Split into positive (inclusion) and negative (exclusion) filters
	positiveByKey := make(map[string][]string) // key → values (OR within key)
	type negFilter struct{ key, value string }
	var negatives []negFilter

	for _, f := range filters {
		if f.Negate {
			negatives = append(negatives, negFilter{f.TagKey, f.TagValue})
		} else {
			positiveByKey[f.TagKey] = append(positiveByKey[f.TagKey], f.TagValue)
		}
	}

	// Require at least one positive filter
	if len(positiveByKey) == 0 {
		return []string{}, 0, nil
	}

	// Build positive WHERE clause: (tag_key=? AND tag_value IN (?,?)) OR ...
	var posArgs []interface{}
	posWhere := ""
	first := true
	for k, vals := range positiveByKey {
		if !first {
			posWhere += " OR "
		}
		first = false
		placeholders := ""
		for i := range vals {
			if i > 0 {
				placeholders += ","
			}
			placeholders += "?"
		}
		posWhere += fmt.Sprintf("(tag_key = ? AND tag_value IN (%s))", placeholders)
		posArgs = append(posArgs, k)
		for _, v := range vals {
			posArgs = append(posArgs, v)
		}
	}

	// Build NOT IN sub-queries for negative filters
	notInSQL := ""
	var notInArgs []interface{}
	for _, neg := range negatives {
		notInSQL += " AND image_path NOT IN (SELECT image_path FROM image_tags WHERE tag_key = ? AND tag_value = ?)"
		notInArgs = append(notInArgs, neg.key, neg.value)
	}

	var baseSQL string
	var baseArgs []interface{}

	if filterMatch == "any" {
		baseSQL = fmt.Sprintf(
			`SELECT DISTINCT image_path FROM image_tags WHERE (%s)%s ORDER BY image_path`,
			posWhere, notInSQL,
		)
		baseArgs = append(posArgs, notInArgs...)
	} else {
		// "all": must match every positive key group
		distinctKeyCount := len(positiveByKey)
		baseSQL = fmt.Sprintf(
			`SELECT image_path FROM image_tags WHERE (%s)%s GROUP BY image_path HAVING COUNT(DISTINCT tag_key) = ? ORDER BY image_path`,
			posWhere, notInSQL,
		)
		baseArgs = append(posArgs, notInArgs...)
		baseArgs = append(baseArgs, distinctKeyCount)
	}

	// Count total using a SQL COUNT to avoid fetching all rows
	countSQL := fmt.Sprintf("SELECT COUNT(*) FROM (%s)", baseSQL)
	var total int
	if err := r.db.Raw(countSQL, baseArgs...).Scan(&total).Error; err != nil {
		return nil, 0, err
	}

	// Fetch paginated paths
	paginatedSQL := baseSQL + " LIMIT ? OFFSET ?"
	paginatedArgs := append(baseArgs, limit, offset)

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
