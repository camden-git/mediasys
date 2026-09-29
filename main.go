package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"runtime/debug"
	"strconv"
	"time"

	"github.com/camden-git/mediasysbackend/config"
	"github.com/camden-git/mediasysbackend/database"
	"github.com/camden-git/mediasysbackend/handlers"
	"github.com/camden-git/mediasysbackend/media"
	"github.com/camden-git/mediasysbackend/realtime"
	"github.com/camden-git/mediasysbackend/repository"
	"github.com/camden-git/mediasysbackend/workers"
	"github.com/go-chi/chi/v5"
	"github.com/joho/godotenv"
	"gorm.io/gorm/logger"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Printf("Info: No .env file found or error loading: %v", err)
	}
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("FATAL: Failed to load configuration: %v", err)
	}

	if memLimitStr := os.Getenv("GOMEMLIMIT"); memLimitStr != "" {
		if memLimit, parseErr := strconv.ParseInt(memLimitStr, 10, 64); parseErr == nil && memLimit > 0 {
			debug.SetMemoryLimit(memLimit)
			log.Printf("GOMEMLIMIT set to %d bytes (%.1f GiB)", memLimit, float64(memLimit)/(1<<30))
		} else {
			log.Printf("Warning: Invalid GOMEMLIMIT value '%s', ignoring", memLimitStr)
		}
	}

	logLevel := logger.Warn
	if cfg.DatabaseDebug {
		logLevel = logger.Info
	}
	gormDB, err := database.InitGormDB(cfg.DatabaseURL, logLevel)
	if err != nil {
		log.Fatalf("FATAL: Failed to initialize database: %v", err)
	}
	sqlDB, err := gormDB.DB()
	if err != nil {
		log.Fatalf("FATAL: Failed to get underlying sql.DB from GORM: %v", err)
	}
	defer sqlDB.Close()

	if err := database.AutoMigrateModels(gormDB); err != nil {
		log.Fatalf("FATAL: Failed to migrate database schema: %v", err)
	}

	mediaStore, err := media.NewStore(context.Background(), media.StoreConfig{
		Endpoint:  cfg.S3Endpoint,
		AccessKey: cfg.S3AccessKey,
		SecretKey: cfg.S3SecretKey,
		Bucket:    cfg.S3Bucket,
		Region:    cfg.S3Region,
		UseSSL:    cfg.S3UseSSL,
	})
	if err != nil {
		log.Fatalf("FATAL: Failed to initialize object store: %v", err)
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
		personRepo,
		faceEmbeddingRepo,
		float32(cfg.FaceRecognitionThreshold),
	)

	imageProcessor := workers.NewImageProcessor(cfg, imageRepo, albumRepo, imageTagRepo, mediaStore, hub)
	imageProcessor.StartDispatcher()
	if cfg.MemoryTrimIntervalMinutes > 0 {
		imageProcessor.StartMemoryTrimmer(cfg.MemoryTrimIntervalMinutes)
	}

	if err := handlers.SyncSuperAdminRole(roleRepo); err != nil {
		log.Fatalf("Failed to sync super admin role: %v", err)
	}

	deps := handlers.AppDependencies{
		Cfg:      cfg,
		Hub:      hub,
		UserRepo: userRepo,
		Store:    mediaStore,
		DB:       gormDB,

		AlbumHandler:        &handlers.AlbumHandler{AlbumRepo: albumRepo, ImageRepo: imageRepo, UserRepo: userRepo, TagRepo: imageTagRepo, Cfg: cfg, ThumbGen: imageProcessor, MediaProcessor: mediaProcessor, Store: mediaStore},
		PersonHandler:       &handlers.PersonHandler{PersonRepo: personRepo, FaceRepo: faceRepo, ImageRepo: imageRepo, Store: mediaStore, Cfg: cfg},
		FaceHandler:         &handlers.FaceHandler{FaceRepo: faceRepo, EmbeddingRepo: faceEmbeddingRepo, PersonRepo: personRepo, ImageRepo: imageRepo, Cfg: cfg, FaceRecognitionService: faceRecognitionService},
		ImagePreviewHandler: &handlers.ImagePreviewHandler{FaceRepo: faceRepo, ImageRepo: imageRepo, Store: mediaStore, Cfg: cfg},
		DebugHandler:        &handlers.DebugHandler{Cfg: cfg, ImageRepo: imageRepo, ImageProcessor: imageProcessor},

		AuthHandler:            handlers.NewAuthHandler(userRepo, inviteCodeRepo, cfg),
		PermissionsHandler:     handlers.NewPermissionsHandler(),
		AdminUserHandler:       handlers.NewAdminUserHandler(userRepo, roleRepo),
		AdminRoleHandler:       handlers.NewAdminRoleHandler(roleRepo),
		AdminInviteCodeHandler: handlers.NewAdminInviteCodeHandler(inviteCodeRepo),
		AdminAlbumHandler: func() *handlers.AdminAlbumHandler {
			h := handlers.NewAdminAlbumHandler(albumRepo, imageRepo, userRepo, roleRepo, imageTagRepo, cfg, imageProcessor, hub)
			h.MediaProcessor = mediaProcessor
			h.Store = mediaStore
			return h
		}(),
		AdminAlbumUserHandler:  handlers.NewAdminAlbumUserHandler(userRepo, albumRepo),
		AdminAlbumGroupHandler: handlers.NewAdminAlbumGroupHandler(albumGroupRepo, cfg, mediaProcessor, mediaStore),
		AlbumGroupHandler:      &handlers.AlbumGroupHandler{GroupRepo: albumGroupRepo, ImageRepo: imageRepo},
		AdminImageTagHandler:   &handlers.AdminImageTagHandler{TagRepo: imageTagRepo, AlbumRepo: albumRepo, ImageRepo: imageRepo},
		AdminCollectionHandler: handlers.NewAdminCollectionHandler(collectionRepo, cfg, mediaProcessor, mediaStore),
		CollectionHandler:      &handlers.CollectionHandler{CollectionRepo: collectionRepo, ImageRepo: imageRepo, Cfg: &cfg},
		SetupHandler:           handlers.NewSetupHandler(userRepo, roleRepo),
	}

	r := chi.NewRouter()
	handlers.RegisterRoutes(r, deps)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	serverAddr := ":" + port
	fmt.Printf("Server starting on http://localhost:%s\n", port)
	log.Printf("Server listening on %s", serverAddr)
	// no read/write timeouts: uploads and archive downloads can be large and slow
	server := &http.Server{
		Addr:              serverAddr,
		Handler:           r,
		ReadHeaderTimeout: 15 * time.Second,
		IdleTimeout:       120 * time.Second,
	}
	log.Fatal(server.ListenAndServe())
}
