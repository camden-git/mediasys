package utils

import (
	"archive/zip"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
)

// CreateAlbumZip creates a ZIP archive of all files (not sub-folders) in a given album folder
// rootDir: The application's root directory
// albumRelativeFolderPath: The path of the album's folder, relative to rootDir
// zipSaveDir: The directory where the final ZIP file should be saved
// zipFilenameBase: The base name for the zip file (e.g., "album_123_archive"). Extension will be added
// Returns: full path to the created zip, size in bytes, error
func CreateAlbumZip(rootDir, albumRelativeFolderPath, zipSaveDir, zipFilenameBase string) (string, int64, error) {
	albumFullPath := filepath.Join(rootDir, albumRelativeFolderPath)
	albumFullPath = filepath.Clean(albumFullPath)

	if _, err := os.Stat(albumFullPath); os.IsNotExist(err) {
		return "", 0, fmt.Errorf("album folder not found: %s", albumFullPath)
	} else if err != nil {
		return "", 0, fmt.Errorf("error stating album folder %s: %w", albumFullPath, err)
	}

	if err := os.MkdirAll(zipSaveDir, 0755); err != nil {
		return "", 0, fmt.Errorf("failed to create zip save directory %s: %w", zipSaveDir, err)
	}

	zipFilePath := filepath.Join(zipSaveDir, zipFilenameBase+".zip")

	zipFile, err := os.Create(zipFilePath)
	if err != nil {
		return "", 0, fmt.Errorf("failed to create zip file %s: %w", zipFilePath, err)
	}
	defer zipFile.Close()

	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

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

		if _, err := io.Copy(writer, fileToZip); err != nil {
			fileToZip.Close()
			log.Printf("zipper: Failed to write file %s to zip: %v. Skipping.", entry.Name(), err)
			continue
		}
		fileToZip.Close()
		foundFiles = true
		log.Printf("zipper: Added %s to archive %s", entry.Name(), zipFilePath)
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
	zipFile.Close()

	zipInfo, err := os.Stat(zipFilePath)
	if err != nil {
		return "", 0, fmt.Errorf("failed to stat created zip file %s: %w", zipFilePath, err)
	}

	log.Printf("Successfully created album zip: %s (Size: %d bytes)", zipFilePath, zipInfo.Size())
	return zipFilePath, zipInfo.Size(), nil
}
