package handlers

import (
	"database/sql"
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
	"github.com/camden-git/mediasysbackend/database"
	"github.com/go-chi/chi/v5"
)

type FaceHandler struct {
	DB  *sql.DB
	Cfg config.Config
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

	if req.PersonID != nil {
		if *req.PersonID <= 0 {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid person_id value"})
			return
		}
		if _, err := database.GetPersonByID(fh.DB, *req.PersonID); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Person with provided person_id not found"})
			} else {
				log.Printf("Error checking person %d before adding face: %v", *req.PersonID, err)
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

	faceID, err := database.AddFace(fh.DB, req.PersonID, imagePathForDB, req.X1, req.Y1, req.X2, req.Y2)
	if err != nil {
		log.Printf("Error adding face (person: %v) to image %s: %v", req.PersonID, imagePathForDB, err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to add face tag"})
		return
	}

	newFace, err := database.GetFaceByID(fh.DB, faceID)
	if err != nil {
		log.Printf("Error fetching newly created face %d: %v", faceID, err)
		writeJSON(w, http.StatusCreated, map[string]interface{}{"message": "Face added successfully", "id": faceID})
		return
	}
	writeJSON(w, http.StatusCreated, newFace)
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
	faces, err := database.ListFacesByImagePath(fh.DB, imagePathForDB)
	if err != nil {
		log.Printf("Error listing faces for image %s: %v", imagePathForDB, err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to retrieve faces for image"})
		return
	}
	if faces == nil {
		faces = []database.Face{}
	}
	writeJSON(w, http.StatusOK, faces)
}

func (fh *FaceHandler) GetFace(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "face_id")
	faceID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid face ID format"})
		return
	}
	face, err := database.GetFaceByID(fh.DB, faceID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
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
	faceID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid face ID format"})
		return
	}

	if _, err := database.GetFaceByID(fh.DB, faceID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
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

	var personIDUpdate *int64
	personIDProvided := false

	if pidVal, ok := reqMap["person_id"]; ok {
		personIDProvided = true
		if pidVal == nil {

		} else if pidFloat, ok := pidVal.(float64); ok {
			pidInt := int64(pidFloat)
			if pidInt > 0 {
				personIDUpdate = &pidInt
			} else {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid value for person_id"})
				return
			}
		} else {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid type for person_id"})
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
		if _, err := database.GetPersonByID(fh.DB, *personIDUpdate); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Person with provided person_id not found"})
			} else {
				log.Printf("Error checking person %d before updating face %d: %v", *personIDUpdate, faceID, err)
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to verify person"})
			}
			return
		}
	}

	var finalPersonID *int64 = nil
	if personIDProvided {
		finalPersonID = personIDUpdate
	}

	err = database.UpdateFace(fh.DB, faceID, finalPersonID, x1Update, y1Update, x2Update, y2Update)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "Face tag not found during update"})
		} else {
			log.Printf("Error updating face %d: %v", faceID, err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to update face tag"})
		}
		return
	}

	updatedFace, err := database.GetFaceByID(fh.DB, faceID)
	if err != nil {
		log.Printf("Error fetching updated face %d: %v", faceID, err)
		writeJSON(w, http.StatusOK, map[string]string{"message": "Face updated successfully"})
		return
	}
	writeJSON(w, http.StatusOK, updatedFace)
}

func (fh *FaceHandler) DeleteFace(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "face_id")
	faceID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid face ID format"})
		return
	}
	err = database.DeleteFace(fh.DB, faceID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
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
	personIDs, err := database.FindPersonIDsByNameOrAlias(fh.DB, query)
	if err != nil {
		log.Printf("Error searching for person IDs with query '%s': %v", query, err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to search for people"})
		return
	}
	if len(personIDs) == 0 {
		writeJSON(w, http.StatusOK, []string{})
		return
	}
	imagePaths, err := database.FindImagesByPersonIDs(fh.DB, personIDs)
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
