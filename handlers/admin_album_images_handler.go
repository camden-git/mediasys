package handlers

import (
	"errors"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/camden-git/mediasysbackend/media"
	"github.com/camden-git/mediasysbackend/models"
	"github.com/camden-git/mediasysbackend/realtime"
	"github.com/camden-git/mediasysbackend/repository"
	"github.com/camden-git/mediasysbackend/workers"
	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"
)

// UploadImages handles multipart folder or multiple file uploads into the album's folder and queues processing
func (h *AdminAlbumHandler) UploadImages(w http.ResponseWriter, r *http.Request) {
	albumIDStr := chi.URLParam(r, "id")
	albumID, err := strconv.ParseUint(albumIDStr, 10, 64)
	if err != nil {
		WriteAPIError(w, http.StatusBadRequest, "InvalidAlbumID", "Invalid album ID")
		return
	}

	album, err := h.AlbumRepo.GetByID(uint(albumID))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			WriteAPIError(w, http.StatusNotFound, "AlbumNotFound", "Album not found")
		} else {
			log.Printf("Error fetching album %d for upload: %v", albumID, err)
			WriteAPIError(w, http.StatusInternalServerError, "AlbumFetchError", "Failed to fetch album")
		}
		return
	}

	if h.ImgProc == nil {
		WriteAPIError(w, http.StatusInternalServerError, "ConfigError", "Image processor not configured")
		return
	}

	reader, err := r.MultipartReader()
	if err != nil {
		WriteAPIError(w, http.StatusBadRequest, "InvalidMultipart", "Invalid multipart form: "+err.Error())
		return
	}

	albumBase := filepath.Join(h.Cfg.RootDirectory, album.FolderPath)
	if err := os.MkdirAll(albumBase, 0755); err != nil {
		WriteAPIError(w, http.StatusInternalServerError, "FolderError", "Failed to ensure album folder")
		return
	}

	const maxFileSize int64 = 32 * 1024 * 1024 // 32 MB

	var relPathsQueue []string
	saved := 0
	failed := []map[string]string{}
	for {
		part, err := reader.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Printf("UploadImages: error reading part: %v", err)
			WriteAPIError(w, http.StatusBadRequest, "MalformedUpload", "Malformed upload data")
			return
		}

		field := part.FormName()
		if field == "relative_path" {
			data, _ := io.ReadAll(part)
			rp := strings.TrimSpace(string(data))
			rp = filepath.Clean(rp)
			rp = filepath.ToSlash(rp)
			// prevent path escape
			rp = strings.TrimPrefix(rp, "./")
			rp = strings.TrimPrefix(rp, "/")
			relPathsQueue = append(relPathsQueue, rp)
			continue
		}

		if field != "files" {
			// ignore unknown fields
			continue
		}

		filename := part.FileName()
		rel := filename
		if len(relPathsQueue) > 0 {
			rel = relPathsQueue[0]
			relPathsQueue = relPathsQueue[1:]
		}
		if rel == "" {
			rel = filename
		}
		rel = filepath.Clean(rel)
		rel = filepath.ToSlash(rel)
		rel = strings.TrimPrefix(rel, "./")
		rel = strings.TrimPrefix(rel, "/")
		// strip top-level folder (e.g., `todo/`) from webkitRelativePath so files land at album root
		if idx := strings.Index(rel, "/"); idx >= 0 {
			rel = rel[idx+1:]
		}

		// Validate image type before writing to disk
		if !media.IsRasterImage(filename) {
			failed = append(failed, map[string]string{"path": rel, "error": "unsupported file type"})
			continue
		}

		destPath := filepath.Join(albumBase, rel)
		// security: ensure inside albumBase
		if !strings.HasPrefix(filepath.Clean(destPath), filepath.Clean(albumBase)) {
			log.Printf("UploadImages: blocked path traversal: %s", destPath)
			failed = append(failed, map[string]string{"path": rel, "error": "invalid path"})
			continue
		}
		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			log.Printf("UploadImages: mkdir error for %s: %v", destPath, err)
			failed = append(failed, map[string]string{"path": rel, "error": "failed to create directory"})
			continue
		}

		// Compute db key once (destPath doesn't change)
		relFromRoot, relErr := filepath.Rel(h.Cfg.RootDirectory, destPath)
		relDBKey := ""
		if relErr == nil {
			relDBKey = filepath.ToSlash(relFromRoot)
		}

		out, err := os.Create(destPath)
		if err != nil {
			log.Printf("UploadImages: create error for %s: %v", destPath, err)
			if h.Hub != nil && relDBKey != "" {
				h.Hub.Broadcast(realtime.Event{Type: "upload", Path: relDBKey, Status: "error", Error: err.Error(), Timestamp: time.Now().Unix()})
			}
			failed = append(failed, map[string]string{"path": rel, "error": "failed to create file"})
			continue
		}
		if h.Hub != nil && relDBKey != "" {
			h.Hub.Broadcast(realtime.Event{Type: "upload", Path: relDBKey, Status: "uploading", Timestamp: time.Now().Unix()})
		}

		n, copyErr := io.Copy(out, io.LimitReader(part, maxFileSize+1))
		out.Close()

		if n > maxFileSize {
			os.Remove(destPath)
			log.Printf("UploadImages: file too large: %s", destPath)
			if h.Hub != nil && relDBKey != "" {
				h.Hub.Broadcast(realtime.Event{Type: "upload", Path: relDBKey, Status: "error", Error: "file exceeds 32 MB limit", Timestamp: time.Now().Unix()})
			}
			failed = append(failed, map[string]string{"path": rel, "error": "file exceeds 32 MB limit"})
			continue
		}
		if copyErr != nil {
			log.Printf("UploadImages: write error for %s: %v", destPath, copyErr)
			os.Remove(destPath)
			if h.Hub != nil && relDBKey != "" {
				h.Hub.Broadcast(realtime.Event{Type: "upload", Path: relDBKey, Status: "error", Error: copyErr.Error(), Timestamp: time.Now().Unix()})
			}
			failed = append(failed, map[string]string{"path": rel, "error": copyErr.Error()})
			continue
		}

		if h.Hub != nil && relDBKey != "" {
			h.Hub.Broadcast(realtime.Event{Type: "upload", Path: relDBKey, Status: "uploaded", Timestamp: time.Now().Unix()})
		}

		if relDBKey == "" {
			log.Printf("UploadImages: failed to compute relative path for %s: %v", destPath, relErr)
			continue
		}

		info, err := os.Stat(destPath)
		if err != nil {
			log.Printf("UploadImages: stat error for %s: %v", destPath, err)
			continue
		}

		var uploadedBy *uint
		if user, ok := r.Context().Value(UserContextKey).(*models.User); ok && user != nil {
			uploadedBy = &user.ID
		}
		if created, err := h.ImageRepo.EnsureExistsWithUploader(relDBKey, info.ModTime().Unix(), uploadedBy); err != nil {
			log.Printf("UploadImages: EnsureExists error for %s: %v", relDBKey, err)
		} else if created && h.TagRepo != nil {
			if tagErr := h.TagRepo.ApplyAlbumDefaultTags(relDBKey, uint(albumID)); tagErr != nil {
				log.Printf("UploadImages: ERROR applying album default tags for %s: %v", relDBKey, tagErr)
			}
		}
		baseJob := workers.ImageJob{OriginalImagePath: destPath, OriginalRelativePath: relDBKey, ModTimeUnix: info.ModTime().Unix()}
		for _, task := range []string{workers.TaskThumbnail, workers.TaskMetadata, workers.TaskDetection} {
			job := baseJob
			job.TaskType = task
			h.ImgProc.QueueJob(job)
		}

		saved++
	}

	WriteAPIResponse(w, http.StatusCreated, map[string]any{"uploaded": saved, "failed": failed})
}

