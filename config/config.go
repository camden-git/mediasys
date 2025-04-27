package config

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
)

const (
	defaultThumbnailQueueSize  = 200
	defaultNumThumbnailWorkers = 4
	defaultThumbnailWidth      = 150
	defaultThumbnailHeight     = 150
)

type Config struct {
	RootDirectory       string
	DatabasePath        string
	ThumbnailDir        string
	ThumbnailWidth      int
	ThumbnailHeight     int
	ThumbnailQueueSize  int
	NumThumbnailWorkers int
}

func getEnvIntOrDefault(envVar string, defaultVal int) int {
	valStr := os.Getenv(envVar)
	if valStr == "" {
		return defaultVal
	}
	val, err := strconv.Atoi(valStr)
	if err != nil || val <= 0 {
		log.Printf("Warning: Invalid %s '%s'. Using default %d. Error: %v", envVar, valStr, defaultVal, err)
		return defaultVal
	}
	return val
}

func LoadConfig() (Config, error) {
	root := os.Getenv("ROOT_DIRECTORY")
	if root == "" {
		root = "."
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return Config{}, fmt.Errorf("failed to get absolute path for root directory '%s': %w", root, err)
	}

	dbPath := os.Getenv("DATABASE_PATH")
	if dbPath == "" {
		dbPath = "thumbnails.db"
	}

	thumbDir := os.Getenv("THUMBNAIL_DIR")
	if thumbDir == "" {
		thumbDir = filepath.Join(".", "generated_thumbnails")
	}
	absThumbDir, err := filepath.Abs(thumbDir)
	if err != nil {
		return Config{}, fmt.Errorf("failed to get absolute path for thumbnail directory '%s': %w", thumbDir, err)
	}

	thumbWidth := getEnvIntOrDefault("THUMBNAIL_WIDTH", defaultThumbnailWidth)
	thumbHeight := getEnvIntOrDefault("THUMBNAIL_HEIGHT", defaultThumbnailHeight)
	queueSize := getEnvIntOrDefault("THUMBNAIL_QUEUE_SIZE", defaultThumbnailQueueSize)
	numWorkers := getEnvIntOrDefault("NUM_THUMBNAIL_WORKERS", defaultNumThumbnailWorkers)

	return Config{
		RootDirectory:       absRoot,
		DatabasePath:        dbPath,
		ThumbnailDir:        absThumbDir,
		ThumbnailWidth:      thumbWidth,
		ThumbnailHeight:     thumbHeight,
		ThumbnailQueueSize:  queueSize,
		NumThumbnailWorkers: numWorkers,
	}, nil
}
