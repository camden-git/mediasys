package handlers

import (
	"encoding/json"
	"errors"
	"image"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/camden-git/mediasysbackend/config"
	"github.com/camden-git/mediasysbackend/media"
	"github.com/camden-git/mediasysbackend/models"
	"github.com/camden-git/mediasysbackend/repository"
	"github.com/disintegration/imaging"
	"github.com/go-chi/chi/v5"
	"gorm.io/gorm" // For gorm.ErrRecordNotFound
)

type PersonHandler struct {
	PersonRepo repository.PersonRepositoryInterface
	FaceRepo   repository.FaceRepositoryInterface
	ImageRepo  repository.ImageRepositoryInterface
	Store      *media.Store
	Cfg        config.Config
}

func (ph *PersonHandler) CreatePerson(w http.ResponseWriter, r *http.Request) {
	var req struct {
		PrimaryName string   `json:"primary_name"`
		Aliases     []string `json:"aliases"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request body: " + err.Error()})
		return
	}

	if strings.TrimSpace(req.PrimaryName) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Missing required field: primary_name"})
		return
	}

	person := models.Person{
		PrimaryName: req.PrimaryName,
	}

	err := ph.PersonRepo.Create(&person)
	if err != nil {
		// GORM might return a more specific error for unique constraints
		log.Printf("Error creating person '%s': %v", req.PrimaryName, err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to create person"})
		return
	}

	// add aliases if provided
	if len(req.Aliases) > 0 {
		for _, aliasName := range req.Aliases {
			if strings.TrimSpace(aliasName) != "" {
				alias := models.Alias{
					PersonID: person.ID,
					Name:     aliasName,
				}
				aliasErr := ph.PersonRepo.AddAlias(&alias)
				if aliasErr != nil {
					log.Printf("Error adding initial alias '%s' for person %d: %v", aliasName, person.ID, aliasErr)
				}
			}
		}
	}

	createdPerson, fetchErr := ph.PersonRepo.GetByID(person.ID)
	if fetchErr != nil {
		log.Printf("Error fetching newly created person %d with aliases: %v", person.ID, fetchErr)
		writeJSON(w, http.StatusCreated, person)
		return
	}

	writeJSON(w, http.StatusCreated, createdPerson)
}

func (ph *PersonHandler) ListPeople(w http.ResponseWriter, r *http.Request) {
	people, err := ph.PersonRepo.ListAll()
	if err != nil {
		log.Printf("Error listing people: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to retrieve people"})
		return
	}
	if people == nil {
		people = []models.Person{}
	}
	writeJSON(w, http.StatusOK, people)
}

func (ph *PersonHandler) GetPerson(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "person_id")
	personID, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid person ID format"})
		return
	}

	person, err := ph.PersonRepo.GetByID(uint(personID))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "Person not found"})
		} else {
			log.Printf("Error getting person %d: %v", personID, err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to retrieve person"})
		}
		return
	}
	// GetByID should preload aliases if defined in repository method
	writeJSON(w, http.StatusOK, person)
}

func (ph *PersonHandler) UpdatePerson(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "person_id")
	personID, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid person ID format"})
		return
	}

	var req struct {
		PrimaryName string `json:"primary_name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request body: " + err.Error()})
		return
	}
	if strings.TrimSpace(req.PrimaryName) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Missing required field: primary_name"})
		return
	}

	personToUpdate, err := ph.PersonRepo.GetByID(uint(personID))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "Person not found"})
		} else {
			log.Printf("Error finding person %d for update: %v", personID, err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to find person for update"})
		}
		return
	}

	personToUpdate.PrimaryName = req.PrimaryName

	err = ph.PersonRepo.Update(personToUpdate)
	if err != nil {
		log.Printf("Error updating person %d: %v", personID, err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to update person"})
		return
	}

	updatedPerson, err := ph.PersonRepo.GetByID(uint(personID))
	if err != nil {
		log.Printf("Error fetching updated person %d: %v", personID, err)
		writeJSON(w, http.StatusOK, map[string]string{"message": "Person updated successfully, but failed to fetch full details."})
		return
	}
	writeJSON(w, http.StatusOK, updatedPerson)
}

