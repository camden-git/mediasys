package config

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
)

const (
	DefaultThumbnailsSubDir = "thumbnails"
	DefaultBannersSubDir    = "album_banners"
	DefaultArchivesSubDir   = "album_archives"
)

const (
	defaultThumbnailQueueSize  = 200
	defaultNumThumbnailWorkers = 4
	defaultThumbnailMaxSize    = 300
)

type Config struct {
	// source directory (where original user files are scanned)
	RootDirectory string

	// database path
	DatabasePath string

	// media storage configuration
	MediaStoragePath string // primary root for generated assets (thumbs, banners, zips)
	ThumbnailsPath   string // full-calculated path for thumbnails
	BannersPath      string // full-calculated path for banners
	ArchivesPath     string // full-calculated path for archives

	// thumbnail generation settings
	ThumbnailMaxSize int

	// worker settings
	ThumbnailQueueSize  int
	NumThumbnailWorkers int

	// face detection model paths (DNN)
	FaceDNNNetConfigPath string
	FaceDNNNetModelPath  string
}

func getEnvOrDefault(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
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
	root := getEnvOrDefault("ROOT_DIRECTORY", ".")
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return Config{}, fmt.Errorf("failed to get absolute path for root directory '%s': %w", root, err)
	}

	dbPath := getEnvOrDefault("DATABASE_PATH", "images.db")

	mediaStorage := getEnvOrDefault("MEDIA_STORAGE_PATH", filepath.Join(".", "media_storage"))
	absMediaStorage, err := filepath.Abs(mediaStorage)
	if err != nil {
		return Config{}, fmt.Errorf("failed to get absolute path for media storage '%s': %w", mediaStorage, err)
	}

	thumbSubDir := getEnvOrDefault("THUMBNAILS_SUBDIR", DefaultThumbnailsSubDir)
	absThumbnailsPath := filepath.Join(absMediaStorage, thumbSubDir)

	bannerSubDir := getEnvOrDefault("BANNERS_SUBDIR", DefaultBannersSubDir)
	absBannersPath := filepath.Join(absMediaStorage, bannerSubDir)

	archiveSubDir := getEnvOrDefault("ARCHIVES_SUBDIR", DefaultArchivesSubDir)
	absArchivesPath := filepath.Join(absMediaStorage, archiveSubDir)

	thumbMaxSize := getEnvIntOrDefault("THUMBNAIL_MAX_SIZE", defaultThumbnailMaxSize)

	queueSize := getEnvIntOrDefault("THUMBNAIL_QUEUE_SIZE", defaultThumbnailQueueSize)
	numWorkers := getEnvIntOrDefault("NUM_THUMBNAIL_WORKERS", defaultNumThumbnailWorkers)

	faceDNNConfig := getEnvOrDefault("FACE_DNN_CONFIG_PATH", "./models/deploy.prototxt.txt")
	faceDNNModel := getEnvOrDefault("FACE_DNN_MODEL_PATH", "./models/res10_300x300_ssd_iter_140000_fp16.caffemodel")

	cfg := Config{
		RootDirectory:        absRoot,
		DatabasePath:         dbPath,
		MediaStoragePath:     absMediaStorage,
		ThumbnailsPath:       absThumbnailsPath,
		BannersPath:          absBannersPath,
		ArchivesPath:         absArchivesPath,
		ThumbnailMaxSize:     thumbMaxSize,
		ThumbnailQueueSize:   queueSize,
		NumThumbnailWorkers:  numWorkers,
		FaceDNNNetConfigPath: faceDNNConfig,
		FaceDNNNetModelPath:  faceDNNModel,
	}

	return cfg, nil
}
