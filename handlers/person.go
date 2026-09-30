package handlers

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/camden-git/mediasysbackend/config"
	"github.com/camden-git/mediasysbackend/media"
	"github.com/camden-git/mediasysbackend/models"
	"github.com/camden-git/mediasysbackend/repository"
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
		WriteAPIError(w, http.StatusBadRequest, "InvalidPayload", "Invalid request body: "+err.Error())
		return
	}

	req.PrimaryName = strings.TrimSpace(req.PrimaryName)
	if req.PrimaryName == "" {
		WriteAPIError(w, http.StatusBadRequest, "ValidationError", "Missing required field: primary_name")
		return
	}

	// trim and de-duplicate the initial aliases
	var aliases []string
	seen := make(map[string]bool, len(req.Aliases))
	for _, aliasName := range req.Aliases {
		aliasName = strings.TrimSpace(aliasName)
		if aliasName != "" && !seen[aliasName] {
			seen[aliasName] = true
			aliases = append(aliases, aliasName)
		}
	}

	person := models.Person{
		PrimaryName: req.PrimaryName,
	}

	if err := ph.PersonRepo.CreateWithAliases(&person, aliases); err != nil {
		log.Printf("Error creating person '%s': %v", req.PrimaryName, err)
		WriteAPIError(w, http.StatusInternalServerError, "PersonCreateError", "Failed to create person")
		return
	}

	createdPerson, fetchErr := ph.PersonRepo.GetByID(person.ID)
	if fetchErr != nil {
		log.Printf("Error fetching newly created person %d with aliases: %v", person.ID, fetchErr)
		WriteAPIResponse(w, http.StatusCreated, person)
		return
	}

	WriteAPIResponse(w, http.StatusCreated, createdPerson)
}

func (ph *PersonHandler) ListPeople(w http.ResponseWriter, r *http.Request) {
	people, err := ph.PersonRepo.ListAll()
	if err != nil {
		log.Printf("Error listing people: %v", err)
		WriteAPIError(w, http.StatusInternalServerError, "PersonListError", "Failed to retrieve people")
		return
	}
	if people == nil {
		people = []models.Person{}
	}
	WriteAPIResponse(w, http.StatusOK, people)
}

func (ph *PersonHandler) GetPerson(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "person_id")
	personID, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		WriteAPIError(w, http.StatusBadRequest, "InvalidID", "Invalid person ID format")
		return
	}

	person, err := ph.PersonRepo.GetPublicByID(uint(personID))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			WriteAPIError(w, http.StatusNotFound, "PersonNotFound", "Person not found")
		} else {
			log.Printf("Error getting person %d: %v", personID, err)
			WriteAPIError(w, http.StatusInternalServerError, "PersonFetchError", "Failed to retrieve person")
		}
		return
	}
	WriteAPIResponse(w, http.StatusOK, person)
}

// GetPersonAdmin returns a person with all of their faces, including faces in hidden albums.
// GET /api/people/{person_id}/admin (requires people.manage)
func (ph *PersonHandler) GetPersonAdmin(w http.ResponseWriter, r *http.Request) {
	personID, err := strconv.ParseUint(chi.URLParam(r, "person_id"), 10, 64)
	if err != nil {
		WriteAPIError(w, http.StatusBadRequest, "InvalidID", "Invalid person ID format")
		return
	}
	person, err := ph.PersonRepo.GetByID(uint(personID))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			WriteAPIError(w, http.StatusNotFound, "PersonNotFound", "Person not found")
		} else {
			log.Printf("Error getting person %d: %v", personID, err)
			WriteAPIError(w, http.StatusInternalServerError, "PersonFetchError", "Failed to retrieve person")
		}
		return
	}
	WriteAPIResponse(w, http.StatusOK, person)
}

