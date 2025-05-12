package utils

import (
	"archive/zip"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
)

// CreateAlbumZip creates a ZIP archive of files in an album folder.
// sourceRootDir: Absolute path to the root where original images reside.
// albumRelativeFolderPath: Path of the album folder relative to sourceRootDir.
// archiveSaveDir: The *full, absolute* path to the directory where the ZIP file should be saved (e.g., cfg.ArchivesPath).
// archiveFilenameBase: The base name for the zip file (e.g., "album_123_archive_ts"). Extension (.zip) will be added.
// Returns: final filename (e.g., "album_123_archive_ts.zip"), size in bytes, error.
func CreateAlbumZip(sourceRootDir, albumRelativeFolderPath, archiveSaveDir, archiveFilenameBase string) (string, int64, error) {

	albumFullPath := filepath.Join(sourceRootDir, albumRelativeFolderPath)
	albumFullPath = filepath.Clean(albumFullPath)

	if _, err := os.Stat(albumFullPath); os.IsNotExist(err) {
		return "", 0, fmt.Errorf("album folder not found: %s", albumFullPath)
	} else if err != nil {
		return "", 0, fmt.Errorf("error stating album folder %s: %w", albumFullPath, err)
	}

	// Ensure archive save directory exists
	if err := os.MkdirAll(archiveSaveDir, 0755); err != nil {
		return "", 0, fmt.Errorf("failed to create zip save directory %s: %w", archiveSaveDir, err)
	}

	// Final zip file path
	zipFilename := archiveFilenameBase + ".zip"
	zipFilePath := filepath.Join(archiveSaveDir, zipFilename)

	// Create the final ZIP file directly
	zipFile, err := os.Create(zipFilePath)
	if err != nil {
		return "", 0, fmt.Errorf("failed to create zip file %s: %w", zipFilePath, err)
	}
	// Defer closing the file handle itself
	defer zipFile.Close()

	zipWriter := zip.NewWriter(zipFile)
	// Defer closing the zip writer *before* the file handle
	defer zipWriter.Close()

	// Walk the album directory
	entries, err := os.ReadDir(albumFullPath)
	if err != nil {
		return "", 0, fmt.Errorf("failed to read album directory %s: %w", albumFullPath, err)
	}

	foundFiles := false
	for _, entry := range entries {
		if entry.IsDir() {
			continue // Skip subdirectories
		}

		filePathInAlbum := filepath.Join(albumFullPath, entry.Name())
		fileToZip, err := os.Open(filePathInAlbum)
		if err != nil {
			log.Printf("zipper: Failed to open file %s for zipping: %v. Skipping.", filePathInAlbum, err)
			continue // Skip this file, try others
		}
		// Ensure fileToZip is closed within the loop iteration
		func() {
			defer fileToZip.Close()
			writer, err := zipWriter.Create(entry.Name()) // Path inside zip is just filename
			if err != nil {
				log.Printf("zipper: Failed to create entry in zip for %s: %v. Skipping.", entry.Name(), err)
				return // Skip this file
			}
			if _, err := io.Copy(writer, fileToZip); err != nil {
				log.Printf("zipper: Failed to write file %s to zip: %v. Skipping.", entry.Name(), err)
				return // Skip this file
			}
			foundFiles = true
			// log.Printf("zipper: Added %s to archive %s", entry.Name(), zipFilePath) // Less verbose log
		}() // Immediately invoke the func to ensure defer runs
	}

	// Close the zip writer *explicitly* here to ensure data is flushed
	// before we stat the file. The defer will also run, but this ensures timing.
	if err := zipWriter.Close(); err != nil {
		// Attempt cleanup on finalize error
		zipFile.Close()
		os.Remove(zipFilePath)
		return "", 0, fmt.Errorf("failed to finalize zip writer for %s: %w", zipFilePath, err)
	}

	// If no files were added, remove the empty zip and return an error
	if !foundFiles {
		zipFile.Close() // Need to close before removing
		os.Remove(zipFilePath)
		return "", 0, fmt.Errorf("no files found in album folder %s to zip", albumFullPath)
	}

	// Get the size of the created ZIP file (Stat needs file handle closed)
	// We closed zipWriter, now close zipFile to allow Stat
	zipFile.Close() // Explicit close before Stat

	zipInfo, err := os.Stat(zipFilePath)
	if err != nil {
		// File might be locked briefly? Unlikely but possible.
		// If Stat fails, we can't return size, but file might exist.
		log.Printf("zipper: Warning - failed to stat created zip file %s: %v", zipFilePath, err)
		return zipFilename, 0, fmt.Errorf("zip created but failed to get size: %w", err) // Return filename but size 0 and error
	}

	log.Printf("Successfully created album zip: %s (Size: %d bytes)", zipFilePath, zipInfo.Size())
	// Return the FILENAME only (relative to archiveSaveDir), size, and nil error
	return zipFilename, zipInfo.Size(), nil
}
