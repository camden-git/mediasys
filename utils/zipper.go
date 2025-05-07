package utils

import (
	"archive/zip"
	"fmt"
	"github.com/google/uuid"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"
)

// CreateAlbumZip creates a ZIP archive of files in an album folder.
// sourceRootDir: The application's root directory for *source* images.
// albumRelativeFolderPath: Path of the album folder relative to sourceRootDir.
// archiveSaveDir: The *full, absolute* path where the ZIP file should be saved (e.g., cfg.ArchivesPath).
// Returns: filename relative to archiveSaveDir, size in bytes, error.
func CreateAlbumZip(sourceRootDir, albumRelativeFolderPath, archiveSaveDir string) (string, int64, error) {
	albumFullPath := filepath.Join(sourceRootDir, albumRelativeFolderPath)
	albumFullPath = filepath.Clean(albumFullPath)

	if _, err := os.Stat(albumFullPath); os.IsNotExist(err) {
		return "", 0, fmt.Errorf("album folder not found: %s", albumFullPath)
	} else if err != nil {
		return "", 0, fmt.Errorf("error stating album folder %s: %w", albumFullPath, err)
	}

	if err := os.MkdirAll(archiveSaveDir, 0755); err != nil {
		return "", 0, fmt.Errorf("failed to create zip save directory %s: %w", archiveSaveDir, err)
	}

	timestamp := time.Now().Unix()
	archiveUUID, _ := uuid.NewRandom()
	zipFilename := fmt.Sprintf("archive_%d_%s.zip", timestamp, archiveUUID.String()[:8])
	zipFilePath := filepath.Join(archiveSaveDir, zipFilename)

	zipFile, err := os.Create(zipFilePath)
	if err != nil {
		return "", 0, fmt.Errorf("failed to create zip file %s: %w", zipFilePath, err)
	}
	defer zipFile.Close()

	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	log.Printf("zipper: Archiving files from %s", albumFullPath)
	entries, err := os.ReadDir(albumFullPath)
	if err != nil {
		return "", 0, fmt.Errorf("failed to read album directory %s: %w", albumFullPath, err)
	}

	foundFiles := false
	for _, entry := range entries {
		if entry.IsDir() {
			continue // skip subdirectories
		}

		filePathInAlbum := filepath.Join(albumFullPath, entry.Name())
		fileToZip, err := os.Open(filePathInAlbum)
		if err != nil {
			log.Printf("zipper: Failed to open file %s for zipping: %v. Skipping.", filePathInAlbum, err)
			continue
		}

		writer, err := zipWriter.Create(entry.Name())
		if err != nil {
			fileToZip.Close()
			log.Printf("zipper: Failed to create entry in zip for %s: %v. Skipping.", entry.Name(), err)
			continue
		}

		_, err = io.Copy(writer, fileToZip)
		fileToZip.Close()
		if err != nil {
			log.Printf("zipper: Failed to write file %s to zip: %v. Skipping.", entry.Name(), err)
			continue
		}
		foundFiles = true
		// log.Printf("zipper: Added %s to archive %s", entry.Name(), zipFilePath)
	}

	if !foundFiles {
		zipWriter.Close()
		zipFile.Close()
		os.Remove(zipFilePath)
		return "", 0, fmt.Errorf("no files found in album folder %s to zip", albumFullPath)
	}

	if err := zipWriter.Close(); err != nil {
		return "", 0, fmt.Errorf("failed to finalize zip writer for %s: %w", zipFilePath, err)
	}
	// file handle closed by defer

	zipInfo, err := os.Stat(zipFilePath)
	if err != nil {
		return "", 0, fmt.Errorf("failed to stat created zip file %s: %w", zipFilePath, err)
	}

	log.Printf("Successfully created album zip: %s (Size: %d bytes)", zipFilePath, zipInfo.Size())
	return zipFilePath, zipInfo.Size(), nil
}
