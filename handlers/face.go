package handlers

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/camden-git/mediasysbackend/config"
	"github.com/camden-git/mediasysbackend/models"
	"github.com/camden-git/mediasysbackend/repository"
	"github.com/camden-git/mediasysbackend/services"
	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"
)

type FaceHandler struct {
	FaceRepo               repository.FaceRepositoryInterface
	PersonRepo             repository.PersonRepositoryInterface
	Cfg                    config.Config
	FaceRecognitionService *services.FaceRecognitionService
}

func (fh *FaceHandler) AddFace(w http.ResponseWriter, r *http.Request) {
	var req struct {
		PersonID  *int64 `json:"person_id"`
		ImagePath string `json:"image_path"`
		X1        int    `json:"x1"`
		Y1        int    `json:"y1"`
		X2        int    `json:"x2"`
		Y2        int    `json:"y2"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request body: " + err.Error()})
		return
	}

	if req.ImagePath == "" || req.X1 < 0 || req.Y1 < 0 || req.X2 <= req.X1 || req.Y2 <= req.Y1 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Missing or invalid required fields (image_path, coordinates)"})
		return
	}

	var personIDUint *uint
	if req.PersonID != nil {
		if *req.PersonID <= 0 {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid person_id value"})
			return
		}
		pid := uint(*req.PersonID)
		personIDUint = &pid
		if _, err := fh.PersonRepo.GetByID(*personIDUint); err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Person with provided person_id not found"})
			} else {
				log.Printf("Error checking person %d before adding face: %v", *personIDUint, err)
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to verify person"})
			}
			return
		}
	}

	cleanRelativePath := filepath.Clean(req.ImagePath)
	if filepath.IsAbs(cleanRelativePath) || strings.HasPrefix(cleanRelativePath, "..") {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "image_path must be relative and cannot use '..'"})
		return
	}
	imagePathForDB := filepath.ToSlash(cleanRelativePath)
	fullImagePath := filepath.Join(fh.Cfg.RootDirectory, imagePathForDB)
	if _, err := os.Stat(fullImagePath); os.IsNotExist(err) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "image_path does not exist: " + imagePathForDB})
		return
	} else if err != nil {
		log.Printf("Error stating image path %s during face add: %v", fullImagePath, err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Could not verify image_path"})
		return
	}

	face := models.Face{
		PersonID:  personIDUint,
		ImagePath: imagePathForDB,
		X1:        req.X1,
		Y1:        req.Y1,
		X2:        req.X2,
		Y2:        req.Y2,
	}
	createErr := fh.FaceRepo.Create(&face)
	if createErr != nil {
		log.Printf("Error adding face (person: %v) to image %s: %v", req.PersonID, imagePathForDB, createErr)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to add face tag"})
		return
	}

	createdFace, fetchErr := fh.FaceRepo.GetByID(face.ID)
	if fetchErr != nil {
		log.Printf("Error fetching newly created face %d: %v", face.ID, fetchErr)
		writeJSON(w, http.StatusCreated, face)
		return
	}
	writeJSON(w, http.StatusCreated, createdFace)
}

func (fh *FaceHandler) ListFacesByImage(w http.ResponseWriter, r *http.Request) {
	imageQueryParam := r.URL.Query().Get("path")
	if imageQueryParam == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Missing required query parameter: path"})
		return
	}
	imagePath, err := url.QueryUnescape(imageQueryParam)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid URL encoding for path parameter"})
		return
	}
	cleanRelativePath := filepath.Clean(imagePath)
	if filepath.IsAbs(cleanRelativePath) || strings.HasPrefix(cleanRelativePath, "..") {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "image_path must be relative and cannot use '..'"})
		return
	}
	imagePathForDB := filepath.ToSlash(cleanRelativePath)
	faces, err := fh.FaceRepo.ListByImagePath(imagePathForDB)
	if err != nil {
		log.Printf("Error listing faces for image %s: %v", imagePathForDB, err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to retrieve faces for image"})
		return
	}
	if faces == nil {
		faces = []models.Face{}
	}
	writeJSON(w, http.StatusOK, faces)
}

func (fh *FaceHandler) GetFace(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "face_id")
	faceID, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid face ID format"})
		return
	}
	face, err := fh.FaceRepo.GetByID(uint(faceID))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "Face tag not found"})
		} else {
			log.Printf("Error getting face %d: %v", faceID, err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to retrieve face tag"})
		}
		return
	}
	writeJSON(w, http.StatusOK, face)
}

func (fh *FaceHandler) UpdateFace(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "face_id")
	faceID, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid face ID format"})
		return
	}

	// check if face exists first
	if _, err := fh.FaceRepo.GetByID(uint(faceID)); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "Face tag not found"})
		} else {
			log.Printf("Error finding face %d for update: %v", faceID, err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to find face tag for update"})
		}
		return
	}

	var reqMap map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&reqMap); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request body: " + err.Error()})
		return
	}

	var personIDUpdate *uint
	personIDProvided := false

	if pidVal, ok := reqMap["person_id"]; ok {
		personIDProvided = true
		if pidVal == nil { // explicitly un-tagging
			// personIDUpdate remains nil
		} else if pidFloat, ok := pidVal.(float64); ok {
			pidUint := uint(pidFloat)
			if pidUint > 0 {
				personIDUpdate = &pidUint
			} else { // person_id: 0 is not valid for tagging
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid non-zero value for person_id"})
				return
			}
		} else {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid type for person_id, expected number or null"})
			return
		}
	}

	var x1Update, y1Update, x2Update, y2Update *int
	if v, ok := reqMap["x1"].(float64); ok {
		i := int(v)
		x1Update = &i
	}
	if v, ok := reqMap["y1"].(float64); ok {
		i := int(v)
		y1Update = &i
	}
	if v, ok := reqMap["x2"].(float64); ok {
		i := int(v)
		x2Update = &i
	}
	if v, ok := reqMap["y2"].(float64); ok {
		i := int(v)
		y2Update = &i
	}

	if personIDProvided && personIDUpdate != nil {
		// *personIDUpdate is uint here because personIDUpdate is *uint
		if _, err := fh.PersonRepo.GetByID(*personIDUpdate); err != nil { // Use PersonRepo
			if errors.Is(err, gorm.ErrRecordNotFound) {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Person with provided person_id not found"})
			} else {
				log.Printf("Error checking person %d before updating face %d: %v", *personIDUpdate, faceID, err)
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to verify person"})
			}
			return
		}
	}

	updateErr := fh.FaceRepo.Update(uint(faceID), personIDUpdate, x1Update, y1Update, x2Update, y2Update)
	if updateErr != nil {
		if errors.Is(updateErr, gorm.ErrRecordNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "Face tag not found during update"})
		} else {
			log.Printf("Error updating face %d: %v", faceID, err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to update face tag"})
		}
		return
	}

	updatedFace, err := fh.FaceRepo.GetByID(uint(faceID))
	if err != nil {
		log.Printf("Error fetching updated face %d: %v", faceID, err)
		// still, the update was successful at DB level
		writeJSON(w, http.StatusOK, map[string]string{"message": "Face updated successfully"})
		return
	}
	writeJSON(w, http.StatusOK, updatedFace)
}

func (fh *FaceHandler) DeleteFace(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "face_id")
	faceID, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid face ID format"})
		return
	}
	err = fh.FaceRepo.Delete(uint(faceID))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "Face tag not found"})
		} else {
			log.Printf("Error deleting face %d: %v", faceID, err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to delete face tag"})
		}
		return
	}
	writeJSON(w, http.StatusNoContent, nil)
}

func (fh *FaceHandler) SearchFacesByPerson(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("query")
	if strings.TrimSpace(query) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Missing required query parameter: query"})
		return
	}

	personIDs, err := fh.PersonRepo.FindPersonIDsByNameOrAlias(query)
	if err != nil {
		log.Printf("Error searching for person IDs with query '%s': %v", query, err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to search for people"})
		return
	}
	if len(personIDs) == 0 {
		writeJSON(w, http.StatusOK, []string{}) // return an empty list of image paths
		return
	}
	imagePaths, err := fh.PersonRepo.FindImagesByPersonIDs(personIDs)
	if err != nil {
		log.Printf("Error finding images for person IDs %v: %v", personIDs, err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to find images associated with person"})
		return
	}
	if imagePaths == nil {
		imagePaths = []string{}
	}
	writeJSON(w, http.StatusOK, imagePaths)
}

// GetSimilarFaces finds faces similar to a given face ID
func (fh *FaceHandler) GetSimilarFaces(w http.ResponseWriter, r *http.Request) {
	if fh.FaceRecognitionService == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "Face recognition service not available"})
		return
	}

	idStr := chi.URLParam(r, "face_id")
	faceID, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid face ID format"})
		return
	}

	// Get limit from query parameter, default to 10
	limitStr := r.URL.Query().Get("limit")
	limit := 10
	if limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	similarFaces, err := fh.FaceRecognitionService.FindSimilarFaces(uint(faceID), limit)
	if err != nil {
		log.Printf("Error finding similar faces for face %d: %v", faceID, err)

		// Check if the error is due to missing face embedding
		if strings.Contains(err.Error(), "failed to get target face embedding") {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "Face does not have an embedding. Face recognition requires embeddings to be generated for faces."})
		} else {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to find similar faces"})
		}
		return
	}

	writeJSON(w, http.StatusOK, similarFaces)
}

// GetUntaggedFaces returns untagged faces with person suggestions
func (fh *FaceHandler) GetUntaggedFaces(w http.ResponseWriter, r *http.Request) {
	if fh.FaceRecognitionService == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "Face recognition service not available"})
		return
	}

	// Get limit from query parameter, default to 20
	limitStr := r.URL.Query().Get("limit")
	limit := 20
	if limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	untaggedFaces, err := fh.FaceRecognitionService.GetUntaggedFacesWithSuggestions(limit)
	if err != nil {
		log.Printf("Error getting untagged faces with suggestions: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to get untagged faces"})
		return
	}

	writeJSON(w, http.StatusOK, untaggedFaces)
}

// TagFace tags a face with a person and optionally auto-tags similar faces
func (fh *FaceHandler) TagFace(w http.ResponseWriter, r *http.Request) {
	if fh.FaceRecognitionService == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "Face recognition service not available"})
		return
	}

	idStr := chi.URLParam(r, "face_id")
	faceID, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid face ID format"})
		return
	}

	var req struct {
		PersonID uint `json:"person_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request body: " + err.Error()})
		return
	}

	if req.PersonID == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "person_id is required and must be greater than 0"})
		return
	}

	// Verify person exists
	if _, err := fh.PersonRepo.GetByID(req.PersonID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Person not found"})
		} else {
			log.Printf("Error verifying person %d: %v", req.PersonID, err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to verify person"})
		}
		return
	}

	// Tag the face with auto-tagging of similar faces
	err = fh.FaceRecognitionService.TagFaceWithPerson(uint(faceID), req.PersonID)
	if err != nil {
		log.Printf("Error tagging face %d with person %d: %v", faceID, req.PersonID, err)

		// Check if the error is due to missing face embedding
		if strings.Contains(err.Error(), "failed to get target face embedding") {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "Face does not have an embedding. Face recognition requires embeddings to be generated for faces."})
		} else {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to tag face"})
		}
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "Face tagged successfully"})
}

