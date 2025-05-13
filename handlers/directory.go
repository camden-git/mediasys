package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/camden-git/mediasysbackend/media"
	"github.com/facette/natsort"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/camden-git/mediasysbackend/config"
	"github.com/camden-git/mediasysbackend/database"
	"github.com/camden-git/mediasysbackend/workers"
)

// FileInfo struct
type FileInfo struct {
	Name            string   `json:"name"`
	Path            string   `json:"path"`
	IsDir           bool     `json:"is_dir"`
	Size            int64    `json:"size"`
	ModTime         int64    `json:"mod_time"`
	ThumbnailPath   *string  `json:"thumbnail_path,omitempty"`
	Width           *int     `json:"width,omitempty"`
	Height          *int     `json:"height,omitempty"`
	Aperture        *float64 `json:"aperture,omitempty"`
	ShutterSpeed    *string  `json:"shutter_speed,omitempty"`
	ISO             *int     `json:"iso,omitempty"`
	FocalLength     *float64 `json:"focal_length,omitempty"`
	LensMake        *string  `json:"lens_make,omitempty"`
	LensModel       *string  `json:"lens_model,omitempty"`
	CameraMake      *string  `json:"camera_make,omitempty"`
	CameraModel     *string  `json:"camera_model,omitempty"`
	TakenAt         *int64   `json:"taken_at,omitempty"`
	ThumbnailStatus string   `json:"thumbnail_status,omitempty"`
	MetadataStatus  string   `json:"metadata_status,omitempty"`
	DetectionStatus string   `json:"detection_status,omitempty"`
}

type DirectoryListing struct {
	Path   string     `json:"path"`
	Files  []FileInfo `json:"files"`
	Parent string     `json:"parent,omitempty"`
}

const thumbnailApiPrefix = "/thumbnails/"

type entryInfo struct {
	entry fs.DirEntry
	info  fs.FileInfo
	err   error
}

func DirectoryHandler(cfg config.Config, db *sql.DB, imgProc *workers.ImageProcessor) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rawRequestedPath := r.URL.Path

		var actualContentPath string
		if strings.HasPrefix(rawRequestedPath, "/api/") {
			actualContentPath = strings.TrimPrefix(rawRequestedPath, "/api")
			log.Printf("[DirectoryHandler] Debug: Stripped /api prefix. Raw: '%s', Effective Content Path: '%s'", rawRequestedPath, actualContentPath)
		} else {
			actualContentPath = rawRequestedPath
		}

		if actualContentPath != "/" && !strings.HasSuffix(actualContentPath, "/") {
			potentialFullPath := filepath.Join(cfg.RootDirectory, actualContentPath)
			potentialFullPath = filepath.Clean(potentialFullPath)

			log.Printf("[DirectoryHandler] Debug: RootDirectory='%s', RequestedPath (from Nginx)='%s', PotentialFullPath='%s'", cfg.RootDirectory, actualContentPath, potentialFullPath)

			if !strings.HasPrefix(potentialFullPath, cfg.RootDirectory) && potentialFullPath != cfg.RootDirectory {
				http.Error(w, "Forbidden", http.StatusForbidden)
				log.Printf("Attempted access outside roo	t directory (pre-stat): Request='%s', Resolved='%s', Root='%s'", actualContentPath, potentialFullPath, cfg.RootDirectory)
				return
			}

			stat, err := os.Stat(potentialFullPath)
			isExistingFile := err == nil && !stat.IsDir()

			if isExistingFile {
				serveFileOrDirectory(w, r, cfg, db, imgProc, actualContentPath, potentialFullPath)
				return
			}
			if err != nil && !os.IsNotExist(err) {
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				log.Printf("Error stating potential file %s: %v", potentialFullPath, err)
				return
			}
			http.Redirect(w, r, actualContentPath+"/", http.StatusMovedPermanently)
			return
		}

		fullPath := filepath.Join(cfg.RootDirectory, actualContentPath)
		fullPath = filepath.Clean(fullPath)
		serveFileOrDirectory(w, r, cfg, db, imgProc, actualContentPath, fullPath)
	}
}

