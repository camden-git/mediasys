package handlers

import (
	"encoding/json"
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
	Success   bool   `json:"success"`
	Message   string `json:"message"`
	ImagePath string `json:"image_path"`
	Queued    bool   `json:"queued"`
	JobID     string `json:"job_id,omitempty"`
	Error     string `json:"error,omitempty"`
}

// QueueFaceDetection re-runs face detection for a specific image
func (dh *DebugHandler) QueueFaceDetection(w http.ResponseWriter, r *http.Request) {
	dbPath := strings.TrimPrefix(r.URL.Query().Get("path"), "/")
	if dbPath == "" {
		http.Error(w, "Missing 'path' query parameter", http.StatusBadRequest)
		return
	}

	response := QueueDetectionResponse{ImagePath: dbPath}
	if !dh.Cfg.FaceRecognitionEnabled {
		response.Message = "Face recognition is disabled"
		dh.sendJSONResponse(w, response, http.StatusConflict)
		return
	}
	if err := dh.ImageRepo.RequeueTask(dbPath, "detection_status"); err != nil {
		response.Message = "Image not found"
		response.Error = err.Error()
		dh.sendJSONResponse(w, response, http.StatusNotFound)
		return
	}
	dh.ImageProcessor.Wake()

	response.Success = true
	response.Queued = true
	response.Message = "Face detection task queued successfully"
	response.JobID = fmt.Sprintf("%s:%s", dbPath, workers.TaskDetection)
	log.Printf("Debug API: Queued face detection for %s", dbPath)
	dh.sendJSONResponse(w, response, http.StatusOK)
}

// GetDetectionStatus returns the current detection status for an image
func (dh *DebugHandler) GetDetectionStatus(w http.ResponseWriter, r *http.Request) {
	relativePath := r.URL.Query().Get("path")
	if relativePath == "" {
		http.Error(w, "Missing 'path' query parameter", http.StatusBadRequest)
		return
	}

	dbPath := strings.TrimPrefix(relativePath, "/")

	// Get image record
	image, err := dh.ImageRepo.GetByPath(dbPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("Image not found: %v", err), http.StatusNotFound)
		return
	}

	statusResponse := map[string]interface{}{
		"image_path":             dbPath,
		"detection_status":       image.DetectionStatus,
		"detection_processed_at": image.DetectionProcessedAt,
		"detection_error":        image.DetectionError,
		"last_modified":          image.LastModified,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(statusResponse)
}

// sendJSONResponse sends a JSON response with the given status code
func (dh *DebugHandler) sendJSONResponse(w http.ResponseWriter, response QueueDetectionResponse, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(response)
}
