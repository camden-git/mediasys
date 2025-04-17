package handlers

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/camden-git/mediasysbackend/config"
	"github.com/camden-git/mediasysbackend/database"
	"github.com/camden-git/mediasysbackend/utils"
	"github.com/camden-git/mediasysbackend/workers"
)

// FileInfo represents a file or directory entry in the API response
type FileInfo struct {
	Name          string `json:"name"`
	Path          string `json:"path"` // relative path from root (URL format)
	IsDir         bool   `json:"is_dir"`
	Size          int64  `json:"size"`
	ModTime       int64  `json:"mod_time"`                 // unix timestamp
	ThumbnailPath string `json:"thumbnail_path,omitempty"` // API path to thumbnail (/thumbnails/uuid.jpg), present only if available
}

// DirectoryListing represents the JSON structure for a directory listing response
type DirectoryListing struct {
	Path   string     `json:"path"` // requested path (URL format)
	Files  []FileInfo `json:"files"`
	Parent string     `json:"parent,omitempty"` // parent path (URL format)
}

// thumbnailApiPrefix is the URL prefix for accessing generated thumbnails
const thumbnailApiPrefix = "/thumbnails/"

// DirectoryHandler creates the main HTTP handler for serving files and directory listings
func DirectoryHandler(cfg config.Config, db *sql.DB, thumbGen *workers.ThumbnailGenerator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		requestedPath := r.URL.Path

		if requestedPath != "/" && !strings.HasSuffix(requestedPath, "/") {
			// construct the potential full filesystem path
			potentialFullPath := filepath.Join(cfg.RootDirectory, requestedPath)
			potentialFullPath = filepath.Clean(potentialFullPath)

			if !strings.HasPrefix(potentialFullPath, cfg.RootDirectory) && potentialFullPath != cfg.RootDirectory {
				http.Error(w, "Forbidden", http.StatusForbidden)
				log.Printf("attempted access outside root directory (pre-stat): Request='%s', Resolved='%s', Root='%s'", requestedPath, potentialFullPath, cfg.RootDirectory)
				return
			}

			stat, err := os.Stat(potentialFullPath)
			isExistingFile := err == nil && !stat.IsDir()

			if isExistingFile {
				serveFileOrDirectory(w, r, cfg, db, thumbGen, requestedPath, potentialFullPath)
				return
			} else if err != nil && !os.IsNotExist(err) {
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				log.Printf("error stating potential file %s: %v", potentialFullPath, err)
				return
			}

			// if it wasn't an existing file (or stat resulted in not found),
			// treat it as a directory and redirect to add the trailing slash
			http.Redirect(w, r, requestedPath+"/", http.StatusMovedPermanently)
			return
		}

		fullPath := filepath.Join(cfg.RootDirectory, requestedPath)
		fullPath = filepath.Clean(fullPath)
		serveFileOrDirectory(w, r, cfg, db, thumbGen, requestedPath, fullPath)
	}
}