func (ph *PersonHandler) DeletePerson(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "person_id")
	personID, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid person ID format"})
		return
	}

	err = ph.PersonRepo.Delete(uint(personID))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "Person not found"})
		} else {
			log.Printf("Error deleting person %d: %v", personID, err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to delete person"})
		}
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (ph *PersonHandler) ListAliases(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "person_id")
	personID, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid person ID format"})
		return
	}

	aliases, err := ph.PersonRepo.ListAliasesByPersonID(uint(personID))
	if err != nil {
		log.Printf("Error listing aliases for person %d: %v", personID, err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to list aliases"})
		return
	}

	writeJSON(w, http.StatusOK, aliases)
}

func (ph *PersonHandler) AddAlias(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "person_id")
	personID, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid person ID format"})
		return
	}

	_, err = ph.PersonRepo.GetByID(uint(personID))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "Person not found"})
		} else {
			log.Printf("Error checking person %d before adding alias: %v", personID, err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to verify person"})
		}
		return
	}

	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request body: " + err.Error()})
		return
	}
	if strings.TrimSpace(req.Name) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Missing required field: name"})
		return
	}

	alias := models.Alias{
		PersonID: uint(personID),
		Name:     req.Name,
	}
	err = ph.PersonRepo.AddAlias(&alias)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") || strings.Contains(err.Error(), "UNIQUE constraint failed") {
			writeJSON(w, http.StatusConflict, map[string]string{"error": "Alias already exists for this person"})
		} else {
			log.Printf("Error adding alias '%s' to person %d: %v", req.Name, personID, err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to add alias"})
		}
		return
	}

	writeJSON(w, http.StatusCreated, alias)
}

