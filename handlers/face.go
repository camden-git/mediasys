package handlers

import (
	"encoding/json"
	"errors"
	"log"
	"math"
	"net/http"
	"strconv"
	"strings"

	"github.com/camden-git/mediasysbackend/config"
	"github.com/camden-git/mediasysbackend/media"
	"github.com/camden-git/mediasysbackend/models"
	"github.com/camden-git/mediasysbackend/repository"
	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"
)

// maxFaceListLimit caps the limit query param on the admin face listing endpoints.
const maxFaceListLimit = 100

// validateFaceBox checks a pixel-space box: non-negative, non-empty and, when the image
// dimensions are known, inside the image.
func validateFaceBox(x1, y1, x2, y2 int, img *models.Image) string {
	if x1 < 0 || y1 < 0 {
		return "Coordinates must not be negative"
	}
	if x2 <= x1 || y2 <= y1 {
		return "x2 must be greater than x1 and y2 greater than y1"
	}
	if img != nil && img.Width != nil && img.Height != nil && (x2 > *img.Width || y2 > *img.Height) {
		return "Bounding box lies outside the image"
	}
	return ""
}

type FaceHandler struct {
	FaceRepo               repository.FaceRepositoryInterface
	EmbeddingRepo          repository.FaceEmbeddingRepositoryInterface
	PersonRepo             repository.PersonRepositoryInterface
	ImageRepo              repository.ImageRepositoryInterface
	Store                  *media.Store
	Cfg                    config.Config
	FaceRecognitionService *FaceRecognitionService
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
		WriteAPIError(w, http.StatusBadRequest, "InvalidPayload", "Invalid request body: "+err.Error())
		return
	}

	if req.ImagePath == "" || req.X1 < 0 || req.Y1 < 0 || req.X2 <= req.X1 || req.Y2 <= req.Y1 {
		WriteAPIError(w, http.StatusBadRequest, "ValidationError", "Missing or invalid required fields (image_path, coordinates)")
		return
	}

	var personIDUint *uint
	if req.PersonID != nil {
		if *req.PersonID <= 0 {
			WriteAPIError(w, http.StatusBadRequest, "ValidationError", "Invalid person_id value")
			return
		}
		pid := uint(*req.PersonID)
		personIDUint = &pid
		if _, err := fh.PersonRepo.GetBasic(*personIDUint); err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				WriteAPIError(w, http.StatusBadRequest, "PersonNotFound", "Person with provided person_id not found")
			} else {
				log.Printf("Error checking person %d before adding face: %v", *personIDUint, err)
				WriteAPIError(w, http.StatusInternalServerError, "PersonFetchError", "Failed to verify person")
			}
			return
		}
	}

	imagePathForDB := strings.TrimLeft(req.ImagePath, "/")
	img, err := fh.ImageRepo.GetByPath(imagePathForDB)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			WriteAPIError(w, http.StatusBadRequest, "ImageNotFound", "image_path does not exist: "+imagePathForDB)
		} else {
			log.Printf("Error checking image %s during face add: %v", imagePathForDB, err)
			WriteAPIError(w, http.StatusInternalServerError, "ImageFetchError", "Could not verify image_path")
		}
		return
	}
	if msg := validateFaceBox(req.X1, req.Y1, req.X2, req.Y2, img); msg != "" {
		WriteAPIError(w, http.StatusBadRequest, "ValidationError", msg)
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
		WriteAPIError(w, http.StatusInternalServerError, "FaceCreateError", "Failed to add face tag")
		return
	}

	createdFace, fetchErr := fh.FaceRepo.GetByID(face.ID)
	if fetchErr != nil {
		log.Printf("Error fetching newly created face %d: %v", face.ID, fetchErr)
		WriteAPIResponse(w, http.StatusCreated, face)
		return
	}
	WriteAPIResponse(w, http.StatusCreated, createdFace)
}

func (fh *FaceHandler) ListFacesByImage(w http.ResponseWriter, r *http.Request) {
	// Query() has already percent-decoded the value; decoding again would corrupt paths containing '%' or '+'
	imagePath := r.URL.Query().Get("path")
	if imagePath == "" {
		WriteAPIError(w, http.StatusBadRequest, "ValidationError", "Missing required query parameter: path")
		return
	}
	imagePathForDB := strings.TrimLeft(imagePath, "/")
	faces, err := fh.FaceRepo.ListByImagePath(imagePathForDB)
	if err != nil {
		log.Printf("Error listing faces for image %s: %v", imagePathForDB, err)
		WriteAPIError(w, http.StatusInternalServerError, "FaceListError", "Failed to retrieve faces for image")
		return
	}
	if faces == nil {
		faces = []models.Face{}
	}
	WriteAPIResponse(w, http.StatusOK, faces)
}

