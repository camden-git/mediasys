package database_test

import (
	"context"
	"database/sql"
	"fmt"
	"math/rand"
	"os"
	"testing"

	"github.com/camden-git/mediasysbackend/database"
)

// TestMigrationsUpgradeAutoMigrateDatabase simulates a database created by the
// pre-goose GORM AutoMigrate code (no goose version table) and checks that
// database.RunMigrations baselines it and brings it up to the current schema
// without losing data. Like TestMigrationsMatchGORMSchema it needs
// TEST_DATABASE_URL (an admin connection) and is skipped without it.
func TestMigrationsUpgradeAutoMigrateDatabase(t *testing.T) {
	adminURL := os.Getenv("TEST_DATABASE_URL")
	if adminURL == "" {
		skipOrFailMissingEnv(t, "TEST_DATABASE_URL not set; skipping upgrade test")
	}
	ctx := context.Background()

	adminDB, err := sql.Open("pgx", adminURL)
	if err != nil {
		t.Fatalf("failed to open admin connection: %v", err)
	}
	defer adminDB.Close()

	dbName := fmt.Sprintf("mediasys_upgrade_test_%d", rand.Intn(1_000_000))
	createDatabase(t, adminDB, dbName)
	defer dropDatabase(t, adminDB, dbName)

	db := legacyAutoMigrate(t, withDBName(t, adminURL, dbName))
	defer db.Close()

	// Make the database look like one created by older code: table-wide unique
	// constraints instead of partial indexes, and no composite listing index.
	for _, stmt := range []string{
		"DROP INDEX idx_albums_name",
		"ALTER TABLE albums ADD CONSTRAINT uni_albums_name UNIQUE (name)",
		"DROP INDEX idx_collections_slug",
		"ALTER TABLE collections ADD CONSTRAINT uni_collections_slug UNIQUE (slug)",
		"DROP INDEX idx_images_album_taken",
		// existing data that must survive the upgrade
		`INSERT INTO albums (name, slug, folder_path, created_at, updated_at) VALUES ('old', 'old', 'old', 1, 1)`,
		`INSERT INTO images (original_path, album_id, object_key, last_modified, created_at)
			VALUES ('old/a.jpg', 1, 'old/a.jpg', 1, 1)`,
		// orphans (no foreign keys existed) that the FK migration must clean up
		`INSERT INTO image_tags (image_path, tag_key, tag_value, source, created_at)
			VALUES ('gone.jpg', 'k', 'v', 'manual', now())`,
		`INSERT INTO albums (name, slug, folder_path, group_id, created_at, updated_at)
			VALUES ('grouped', 'grouped', 'grouped', 999, 1, 1)`,
	} {
		if _, err := db.Exec(stmt); err != nil {
			t.Fatalf("legacy setup %q failed: %v", stmt, err)
		}
	}

	// Without the baseline, migration 00001 would fail with "already exists".
	if err := database.RunMigrations(ctx, db); err != nil {
		t.Fatalf("RunMigrations on AutoMigrate-era database failed: %v", err)
	}
	// Running again must be a no-op.
	if err := database.RunMigrations(ctx, db); err != nil {
		t.Fatalf("second RunMigrations failed: %v", err)
	}

	var applied int
	if err := db.QueryRow("SELECT count(*) FROM goose_db_version WHERE version_id = 1 AND is_applied").Scan(&applied); err != nil {
		t.Fatalf("failed to read goose_db_version: %v", err)
	}
	if applied != 1 {
		t.Errorf("expected version 1 to be recorded exactly once, got %d", applied)
	}

	var legacyConstraints int
	if err := db.QueryRow(
		"SELECT count(*) FROM pg_constraint WHERE conname IN ('uni_albums_name', 'uni_collections_slug')",
	).Scan(&legacyConstraints); err != nil {
		t.Fatalf("failed to query constraints: %v", err)
	}
	if legacyConstraints != 0 {
		t.Errorf("legacy unique constraints were not dropped")
	}

	for _, idx := range []string{"idx_albums_name", "idx_collections_slug", "idx_images_album_taken"} {
		var exists bool
		if err := db.QueryRow("SELECT to_regclass($1) IS NOT NULL", idx).Scan(&exists); err != nil {
			t.Fatalf("failed to check index %s: %v", idx, err)
		}
		if !exists {
			t.Errorf("index %s was not reconciled", idx)
		}
	}

	// partial unique index semantics: a soft-deleted album's name can be reused.
	if _, err := db.Exec(`UPDATE albums SET deleted_at = now() WHERE name = 'old'`); err != nil {
		t.Fatalf("soft delete failed: %v", err)
	}
	if _, err := db.Exec(
		`INSERT INTO albums (name, slug, folder_path, created_at, updated_at) VALUES ('old', 'old', 'old', 1, 1)`,
	); err != nil {
		t.Errorf("re-using a soft-deleted album name failed: %v", err)
	}

	var orphanTags, danglingGroups int
	if err := db.QueryRow("SELECT count(*) FROM image_tags").Scan(&orphanTags); err != nil {
		t.Fatalf("failed to count image_tags: %v", err)
	}
	if err := db.QueryRow("SELECT count(*) FROM albums WHERE group_id IS NOT NULL").Scan(&danglingGroups); err != nil {
		t.Fatalf("failed to count albums: %v", err)
	}
	if orphanTags != 0 || danglingGroups != 0 {
		t.Errorf("orphans not cleaned up: image_tags=%d albums with dangling group=%d", orphanTags, danglingGroups)
	}

	var images int
	if err := db.QueryRow("SELECT count(*) FROM images").Scan(&images); err != nil {
		t.Fatalf("failed to count images: %v", err)
	}
	if images != 1 {
		t.Errorf("expected existing image row to survive upgrade, got %d rows", images)
	}
}
