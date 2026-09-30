package handlers

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/camden-git/mediasysbackend/database"
	"github.com/camden-git/mediasysbackend/media"
	"github.com/camden-git/mediasysbackend/models"
	"github.com/camden-git/mediasysbackend/realtime"
	"github.com/camden-git/mediasysbackend/repository"
	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"
)

var errUploadTooLarge = errors.New("file exceeds upload size limit")

// limitedReader fails once more than n bytes have been read.
type limitedReader struct {
	r io.Reader
	n int64
}

func (l *limitedReader) Read(p []byte) (int, error) {
	n, err := l.r.Read(p)
	l.n -= int64(n)
	if l.n < 0 {
		return n, errUploadTooLarge
	}
	return n, err
}

// cleanUploadPath normalises a client supplied relative path and strips the
// top-level folder that browsers add for directory uploads.
func cleanUploadPath(rel, filename string) (string, bool) {
	if rel == "" {
		rel = filename
	}
	rel = strings.ReplaceAll(rel, "\\", "/")
	if strings.Contains(rel, "/") {
		rel = rel[strings.Index(rel, "/")+1:]
	}
	rel = path.Clean("/" + rel)[1:]
	if rel == "" || rel == "." || strings.HasPrefix(rel, "..") {
		return "", false
	}
	return rel, true
}

// invalidateAlbumZip drops the album's archive after its images changed so a
// stale zip is never served.
func (h *AdminAlbumHandler) invalidateAlbumZip(albumID uint) {
	oldZip, err := h.AlbumRepo.InvalidateZip(albumID)
	if err != nil {
		log.Printf("Error invalidating zip for album %d: %v", albumID, err)
		return
	}
	if oldZip != nil && *oldZip != "" {
		_ = h.Store.Delete(context.Background(), *oldZip)
	}
}

// UploadImages streams multipart uploads straight into object storage and queues processing
func (h *AdminAlbumHandler) UploadImages(w http.ResponseWriter, r *http.Request) {
	albumID, err := strconv.ParseUint(chi.URLParam(r, "id"), 10, 64)
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

	reader, err := r.MultipartReader()
	if err != nil {
		WriteAPIError(w, http.StatusBadRequest, "InvalidMultipart", "Invalid multipart form: "+err.Error())
		return
	}

	var uploadedBy *uint
	if user, ok := r.Context().Value(UserContextKey).(*models.User); ok && user != nil {
		uploadedBy = &user.ID
	}

	broadcast := func(p, status, errMsg string) {
		if h.Hub != nil {
			h.Hub.Broadcast(realtime.Event{Type: "upload", AlbumID: album.ID, Path: p, Status: status, Error: errMsg, Timestamp: time.Now().Unix()})
		}
	}

	folder := strings.Trim(album.FolderPath, "/")
	// relative_path applies only to the file part that immediately follows it
	pendingRel, hasPendingRel := "", false
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

		switch part.FormName() {
		case "relative_path":
			if hasPendingRel {
				WriteAPIError(w, http.StatusBadRequest, "MalformedUpload", "relative_path must be immediately followed by its file")
				return
			}
			data, _ := io.ReadAll(io.LimitReader(part, 4096))
			pendingRel, hasPendingRel = strings.TrimSpace(string(data)), true
			continue
		case "files":
		default:
			hasPendingRel = false
			continue
		}

		filename := part.FileName()
		rawRel := ""
		if hasPendingRel {
			rawRel, hasPendingRel = pendingRel, false
		}
		rel, ok := cleanUploadPath(rawRel, filename)
		if !ok {
			failed = append(failed, map[string]string{"path": rawRel, "error": "invalid path"})
			continue
		}
		if !media.IsRasterImage(filename) {
			failed = append(failed, map[string]string{"path": rel, "error": "unsupported file type"})
			continue
		}

		imagePath := folder + "/" + rel
		owner, lookupErr := h.ImageRepo.GetByPath(imagePath)
		if lookupErr == nil && owner.AlbumID != album.ID {
			failed = append(failed, map[string]string{"path": rel, "error": "path belongs to another album"})
			continue
		}
		if lookupErr != nil && !errors.Is(lookupErr, gorm.ErrRecordNotFound) {
			log.Printf("UploadImages: failed to look up %s: %v", imagePath, lookupErr)
			failed = append(failed, map[string]string{"path": rel, "error": "failed to save image record"})
			continue
		}
		objectKey := media.OriginalKey(imagePath)
		broadcast(imagePath, "uploading", "")

		size, putErr := h.Store.Put(r.Context(), objectKey, &limitedReader{r: part, n: h.Cfg.MaxUploadSize}, -1, media.ContentTypeFor(filename))
		if putErr != nil {
			msg := putErr.Error()
			if errors.Is(putErr, errUploadTooLarge) {
				msg = fmt.Sprintf("file exceeds %d MB limit", h.Cfg.MaxUploadSize>>20)
			}
			log.Printf("UploadImages: failed to store %s: %v", imagePath, putErr)
			broadcast(imagePath, "error", msg)
			failed = append(failed, map[string]string{"path": rel, "error": msg})
			continue
		}

		now := time.Now().Unix()
		detectionStatus := database.StatusPending
		if !h.Cfg.FaceRecognitionEnabled {
			detectionStatus = database.StatusNotRequired
		}
		img := &models.Image{
			OriginalPath:     imagePath,
			AlbumID:          album.ID,
			ObjectKey:        objectKey,
			Size:             size,
			ContentType:      media.ContentTypeFor(filename),
			LastModified:     now,
			CreatedAt:        now,
			UploadedByUserID: uploadedBy,
			MetadataStatus:   database.StatusPending,
			ThumbnailStatus:  database.StatusPending,
			PreviewStatus:    database.StatusPending,
			DetectionStatus:  detectionStatus,
		}
		previous, dbErr := h.ImageRepo.Upsert(img)
		if errors.Is(dbErr, repository.ErrPathOwnedByOtherAlbum) {
			// lost a race with another album; its original must stay untouched
			failed = append(failed, map[string]string{"path": rel, "error": "path belongs to another album"})
			continue
		}
		if dbErr != nil {
			log.Printf("UploadImages: failed to record %s: %v", imagePath, dbErr)
			_ = h.Store.Delete(context.Background(), objectKey)
			broadcast(imagePath, "error", "failed to save image record")
			failed = append(failed, map[string]string{"path": rel, "error": "failed to save image record"})
			continue
		}
		if previous != nil {
			// replaced an existing file: drop its old generated assets
			var stale []string
			for _, k := range repository.ImageObjectKeys([]models.Image{*previous}) {
				if k != objectKey {
					stale = append(stale, k)
				}
			}
			h.Store.DeleteKeys(context.Background(), stale)
		}
		if h.TagRepo != nil {
			if tagErr := h.TagRepo.ApplyAlbumDefaultTags(imagePath, album.ID); tagErr != nil {
				log.Printf("UploadImages: failed to apply default tags for %s: %v", imagePath, tagErr)
			}
		}

		broadcast(imagePath, "uploaded", "")
		h.ImgProc.Wake()
		saved++
	}
	if saved > 0 {
		h.invalidateAlbumZip(album.ID)
	}

	WriteAPIResponse(w, http.StatusCreated, map[string]any{"uploaded": saved, "failed": failed})
}

