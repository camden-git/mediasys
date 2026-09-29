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

// AutoMigrateModels creates or updates the schema for all models.
func AutoMigrateModels(db *gorm.DB) error {
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
	log.Println("GORM AutoMigrate completed successfully.")
	return nil
}