// AutoTagFace automatically tags a face based on similar faces
func (fh *FaceHandler) AutoTagFace(w http.ResponseWriter, r *http.Request) {
	if fh.FaceRecognitionService == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "Face recognition service not available"})
		return
	}

	idStr := chi.URLParam(r, "face_id")
	faceID, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid face ID format"})
		return
	}

	// Get person suggestion for the face
	personID, personName, confidence, err := fh.FaceRecognitionService.SuggestPersonForFace(uint(faceID))
	if err != nil {
		log.Printf("Error suggesting person for face %d: %v", faceID, err)

		// Check if the error is due to missing face embedding
		if strings.Contains(err.Error(), "failed to get target face embedding") {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "Face does not have an embedding. Face recognition requires embeddings to be generated for faces."})
		} else {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to suggest person for face"})
		}
		return
	}

	if personID == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "No suitable person found for this face"})
		return
	}

	// Tag the face with the suggested person
	err = fh.FaceRecognitionService.TagFaceWithPerson(uint(faceID), *personID)
	if err != nil {
		log.Printf("Error auto-tagging face %d with person %d: %v", faceID, *personID, err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to auto-tag face"})
		return
	}

	response := map[string]interface{}{
		"message":    "Face auto-tagged successfully",
		"person_id":  *personID,
		"confidence": confidence,
	}
	if personName != nil {
		response["person_name"] = *personName
	}

	writeJSON(w, http.StatusOK, response)
}