// ListAlbumImages lists every image in an album for the admin view
func (h *AdminAlbumHandler) ListAlbumImages(w http.ResponseWriter, r *http.Request) {
	albumID, err := strconv.ParseUint(chi.URLParam(r, "id"), 10, 64)
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

	sortOrder := album.SortOrder
	if !database.IsValidSortOrder(sortOrder) {
		sortOrder = database.DefaultSortOrder
	}

	// the admin view is not paginated: a negative limit returns every image
	images, total, err := h.ImageRepo.ListByAlbumPaged(album.ID, nil, sortOrder, 0, -1)
	if err != nil {
		log.Printf("Error listing images for album %d: %v", album.ID, err)
		WriteAPIError(w, http.StatusInternalServerError, "ImageListError", "Failed to list album contents")
		return
	}

	files := make([]FileInfo, 0, len(images))
	for i := range images {
		files = append(files, imageToFileInfo(&images[i]))
	}
	WriteAPIResponse(w, http.StatusOK, DirectoryListing{Path: "/" + album.FolderPath, Files: files, Total: total})
}

// DeleteAlbumImage deletes a single image, its related records and all of its stored objects
func (h *AdminAlbumHandler) DeleteAlbumImage(w http.ResponseWriter, r *http.Request) {
	albumID, err := strconv.ParseUint(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		WriteAPIError(w, http.StatusBadRequest, "InvalidAlbumID", "Invalid album ID")
		return
	}

	relPath := strings.TrimPrefix(r.URL.Query().Get("path"), "/")
	if relPath == "" {
		WriteAPIError(w, http.StatusBadRequest, "MissingPath", "Missing 'path' query parameter")
		return
	}

	img, err := h.ImageRepo.GetByPath(relPath)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			WriteAPIError(w, http.StatusNotFound, "ImageNotFound", "Image not found")
		} else {
			log.Printf("Error loading image %s for delete: %v", relPath, err)
			WriteAPIError(w, http.StatusInternalServerError, "ImageFetchError", "Failed to load image")
		}
		return
	}
	if img.AlbumID != uint(albumID) {
		WriteAPIError(w, http.StatusForbidden, "PathOutsideAlbum", "Image is not within the specified album")
		return
	}

	keys, err := h.ImageRepo.DeleteImages([]string{relPath})
	if err != nil {
		log.Printf("Error deleting image %s: %v", relPath, err)
		WriteAPIError(w, http.StatusInternalServerError, "DeleteRecordError", "Failed to delete image record")
		return
	}
	h.Store.DeleteKeys(context.Background(), keys)
	h.invalidateAlbumZip(img.AlbumID)

	w.WriteHeader(http.StatusNoContent)
}
