package database

import (
	"fmt"
	"log"
	"os"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/camden-git/mediasysbackend/models"
)

// InitGormDB connects to Postgres using the given DSN and returns a GORM instance.
// It retries for a short while so the app can start alongside the database container.
func InitGormDB(dsn string, logLevel logger.LogLevel) (*gorm.DB, error) {
	gormLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags),
		logger.Config{
			SlowThreshold:             time.Second,
			LogLevel:                  logLevel,
			IgnoreRecordNotFoundError: true,
			Colorful:                  true,
		},
	)

	var db *gorm.DB
	var err error
	for attempt := 1; attempt <= 30; attempt++ {
		db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
			Logger:                                   gormLogger,
			DisableForeignKeyConstraintWhenMigrating: true,
		})
		if err == nil {
			break
		}
		log.Printf("database: connect attempt %d failed: %v", attempt, err)
		time.Sleep(2 * time.Second)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to connect to postgres: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying sql.DB from GORM: %w", err)
	}
	sqlDB.SetMaxOpenConns(20)
	sqlDB.SetMaxIdleConns(5)
	sqlDB.SetConnMaxLifetime(30 * time.Minute)

	log.Println("GORM connected to postgres")
	return db, nil
}

// EnsureNaturalSortCollation creates the ICU collation used by SQLOrderClause for
// filename_nat (natural filename sort, e.g. "img2" before "img10"), if it doesn't
// already exist. Requires a Postgres build with ICU support, which is standard on
// the official postgres image (both debian and alpine variants, and images built
// on top of it such as pgvector/pgvector) since Postgres 15.
func EnsureNaturalSortCollation(db *gorm.DB) error {
	stmt := fmt.Sprintf(
		"CREATE COLLATION IF NOT EXISTS %s (provider = icu, locale = 'und-u-kn-true')",
		NaturalSortCollation,
	)
	if err := db.Exec(stmt).Error; err != nil {
		return fmt.Errorf("failed to create natural sort collation: %w", err)
	}
	return nil
}

// AutoMigrateModels creates or updates the schema for all models.
func AutoMigrateModels(db *gorm.DB) error {
	if err := db.Exec("CREATE EXTENSION IF NOT EXISTS vector").Error; err != nil {
		return fmt.Errorf("failed to create pgvector extension: %w", err)
	}
	if err := EnsureNaturalSortCollation(db); err != nil {
		return err
	}

	err := db.AutoMigrate(
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
	)
	if err != nil {
		return fmt.Errorf("GORM AutoMigrate failed: %w", err)
	}

	// HNSW index for approximate nearest-neighbour cosine similarity search over face
	// embeddings. Faces are compared by cosine similarity (see media.FaceRecognitionModel
	// .CalculateSimilarity and handlers.FaceRecognitionService.CalculateSimilarity), so use
	// vector_cosine_ops to match that semantics.
	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_face_embeddings_embedding_hnsw_cosine " +
		"ON face_embeddings USING hnsw (embedding vector_cosine_ops)").Error; err != nil {
		return fmt.Errorf("failed to create face embedding HNSW index: %w", err)
	}

	log.Println("GORM AutoMigrate completed successfully.")
	return nil
}
