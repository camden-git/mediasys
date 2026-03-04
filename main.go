package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
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

	storagePaths := []string{cfg.ThumbnailsPath, cfg.BannersPath, cfg.ArchivesPath, cfg.PreviewsPath, filepath.Dir(cfg.DatabasePath)}
	for _, p := range storagePaths {
		log.Printf("Ensuring storage directory exists: %s", p)
		if err := os.MkdirAll(p, 0755); err != nil {
			log.Fatalf("FATAL: Failed to create storage directory %s: %v", p, err)
		}
	}

	gormDB, err := database.InitGormDB(cfg.DatabasePath)
	if err != nil {
		log.Fatalf("FATAL: Failed to initialize GORM database: %v", err)
	}
	sqlDB, err := gormDB.DB()
	if err != nil {
		log.Fatalf("FATAL: Failed to get underlying sql.DB from GORM: %v", err)
	}
	defer sqlDB.Close()

	log.Println("Running GORM AutoMigrate...")
	if err := database.AutoMigrateModels(gormDB); err != nil {
		log.Fatalf("FATAL: Failed to auto-migrate GORM models: %v", err)
	}
	log.Println("GORM AutoMigrate completed.")

	log.Println("Running database migrations...")
	if err := database.RunMigrations(gormDB); err != nil {
		log.Fatalf("FATAL: Failed to run database migrations: %v", err)
	}
	log.Println("Database migrations completed.")

	mediaSubDirs := map[media.AssetType]string{
		media.AssetTypeThumbnail: filepath.Base(cfg.ThumbnailsPath),
		media.AssetTypeBanner:    filepath.Base(cfg.BannersPath),
		media.AssetTypeArchive:   filepath.Base(cfg.ArchivesPath),
		media.AssetTypePreview:   filepath.Base(cfg.PreviewsPath),
	}
	mediaStore, err := media.NewLocalStorage(cfg.MediaStoragePath, mediaSubDirs)
	if err != nil {
		log.Fatalf("FATAL: Failed to initialize media store: %v", err)
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

	log.Printf("Initializing image processor worker pool (Workers: %d, Queue Size: %d, Detection Workers: %d, Detection Queue Size: %d)...",
		cfg.NumThumbnailWorkers, cfg.ThumbnailQueueSize, cfg.NumDetectionWorkers, cfg.DetectionQueueSize)
	imageProcessor := workers.NewImageProcessor(
		cfg,
		imageRepo,
		albumRepo,
		faceRepo,
		imageTagRepo,
		cfg.ThumbnailQueueSize,
		cfg.NumThumbnailWorkers,
		cfg.NumDetectionWorkers,
		cfg.DetectionQueueSize,
		hub,
	)

	imageProcessor.StartPreviewCleanup()
	if cfg.MemoryTrimIntervalMinutes > 0 {
		imageProcessor.StartMemoryTrimmer(cfg.MemoryTrimIntervalMinutes)
	}

	log.Printf("Serving files from root: %s", cfg.RootDirectory)
	log.Printf("Using database: %s", cfg.DatabasePath)
	log.Printf("Storing thumbnails in: %s", cfg.ThumbnailsPath)
	log.Printf("Thumbnail max size (longest side): %dpx", cfg.ThumbnailMaxSize)

	if err := handlers.SyncSuperAdminRole(roleRepo); err != nil {
		log.Fatalf("Failed to sync super admin role: %v", err)
	}

	deps := handlers.AppDependencies{
		Cfg:      cfg,
		Hub:      hub,
		UserRepo: userRepo,

		AlbumHandler:        &handlers.AlbumHandler{AlbumRepo: albumRepo, ImageRepo: imageRepo, UserRepo: userRepo, TagRepo: imageTagRepo, Cfg: cfg, ThumbGen: imageProcessor, MediaProcessor: mediaProcessor},
		PersonHandler:       &handlers.PersonHandler{PersonRepo: personRepo, FaceRepo: faceRepo, Cfg: cfg},
		FaceHandler:         &handlers.FaceHandler{FaceRepo: faceRepo, EmbeddingRepo: faceEmbeddingRepo, PersonRepo: personRepo, ImageRepo: imageRepo, Cfg: cfg, FaceRecognitionService: faceRecognitionService},
		ImagePreviewHandler: &handlers.ImagePreviewHandler{FaceRepo: faceRepo, ImageRepo: imageRepo, Cfg: cfg},
		DebugHandler:        &handlers.DebugHandler{Cfg: cfg, ImageRepo: imageRepo, ImageProcessor: imageProcessor},

		AuthHandler:            handlers.NewAuthHandler(userRepo, inviteCodeRepo, cfg),
		PermissionsHandler:     handlers.NewPermissionsHandler(),
		AdminUserHandler:       handlers.NewAdminUserHandler(userRepo, roleRepo),
		AdminRoleHandler:       handlers.NewAdminRoleHandler(roleRepo),
		AdminInviteCodeHandler: handlers.NewAdminInviteCodeHandler(inviteCodeRepo),
		AdminAlbumHandler: func() *handlers.AdminAlbumHandler {
			h := handlers.NewAdminAlbumHandler(albumRepo, imageRepo, userRepo, roleRepo, imageTagRepo, cfg, imageProcessor, hub)
			h.MediaProcessor = mediaProcessor
			return h
		}(),
		AdminAlbumUserHandler:  handlers.NewAdminAlbumUserHandler(userRepo, albumRepo),
		AdminAlbumGroupHandler: handlers.NewAdminAlbumGroupHandler(albumGroupRepo, cfg, mediaProcessor),
		AlbumGroupHandler:      &handlers.AlbumGroupHandler{GroupRepo: albumGroupRepo, ImageRepo: imageRepo},
		AdminImageTagHandler:   &handlers.AdminImageTagHandler{TagRepo: imageTagRepo, AlbumRepo: albumRepo, ImageRepo: imageRepo},
		AdminCollectionHandler: handlers.NewAdminCollectionHandler(collectionRepo, cfg, mediaProcessor),
		CollectionHandler:      &handlers.CollectionHandler{CollectionRepo: collectionRepo, ImageRepo: imageRepo, Cfg: &cfg},
		SetupHandler:           handlers.NewSetupHandler(gormDB, userRepo, roleRepo),
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
	server := &http.Server{
		Addr:         serverAddr,
		Handler:      r,
		ReadTimeout:  60 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}
	log.Fatal(server.ListenAndServe())
}