func (fh *FaceHandler) GetFace(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "face_id")
	faceID, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		WriteAPIError(w, http.StatusBadRequest, "InvalidID", "Invalid face ID format")
		return
	}
	face, err := fh.FaceRepo.GetByID(uint(faceID))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			WriteAPIError(w, http.StatusNotFound, "FaceNotFound", "Face tag not found")
		} else {
			log.Printf("Error getting face %d: %v", faceID, err)
			WriteAPIError(w, http.StatusInternalServerError, "FaceFetchError", "Failed to retrieve face tag")
		}
		return
	}
	WriteAPIResponse(w, http.StatusOK, face)
}

func (fh *FaceHandler) UpdateFace(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "face_id")
	faceID, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		WriteAPIError(w, http.StatusBadRequest, "InvalidID", "Invalid face ID format")
		return
	}

	// check if face exists first
	existing, err := fh.FaceRepo.GetByID(uint(faceID))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			WriteAPIError(w, http.StatusNotFound, "FaceNotFound", "Face tag not found")
		} else {
			log.Printf("Error finding face %d for update: %v", faceID, err)
			WriteAPIError(w, http.StatusInternalServerError, "FaceFetchError", "Failed to find face tag for update")
		}
		return
	}

	var reqMap map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&reqMap); err != nil {
		WriteAPIError(w, http.StatusBadRequest, "InvalidPayload", "Invalid request body: "+err.Error())
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
				WriteAPIError(w, http.StatusBadRequest, "ValidationError", "Invalid non-zero value for person_id")
				return
			}
		} else {
			WriteAPIError(w, http.StatusBadRequest, "ValidationError", "Invalid type for person_id, expected number or null")
			return
		}
	}

	coords := map[string]*int{"x1": nil, "y1": nil, "x2": nil, "y2": nil}
	for key := range coords {
		raw, ok := reqMap[key]
		if !ok {
			continue
		}
		v, isNum := raw.(float64)
		if !isNum || math.IsNaN(v) || math.IsInf(v, 0) || v != math.Trunc(v) || math.Abs(v) > math.MaxInt32 {
			WriteAPIError(w, http.StatusBadRequest, "ValidationError", "Invalid value for "+key+", expected an integer")
			return
		}
		i := int(v)
		coords[key] = &i
	}
	x1Update, y1Update, x2Update, y2Update := coords["x1"], coords["y1"], coords["x2"], coords["y2"]

	if x1Update != nil || y1Update != nil || x2Update != nil || y2Update != nil {
		x1, y1, x2, y2 := existing.X1, existing.Y1, existing.X2, existing.Y2
		if x1Update != nil {
			x1 = *x1Update
		}
		if y1Update != nil {
			y1 = *y1Update
		}
		if x2Update != nil {
			x2 = *x2Update
		}
		if y2Update != nil {
			y2 = *y2Update
		}
		img, imgErr := fh.ImageRepo.GetByPath(existing.ImagePath)
		if imgErr != nil && !errors.Is(imgErr, gorm.ErrRecordNotFound) {
			log.Printf("Error fetching image %s for face %d update: %v", existing.ImagePath, faceID, imgErr)
			WriteAPIError(w, http.StatusInternalServerError, "ImageFetchError", "Could not verify image bounds")
			return
		}
		if msg := validateFaceBox(x1, y1, x2, y2, img); msg != "" {
			WriteAPIError(w, http.StatusBadRequest, "ValidationError", msg)
			return
		}
	}

	if personIDProvided && personIDUpdate != nil {
		// *personIDUpdate is uint here because personIDUpdate is *uint
		if _, err := fh.PersonRepo.GetBasic(*personIDUpdate); err != nil { // Use PersonRepo
			if errors.Is(err, gorm.ErrRecordNotFound) {
				WriteAPIError(w, http.StatusBadRequest, "PersonNotFound", "Person with provided person_id not found")
			} else {
				log.Printf("Error checking person %d before updating face %d: %v", *personIDUpdate, faceID, err)
				WriteAPIError(w, http.StatusInternalServerError, "PersonFetchError", "Failed to verify person")
			}
			return
		}
	}

	updateErr := fh.FaceRepo.Update(uint(faceID), personIDUpdate, x1Update, y1Update, x2Update, y2Update)
	if updateErr != nil {
		if errors.Is(updateErr, gorm.ErrRecordNotFound) {
			WriteAPIError(w, http.StatusNotFound, "FaceNotFound", "Face tag not found during update")
		} else {
			log.Printf("Error updating face %d: %v", faceID, updateErr)
			WriteAPIError(w, http.StatusInternalServerError, "FaceUpdateError", "Failed to update face tag")
		}
		return
	}

	// an explicit null person_id un-tags the face (Update ignores a nil person)
	if personIDProvided && personIDUpdate == nil {
		if err := fh.FaceRepo.UntagFace(uint(faceID)); err != nil {
			log.Printf("Error untagging face %d: %v", faceID, err)
			WriteAPIError(w, http.StatusInternalServerError, "FaceUpdateError", "Failed to update face tag")
			return
		}
	}

	updatedFace, err := fh.FaceRepo.GetByID(uint(faceID))
	if err != nil {
		log.Printf("Error fetching updated face %d: %v", faceID, err)
		WriteAPIError(w, http.StatusInternalServerError, "FaceFetchError", "Face was updated but could not be reloaded")
		return
	}
	WriteAPIResponse(w, http.StatusOK, updatedFace)
}