// DebugFaces returns debug information about faces in the database
func (fh *FaceHandler) DebugFaces(w http.ResponseWriter, r *http.Request) {
	if fh.FaceRecognitionService == nil {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"error": "Face recognition service not available",
		})
		return
	}

	// Get face 485 and its embedding
	face485, err := fh.FaceRepo.GetByID(485)
	if err != nil {
		log.Printf("Error getting face 485: %v", err)
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"error": "Face 485 not found",
		})
		return
	}

	// Get the embedding for face 485
	embedding485, err := fh.FaceRecognitionService.GetEmbeddingRepo().GetByFaceID(485)
	if err != nil {
		log.Printf("Error getting embedding for face 485: %v", err)
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"face_485":        face485,
			"embedding_error": err.Error(),
		})
		return
	}

	// Get embeddings for the similar faces
	embedding481, err := fh.FaceRecognitionService.GetEmbeddingRepo().GetByFaceID(481)
	if err != nil {
		log.Printf("Error getting embedding for face 481: %v", err)
	}

	embedding483, err := fh.FaceRecognitionService.GetEmbeddingRepo().GetByFaceID(483)
	if err != nil {
		log.Printf("Error getting embedding for face 483: %v", err)
	}

	// Calculate similarities manually
	var similarities []map[string]interface{}

	if embedding481 != nil {
		similarity := fh.FaceRecognitionService.CalculateSimilarity(
			embedding485.GetEmbedding(),
			embedding481.GetEmbedding(),
		)
		similarities = append(similarities, map[string]interface{}{
			"face_id":          481,
			"similarity":       similarity,
			"embedding_length": len(embedding481.GetEmbedding()),
		})
	}

	if embedding483 != nil {
		similarity := fh.FaceRecognitionService.CalculateSimilarity(
			embedding485.GetEmbedding(),
			embedding483.GetEmbedding(),
		)
		similarities = append(similarities, map[string]interface{}{
			"face_id":          483,
			"similarity":       similarity,
			"embedding_length": len(embedding483.GetEmbedding()),
		})
	}

	// Check if embeddings are identical
	embedding485Vector := embedding485.GetEmbedding()
	var embedding485Head, embedding481Head []float32
	for i := 0; i < 10 && i < len(embedding485Vector); i++ {
		embedding485Head = append(embedding485Head, embedding485Vector[i])
	}
	var identicalCount int
	if embedding481 != nil {
		embedding481Vector := embedding481.GetEmbedding()
		for i := 0; i < 10 && i < len(embedding481Vector); i++ {
			embedding481Head = append(embedding481Head, embedding481Vector[i])
		}
		if len(embedding485Vector) == len(embedding481Vector) {
			identical := true
			for i := 0; i < len(embedding485Vector); i++ {
				if embedding485Vector[i] != embedding481Vector[i] {
					identical = false
					break
				}
			}
			if identical {
				identicalCount++
			}
		}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"face_485":             face485,
		"embedding_485_length": len(embedding485Vector),
		"embedding_485_head":   embedding485Head,
		"embedding_481_head":   embedding481Head,
		"similarities":         similarities,
		"identical_embeddings": identicalCount,
		"threshold":            fh.FaceRecognitionService.GetSimilarityThreshold(),
	})
}
