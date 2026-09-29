package database_test

import (
	"context"
	"database/sql"
	"fmt"
	"math/rand"
	"net/url"
	"os"
	"sort"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"github.com/camden-git/mediasysbackend/database"
	"github.com/camden-git/mediasysbackend/models"
)

// TestMigrationsMatchGORMSchema is a safety net for database/migrations/: it applies
// the versioned SQL migrations to one empty database, runs the legacy GORM
// AutoMigrate path (as it used to run at startup, see git history for
// database.AutoMigrateModels) against a second empty database built straight from
// the models package structs, and asserts the two resulting schemas agree on
// tables, columns (name/type/nullability) and indexes.
//
// This only runs when TEST_DATABASE_URL is set, since it needs a real Postgres
// server (with the ability to CREATE/DROP DATABASE and CREATE EXTENSION vector) to
// connect to; it is skipped otherwise. Point it at a throwaway pgvector/pgvector
// server, e.g.:
//
//	TEST_DATABASE_URL="postgres://mediasys:mediasys@localhost:5432/postgres?sslmode=disable" go test ./database/...
//
// The user/db in the URL only needs to be an admin connection used to create two
// scratch databases on the same server; it does not need to be the app's database.
func TestMigrationsMatchGORMSchema(t *testing.T) {
	adminURL := os.Getenv("TEST_DATABASE_URL")
	if adminURL == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping schema comparison test (see doc comment for how to run it)")
	}

	ctx := context.Background()

	adminDB, err := sql.Open("pgx", adminURL)
	if err != nil {
		t.Fatalf("failed to open admin connection: %v", err)
	}
	defer adminDB.Close()
	if err := adminDB.PingContext(ctx); err != nil {
		t.Fatalf("failed to ping admin connection: %v", err)
	}

	suffix := rand.Intn(1_000_000)
	migratedDBName := fmt.Sprintf("mediasys_schema_test_migrated_%d", suffix)
	autoDBName := fmt.Sprintf("mediasys_schema_test_automigrate_%d", suffix)

	createDatabase(t, adminDB, migratedDBName)
	defer dropDatabase(t, adminDB, migratedDBName)
	createDatabase(t, adminDB, autoDBName)
	defer dropDatabase(t, adminDB, autoDBName)

	migratedDSN := withDBName(t, adminURL, migratedDBName)
	autoDSN := withDBName(t, adminURL, autoDBName)

	// database created by the goose migrations under database/migrations/.
	migratedSQLDB, err := sql.Open("pgx", migratedDSN)
	if err != nil {
		t.Fatalf("failed to open migrated db: %v", err)
	}
	defer migratedSQLDB.Close()
	if err := database.RunMigrations(ctx, migratedSQLDB); err != nil {
		t.Fatalf("failed to run migrations: %v", err)
	}

	// database created by the legacy GORM AutoMigrate path, driven directly off the
	// models package structs (mirrors the old database.AutoMigrateModels).
	autoGormDB, err := gorm.Open(postgres.Open(autoDSN), &gorm.Config{
		Logger:                                   gormlogger.Default.LogMode(gormlogger.Silent),
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		t.Fatalf("failed to open automigrate db: %v", err)
	}
	if err := autoGormDB.Exec("CREATE EXTENSION IF NOT EXISTS vector").Error; err != nil {
		t.Fatalf("failed to create vector extension: %v", err)
	}
	if err := autoGormDB.Exec(
		"CREATE COLLATION IF NOT EXISTS natural_sort (provider = icu, locale = 'und-u-kn-true')",
	).Error; err != nil {
		t.Fatalf("failed to create natural_sort collation: %v", err)
	}
	if err := autoGormDB.AutoMigrate(
		&models.AlbumGroup{},
		&models.Person{},
		&models.Alias{},
		&models.Face{},
		&models.FaceEmbedding{},
		&models.Image{},
		&models.Album{},
		&models.AlbumBanner{},
		&models.User{},
		&models.UserAlbumPermission{},
		&models.Role{},
		&models.UserRole{},
		&models.RoleAlbumPermission{},
		&models.InviteCode{},
		&models.ImageTag{},
		&models.AlbumDefaultTag{},
		&models.Collection{},
		&models.CollectionTagFilter{},
		&models.CollectionBanner{},
	); err != nil {
		t.Fatalf("GORM AutoMigrate failed: %v", err)
	}
	if err := autoGormDB.Exec(
		"CREATE INDEX IF NOT EXISTS idx_face_embeddings_embedding_hnsw_cosine " +
			"ON face_embeddings USING hnsw (embedding vector_cosine_ops)",
	).Error; err != nil {
		t.Fatalf("failed to create HNSW index: %v", err)
	}
	autoSQLDB, err := autoGormDB.DB()
	if err != nil {
		t.Fatalf("failed to get underlying sql.DB: %v", err)
	}
	defer autoSQLDB.Close()

	migratedTables := fetchTables(t, migratedSQLDB)
	autoTables := fetchTables(t, autoSQLDB)
	if diff := diffStringSlices(migratedTables, autoTables); diff != "" {
		t.Errorf("table sets differ between migrations and AutoMigrate:\n%s", diff)
	}

	migratedColumns := fetchColumns(t, migratedSQLDB)
	autoColumns := fetchColumns(t, autoSQLDB)
	if diff := diffStringSlices(migratedColumns, autoColumns); diff != "" {
		t.Errorf("columns differ between migrations and AutoMigrate:\n%s", diff)
	}

	migratedIndexes := fetchIndexes(t, migratedSQLDB)
	autoIndexes := fetchIndexes(t, autoSQLDB)
	if diff := diffStringSlices(migratedIndexes, autoIndexes); diff != "" {
		t.Errorf("indexes differ between migrations and AutoMigrate:\n%s", diff)
	}
}