func (fh *FaceHandler) DeleteFace(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "face_id")
	faceID, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		WriteAPIError(w, http.StatusBadRequest, "InvalidID", "Invalid face ID format")
		return
	}
	// the repository removes the face and its embedding in one transaction
	err = fh.FaceRepo.Delete(uint(faceID))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			WriteAPIError(w, http.StatusNotFound, "FaceNotFound", "Face tag not found")
		} else {
			log.Printf("Error deleting face %d: %v", faceID, err)
			WriteAPIError(w, http.StatusInternalServerError, "FaceDeleteError", "Failed to delete face tag")
		}
		return
	}
	deleteFaceThumbnails(fh.Store, uint(faceID))
	w.WriteHeader(http.StatusNoContent)
}

// ServeFaceThumbnail serves a small cached JPEG crop of a face.
// GET /api/faces/{face_id}/thumbnail.jpg
func (fh *FaceHandler) ServeFaceThumbnail(w http.ResponseWriter, r *http.Request) {
	faceID, err := strconv.ParseUint(chi.URLParam(r, "face_id"), 10, 64)
	if err != nil {
		WriteAPIError(w, http.StatusBadRequest, "InvalidID", "Invalid face ID format")
		return
	}
	face, err := fh.FaceRepo.GetByID(uint(faceID))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			WriteAPIError(w, http.StatusNotFound, "FaceNotFound", "Face tag not found")
		} else {
			log.Printf("Error getting face %d for thumbnail: %v", faceID, err)
			WriteAPIError(w, http.StatusInternalServerError, "FaceFetchError", "Failed to retrieve face tag")
		}
		return
	}
	serveFaceThumbnail(w, r, fh.Store, fh.ImageRepo, face)
}

func (fh *FaceHandler) SearchFacesByPerson(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("query")
	if strings.TrimSpace(query) == "" {
		WriteAPIError(w, http.StatusBadRequest, "ValidationError", "Missing required query parameter: query")
		return
	}

	personIDs, err := fh.PersonRepo.FindPersonIDsByNameOrAlias(query)
	if err != nil {
		log.Printf("Error searching for person IDs with query '%s': %v", query, err)
		WriteAPIError(w, http.StatusInternalServerError, "PersonSearchError", "Failed to search for people")
		return
	}
	if len(personIDs) == 0 {
		WriteAPIResponse(w, http.StatusOK, []repository.PersonImageResult{})
		return
	}
	offset, limit := parsePagination(r, 100, 500)
	images, _, err := fh.PersonRepo.FindImagesByPersonIDs(personIDs, offset, limit)
	if err != nil {
		log.Printf("Error finding images for person IDs %v: %v", personIDs, err)
		WriteAPIError(w, http.StatusInternalServerError, "ImageListError", "Failed to find images associated with person")
		return
	}
	if images == nil {
		images = []repository.PersonImageResult{}
	}
	WriteAPIResponse(w, http.StatusOK, images)
}

