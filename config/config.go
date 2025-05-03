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
	defaultThumbnailMaxSize    = 300
)

type Config struct {
	RootDirectory        string
	DatabasePath         string
	ThumbnailDir         string
	ThumbnailMaxSize     int
	ThumbnailQueueSize   int
	NumWorkers           int
	FaceDNNNetConfigPath string
	FaceDNNNetModelPath  string
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

	thumbMaxSize := getEnvIntOrDefault("THUMBNAIL_MAX_SIZE", defaultThumbnailMaxSize)
	queueSize := getEnvIntOrDefault("THUMBNAIL_QUEUE_SIZE", defaultThumbnailQueueSize)
	numWorkers := getEnvIntOrDefault("NUM_WORKERS", defaultNumThumbnailWorkers)

	faceDNNConfig := os.Getenv("FACE_DNN_CONFIG_PATH")
	faceDNNModel := os.Getenv("FACE_DNN_MODEL_PATH")

	if _, err := os.Stat(faceDNNConfig); os.IsNotExist(err) {
		log.Printf("Warning: Face DNN config file not found at '%s'. Face detection will fail.", faceDNNConfig)
	}
	if _, err := os.Stat(faceDNNModel); os.IsNotExist(err) {
		log.Printf("Warning: Face DNN model file not found at '%s'. Face detection will fail.", faceDNNModel)
	}

	return Config{
		RootDirectory:        absRoot,
		DatabasePath:         dbPath,
		ThumbnailDir:         absThumbDir,
		ThumbnailMaxSize:     thumbMaxSize,
		ThumbnailQueueSize:   queueSize,
		NumWorkers:           numWorkers,
		FaceDNNNetConfigPath: faceDNNConfig,
		FaceDNNNetModelPath:  faceDNNModel,
	}, nil
}
