// Package app wires together the database, object store, background workers and
// HTTP router from a config.Config. main.go and the end-to-end tests both build
// their App through New so there is exactly one place that knows how to
// assemble the dependency graph.
package app

import (
	"context"
	"fmt"

	"github.com/camden-git/mediasysbackend/config"
	"github.com/camden-git/mediasysbackend/database"
	"github.com/camden-git/mediasysbackend/handlers"
	"github.com/camden-git/mediasysbackend/media"
	"github.com/camden-git/mediasysbackend/realtime"
	"github.com/camden-git/mediasysbackend/repository"
	"github.com/camden-git/mediasysbackend/workers"
	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// App holds every long-lived dependency the server needs, plus the fully
// assembled router.
type App struct {
	Cfg            config.Config
	Router         chi.Router
	DB             *gorm.DB
	Store          *media.Store
	Hub            *realtime.Hub
	ImageProcessor *workers.ImageProcessor
}

// New builds the full dependency graph for the given config: it connects to
// Postgres, runs migrations, connects to the object store, starts the
// background workers and registers every HTTP route. Callers must call
// Close when done to release the database connection.
func New(ctx context.Context, cfg config.Config) (*App, error) {
	handlers.SetJWTSecret(cfg.JWTSecret)

	logLevel := logger.Warn
	if cfg.DatabaseDebug {
		logLevel = logger.Info
	}
	gormDB, err := database.InitGormDB(cfg.DatabaseURL, logLevel)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}
	sqlDB, err := gormDB.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying sql.DB from GORM: %w", err)
	}

	if err := database.RunMigrations(ctx, sqlDB); err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("failed to migrate database schema: %w", err)
	}

	mediaStore, err := media.NewStore(ctx, media.StoreConfig{
		Endpoint:  cfg.S3Endpoint,
		AccessKey: cfg.S3AccessKey,
		SecretKey: cfg.S3SecretKey,
		Bucket:    cfg.S3Bucket,
		Region:    cfg.S3Region,
		UseSSL:    cfg.S3UseSSL,
	})
	if err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("failed to initialize object store: %w", err)
	}
	mediaProcessor := media.NewProcessor(mediaStore)

	hub := realtime.NewHub()
	go hub.Run()

	albumRepo := repository.NewAlbumRepository(gormDB)
	albumGroupRepo := repository.NewAlbumGroupRepository(gormDB)
	personRepo := repository.NewPersonRepository(gormDB)
	faceRepo := repository.NewFaceRepository(gormDB)
	faceEmbeddingRepo := repository.NewFaceEmbeddingRepository(gormDB)
	imageRepo := repository.NewImageRepository(gormDB)
	userRepo := repository.NewGormUserRepository(gormDB)
	roleRepo := repository.NewGormRoleRepository(gormDB)
	inviteCodeRepo := repository.NewGormInviteCodeRepository(gormDB)
	imageTagRepo := repository.NewImageTagRepository(gormDB)
	collectionRepo := repository.NewCollectionRepository(gormDB)

	faceRecognitionService := handlers.NewFaceRecognitionService(
		faceRepo,
		faceEmbeddingRepo,
		float32(cfg.FaceRecognitionThreshold),
	)

	imageProcessor := workers.NewImageProcessor(cfg, imageRepo, albumRepo, imageTagRepo, mediaStore, hub)
	imageProcessor.StartDispatcher()
	if cfg.MemoryTrimIntervalMinutes > 0 {
		imageProcessor.StartMemoryTrimmer(cfg.MemoryTrimIntervalMinutes)
	}

	if err := handlers.SyncSuperAdminRole(roleRepo); err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("failed to sync super admin role: %w", err)
	}

	deps := handlers.AppDependencies{
		Cfg:       cfg,
		Hub:       hub,
		UserRepo:  userRepo,
		AlbumRepo: albumRepo,
		Store:     mediaStore,
		DB:        gormDB,

		AlbumHandler:        &handlers.AlbumHandler{AlbumRepo: albumRepo, ImageRepo: imageRepo, UserRepo: userRepo, ThumbGen: imageProcessor, Store: mediaStore},
		PersonHandler:       &handlers.PersonHandler{PersonRepo: personRepo, FaceRepo: faceRepo, ImageRepo: imageRepo, Store: mediaStore, Cfg: cfg},
		FaceHandler:         &handlers.FaceHandler{FaceRepo: faceRepo, EmbeddingRepo: faceEmbeddingRepo, PersonRepo: personRepo, ImageRepo: imageRepo, Store: mediaStore, Cfg: cfg, FaceRecognitionService: faceRecognitionService},
		ImagePreviewHandler: &handlers.ImagePreviewHandler{FaceRepo: faceRepo, ImageRepo: imageRepo, Store: mediaStore, Cfg: cfg},
		DebugHandler:        &handlers.DebugHandler{Cfg: cfg, ImageRepo: imageRepo, ImageProcessor: imageProcessor},

		AuthHandler:            handlers.NewAuthHandler(userRepo, inviteCodeRepo, cfg),
		PermissionsHandler:     handlers.NewPermissionsHandler(),
		AdminUserHandler:       handlers.NewAdminUserHandler(userRepo, roleRepo),
		AdminRoleHandler:       handlers.NewAdminRoleHandler(roleRepo),
		AdminInviteCodeHandler: handlers.NewAdminInviteCodeHandler(inviteCodeRepo),
		AdminAlbumHandler: func() *handlers.AdminAlbumHandler {
			h := handlers.NewAdminAlbumHandler(albumRepo, imageRepo, userRepo, imageTagRepo, cfg, imageProcessor, hub)
			h.MediaProcessor = mediaProcessor
			h.Store = mediaStore
			return h
		}(),
		AdminAlbumUserHandler:  handlers.NewAdminAlbumUserHandler(userRepo, albumRepo),
		AdminAlbumGroupHandler: handlers.NewAdminAlbumGroupHandler(albumGroupRepo, mediaProcessor, mediaStore),
		AlbumGroupHandler:      &handlers.AlbumGroupHandler{GroupRepo: albumGroupRepo, ImageRepo: imageRepo},
		AdminImageTagHandler:   &handlers.AdminImageTagHandler{TagRepo: imageTagRepo, AlbumRepo: albumRepo, ImageRepo: imageRepo},
		AdminCollectionHandler: handlers.NewAdminCollectionHandler(collectionRepo, mediaProcessor, mediaStore),
		CollectionHandler:      &handlers.CollectionHandler{CollectionRepo: collectionRepo},
		SetupHandler:           handlers.NewSetupHandler(userRepo, roleRepo),
	}

	r := chi.NewRouter()
	handlers.RegisterRoutes(r, deps)

	return &App{
		Cfg:            cfg,
		Router:         r,
		DB:             gormDB,
		Store:          mediaStore,
		Hub:            hub,
		ImageProcessor: imageProcessor,
	}, nil
}

// Close releases the database connection and stops background workers.
func (a *App) Close() error {
	if a.ImageProcessor != nil {
		a.ImageProcessor.Stop()
	}
	if a.DB != nil {
		if sqlDB, err := a.DB.DB(); err == nil {
			return sqlDB.Close()
		}
	}
	return nil
}