// GetSimilarFaces finds faces similar to a given face ID
func (fh *FaceHandler) GetSimilarFaces(w http.ResponseWriter, r *http.Request) {
	if fh.FaceRecognitionService == nil {
		WriteAPIError(w, http.StatusServiceUnavailable, "ServiceUnavailable", "Face recognition service not available")
		return
	}

	idStr := chi.URLParam(r, "face_id")
	faceID, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		WriteAPIError(w, http.StatusBadRequest, "InvalidID", "Invalid face ID format")
		return
	}

	// Get limit from query parameter, default to 10
	limitStr := r.URL.Query().Get("limit")
	limit := 10
	if limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
			limit = min(parsedLimit, maxFaceListLimit)
		}
	}

	similarFaces, err := fh.FaceRecognitionService.FindSimilarFaces(uint(faceID), limit)
	if err != nil {
		log.Printf("Error finding similar faces for face %d: %v", faceID, err)

		// Check if the error is due to missing face embedding
		if strings.Contains(err.Error(), "failed to get target face embedding") {
			WriteAPIError(w, http.StatusNotFound, "FaceNoEmbedding", "Face does not have an embedding. Face recognition requires embeddings to be generated for faces.")
		} else {
			WriteAPIError(w, http.StatusInternalServerError, "FaceSearchError", "Failed to find similar faces")
		}
		return
	}

	WriteAPIResponse(w, http.StatusOK, similarFaces)
}

// GetUntaggedFaces returns untagged faces with person suggestions
func (fh *FaceHandler) GetUntaggedFaces(w http.ResponseWriter, r *http.Request) {
	if fh.FaceRecognitionService == nil {
		WriteAPIError(w, http.StatusServiceUnavailable, "ServiceUnavailable", "Face recognition service not available")
		return
	}

	q := r.URL.Query()

	// Get limit from query parameter, default to 20
	limit := 20
	if limitStr := q.Get("limit"); limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
			limit = min(parsedLimit, maxFaceListLimit)
		}
	}

	// Build filter
	filter := repository.UntaggedFaceFilter{
		SortBy:    "created_at",
		SortOrder: "desc",
	}

	if v := q.Get("min_quality"); v != "" {
		if f, err := strconv.ParseFloat(v, 32); err == nil {
			f32 := float32(f)
			filter.MinQuality = &f32
		}
	}
	if v := q.Get("min_confidence"); v != "" {
		if f, err := strconv.ParseFloat(v, 32); err == nil {
			f32 := float32(f)
			filter.MinConfidence = &f32
		}
	}
	if v := q.Get("sort_by"); v == "quality" || v == "confidence" || v == "created_at" {
		filter.SortBy = v
	}
	if v := q.Get("sort_order"); v == "asc" || v == "desc" {
		filter.SortOrder = v
	}
	if q.Get("group_by_image") == "true" {
		filter.GroupByImage = true
	}

	var albumID *uint
	if v := q.Get("album_id"); v != "" {
		id, err := strconv.ParseUint(v, 10, 64)
		if err != nil || id == 0 {
			WriteAPIError(w, http.StatusBadRequest, "ValidationError", "Invalid album_id")
			return
		}
		aid := uint(id)
		albumID = &aid
	}

	untaggedFaces, err := fh.FaceRecognitionService.GetUntaggedFacesWithSuggestions(limit, filter, albumID)
	if err != nil {
		log.Printf("Error getting untagged faces with suggestions: %v", err)
		WriteAPIError(w, http.StatusInternalServerError, "FaceListError", "Failed to get untagged faces")
		return
	}

	WriteAPIResponse(w, http.StatusOK, untaggedFaces)
}