// ListPersonImages returns a page of the visible images containing the person.
// GET /api/people/{person_id}/images?offset=&limit=
func (ph *PersonHandler) ListPersonImages(w http.ResponseWriter, r *http.Request) {
	personID, err := strconv.ParseUint(chi.URLParam(r, "person_id"), 10, 64)
	if err != nil {
		WriteAPIError(w, http.StatusBadRequest, "InvalidID", "Invalid person ID format")
		return
	}
	if _, err := ph.PersonRepo.GetBasic(uint(personID)); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			WriteAPIError(w, http.StatusNotFound, "PersonNotFound", "Person not found")
		} else {
			log.Printf("Error getting person %d: %v", personID, err)
			WriteAPIError(w, http.StatusInternalServerError, "PersonFetchError", "Failed to retrieve person")
		}
		return
	}
	offset, limit := parsePagination(r, 60, 200)
	images, total, err := ph.PersonRepo.FindImagesByPersonIDs([]uint{uint(personID)}, offset, limit)
	if err != nil {
		log.Printf("Error listing images for person %d: %v", personID, err)
		WriteAPIError(w, http.StatusInternalServerError, "ImageListError", "Failed to list images")
		return
	}
	if images == nil {
		images = []repository.PersonImageResult{}
	}
	WriteAPIResponse(w, http.StatusOK, map[string]any{
		"items":    images,
		"total":    total,
		"offset":   offset,
		"limit":    limit,
		"has_more": int64(offset+len(images)) < total,
	})
}

// parsePagination reads offset/limit query params, applying defaultLimit and capping at maxLimit.
func parsePagination(r *http.Request, defaultLimit, maxLimit int) (offset, limit int) {
	limit = defaultLimit
	if v, err := strconv.Atoi(r.URL.Query().Get("limit")); err == nil && v > 0 {
		limit = min(v, maxLimit)
	}
	if v, err := strconv.Atoi(r.URL.Query().Get("offset")); err == nil && v > 0 {
		offset = v
	}
	return offset, limit
}

