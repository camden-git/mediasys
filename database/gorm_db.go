package database

import (
	"fmt"
	"log"
	"os"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/camden-git/mediasysbackend/models"
)

// InitGormDB initializes and returns a GORM database instance
func InitGormDB(dataSourceName string) (*gorm.DB, error) {
	gormLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags), // io writer
		logger.Config{
			SlowThreshold:             time.Second,
			LogLevel:                  logger.Info,
			IgnoreRecordNotFoundError: true,
			Colorful:                  true,
		},
	)

	db, err := gorm.Open(sqlite.Open(dataSourceName), &gorm.Config{
		Logger: gormLogger,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to connect to database using GORM: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying sql.DB from GORM: %w", err)
	}

	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	log.Println("GORM Database initialized successfully at", dataSourceName)
	return db, nil
}

// AutoMigrateModels can be called after InitGormDB to migrate schemas
// It's placed here for convenience but should be called selectively
func AutoMigrateModels(db *gorm.DB) error {
	err := db.AutoMigrate(
		&models.Person{},
		&models.Alias{},
		&models.Face{},
		&models.FaceEmbedding{},
		&models.Image{},
		&models.Album{},
		&models.User{},
		&models.UserAlbumPermission{},
		&models.Role{},
		&models.UserRole{},
		&models.RoleAlbumPermission{},
		&models.InviteCode{},
	)
	if err != nil {
		return fmt.Errorf("GORM AutoMigrate failed: %w", err)
	}
	log.Println("GORM AutoMigrate completed successfully.")
	return nil
}
