package repository

import (
	"fmt"
	"time"

	"github.com/camden-git/mediasysbackend/database"
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

func (r *GormCollectionRepository) Update(collectionID uint, name, slug string, description *string, isPublic bool, filterMatch string, sortOrder string) error {
	updates := map[string]interface{}{
		"name":         name,
		"slug":         slug,
		"description":  description,
		"is_public":    isPublic,
		"filter_match": filterMatch,
		"sort_order":   sortOrder,
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

// GetInheritedBannerPaths returns banner image paths from all albums that
// contain images in this collection.
func (r *GormCollectionRepository) GetInheritedBannerPaths(collectionID uint) ([]string, error) {
	subSQL, args, err := r.buildFilterSubquery(collectionID)
	if err != nil {
		return nil, err
	}
	if subSQL == "" {
		return []string{}, nil
	}

	whereSQL := fmt.Sprintf("original_path IN (%s)", subSQL)
	var result []string
	err = r.db.Table("album_banners").
		Joins("JOIN albums ON album_banners.album_id = albums.id").
		Where("albums.deleted_at IS NULL AND albums.id IN (?)",
			r.db.Model(&models.Image{}).Distinct("album_id").Where(whereSQL, args...)).
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

// buildFilterSubquery returns a SELECT statement (and its bind args) over
// image_tags that yields the image_path values matching the collection's tag
// filters. It returns an empty sql string when the collection has no filters
// (or no positive/inclusion filter), which callers must treat as "matches
// nothing" without running a query.
//
// Positive filters (Negate=false): image must have the tag.
// Negative filters (Negate=true): image must NOT have the tag.
// FilterMatch="all": image must match every positive inclusion group (AND across keys, OR within key).
// FilterMatch="any": image must match at least one positive inclusion group (OR across groups).
// Negative filters always apply regardless of filter_match mode.
func (r *GormCollectionRepository) buildFilterSubquery(collectionID uint) (string, []interface{}, error) {
	// Load filters
	var filters []models.CollectionTagFilter
	if err := r.db.Where("collection_id = ?", collectionID).Find(&filters).Error; err != nil {
		return "", nil, err
	}
	if len(filters) == 0 {
		return "", nil, nil
	}

	// Load filter_match from collection
	var coll struct {
		FilterMatch string
	}
	if err := r.db.Model(&models.Collection{}).Select("filter_match").Where("id = ?", collectionID).Scan(&coll).Error; err != nil {
		return "", nil, err
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
		return "", nil, nil
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
			`SELECT DISTINCT image_path FROM image_tags WHERE (%s)%s`,
			posWhere, notInSQL,
		)
		baseArgs = append(posArgs, notInArgs...)
	} else {
		// "all": must match every positive key group
		distinctKeyCount := len(positiveByKey)
		baseSQL = fmt.Sprintf(
			`SELECT image_path FROM image_tags WHERE (%s)%s GROUP BY image_path HAVING COUNT(DISTINCT tag_key) = ?`,
			posWhere, notInSQL,
		)
		baseArgs = append(posArgs, notInArgs...)
		baseArgs = append(baseArgs, distinctKeyCount)
	}

	return baseSQL, baseArgs, nil
}

// GetImagePathsMatchingFilters returns all image paths matching the collection's tag filters.
// Prefer ListImages for user-facing listings: it sorts and pages in SQL instead of
// materializing every matching path into Go and passing them back through IN (...).
func (r *GormCollectionRepository) GetImagePathsMatchingFilters(collectionID uint) ([]string, error) {
	subSQL, args, err := r.buildFilterSubquery(collectionID)
	if err != nil {
		return nil, err
	}
	if subSQL == "" {
		return []string{}, nil
	}

	var rows []struct{ ImagePath string }
	if err := r.db.Raw(subSQL, args...).Scan(&rows).Error; err != nil {
		return nil, err
	}

	paths := make([]string, 0, len(rows))
	for _, row := range rows {
		paths = append(paths, row.ImagePath)
	}
	return paths, nil
}

// ListImages returns a sorted, paginated page of images matching the collection's tag
// filters, plus the total number of matching images. The filter is applied as a
// subquery join (images.original_path IN (SELECT ... FROM image_tags ...)) so the
// matching path set never needs to round-trip through Go, and sorting/paging happen
// in SQL.
func (r *GormCollectionRepository) ListImages(collectionID uint, sortOrder string, offset, limit int) ([]models.Image, int, error) {
	subSQL, args, err := r.buildFilterSubquery(collectionID)
	if err != nil {
		return nil, 0, err
	}
	if subSQL == "" {
		return []models.Image{}, 0, nil
	}

	whereSQL := fmt.Sprintf("original_path IN (%s)", subSQL)
	db := r.db.Model(&models.Image{}).Where(whereSQL, args...)

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count collection %d images: %w", collectionID, err)
	}

	var images []models.Image
	if err := db.Order(database.SQLOrderClause(sortOrder)).Offset(offset).Limit(limit).Find(&images).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to list collection %d images: %w", collectionID, err)
	}
	return images, int(total), nil
}
