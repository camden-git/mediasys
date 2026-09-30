package handlers

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/camden-git/mediasysbackend/config"
	"github.com/camden-git/mediasysbackend/repository"
	"github.com/camden-git/mediasysbackend/workers"
)

type DebugHandler struct {
	Cfg            config.Config
	ImageRepo      repository.ImageRepositoryInterface
	ImageProcessor *workers.ImageProcessor
}

type QueueDetectionResponse struct {
	ImagePath string `json:"image_path"`
	Queued    bool   `json:"queued"`
	JobID     string `json:"job_id"`
}

// QueueFaceDetection re-runs face detection for a specific image
func (dh *DebugHandler) QueueFaceDetection(w http.ResponseWriter, r *http.Request) {
	dbPath := strings.TrimPrefix(r.URL.Query().Get("path"), "/")
	if dbPath == "" {
		WriteAPIError(w, http.StatusBadRequest, "ValidationError", "Missing 'path' query parameter")
		return
	}

	if !dh.Cfg.FaceRecognitionEnabled {
		WriteAPIError(w, http.StatusConflict, "FaceRecognitionDisabled", "Face recognition is disabled")
		return
	}
	if err := dh.ImageRepo.RequeueTask(dbPath, "detection_status"); err != nil {
		WriteAPIError(w, http.StatusNotFound, "ImageNotFound", "Image not found: "+err.Error())
		return
	}
	dh.ImageProcessor.Wake()

	log.Printf("Debug API: Queued face detection for %s", dbPath)
	WriteAPIResponse(w, http.StatusOK, QueueDetectionResponse{
		ImagePath: dbPath,
		Queued:    true,
		JobID:     fmt.Sprintf("%s:%s", dbPath, workers.TaskDetection),
	})
}

// GetDetectionStatus returns the current detection status for an image
func (dh *DebugHandler) GetDetectionStatus(w http.ResponseWriter, r *http.Request) {
	relativePath := r.URL.Query().Get("path")
	if relativePath == "" {
		WriteAPIError(w, http.StatusBadRequest, "ValidationError", "Missing 'path' query parameter")
		return
	}

	dbPath := strings.TrimPrefix(relativePath, "/")

	// Get image record
	image, err := dh.ImageRepo.GetByPath(dbPath)
	if err != nil {
		WriteAPIError(w, http.StatusNotFound, "ImageNotFound", fmt.Sprintf("Image not found: %v", err))
		return
	}

	statusResponse := map[string]interface{}{
		"image_path":             dbPath,
		"detection_status":       image.DetectionStatus,
		"detection_processed_at": image.DetectionProcessedAt,
		"detection_error":        image.DetectionError,
		"last_modified":          image.LastModified,
	}

	WriteAPIResponse(w, http.StatusOK, statusResponse)
}