// serveFileOrDirectory handles the core logic of serving a file or listing a directory's content
func serveFileOrDirectory(w http.ResponseWriter, r *http.Request, cfg config.Config, db *sql.DB, thumbGen *workers.ThumbnailGenerator, requestedPath, fullPath string) {
	cleanedFullPath := filepath.Clean(fullPath)
	if !strings.HasPrefix(cleanedFullPath, cfg.RootDirectory) {
		isRootItself := cleanedFullPath == filepath.Clean(cfg.RootDirectory)
		if !isRootItself {
			http.Error(w, "Forbidden", http.StatusForbidden)
			log.Printf("attempted access outside root directory: Request='%s', Resolved='%s', Cleaned='%s', Root='%s'", requestedPath, fullPath, cleanedFullPath, cfg.RootDirectory)
			return
		}
	}

	fileInfo, err := os.Stat(cleanedFullPath)
	if os.IsNotExist(err) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		log.Printf("error stating file/dir %s: %v", cleanedFullPath, err)
		return
	}

	if !fileInfo.IsDir() {
		http.ServeFile(w, r, cleanedFullPath)
		return
	}

	dirEntries, err := os.ReadDir(cleanedFullPath)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		log.Printf("error reading directory %s: %v", cleanedFullPath, err)
		return
	}

	sort.Slice(dirEntries, func(i, j int) bool {
		iIsDir := dirEntries[i].IsDir()
		jIsDir := dirEntries[j].IsDir()
		if iIsDir != jIsDir {
			return iIsDir // true puts directories first
		}
		return strings.ToLower(dirEntries[i].Name()) < strings.ToLower(dirEntries[j].Name())
	})

	fileInfos := make([]FileInfo, 0, len(dirEntries))
	for _, entry := range dirEntries {
		name := entry.Name()
		entryRelativePath := "/" + strings.TrimPrefix(filepath.ToSlash(filepath.Join(requestedPath, name)), "/")
		entryFullPath := filepath.Join(cleanedFullPath, name)

		entryStat, err := os.Stat(entryFullPath)
		if err != nil {
			log.Printf("error stating directory entry %s: %v. skipping.", entryFullPath, err)
			continue
		}

		isDir := entryStat.IsDir()
		modTimeUnix := entryStat.ModTime().Unix()

		apiFileInfo := FileInfo{
			Name:    name,
			Path:    entryRelativePath,
			IsDir:   isDir,
			Size:    entryStat.Size(),
			ModTime: modTimeUnix,
		}

		if !isDir && utils.IsRasterImage(name) {
			dbKeyPath := strings.TrimPrefix(entryRelativePath, "/")

			dbThumbInfo, err := database.GetThumbnailInfo(db, dbKeyPath)

			shouldQueueGeneration := false
			thumbnailFileExists := false
			var existingThumbPathOnDisk string

			if err == sql.ErrNoRows {
				shouldQueueGeneration = true
				log.Printf("DB record for '%s' not found, queuing generation", dbKeyPath)
			} else if err != nil {
				log.Printf("ERROR querying thumbnail DB for '%s': %v", dbKeyPath, err)
			} else {
				// Thumbnail record exists in the database. Check if it's up-to-date and file exists.
				existingThumbPathOnDisk = dbThumbInfo.ThumbnailPath
				if modTimeUnix > dbThumbInfo.LastModified {
					// The original file has been modified since the thumbnail was generated. Regenerate.
					shouldQueueGeneration = true
					log.Printf("DB record for '%s' outdated (FileMod: %d > DBMod: %d), queuing regeneration.", dbKeyPath, modTimeUnix, dbThumbInfo.LastModified)
					// Note: Old thumbnail file remains on disk until potentially overwritten or cleaned up later.
				} else {
					// DB record is current. Check if the thumbnail file physically exists.
					if _, statErr := os.Stat(existingThumbPathOnDisk); statErr == nil {
						// File exists on disk and DB record is current.
						thumbnailFileExists = true
					} else {
						// File referenced in DB doesn't exist on disk. Regenerate.
						log.Printf("Thumbnail file '%s' for '%s' not found on disk (error: %v), queuing regeneration.", existingThumbPathOnDisk, dbKeyPath, statErr)
						shouldQueueGeneration = true
					}
				}
			}

			if thumbnailFileExists {
				thumbFilename := filepath.Base(existingThumbPathOnDisk) // extract "uuid.jpg"
				apiFileInfo.ThumbnailPath = thumbnailApiPrefix + thumbFilename
			}

			if shouldQueueGeneration {
				job := workers.ThumbnailJob{
					OriginalImagePath:    entryFullPath, // full path to the source image
					OriginalRelativePath: dbKeyPath,     // relative path (DB key)
					ModTimeUnix:          modTimeUnix,   // mod time of the source image
				}

				queued := thumbGen.QueueJob(job)
				if !queued {
					log.Printf("failed to queue thumbnail job for %s (maybe queue full or already pending)", dbKeyPath)
				}
			}
		}

		fileInfos = append(fileInfos, apiFileInfo)
	}

	listing := DirectoryListing{
		Path:  requestedPath,
		Files: fileInfos,
	}

	if requestedPath != "/" && requestedPath != "" {
		parent := filepath.ToSlash(filepath.Dir(strings.TrimSuffix(requestedPath, "/")))
		if parent == "." { // dir of "/something" is "."
			parent = "/"
		}

		if !strings.HasSuffix(parent, "/") && parent != "/" {
			parent += "/"
		}
		listing.Parent = parent
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")

	if err := json.NewEncoder(w).Encode(listing); err != nil {
		log.Printf("Error encoding JSON response for path %s: %v", requestedPath, err)
	}
}
