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

	thumbWidthStr := os.Getenv("THUMBNAIL_WIDTH")
	thumbHeightStr := os.Getenv("THUMBNAIL_HEIGHT")

	thumbWidth := defaultThumbnailWidth
	if thumbWidthStr != "" {
		parsedWidth, err := strconv.Atoi(thumbWidthStr)
		if err != nil || parsedWidth <= 0 {
			log.Printf("Warning: Invalid THUMBNAIL_WIDTH '%s'. Using default %d. Error: %v", thumbWidthStr, defaultThumbnailWidth, err)
		} else {
			thumbWidth = parsedWidth
		}
	}

	thumbHeight := defaultThumbnailHeight
	if thumbHeightStr != "" {
		parsedHeight, err := strconv.Atoi(thumbHeightStr)
		if err != nil || parsedHeight <= 0 {
			log.Printf("Warning: Invalid THUMBNAIL_HEIGHT '%s'. Using default %d. Error: %v", thumbHeightStr, defaultThumbnailHeight, err)
		} else {
			thumbHeight = parsedHeight
		}
	}

	queueSizeStr := os.Getenv("THUMBNAIL_QUEUE_SIZE")
	queueSize := defaultThumbnailQueueSize
	if queueSizeStr != "" {
		parsedSize, err := strconv.Atoi(queueSizeStr)
		if err != nil || parsedSize <= 0 {
			log.Printf("Warning: Invalid THUMBNAIL_QUEUE_SIZE '%s'. Using default %d. Error: %v", queueSizeStr, defaultThumbnailQueueSize, err)
		} else {
			queueSize = parsedSize
		}
	}

	numWorkersStr := os.Getenv("NUM_THUMBNAIL_WORKERS")
	numWorkers := defaultNumThumbnailWorkers
	if numWorkersStr != "" {
		parsedNum, err := strconv.Atoi(numWorkersStr)
		if err != nil || parsedNum <= 0 {
			log.Printf("Warning: Invalid NUM_THUMBNAIL_WORKERS '%s'. Using default %d. Error: %v", numWorkersStr, defaultNumThumbnailWorkers, err)
		} else {
			numWorkers = parsedNum
		}
	}

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