func createDatabase(t *testing.T, adminDB *sql.DB, name string) {
	t.Helper()
	if _, err := adminDB.Exec(fmt.Sprintf("CREATE DATABASE %s", pgQuoteIdent(name))); err != nil {
		t.Fatalf("failed to create database %s: %v", name, err)
	}
}

func dropDatabase(t *testing.T, adminDB *sql.DB, name string) {
	t.Helper()
	if _, err := adminDB.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s WITH (FORCE)", pgQuoteIdent(name))); err != nil {
		t.Logf("warning: failed to drop database %s: %v", name, err)
	}
}

func pgQuoteIdent(name string) string {
	return `"` + name + `"`
}

// withDBName returns dsn with its path (database name) replaced by name.
func withDBName(t *testing.T, dsn, name string) string {
	t.Helper()
	u, err := url.Parse(dsn)
	if err != nil {
		t.Fatalf("failed to parse TEST_DATABASE_URL: %v", err)
	}
	u.Path = "/" + name
	return u.String()
}

// fetchTables returns the sorted list of base tables in the public schema.
func fetchTables(t *testing.T, db *sql.DB) []string {
	t.Helper()
	rows, err := db.Query(
		"SELECT table_name FROM information_schema.tables " +
			"WHERE table_schema = 'public' AND table_type = 'BASE TABLE' AND table_name != 'goose_db_version'",
	)
	if err != nil {
		t.Fatalf("failed to query tables: %v", err)
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatalf("failed to scan table name: %v", err)
		}
		tables = append(tables, name)
	}
	sort.Strings(tables)
	return tables
}

// fetchColumns returns one descriptive string per column: table, column name,
// data type (udt_name captures e.g. "vector" which data_type alone doesn't), and
// nullability.
func fetchColumns(t *testing.T, db *sql.DB) []string {
	t.Helper()
	rows, err := db.Query(`
		SELECT table_name, column_name, udt_name, is_nullable
		FROM information_schema.columns
		WHERE table_schema = 'public' AND table_name != 'goose_db_version'
	`)
	if err != nil {
		t.Fatalf("failed to query columns: %v", err)
	}
	defer rows.Close()

	var columns []string
	for rows.Next() {
		var table, column, udtName, nullable string
		if err := rows.Scan(&table, &column, &udtName, &nullable); err != nil {
			t.Fatalf("failed to scan column: %v", err)
		}
		columns = append(columns, fmt.Sprintf("%s.%s type=%s nullable=%s", table, column, udtName, nullable))
	}
	sort.Strings(columns)
	return columns
}

// fetchIndexes returns the canonical (Postgres-reconstructed) definition of every
// index in the public schema, which is independent of how the index was originally
// created (hand-written SQL vs GORM-generated SQL should produce byte-identical
// indexdef text for equivalent indexes).
func fetchIndexes(t *testing.T, db *sql.DB) []string {
	t.Helper()
	rows, err := db.Query(
		"SELECT tablename, indexname, indexdef FROM pg_indexes " +
			"WHERE schemaname = 'public' AND tablename != 'goose_db_version'",
	)
	if err != nil {
		t.Fatalf("failed to query indexes: %v", err)
	}
	defer rows.Close()

	var indexes []string
	for rows.Next() {
		var table, name, def string
		if err := rows.Scan(&table, &name, &def); err != nil {
			t.Fatalf("failed to scan index: %v", err)
		}
		indexes = append(indexes, fmt.Sprintf("%s.%s: %s", table, name, def))
	}
	sort.Strings(indexes)
	return indexes
}

// diffStringSlices returns a human-readable diff of two sorted slices, or "" if
// they're equal.
func diffStringSlices(a, b []string) string {
	setA := make(map[string]bool, len(a))
	for _, v := range a {
		setA[v] = true
	}
	setB := make(map[string]bool, len(b))
	for _, v := range b {
		setB[v] = true
	}

	var diff string
	for _, v := range a {
		if !setB[v] {
			diff += fmt.Sprintf("  only in migrations: %s\n", v)
		}
	}
	for _, v := range b {
		if !setA[v] {
			diff += fmt.Sprintf("  only in AutoMigrate: %s\n", v)
		}
	}
	return diff
}
