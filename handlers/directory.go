package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
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

type FileInfo struct {
	Name          string `json:"name"`
	Path          string `json:"path"`
	IsDir         bool   `json:"is_dir"`
	Size          int64  `json:"size"`
	ModTime       int64  `json:"mod_time"`
	ThumbnailPath string `json:"thumbnail_path,omitempty"`
}

type DirectoryListing struct {
	Path   string     `json:"path"`
	Files  []FileInfo `json:"files"`
	Parent string     `json:"parent,omitempty"`
}

const thumbnailApiPrefix = "/thumbnails/"

func DirectoryHandler(cfg config.Config, db *sql.DB, thumbGen *workers.ThumbnailGenerator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		requestedPath := r.URL.Path

		if requestedPath != "/" && !strings.HasSuffix(requestedPath, "/") {
			potentialFullPath := filepath.Join(cfg.RootDirectory, requestedPath)
			potentialFullPath = filepath.Clean(potentialFullPath)

			if !strings.HasPrefix(potentialFullPath, cfg.RootDirectory) && potentialFullPath != cfg.RootDirectory {
				http.Error(w, "Forbidden", http.StatusForbidden)
				log.Printf("Attempted access outside root directory (pre-stat): Request='%s', Resolved='%s', Root='%s'", requestedPath, potentialFullPath, cfg.RootDirectory)
				return
			}

			stat, err := os.Stat(potentialFullPath)
			isExistingFile := err == nil && !stat.IsDir()

			if isExistingFile {
				serveFileOrDirectory(w, r, cfg, db, thumbGen, requestedPath, potentialFullPath)
				return
			} else if err != nil && !os.IsNotExist(err) {
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				log.Printf("Error stating potential file %s: %v", potentialFullPath, err)
				return
			}
			http.Redirect(w, r, requestedPath+"/", http.StatusMovedPermanently)
			return
		}

		fullPath := filepath.Join(cfg.RootDirectory, requestedPath)
		fullPath = filepath.Clean(fullPath)
		serveFileOrDirectory(w, r, cfg, db, thumbGen, requestedPath, fullPath)
	}
}

func serveFileOrDirectory(w http.ResponseWriter, r *http.Request, cfg config.Config, db *sql.DB, thumbGen *workers.ThumbnailGenerator, requestedPath, fullPath string) {
	cleanedFullPath := filepath.Clean(fullPath)
	if !strings.HasPrefix(cleanedFullPath, cfg.RootDirectory) {
		isRootItself := cleanedFullPath == filepath.Clean(cfg.RootDirectory)
		if !isRootItself {
			http.Error(w, "Forbidden", http.StatusForbidden)
			log.Printf("Attempted access outside root directory: Request='%s', Resolved='%s', Cleaned='%s', Root='%s'", requestedPath, fullPath, cleanedFullPath, cfg.RootDirectory)
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
		log.Printf("Error stating file/dir %s: %v", cleanedFullPath, err)
		return
	}

	if !fileInfo.IsDir() {
		http.ServeFile(w, r, cleanedFullPath)
		return
	}

	fileInfos, err := listDirectoryContents(cleanedFullPath, requestedPath, cfg, db, thumbGen)
	if err != nil {
		if os.IsPermission(err) {
			http.Error(w, "Forbidden", http.StatusForbidden)
		} else {
			log.Printf("Error listing directory contents for %s (request path %s): %v", cleanedFullPath, requestedPath, err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
		return
	}

	listing := DirectoryListing{
		Path:  requestedPath,
		Files: fileInfos,
	}

	if requestedPath != "/" && requestedPath != "" {
		parent := filepath.ToSlash(filepath.Dir(strings.TrimSuffix(requestedPath, "/")))
		if parent == "." {
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

func listDirectoryContents(baseDirFullPath string, requestPathPrefix string, cfg config.Config, db *sql.DB, thumbGen *workers.ThumbnailGenerator) ([]FileInfo, error) {

	dirEntries, err := os.ReadDir(baseDirFullPath)
	if err != nil {
		return nil, fmt.Errorf("reading directory %s: %w", baseDirFullPath, err)
	}

	sort.Slice(dirEntries, func(i, j int) bool {
		iIsDir := dirEntries[i].IsDir()
		jIsDir := dirEntries[j].IsDir()
		if iIsDir != jIsDir {
			return iIsDir
		}
		return strings.ToLower(dirEntries[i].Name()) < strings.ToLower(dirEntries[j].Name())
	})

	fileInfos := make([]FileInfo, 0, len(dirEntries))
	for _, entry := range dirEntries {
		name := entry.Name()
		entryFullPath := filepath.Join(baseDirFullPath, name)

		prefix := strings.TrimSuffix(requestPathPrefix, "/")
		if prefix == "" {
			prefix = "/"
		}
		entryRelativePath := prefix + "/" + name
		entryRelativePath = "/" + strings.TrimPrefix(entryRelativePath, "/")

		entryStat, err := os.Stat(entryFullPath)
		if err != nil {
			log.Printf("Error stating directory entry %s: %v. Skipping.", entryFullPath, err)
			continue
		}

		isDir := entryStat.IsDir()
		modTimeUnix := entryStat.ModTime().Unix()

		if isDir && !strings.HasSuffix(entryRelativePath, "/") {
			entryRelativePath += "/"
		}

		apiFileInfo := FileInfo{
			Name:    name,
			Path:    entryRelativePath,
			IsDir:   isDir,
			Size:    entryStat.Size(),
			ModTime: modTimeUnix,
		}

		if !isDir && utils.IsRasterImage(name) {
			relPathFromRoot, err := filepath.Rel(cfg.RootDirectory, entryFullPath)
			if err != nil {
				log.Printf("CRITICAL: Error creating relative path for DB key (%s relative to %s): %v. Skipping thumbnail.", entryFullPath, cfg.RootDirectory, err)
			} else {
				dbKeyPath := filepath.ToSlash(relPathFromRoot)

				dbThumbInfo, err := database.GetThumbnailInfo(db, dbKeyPath)

				shouldQueueGeneration := false
				thumbnailFileExists := false
				var existingThumbPathOnDisk string

				if err == sql.ErrNoRows {
					shouldQueueGeneration = true
				} else if err != nil {
					log.Printf("ERROR querying thumbnail DB for '%s': %v. Skipping thumbnail.", dbKeyPath, err)
				} else {
					existingThumbPathOnDisk = dbThumbInfo.ThumbnailPath
					if modTimeUnix > dbThumbInfo.LastModified {
						shouldQueueGeneration = true
					} else {
						if _, statErr := os.Stat(existingThumbPathOnDisk); statErr == nil {
							thumbnailFileExists = true
						} else {
							log.Printf("Thumbnail file '%s' for '%s' not found on disk (error: %v), queuing regeneration.", existingThumbPathOnDisk, dbKeyPath, statErr)
							shouldQueueGeneration = true
						}
					}
				}

				if thumbnailFileExists {
					thumbFilename := filepath.Base(existingThumbPathOnDisk)
					apiFileInfo.ThumbnailPath = thumbnailApiPrefix + thumbFilename
				}

				if shouldQueueGeneration {
					job := workers.ThumbnailJob{
						OriginalImagePath:    entryFullPath,
						OriginalRelativePath: dbKeyPath,
						ModTimeUnix:          modTimeUnix,
					}
					thumbGen.QueueJob(job)
				}
			}
		}

		fileInfos = append(fileInfos, apiFileInfo)
	}

	return fileInfos, nil
}
