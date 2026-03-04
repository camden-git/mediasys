package database

import (
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/camden-git/mediasysbackend/models"
	"gorm.io/gorm"
)

// Migration represents a database migration
type Migration struct {
	ID         uint      `gorm:"primaryKey;autoIncrement"`
	Name       string    `gorm:"not null;unique"`
	ExecutedAt time.Time `gorm:"not null"`
}

// TableName explicitly sets the table name for GORM.
func (Migration) TableName() string {
	return "migrations"
}

// RunMigrations runs all pending database migrations
func RunMigrations(db *gorm.DB) error {
	log.Println("Starting database migrations...")

	// Create migrations table if it doesn't exist
	if err := db.AutoMigrate(&Migration{}); err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}

	// Define all migrations
	migrations := []func(*gorm.DB) error{
		migrateSortOrderValues,
		migrateAddTagsAndCollections,
		migrateAddPersonKeyPhoto,
		migrateCollectionFilterExpansion,
		migrateAddImageTagsImagePathIndex,
		migrateAddMultiBanners,
	}

	migrationNames := []string{
		"fix_sort_order_values",
		"add_tags_and_collections",
		"add_person_key_photo",
		"collection_filter_expansion",
		"add_image_tags_image_path_index",
		"add_multi_banners",
	}

	// Run pending migrations
	for i, migration := range migrations {
		migrationName := migrationNames[i]

		// Check if migration has already been executed
		var existingMigration Migration
		result := db.Where("name = ?", migrationName).First(&existingMigration)
		if result.Error == nil {
			log.Printf("Migration '%s' already executed, skipping", migrationName)
			continue
		}
		if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return fmt.Errorf("failed to check migration status for '%s': %w", migrationName, result.Error)
		}

		// Execute migration
		log.Printf("Executing migration: %s", migrationName)
		if err := migration(db); err != nil {
			return fmt.Errorf("migration '%s' failed: %w", migrationName, err)
		}

		// Record migration as executed
		migrationRecord := Migration{
			Name:       migrationName,
			ExecutedAt: time.Now(),
		}
		if err := db.Create(&migrationRecord).Error; err != nil {
			return fmt.Errorf("failed to record migration '%s': %w", migrationName, err)
		}

		log.Printf("Migration '%s' completed successfully", migrationName)
	}

	log.Println("All database migrations completed successfully")
	return nil
}

