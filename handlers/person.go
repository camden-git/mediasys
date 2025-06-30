package handlers

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/camden-git/mediasysbackend/models"
	"github.com/camden-git/mediasysbackend/repository"
	"github.com/go-chi/chi/v5"
	"gorm.io/gorm" // For gorm.ErrRecordNotFound
)

type PersonHandler struct {
	PersonRepo repository.PersonRepositoryInterface
	// GormDB *gorm.DB
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
	writeJSON(w, http.StatusNoContent, nil)
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
	writeJSON(w, http.StatusNoContent, nil)
}
