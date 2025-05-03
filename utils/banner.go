package utils

import (
	"fmt"
	"github.com/google/uuid"
	"image"
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/disintegration/imaging"
)

// TODO: maybe throw this into the env?
const (
	BannerTargetWidth   = 2000
	BannerJpegQuality   = 80
	BannerFileExtension = ".jpg"
)

// ProcessAndSaveBanner takes uploaded file data, processes it, and saves it with a UUID filename.
// returns the final UUID-based filename (without directory) or an error.
func ProcessAndSaveBanner(fileData io.Reader, targetDir string) (string, error) {
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create banner directory %s: %w", targetDir, err)
	}

	img, format, err := image.Decode(fileData)
	if err != nil {
		return "", fmt.Errorf("failed to decode uploaded banner image: %w", err)
	}
	log.Printf("Decoded uploaded banner (format: %s)", format)

	processedImg := imaging.Resize(img, BannerTargetWidth, 0, imaging.Lanczos)

	bannerUUID, err := uuid.NewRandom()
	if err != nil {
		return "", fmt.Errorf("failed to generate UUID for banner filename: %w", err)
	}
	targetFilename := bannerUUID.String() + BannerFileExtension

	savePath := filepath.Join(targetDir, targetFilename)

	err = imaging.Save(processedImg, savePath, imaging.JPEGQuality(BannerJpegQuality))
	if err != nil {
		return "", fmt.Errorf("failed to save processed banner to %s: %w", savePath, err)
	}

	log.Printf("Saved processed banner (UUID: %s) to: %s", bannerUUID.String(), savePath)
	return targetFilename, nil
}