func (ph *PersonHandler) UpdatePerson(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "person_id")
	personID, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		WriteAPIError(w, http.StatusBadRequest, "InvalidID", "Invalid person ID format")
		return
	}

	var req struct {
		PrimaryName string `json:"primary_name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteAPIError(w, http.StatusBadRequest, "InvalidPayload", "Invalid request body: "+err.Error())
		return
	}
	req.PrimaryName = strings.TrimSpace(req.PrimaryName)
	if req.PrimaryName == "" {
		WriteAPIError(w, http.StatusBadRequest, "ValidationError", "Missing required field: primary_name")
		return
	}

	personToUpdate, err := ph.PersonRepo.GetBasic(uint(personID))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			WriteAPIError(w, http.StatusNotFound, "PersonNotFound", "Person not found")
		} else {
			log.Printf("Error finding person %d for update: %v", personID, err)
			WriteAPIError(w, http.StatusInternalServerError, "PersonFetchError", "Failed to find person for update")
		}
		return
	}

	personToUpdate.PrimaryName = req.PrimaryName

	err = ph.PersonRepo.Update(personToUpdate)
	if err != nil {
		log.Printf("Error updating person %d: %v", personID, err)
		WriteAPIError(w, http.StatusInternalServerError, "PersonUpdateError", "Failed to update person")
		return
	}

	updatedPerson, err := ph.PersonRepo.GetByID(uint(personID))
	if err != nil {
		log.Printf("Error fetching updated person %d: %v", personID, err)
		WriteAPIResponse(w, http.StatusOK, personToUpdate)
		return
	}
	WriteAPIResponse(w, http.StatusOK, updatedPerson)
}

func (ph *PersonHandler) DeletePerson(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "person_id")
	personID, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		WriteAPIError(w, http.StatusBadRequest, "InvalidID", "Invalid person ID format")
		return
	}

	err = ph.PersonRepo.Delete(uint(personID))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			WriteAPIError(w, http.StatusNotFound, "PersonNotFound", "Person not found")
		} else {
			log.Printf("Error deleting person %d: %v", personID, err)
			WriteAPIError(w, http.StatusInternalServerError, "PersonDeleteError", "Failed to delete person")
		}
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (ph *PersonHandler) ListAliases(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "person_id")
	personID, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		WriteAPIError(w, http.StatusBadRequest, "InvalidID", "Invalid person ID format")
		return
	}

	aliases, err := ph.PersonRepo.ListAliasesByPersonID(uint(personID))
	if err != nil {
		log.Printf("Error listing aliases for person %d: %v", personID, err)
		WriteAPIError(w, http.StatusInternalServerError, "AliasListError", "Failed to list aliases")
		return
	}

	WriteAPIResponse(w, http.StatusOK, aliases)
}

func (ph *PersonHandler) AddAlias(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "person_id")
	personID, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		WriteAPIError(w, http.StatusBadRequest, "InvalidID", "Invalid person ID format")
		return
	}

	_, err = ph.PersonRepo.GetBasic(uint(personID))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			WriteAPIError(w, http.StatusNotFound, "PersonNotFound", "Person not found")
		} else {
			log.Printf("Error checking person %d before adding alias: %v", personID, err)
			WriteAPIError(w, http.StatusInternalServerError, "PersonFetchError", "Failed to verify person")
		}
		return
	}

	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteAPIError(w, http.StatusBadRequest, "InvalidPayload", "Invalid request body: "+err.Error())
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		WriteAPIError(w, http.StatusBadRequest, "ValidationError", "Missing required field: name")
		return
	}

	alias := models.Alias{
		PersonID: uint(personID),
		Name:     req.Name,
	}
	err = ph.PersonRepo.AddAlias(&alias)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") || strings.Contains(err.Error(), "UNIQUE constraint failed") {
			WriteAPIError(w, http.StatusConflict, "AliasConflict", "Alias already exists for this person")
		} else {
			log.Printf("Error adding alias '%s' to person %d: %v", req.Name, personID, err)
			WriteAPIError(w, http.StatusInternalServerError, "AliasCreateError", "Failed to add alias")
		}
		return
	}

	WriteAPIResponse(w, http.StatusCreated, alias)
}

func (ph *PersonHandler) SearchPeople(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	if strings.TrimSpace(q) == "" {
		WriteAPIResponse(w, http.StatusOK, []models.Person{})
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
		WriteAPIError(w, http.StatusInternalServerError, "PersonSearchError", "Failed to search people")
		return
	}
	if people == nil {
		people = []models.Person{}
	}
	WriteAPIResponse(w, http.StatusOK, people)
}

func (ph *PersonHandler) DeleteAlias(w http.ResponseWriter, r *http.Request) {
	aliasIdStr := chi.URLParam(r, "alias_id")
	aliasID, err := strconv.ParseUint(aliasIdStr, 10, 64)
	if err != nil {
		WriteAPIError(w, http.StatusBadRequest, "InvalidID", "Invalid alias ID format")
		return
	}

	err = ph.PersonRepo.DeleteAlias(uint(aliasID))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			WriteAPIError(w, http.StatusNotFound, "AliasNotFound", "Alias not found")
		} else {
			log.Printf("Error deleting alias %d: %v", aliasID, err)
			WriteAPIError(w, http.StatusInternalServerError, "AliasDeleteError", "Failed to delete alias")
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

	person, err := ph.PersonRepo.GetBasic(uint(personID))
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

// ServeKeyPhoto serves the cached JPEG crop (max 256px) of the person's key photo face.
// GET /api/people/{person_id}/key-photo.jpg
func (ph *PersonHandler) ServeKeyPhoto(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "person_id")
	personID, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		WriteAPIError(w, http.StatusBadRequest, "InvalidID", "invalid person ID")
		return
	}

	person, err := ph.PersonRepo.GetBasic(uint(personID))
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

	if face.PersonID == nil || *face.PersonID != person.ID {
		WriteAPIError(w, http.StatusNotFound, "NoKeyPhoto", "person has no key photo set")
		return
	}

	serveFaceThumbnail(w, r, ph.Store, ph.ImageRepo, face)
}