// TagFace tags a face with a person and optionally auto-tags similar faces
func (fh *FaceHandler) TagFace(w http.ResponseWriter, r *http.Request) {
	if fh.FaceRecognitionService == nil {
		WriteAPIError(w, http.StatusServiceUnavailable, "ServiceUnavailable", "Face recognition service not available")
		return
	}

	idStr := chi.URLParam(r, "face_id")
	faceID, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		WriteAPIError(w, http.StatusBadRequest, "InvalidID", "Invalid face ID format")
		return
	}

	var req struct {
		PersonID uint `json:"person_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteAPIError(w, http.StatusBadRequest, "InvalidPayload", "Invalid request body: "+err.Error())
		return
	}

	if req.PersonID == 0 {
		WriteAPIError(w, http.StatusBadRequest, "ValidationError", "person_id is required and must be greater than 0")
		return
	}

	// Verify person exists
	if _, err := fh.PersonRepo.GetBasic(req.PersonID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			WriteAPIError(w, http.StatusBadRequest, "PersonNotFound", "Person not found")
		} else {
			log.Printf("Error verifying person %d: %v", req.PersonID, err)
			WriteAPIError(w, http.StatusInternalServerError, "PersonFetchError", "Failed to verify person")
		}
		return
	}

	// Tag the face with auto-tagging of similar faces; manual tagging counts as confirmed
	err = fh.FaceRecognitionService.TagFaceWithPerson(uint(faceID), req.PersonID, true)
	if err != nil {
		log.Printf("Error tagging face %d with person %d: %v", faceID, req.PersonID, err)

		// Check if the error is due to missing face embedding
		if strings.Contains(err.Error(), "failed to get target face embedding") {
			WriteAPIError(w, http.StatusNotFound, "FaceNoEmbedding", "Face does not have an embedding. Face recognition requires embeddings to be generated for faces.")
		} else {
			WriteAPIError(w, http.StatusInternalServerError, "FaceUpdateError", "Failed to tag face")
		}
		return
	}

	WriteAPIResponse(w, http.StatusOK, map[string]string{"message": "Face tagged successfully"})
}

// AutoTagFace automatically tags a face based on similar faces
func (fh *FaceHandler) AutoTagFace(w http.ResponseWriter, r *http.Request) {
	if fh.FaceRecognitionService == nil {
		WriteAPIError(w, http.StatusServiceUnavailable, "ServiceUnavailable", "Face recognition service not available")
		return
	}

	idStr := chi.URLParam(r, "face_id")
	faceID, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		WriteAPIError(w, http.StatusBadRequest, "InvalidID", "Invalid face ID format")
		return
	}

	face, err := fh.FaceRepo.GetByID(uint(faceID))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			WriteAPIError(w, http.StatusNotFound, "FaceNotFound", "Face tag not found")
		} else {
			log.Printf("Error getting face %d for auto-tag: %v", faceID, err)
			WriteAPIError(w, http.StatusInternalServerError, "FaceFetchError", "Failed to retrieve face tag")
		}
		return
	}
	if face.PersonID != nil {
		WriteAPIError(w, http.StatusConflict, "FaceAlreadyTagged", "Face is already tagged with a person")
		return
	}

	// Get person suggestion for the face
	personID, personName, confidence, err := fh.FaceRecognitionService.SuggestPersonForFace(uint(faceID))
	if err != nil {
		log.Printf("Error suggesting person for face %d: %v", faceID, err)

		// Check if the error is due to missing face embedding
		if strings.Contains(err.Error(), "failed to get target face embedding") {
			WriteAPIError(w, http.StatusNotFound, "FaceNoEmbedding", "Face does not have an embedding. Face recognition requires embeddings to be generated for faces.")
		} else {
			WriteAPIError(w, http.StatusInternalServerError, "FaceSuggestError", "Failed to suggest person for face")
		}
		return
	}

	if personID == nil {
		WriteAPIError(w, http.StatusNotFound, "PersonNotFound", "No suitable person found for this face")
		return
	}

	// the suggestion may reference a person that has since been deleted
	if _, err := fh.PersonRepo.GetBasic(*personID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			WriteAPIError(w, http.StatusNotFound, "PersonNotFound", "No suitable person found for this face")
		} else {
			log.Printf("Error verifying suggested person %d for face %d: %v", *personID, faceID, err)
			WriteAPIError(w, http.StatusInternalServerError, "PersonFetchError", "Failed to verify person")
		}
		return
	}

	// Tag the face with the suggested person; auto-tagging is never confirmed
	err = fh.FaceRecognitionService.TagFaceWithPerson(uint(faceID), *personID, false)
	if err != nil {
		log.Printf("Error auto-tagging face %d with person %d: %v", faceID, *personID, err)
		WriteAPIError(w, http.StatusInternalServerError, "FaceUpdateError", "Failed to auto-tag face")
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

	WriteAPIResponse(w, http.StatusOK, response)
}

// SuggestFace returns the best person suggestion for a face without tagging it
func (fh *FaceHandler) SuggestFace(w http.ResponseWriter, r *http.Request) {
	if fh.FaceRecognitionService == nil {
		WriteAPIError(w, http.StatusServiceUnavailable, "ServiceUnavailable", "Face recognition service not available")
		return
	}

	idStr := chi.URLParam(r, "face_id")
	faceID, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		WriteAPIError(w, http.StatusBadRequest, "InvalidID", "Invalid face ID format")
		return
	}

	suggestion, err := fh.FaceRecognitionService.SuggestPerson(uint(faceID))
	if err != nil {
		if strings.Contains(err.Error(), "failed to get target face embedding") {
			WriteAPIError(w, http.StatusNotFound, "FaceNoEmbedding", "Face does not have an embedding")
		} else {
			log.Printf("Error finding similar faces for suggest on face %d: %v", faceID, err)
			WriteAPIError(w, http.StatusInternalServerError, "FaceSuggestError", "Failed to compute suggestion")
		}
		return
	}

	WriteAPIResponse(w, http.StatusOK, map[string]interface{}{
		"suggested_person_id":   suggestion.PersonID,
		"suggested_person_name": suggestion.PersonName,
		"suggestion_count":      suggestion.Count,
		"confidence":            suggestion.Similarity,
	})
}