// migrateAddTagsAndCollections creates the image_tags, album_default_tags, collections,
// and collection_tag_filters tables and their indexes.
func migrateAddTagsAndCollections(db *gorm.DB) error {
	log.Println("Running tags and collections schema migration...")
	if err := db.AutoMigrate(
		&models.ImageTag{},
		&models.AlbumDefaultTag{},
		&models.Collection{},
		&models.CollectionTagFilter{},
	); err != nil {
		return fmt.Errorf("failed to auto-migrate tag/collection tables: %w", err)
	}

	// Unique index: (image_path, tag_key, tag_value, source)
	if err := db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS idx_image_tags_unique
		ON image_tags (image_path, tag_key, tag_value, source)`).Error; err != nil {
		return fmt.Errorf("failed to create unique index on image_tags: %w", err)
	}
	// Lookup index: (tag_key, tag_value)
	if err := db.Exec(`CREATE INDEX IF NOT EXISTS idx_image_tags_key_value
		ON image_tags (tag_key, tag_value)`).Error; err != nil {
		return fmt.Errorf("failed to create key/value index on image_tags: %w", err)
	}

	// Unique index on album_default_tags
	if err := db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS idx_album_default_tags_unique
		ON album_default_tags (album_id, tag_key, tag_value)`).Error; err != nil {
		return fmt.Errorf("failed to create unique index on album_default_tags: %w", err)
	}

	// Unique index on collection_tag_filters
	if err := db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS idx_collection_tag_filters_unique
		ON collection_tag_filters (collection_id, tag_key, tag_value)`).Error; err != nil {
		return fmt.Errorf("failed to create unique index on collection_tag_filters: %w", err)
	}

	log.Println("Tags and collections migration completed.")
	return nil
}

// migrateAddPersonKeyPhoto adds the key_photo_face_id column to the people table.
func migrateAddPersonKeyPhoto(db *gorm.DB) error {
	log.Println("Running add_person_key_photo migration...")
	if err := db.AutoMigrate(&models.Person{}); err != nil {
		return fmt.Errorf("failed to auto-migrate people table: %w", err)
	}
	log.Println("add_person_key_photo migration completed.")
	return nil
}

// migrateCollectionFilterExpansion adds FilterMatch to collections and Negate to collection_tag_filters.
func migrateCollectionFilterExpansion(db *gorm.DB) error {
	log.Println("Running collection filter expansion migration...")
	if err := db.AutoMigrate(&models.Collection{}, &models.CollectionTagFilter{}); err != nil {
		return fmt.Errorf("failed to auto-migrate: %w", err)
	}
	log.Println("Collection filter expansion migration completed.")
	return nil
}

// migrateAddImageTagsImagePathIndex adds a lookup index on image_tags(image_path).
func migrateAddImageTagsImagePathIndex(db *gorm.DB) error {
	log.Println("Running add_image_tags_image_path_index migration...")
	if err := db.Exec(`CREATE INDEX IF NOT EXISTS idx_image_tags_image_path ON image_tags (image_path)`).Error; err != nil {
		return fmt.Errorf("failed to create image_path index on image_tags: %w", err)
	}
	log.Println("add_image_tags_image_path_index migration completed.")
	return nil
}

// migrateAddMultiBanners creates album_banners and collection_banners tables,
// migrates existing single banner_image_path values, and adds inherit_banners_from_albums to collections.
func migrateAddMultiBanners(db *gorm.DB) error {
	log.Println("Running add_multi_banners migration...")

	// Create new tables
	if err := db.AutoMigrate(&models.AlbumBanner{}, &models.CollectionBanner{}); err != nil {
		return fmt.Errorf("failed to auto-migrate banner tables: %w", err)
	}

	// Migrate existing album banner_image_path values into album_banners (skip if column doesn't exist)
	var albumColCount int
	db.Raw(`SELECT COUNT(*) FROM pragma_table_info('albums') WHERE name = 'banner_image_path'`).Scan(&albumColCount)
	if albumColCount > 0 {
		if err := db.Exec(`
			INSERT OR IGNORE INTO album_banners (album_id, image_path, sort_order, created_at)
			SELECT id, banner_image_path, 0, strftime('%s', 'now')
			FROM albums
			WHERE banner_image_path IS NOT NULL AND banner_image_path != ''
		`).Error; err != nil {
			return fmt.Errorf("failed to migrate album banners: %w", err)
		}
	}

	// Migrate existing collection banner_image_path values into collection_banners (skip if column doesn't exist)
	var collColCount int
	db.Raw(`SELECT COUNT(*) FROM pragma_table_info('collections') WHERE name = 'banner_image_path'`).Scan(&collColCount)
	if collColCount > 0 {
		if err := db.Exec(`
			INSERT OR IGNORE INTO collection_banners (collection_id, image_path, sort_order, created_at)
			SELECT id, banner_image_path, 0, strftime('%s', 'now')
			FROM collections
			WHERE banner_image_path IS NOT NULL AND banner_image_path != ''
		`).Error; err != nil {
			return fmt.Errorf("failed to migrate collection banners: %w", err)
		}
	}

	// Add inherit_banners_from_albums to collections (AutoMigrate handles SQLite ALTER TABLE)
	if err := db.AutoMigrate(&models.Collection{}); err != nil {
		return fmt.Errorf("failed to add inherit_banners_from_albums column: %w", err)
	}

	log.Println("add_multi_banners migration completed.")
	return nil
}

// migrateSortOrderValues fixes the sort_order column values from 'name_asc' to 'filename_asc'
func migrateSortOrderValues(db *gorm.DB) error {
	log.Println("Running sort_order values migration...")

	// Update albums with old 'name_asc' sort order to 'filename_asc'
	result := db.Model(&models.Album{}).Where("sort_order = ?", "name_asc").Update("sort_order", "filename_asc")
	if result.Error != nil {
		return fmt.Errorf("failed to update albums with old sort order: %w", result.Error)
	}

	log.Printf("Updated %d albums from 'name_asc' to 'filename_asc'", result.RowsAffected)

	// Also check for any other invalid sort orders and set them to default
	validSortOrders := []string{"filename_asc", "filename_nat", "date_desc", "date_asc"}

	// This will update any albums that don't have a valid sort order
	result = db.Model(&models.Album{}).Where("sort_order NOT IN ?", validSortOrders).Update("sort_order", "filename_asc")
	if result.Error != nil {
		return fmt.Errorf("failed to fix invalid sort orders: %w", result.Error)
	}
	if result.RowsAffected > 0 {
		log.Printf("Fixed %d albums with invalid sort orders", result.RowsAffected)
	}

	return nil
}
