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
	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"
)

type FaceHandler struct {
	FaceRepo   repository.FaceRepositoryInterface
	PersonRepo repository.PersonRepositoryInterface
	Cfg        config.Config
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
		writeJSON(w, http.StatusOK, []string{}) // return empty list of image paths
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