func (ph *PersonHandler) SearchPeople(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	if strings.TrimSpace(q) == "" {
		writeJSON(w, http.StatusOK, []models.Person{})
		return
	}

	limit := 5
	if ls := r.URL.Query().Get("limit"); ls != "" {
		if v, err := strconv.Atoi(ls); err == nil && v > 0 {
			if v > 50 {
				v = 50
			}
			limit = v
		}
	}

	people, err := ph.PersonRepo.SearchByNameOrAlias(q, limit)
	if err != nil {
		log.Printf("Error searching people for '%s': %v", q, err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to search people"})
		return
	}
	if people == nil {
		people = []models.Person{}
	}
	writeJSON(w, http.StatusOK, people)
}

func (ph *PersonHandler) DeleteAlias(w http.ResponseWriter, r *http.Request) {
	aliasIdStr := chi.URLParam(r, "alias_id")
	aliasID, err := strconv.ParseUint(aliasIdStr, 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid alias ID format"})
		return
	}

	err = ph.PersonRepo.DeleteAlias(uint(aliasID))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "Alias not found"})
		} else {
			log.Printf("Error deleting alias %d: %v", aliasID, err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to delete alias"})
		}
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// SetKeyPhoto sets (or clears) the key photo for a person.
// PUT /api/people/{person_id}/key-photo
// Body: {"face_id": N}  (N=0 or omitted clears the key photo)
func (ph *PersonHandler) SetKeyPhoto(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "person_id")
	personID, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		WriteAPIError(w, http.StatusBadRequest, "InvalidID", "invalid person ID")
		return
	}

	var req struct {
		FaceID uint `json:"face_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteAPIError(w, http.StatusBadRequest, "InvalidBody", "invalid request body")
		return
	}

	person, err := ph.PersonRepo.GetByID(uint(personID))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			WriteAPIError(w, http.StatusNotFound, "PersonNotFound", "person not found")
		} else {
			log.Printf("SetKeyPhoto: error getting person %d: %v", personID, err)
			WriteAPIError(w, http.StatusInternalServerError, "DBError", "failed to retrieve person")
		}
		return
	}

	var faceID *uint
	if req.FaceID != 0 {
		face, err := ph.FaceRepo.GetByID(req.FaceID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				WriteAPIError(w, http.StatusNotFound, "FaceNotFound", "face not found")
			} else {
				log.Printf("SetKeyPhoto: error getting face %d: %v", req.FaceID, err)
				WriteAPIError(w, http.StatusInternalServerError, "DBError", "failed to retrieve face")
			}
			return
		}
		if face.PersonID == nil || *face.PersonID != person.ID {
			WriteAPIError(w, http.StatusBadRequest, "FacePersonMismatch", "face does not belong to this person")
			return
		}
		faceID = &req.FaceID
	}

	if err := ph.PersonRepo.UpdateKeyPhoto(uint(personID), faceID); err != nil {
		log.Printf("SetKeyPhoto: error updating key photo for person %d: %v", personID, err)
		WriteAPIError(w, http.StatusInternalServerError, "DBError", "failed to update key photo")
		return
	}

	updatedPerson, err := ph.PersonRepo.GetByID(uint(personID))
	if err != nil {
		log.Printf("SetKeyPhoto: error fetching updated person %d: %v", personID, err)
		WriteAPIError(w, http.StatusInternalServerError, "DBError", "failed to retrieve updated person")
		return
	}
	WriteAPIResponse(w, http.StatusOK, updatedPerson)
}

// ServeKeyPhoto serves a 128×128 JPEG thumbnail cropped from the person's key photo face.
// GET /api/people/{person_id}/key-photo.jpg
func (ph *PersonHandler) ServeKeyPhoto(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "person_id")
	personID, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		WriteAPIError(w, http.StatusBadRequest, "InvalidID", "invalid person ID")
		return
	}

	person, err := ph.PersonRepo.GetByID(uint(personID))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			WriteAPIError(w, http.StatusNotFound, "PersonNotFound", "person not found")
		} else {
			log.Printf("ServeKeyPhoto: error getting person %d: %v", personID, err)
			WriteAPIError(w, http.StatusInternalServerError, "DBError", "failed to retrieve person")
		}
		return
	}

	if person.KeyPhotoFaceID == nil {
		WriteAPIError(w, http.StatusNotFound, "NoKeyPhoto", "person has no key photo set")
		return
	}

	face, err := ph.FaceRepo.GetByID(*person.KeyPhotoFaceID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			WriteAPIError(w, http.StatusNotFound, "FaceNotFound", "key photo face not found")
		} else {
			log.Printf("ServeKeyPhoto: error getting face %d: %v", *person.KeyPhotoFaceID, err)
			WriteAPIError(w, http.StatusInternalServerError, "DBError", "failed to retrieve face")
		}
		return
	}

	img, err := ph.ImageRepo.GetByPath(face.ImagePath)
	if err != nil {
		WriteAPIError(w, http.StatusNotFound, "ImageNotFound", "key photo image not found")
		return
	}
	obj, _, err := ph.Store.Get(r.Context(), img.ObjectKey)
	if err != nil {
		log.Printf("ServeKeyPhoto: failed to open image %s: %v", img.ObjectKey, err)
		WriteAPIError(w, http.StatusInternalServerError, "ImageOpenError", "could not open image")
		return
	}
	defer obj.Close()
	src, err := imaging.Decode(obj, imaging.AutoOrientation(true))
	if err != nil {
		log.Printf("ServeKeyPhoto: failed to decode image %s: %v", img.ObjectKey, err)
		WriteAPIError(w, http.StatusInternalServerError, "ImageOpenError", "could not open image")
		return
	}

	// Compute padded square crop centered on face bounding box
	cx := (face.X1 + face.X2) / 2
	cy := (face.Y1 + face.Y2) / 2
	faceW := face.X2 - face.X1
	faceH := face.Y2 - face.Y1
	side := faceW
	if faceH > side {
		side = faceH
	}
	// Add 20% padding on each side
	paddedSide := int(float64(side) * 1.4)
	if paddedSide < 1 {
		paddedSide = 1
	}
	x1 := cx - paddedSide/2
	y1 := cy - paddedSide/2
	x2 := x1 + paddedSide
	y2 := y1 + paddedSide

	// Clamp to image bounds
	imgW := src.Bounds().Dx()
	imgH := src.Bounds().Dy()
	if x1 < 0 {
		x1 = 0
	}
	if y1 < 0 {
		y1 = 0
	}
	if x2 > imgW {
		x2 = imgW
	}
	if y2 > imgH {
		y2 = imgH
	}

	crop := imaging.Crop(src, image.Rect(x1, y1, x2, y2))
	thumb := imaging.Fill(crop, 128, 128, imaging.Center, imaging.Lanczos)

	w.Header().Set("Content-Type", "image/jpeg")
	w.Header().Set("Cache-Control", "public, max-age=604800")

	if err := imaging.Encode(w, thumb, imaging.JPEG, imaging.JPEGQuality(85)); err != nil {
		log.Printf("ServeKeyPhoto: encode error: %v", err)
	}
}
