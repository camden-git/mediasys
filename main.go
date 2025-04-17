package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/camden-git/mediasysbackend/config"
	"github.com/camden-git/mediasysbackend/database"
	"github.com/camden-git/mediasysbackend/handlers"
	"github.com/camden-git/mediasysbackend/workers"
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

	db, err := database.InitDB(cfg.DatabasePath)
	if err != nil {
		log.Fatalf("FATAL: Failed to initialize database: %v", err)
	}
	defer db.Close()

	log.Printf("ensuring thumbnail directory exists: %s", cfg.ThumbnailDir)
	err = os.MkdirAll(cfg.ThumbnailDir, 0755)
	if err != nil {
		log.Fatalf("FATAL: Failed to create thumbnail directory %s: %v", cfg.ThumbnailDir, err)
	}

	log.Printf("initializing thumbnail worker pool (workers: %d, queue Size: %d)...", cfg.NumThumbnailWorkers, cfg.ThumbnailQueueSize) // <-- Use cfg values in log
	thumbGenerator := workers.NewThumbnailGenerator(cfg, db, cfg.ThumbnailQueueSize, cfg.NumThumbnailWorkers)

	log.Printf("serving files from root: %s", cfg.RootDirectory)
	log.Printf("using database: %s", cfg.DatabasePath)
	log.Printf("storing thumbnails in: %s", cfg.ThumbnailDir)

	http.HandleFunc("/", handlers.DirectoryHandler(cfg, db, thumbGenerator))

	http.HandleFunc("/thumbnails/", handlers.ThumbnailServer(cfg.ThumbnailDir))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	serverAddr := ":" + port
	fmt.Printf("server starting on http://localhost:%s\n", port)
	log.Printf("server listening on %s", serverAddr)

	log.Fatal(http.ListenAndServe(serverAddr, nil))
}
