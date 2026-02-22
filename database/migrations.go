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
	}

	migrationNames := []string{
		"fix_sort_order_values",
		"add_tags_and_collections",
		"add_person_key_photo",
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