// ListAlbumImages lists files within the album folder for admin, including metadata and thumbnail info
func (h *AdminAlbumHandler) ListAlbumImages(w http.ResponseWriter, r *http.Request) {
	albumIDStr := chi.URLParam(r, "id")
	albumID, err := strconv.ParseUint(albumIDStr, 10, 64)
	if err != nil {
		WriteAPIError(w, http.StatusBadRequest, "InvalidAlbumID", "Invalid album ID")
		return
	}

	album, err := h.AlbumRepo.GetByID(uint(albumID))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			WriteAPIError(w, http.StatusNotFound, "AlbumNotFound", "Album not found")
		} else {
			log.Printf("Error getting album %d for image listing: %v", albumID, err)
			WriteAPIError(w, http.StatusInternalServerError, "AlbumFetchError", "Failed to retrieve album")
		}
		return
	}

	albumFullPath := filepath.Join(h.Cfg.RootDirectory, album.FolderPath)
	albumFullPath = filepath.Clean(albumFullPath)
	if !strings.HasPrefix(albumFullPath, h.Cfg.RootDirectory) {
		WriteAPIError(w, http.StatusInternalServerError, "AlbumConfigError", "Album configuration error")
		return
	}

	files, totalCount, err := listDirectoryContents(albumFullPath, "/"+album.FolderPath, h.Cfg, h.ImageRepo, h.ImgProc, h.TagRepo, album.ID, album.SortOrder, -1, -1)
	if err != nil {
		if os.IsNotExist(err) {
			WriteAPIError(w, http.StatusNotFound, "FolderNotFound", "Album folder not found on disk: "+album.FolderPath)
		} else if os.IsPermission(err) {
			WriteAPIError(w, http.StatusForbidden, "FolderPermission", "Permission denied accessing album folder")
		} else {
			log.Printf("Error listing contents for album %d (%s): %v", album.ID, album.FolderPath, err)
			WriteAPIError(w, http.StatusInternalServerError, "FolderListError", "Failed to list album contents")
		}
		return
	}

	WriteAPIResponse(w, http.StatusOK, DirectoryListing{Path: "/" + album.FolderPath, Files: files, Total: totalCount, Offset: 0, Limit: len(files), HasMore: false})
}