func serveFileOrDirectory(w http.ResponseWriter, r *http.Request, cfg config.Config, db *sql.DB, imgProc *workers.ImageProcessor, requestedPath, fullPath string) {
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

	fileInfos, err := listDirectoryContents(cleanedFullPath, requestedPath, cfg, db, imgProc, database.DefaultSortOrder)
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

func listDirectoryContents(baseDirFullPath string, requestPathPrefix string, cfg config.Config, db database.Querier, imgProc *workers.ImageProcessor, sortOrder string) ([]FileInfo, error) {
	dirEntries, err := os.ReadDir(baseDirFullPath)
	if err != nil {
		return nil, fmt.Errorf("reading directory %s: %w", baseDirFullPath, err)
	}

	entriesWithInfo := make([]entryInfo, 0, len(dirEntries))
	for _, entry := range dirEntries {
		entryFullPath := filepath.Join(baseDirFullPath, entry.Name())
		info, statErr := os.Stat(entryFullPath)
		entriesWithInfo = append(entriesWithInfo, entryInfo{
			entry: entry,
			info:  info, // can be nil on error
			err:   statErr,
		})
	}

	sort.SliceStable(entriesWithInfo, func(i, j int) bool {
		ei := entriesWithInfo[i]
		ej := entriesWithInfo[j]

		if ei.err != nil {
			return false
		} // put errored i after valid j
		if ej.err != nil {
			return true
		} // put valid i before errored j

		isDirI := ei.entry.IsDir()
		isDirJ := ej.entry.IsDir()
		if isDirI != isDirJ {
			return isDirI
		}

		switch sortOrder {
		case database.SortDateDesc:
			// sort by ModTime descending (newest first)
			return ei.info.ModTime().After(ej.info.ModTime())
		case database.SortDateAsc:
			// sort by ModTime ascending (oldest first)
			return ei.info.ModTime().Before(ej.info.ModTime())
		case database.SortFilenameNat:
			return natsort.Compare(ei.entry.Name(), ej.entry.Name())
		case database.SortFilenameAsc:
			fallthrough
		default:
			// sort by Name ascending (case-insensitive)
			return strings.ToLower(ei.entry.Name()) < strings.ToLower(ej.entry.Name())
		}
	})

	fileInfos := make([]FileInfo, 0, len(entriesWithInfo))
	for _, ei := range entriesWithInfo {
		// skip entries that had stat errors
		if ei.err != nil {
			log.Printf("Error stating directory entry %s: %v. Skipping.", filepath.Join(baseDirFullPath, ei.entry.Name()), ei.err)
			continue
		}

		entry := ei.entry
		info := ei.info
		name := entry.Name()
		entryFullPath := filepath.Join(baseDirFullPath, name)
		isDir := info.IsDir()
		modTimeUnix := info.ModTime().Unix()

		prefix := strings.TrimSuffix(requestPathPrefix, "/")
		if prefix == "" {
			prefix = "/"
		}
		entryRelativePath := "/" + strings.TrimPrefix(prefix+"/"+name, "/")
		if isDir && !strings.HasSuffix(entryRelativePath, "/") {
			entryRelativePath += "/"
		}

		apiFileInfo := FileInfo{
			Name:    name,
			Path:    entryRelativePath,
			IsDir:   isDir,
			Size:    info.Size(),
			ModTime: modTimeUnix,
		}

		if !isDir && media.IsRasterImage(name) {
			relPathFromRoot, err := filepath.Rel(cfg.RootDirectory, entryFullPath)
			if err != nil {
				log.Printf("CRITICAL: Error creating relative path for DB key (%s relative to %s): %v. Skipping image processing.", entryFullPath, cfg.RootDirectory, err)
				fileInfos = append(fileInfos, apiFileInfo)
				continue
			}
			dbKeyPath := filepath.ToSlash(relPathFromRoot)

			var imageInfo database.Image
			var recordExists bool = true

			imageInfo, err = database.GetImageInfo(db, dbKeyPath)

			if err == sql.ErrNoRows {
				recordExists = false
				// ensure record exists with pending statuses before queuing tasks
				created, ensureErr := database.EnsureImageRecordExists(db, dbKeyPath, modTimeUnix)
				if ensureErr != nil {
					log.Printf("ERROR ensuring image record exists for %s: %v", dbKeyPath, ensureErr)

					fileInfos = append(fileInfos, apiFileInfo)
					continue
				}
				if created {
					// fetch again to get the initialized record with pending statuses
					imageInfo, err = database.GetImageInfo(db, dbKeyPath)
					if err != nil {
						log.Printf("ERROR fetching newly created image record for %s: %v", dbKeyPath, err)
					}
				} else {
					log.Printf("WARNING: EnsureImageRecordExists inconsistency for %s.", dbKeyPath)
					imageInfo, err = database.GetImageInfo(db, dbKeyPath)
					if err != nil {
						log.Printf("ERROR fetching image record for %s after Ensure inconsistency: %v", dbKeyPath, err)
						fileInfos = append(fileInfos, apiFileInfo)
						continue
					}
				}

				if err == nil {
					recordExists = true
				}

			} else if err != nil {
				log.Printf("ERROR querying initial image DB for '%s': %v. Skipping further processing.", dbKeyPath, err)
				fileInfos = append(fileInfos, apiFileInfo)
				continue
			}

			if recordExists {
				apiFileInfo.ThumbnailStatus = imageInfo.ThumbnailStatus
				apiFileInfo.MetadataStatus = imageInfo.MetadataStatus
				apiFileInfo.DetectionStatus = imageInfo.DetectionStatus
				apiFileInfo.Width = imageInfo.Width
				apiFileInfo.Height = imageInfo.Height
				apiFileInfo.Aperture = imageInfo.Aperture
				apiFileInfo.ShutterSpeed = imageInfo.ShutterSpeed
				apiFileInfo.ISO = imageInfo.ISO
				apiFileInfo.FocalLength = imageInfo.FocalLength
				apiFileInfo.LensMake = imageInfo.LensMake
				apiFileInfo.LensModel = imageInfo.LensModel
				apiFileInfo.CameraMake = imageInfo.CameraMake
				apiFileInfo.CameraModel = imageInfo.CameraModel
				apiFileInfo.TakenAt = imageInfo.TakenAt

				if imageInfo.ThumbnailPath != nil && imageInfo.ThumbnailStatus == database.StatusDone {
					thumbFilename := filepath.Base(*imageInfo.ThumbnailPath)
					fullThumbURL := thumbnailApiPrefix + thumbFilename
					apiFileInfo.ThumbnailPath = &fullThumbURL
				}
			} else {
				apiFileInfo.ThumbnailStatus = database.StatusPending
				apiFileInfo.MetadataStatus = database.StatusPending
				apiFileInfo.DetectionStatus = database.StatusPending
			}

			queueThumbnail := false
			queueMetadata := false
			queueDetection := false

			if !recordExists {
				queueThumbnail = true
				queueMetadata = true
				queueDetection = true
				log.Printf("Queuing all tasks for new image record: %s", dbKeyPath)
			} else if modTimeUnix > imageInfo.LastModified {
				// file is newer than last DB update, re-queue everything
				queueThumbnail = true
				queueMetadata = true
				queueDetection = true
				log.Printf("Queuing all tasks for updated image file: %s (ModTime: %d > DB: %d)", dbKeyPath, modTimeUnix, imageInfo.LastModified)
			} else {
				// file not newer, check individual task statuses
				if imageInfo.ThumbnailStatus != database.StatusDone && (imageInfo.ThumbnailStatus == database.StatusPending || imageInfo.ThumbnailStatus == database.StatusProcessing) {
					queueThumbnail = true
					log.Printf("Re-queuing thumbnail task for %s (status: %s)", dbKeyPath, imageInfo.ThumbnailStatus)
				}
				if imageInfo.MetadataStatus != database.StatusDone && (imageInfo.MetadataStatus == database.StatusPending || imageInfo.MetadataStatus == database.StatusProcessing) {
					queueMetadata = true
					log.Printf("Re-queuing metadata task for %s (status: %s)", dbKeyPath, imageInfo.MetadataStatus)
				}
				if imageInfo.DetectionStatus != database.StatusDone && (imageInfo.DetectionStatus == database.StatusPending || imageInfo.DetectionStatus == database.StatusProcessing) {
					queueDetection = true
					log.Printf("Re-queuing detection task for %s (status: %s)", dbKeyPath, imageInfo.DetectionStatus)
				}
			}

			if queueThumbnail || queueMetadata || queueDetection {
				baseJob := workers.ImageJob{
					OriginalImagePath:    entryFullPath,
					OriginalRelativePath: dbKeyPath,
					ModTimeUnix:          modTimeUnix,
				}

				if queueThumbnail {
					thumbJob := baseJob
					thumbJob.TaskType = workers.TaskThumbnail
					imgProc.QueueJob(thumbJob)
				}
				if queueMetadata {
					metaJob := baseJob
					metaJob.TaskType = workers.TaskMetadata
					imgProc.QueueJob(metaJob)
				}
				if queueDetection {
					detectJob := baseJob
					detectJob.TaskType = workers.TaskDetection
					imgProc.QueueJob(detectJob)
				}
			}
		}

		fileInfos = append(fileInfos, apiFileInfo)
	}

	return fileInfos, nil
}
