package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
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
	db, err := database.InitDB(cfg.DatabasePath)
	if err != nil {
		log.Fatalf("FATAL: Failed to initialize database: %v", err)
	}
	defer db.Close()
	err = os.MkdirAll(cfg.ThumbnailDir, 0755)
	if err != nil {
		log.Fatalf("FATAL: Failed to create thumbnail directory %s: %v", cfg.ThumbnailDir, err)
	}
	thumbGenerator := workers.NewThumbnailGenerator(cfg, db, cfg.ThumbnailQueueSize, cfg.NumThumbnailWorkers)

	log.Printf("Serving files from root: %s", cfg.RootDirectory)
	log.Printf("Using database: %s", cfg.DatabasePath)
	log.Printf("Storing thumbnails in: %s", cfg.ThumbnailDir)
	log.Printf("Thumbnail dimensions (Max WxH): %dx%d", cfg.ThumbnailWidth, cfg.ThumbnailHeight)

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

	albumHandler := &handlers.AlbumHandler{DB: db, Cfg: cfg, ThumbGen: thumbGenerator}
	personHandler := &handlers.PersonHandler{DB: db}
	faceHandler := &handlers.FaceHandler{DB: db, Cfg: cfg}

	r.Route("/api", func(r chi.Router) {
		r.Route("/albums", func(r chi.Router) {
			r.Post("/", albumHandler.CreateAlbum)
			r.Get("/", albumHandler.ListAlbums)
			r.Route("/{album_identifier}", func(r chi.Router) {
				r.Get("/", albumHandler.GetAlbum)
				r.Put("/", albumHandler.UpdateAlbum)
				r.Delete("/", albumHandler.DeleteAlbum)
				r.Get("/contents", albumHandler.GetAlbumContents)
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
	})

	r.Get("/thumbnails/*", handlers.ThumbnailServer(cfg.ThumbnailDir))
	r.Get("/*", handlers.DirectoryHandler(cfg, db, thumbGenerator))

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