// DeleteAlbumImage deletes a single image file within an album and removes DB records and generated assets
func (h *AdminAlbumHandler) DeleteAlbumImage(w http.ResponseWriter, r *http.Request) {
	albumIDStr := chi.URLParam(r, "id")
	albumID, err := strconv.ParseUint(albumIDStr, 10, 64)
	if err != nil {
		WriteAPIError(w, http.StatusBadRequest, "InvalidAlbumID", "Invalid album ID")
		return
	}

	album, err := h.AlbumRepo.GetByID(uint(albumID))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			WriteAPIError(w, http.StatusNotFound, "AlbumNotFound", "Album not found")
		} else {
			log.Printf("Error getting album %d for image delete: %v", albumID, err)
			WriteAPIError(w, http.StatusInternalServerError, "AlbumFetchError", "Failed to retrieve album")
		}
		return
	}

	relPath := r.URL.Query().Get("path")
	if relPath == "" {
		WriteAPIError(w, http.StatusBadRequest, "MissingPath", "Missing 'path' query parameter")
		return
	}
	// Normalize to forward slashes and strip any leading slash
	relPath = filepath.ToSlash(strings.TrimPrefix(relPath, "/"))
	// Security: ensure the path is under the album folder
	if !(relPath == album.FolderPath || strings.HasPrefix(relPath, album.FolderPath+"/")) {
		WriteAPIError(w, http.StatusForbidden, "PathOutsideAlbum", "Image path is not within the specified album")
		return
	}

	// Try to get image DB record to find generated asset paths
	var existingThumbPath, existingPreviewPath *string
	if img, err := h.ImageRepo.GetByPath(relPath); err == nil && img != nil {
		existingThumbPath = img.ThumbnailPath
		existingPreviewPath = img.PreviewPath
	}

	// Delete the original file from disk
	fullPath := filepath.Join(h.Cfg.RootDirectory, relPath)
	if err := os.Remove(fullPath); err != nil && !os.IsNotExist(err) {
		log.Printf("Error deleting original image '%s': %v", fullPath, err)
		WriteAPIError(w, http.StatusInternalServerError, "DeleteError", "Failed to delete original image")
		return
	}

	// Best-effort delete of generated thumbnail and preview assets if known
	if existingThumbPath != nil && *existingThumbPath != "" {
		thumbFull := filepath.Join(h.Cfg.MediaStoragePath, filepath.FromSlash(*existingThumbPath))
		if err := os.Remove(thumbFull); err != nil && !os.IsNotExist(err) {
			log.Printf("Warning: failed to delete thumbnail asset '%s': %v", thumbFull, err)
		}
	}
	if existingPreviewPath != nil && *existingPreviewPath != "" {
		previewFull := filepath.Join(h.Cfg.MediaStoragePath, filepath.FromSlash(*existingPreviewPath))
		if err := os.Remove(previewFull); err != nil && !os.IsNotExist(err) {
			log.Printf("Warning: failed to delete preview asset '%s': %v", previewFull, err)
		}
	}

	// Delete DB records: image row, faces and embeddings for this image using GORM and a transaction
	if repo, ok := h.ImageRepo.(*repository.ImageRepository); ok && repo != nil {
		if err := repo.DB.Transaction(func(tx *gorm.DB) error {
			var faceIDs []uint
			if err := tx.Model(&models.Face{}).Where("image_path = ?", relPath).Pluck("id", &faceIDs).Error; err != nil {
				return err
			}
			if len(faceIDs) > 0 {
				if err := tx.Where("face_id IN ?", faceIDs).Delete(&models.FaceEmbedding{}).Error; err != nil {
					return err
				}
			}
			if err := tx.Where("image_path = ?", relPath).Delete(&models.Face{}).Error; err != nil {
				return err
			}
			if err := tx.Where("original_path = ?", relPath).Delete(&models.Image{}).Error; err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
			return nil
		}); err != nil {
			log.Printf("Error deleting image and related records for %s: %v", relPath, err)
			WriteAPIError(w, http.StatusInternalServerError, "DeleteRecordError", "Failed to delete image record")
			return
		}
	} else {
		if err := h.ImageRepo.Delete(relPath); err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			log.Printf("Error deleting image DB record %s: %v", relPath, err)
			WriteAPIError(w, http.StatusInternalServerError, "DeleteRecordError", "Failed to delete image record")
			return
		}
	}

	w.WriteHeader(http.StatusNoContent)
}
