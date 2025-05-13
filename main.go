package main

import (
	"fmt"
	"github.com/camden-git/mediasysbackend/media"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/camden-git/mediasysbackend/config"
	"github.com/camden-git/mediasysbackend/database"
	"github.com/camden-git/mediasysbackend/handlers"
	"github.com/camden-git/mediasysbackend/workers"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/joho/godotenv"
	"github.com/rs/cors"
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

	storagePaths := []string{cfg.ThumbnailsPath, cfg.BannersPath, cfg.ArchivesPath, filepath.Dir(cfg.DatabasePath)}
	for _, p := range storagePaths {
		log.Printf("Ensuring storage directory exists: %s", p)
		if err := os.MkdirAll(p, 0755); err != nil {
			log.Fatalf("FATAL: Failed to create storage directory %s: %v", p, err)
		}
	}

	db, err := database.InitDB(cfg.DatabasePath)
	if err != nil {
		log.Fatalf("FATAL: Failed to initialize database: %v", err)
	}
	defer db.Close()

	mediaSubDirs := map[media.AssetType]string{
		media.AssetTypeThumbnail: filepath.Base(cfg.ThumbnailsPath),
		media.AssetTypeBanner:    filepath.Base(cfg.BannersPath),
		media.AssetTypeArchive:   filepath.Base(cfg.ArchivesPath),
	}
	mediaStore, err := media.NewLocalStorage(cfg.MediaStoragePath, mediaSubDirs)
	if err != nil {
		log.Fatalf("FATAL: Failed to initialize media store: %v", err)
	}
	mediaProcessor := media.NewProcessor(mediaStore)

	log.Printf("Initializing image processor worker pool (Workers: %d, Queue Size: %d)...", cfg.NumThumbnailWorkers, cfg.ThumbnailQueueSize)

	imageProcessor := workers.NewImageProcessor(cfg, db, cfg.ThumbnailQueueSize, cfg.NumThumbnailWorkers)

	log.Printf("Serving files from root: %s", cfg.RootDirectory)
	log.Printf("Using database: %s", cfg.DatabasePath)
	log.Printf("Storing thumbnails in: %s", cfg.ThumbnailsPath)
	log.Printf("Thumbnail max size (longest side): %dpx", cfg.ThumbnailMaxSize)

	r := chi.NewRouter()

	corsOptions := cors.Options{
		AllowedOrigins:   []string{"http://localhost:5173"}, //TODO: configurable
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}

	corsHandler := cors.New(corsOptions)

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))
	r.Use(corsHandler.Handler)

	albumHandler := &handlers.AlbumHandler{DB: db, Cfg: cfg, ThumbGen: imageProcessor, MediaProcessor: mediaProcessor}
	personHandler := &handlers.PersonHandler{DB: db}
	faceHandler := &handlers.FaceHandler{DB: db, Cfg: cfg}
	imagePreviewHandler := &handlers.ImagePreviewHandler{DB: db, Cfg: cfg}

	r.Route("/api", func(r chi.Router) {
		r.Route("/albums", func(r chi.Router) {
			r.Post("/", albumHandler.CreateAlbum)
			r.Get("/", albumHandler.ListAlbums)
			r.Route("/{album_identifier}", func(r chi.Router) {
				r.Get("/", albumHandler.GetAlbum)
				r.Put("/", albumHandler.UpdateAlbum)
				r.Delete("/", albumHandler.DeleteAlbum)
				r.Get("/contents", albumHandler.GetAlbumContents)
				r.Put("/banner", albumHandler.UploadAlbumBanner)
				r.Put("/sort_order", albumHandler.UpdateAlbumSortOrder)
				r.Post("/zip", albumHandler.RequestAlbumZipGeneration)
				r.Get("/zip", albumHandler.DownloadAlbumZip)
			})
		})

		r.Route("/people", func(r chi.Router) {
			r.Post("/", personHandler.CreatePerson)
			r.Get("/", personHandler.ListPeople)
			r.Route("/{person_id}", func(r chi.Router) {
				r.Get("/", personHandler.GetPerson)
				r.Put("/", personHandler.UpdatePerson)
				r.Delete("/", personHandler.DeletePerson)
				r.Route("/aliases", func(r chi.Router) {
					r.Post("/", personHandler.AddAlias)
					r.Delete("/{alias_id}", personHandler.DeleteAlias)
				})
			})
		})

		r.Route("/images/faces", func(r chi.Router) {
			r.Post("/", faceHandler.AddFace)
			r.Get("/", faceHandler.ListFacesByImage)
		})

		r.Route("/faces", func(r chi.Router) {
			r.Route("/{face_id}", func(r chi.Router) {
				r.Get("/", faceHandler.GetFace)
				r.Put("/", faceHandler.UpdateFace)
				r.Delete("/", faceHandler.DeleteFace)
			})
		})

		r.Route("/search/faces", func(r chi.Router) {
			r.Get("/", faceHandler.SearchFacesByPerson)
		})

		thumbnailSubDir := filepath.Base(cfg.ThumbnailsPath)
		r.Get(fmt.Sprintf("/%s/*", thumbnailSubDir), handlers.AssetServer(cfg.MediaStoragePath, thumbnailSubDir))
		log.Printf("Registered thumbnail server at /%s/*", thumbnailSubDir)

		bannerSubDir := filepath.Base(cfg.BannersPath)
		r.Get(fmt.Sprintf("/%s/*", bannerSubDir), handlers.AssetServer(cfg.MediaStoragePath, bannerSubDir))
		log.Printf("Registered banner server at /%s/*", bannerSubDir)

		archiveSubDir := filepath.Base(cfg.ArchivesPath)
		r.Get(fmt.Sprintf("/%s/*", archiveSubDir), handlers.AssetServer(cfg.MediaStoragePath, archiveSubDir))
		log.Printf("Registered archive server at /%s/*", archiveSubDir)
	})

	r.Route("/debug", func(r chi.Router) {
		// GET /debug/image_with_faces?path=relative/path/to/image.jpg
		r.Get("/image_with_faces", imagePreviewHandler.ServeImageWithFaces)
	})

	r.Get("/*", handlers.DirectoryHandler(cfg, db, imageProcessor))

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
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}
	log.Fatal(server.ListenAndServe())
}
