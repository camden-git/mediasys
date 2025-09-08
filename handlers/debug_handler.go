package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

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

// QueueFaceDetection queues a face detection task for a specific image
func (dh *DebugHandler) QueueFaceDetection(w http.ResponseWriter, r *http.Request) {
	relativePath := r.URL.Query().Get("path")
	if relativePath == "" {
		http.Error(w, "Missing 'path' query parameter", http.StatusBadRequest)
		return
	}

	decodedPath, err := url.QueryUnescape(relativePath)
	if err != nil {
		http.Error(w, "Invalid URL encoding for path parameter", http.StatusBadRequest)
		return
	}

	cleanRelativePath := filepath.Clean(decodedPath)
	if filepath.IsAbs(cleanRelativePath) || strings.HasPrefix(cleanRelativePath, "..") {
		http.Error(w, "Invalid path: must be relative, no '..'", http.StatusBadRequest)
		return
	}

	dbPath := filepath.ToSlash(cleanRelativePath)
	fullPath := filepath.Join(dh.Cfg.RootDirectory, dbPath)

	response := QueueDetectionResponse{
		Success:   false,
		ImagePath: dbPath,
	}

	// Check if file exists
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		response.Message = "Image file not found"
		response.Error = fmt.Sprintf("File does not exist: %s", fullPath)
		dh.sendJSONResponse(w, response, http.StatusNotFound)
		return
	} else if err != nil {
		response.Message = "Error checking file"
		response.Error = fmt.Sprintf("Failed to stat file: %v", err)
		dh.sendJSONResponse(w, response, http.StatusInternalServerError)
		return
	}

	// Get file modification time
	fileInfo, err := os.Stat(fullPath)
	if err != nil {
		response.Message = "Error getting file info"
		response.Error = fmt.Sprintf("Failed to get file info: %v", err)
		dh.sendJSONResponse(w, response, http.StatusInternalServerError)
		return
	}

	modTimeUnix := fileInfo.ModTime().Unix()

	// Ensure image record exists in database
	exists, err := dh.ImageRepo.EnsureExists(dbPath, modTimeUnix)
	if err != nil {
		response.Message = "Database error"
		response.Error = fmt.Sprintf("Failed to ensure image record exists: %v", err)
		dh.sendJSONResponse(w, response, http.StatusInternalServerError)
		return
	}

	if !exists {
		log.Printf("Created new image record for: %s", dbPath)
	}

	// Create detection job
	detectionJob := workers.ImageJob{
		OriginalImagePath:    fullPath,
		OriginalRelativePath: dbPath,
		ModTimeUnix:          modTimeUnix,
		TaskType:             workers.TaskDetection,
	}

	// Queue the job
	queued := dh.ImageProcessor.QueueJob(detectionJob)
	if queued {
		response.Success = true
		response.Message = "Face detection task queued successfully"
		response.Queued = true
		response.JobID = fmt.Sprintf("%s:%s", dbPath, workers.TaskDetection)

		log.Printf("Debug API: Queued face detection for %s", dbPath)
	} else {
		response.Success = false
		response.Message = "Failed to queue face detection task"
		response.Queued = false
		response.Error = "Job queue is full or task already pending"

		log.Printf("Debug API: Failed to queue face detection for %s", dbPath)
	}

	dh.sendJSONResponse(w, response, http.StatusOK)
}

// GetDetectionStatus returns the current detection status for an image
func (dh *DebugHandler) GetDetectionStatus(w http.ResponseWriter, r *http.Request) {
	relativePath := r.URL.Query().Get("path")
	if relativePath == "" {
		http.Error(w, "Missing 'path' query parameter", http.StatusBadRequest)
		return
	}

	decodedPath, err := url.QueryUnescape(relativePath)
	if err != nil {
		http.Error(w, "Invalid URL encoding for path parameter", http.StatusBadRequest)
		return
	}

	cleanRelativePath := filepath.Clean(decodedPath)
	if filepath.IsAbs(cleanRelativePath) || strings.HasPrefix(cleanRelativePath, "..") {
		http.Error(w, "Invalid path: must be relative, no '..'", http.StatusBadRequest)
		return
	}

	dbPath := filepath.ToSlash(cleanRelativePath)

	// Get image record
	image, err := dh.ImageRepo.GetByPath(dbPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("Image not found: %v", err), http.StatusNotFound)
		return
	}

	statusResponse := map[string]interface{}{
		"image_path":              dbPath,
		"detection_status":        image.DetectionStatus,
		"detection_processed_at":  image.DetectionProcessedAt,
		"detection_error":         image.DetectionError,
		"last_modified":           image.LastModified,
		"last_modified_formatted": time.Unix(image.LastModified, 0).Format(time.RFC3339),
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
