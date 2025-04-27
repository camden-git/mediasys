package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/camden-git/mediasysbackend/database"
	"github.com/go-chi/chi/v5"
)

type PersonHandler struct {
	DB *sql.DB
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

	personID, err := database.CreatePerson(ph.DB, req.PrimaryName)
	if err != nil {
		log.Printf("Error creating person '%s': %v", req.PrimaryName, err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to create person"})
		return
	}

	if len(req.Aliases) > 0 {
		for _, aliasName := range req.Aliases {
			if strings.TrimSpace(aliasName) != "" {
				_, aliasErr := database.AddAlias(ph.DB, personID, aliasName)
				if aliasErr != nil {
					log.Printf("Error adding initial alias '%s' for person %d: %v", aliasName, personID, aliasErr)
				}
			}
		}
	}

	person, err := database.GetPersonByID(ph.DB, personID)
	if err != nil {
		log.Printf("Error fetching newly created person %d: %v", personID, err)
		writeJSON(w, http.StatusCreated, map[string]interface{}{"message": "Person created successfully", "id": personID})
		return
	}
	aliases, err := database.ListAliasesByPersonID(ph.DB, personID)
	if err != nil {
		log.Printf("Error fetching aliases for newly created person %d: %v", personID, err)
	} else {
		person.Aliases = aliases
	}

	writeJSON(w, http.StatusCreated, person)
}

func (ph *PersonHandler) ListPeople(w http.ResponseWriter, r *http.Request) {
	people, err := database.ListPeople(ph.DB)
	if err != nil {
		log.Printf("Error listing people: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to retrieve people"})
		return
	}
	if people == nil {
		people = []database.Person{}
	}
	writeJSON(w, http.StatusOK, people)
}

func (ph *PersonHandler) GetPerson(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "person_id")
	personID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid person ID format"})
		return
	}

	person, err := database.GetPersonByID(ph.DB, personID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "Person not found"})
		} else {
			log.Printf("Error getting person %d: %v", personID, err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to retrieve person"})
		}
		return
	}

	aliases, err := database.ListAliasesByPersonID(ph.DB, personID)
	if err != nil {
		log.Printf("Error fetching aliases for person %d: %v", personID, err)
	} else {
		person.Aliases = aliases
	}

	writeJSON(w, http.StatusOK, person)
}

func (ph *PersonHandler) UpdatePerson(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "person_id")
	personID, err := strconv.ParseInt(idStr, 10, 64)
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

	err = database.UpdatePerson(ph.DB, personID, req.PrimaryName)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "Person not found"})
		} else {
			log.Printf("Error updating person %d: %v", personID, err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to update person"})
		}
		return
	}

	updatedPerson, err := database.GetPersonByID(ph.DB, personID)
	if err != nil {
		log.Printf("Error fetching updated person %d: %v", personID, err)
		writeJSON(w, http.StatusOK, map[string]string{"message": "Person updated successfully"})
		return
	}
	aliases, err := database.ListAliasesByPersonID(ph.DB, personID)
	if err == nil {
		updatedPerson.Aliases = aliases
	}

	writeJSON(w, http.StatusOK, updatedPerson)
}

func (ph *PersonHandler) DeletePerson(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "person_id")
	personID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid person ID format"})
		return
	}

	err = database.DeletePerson(ph.DB, personID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
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
	personID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid person ID format"})
		return
	}

	if _, err := database.GetPersonByID(ph.DB, personID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
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

	aliasID, err := database.AddAlias(ph.DB, personID, req.Name)
	if err != nil {
		if strings.Contains(err.Error(), "already exists") {
			writeJSON(w, http.StatusConflict, map[string]string{"error": "Alias already exists for this person"})
		} else {
			log.Printf("Error adding alias '%s' to person %d: %v", req.Name, personID, err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to add alias"})
		}
		return
	}

	writeJSON(w, http.StatusCreated, map[string]interface{}{"id": aliasID, "person_id": personID, "name": req.Name})
}

func (ph *PersonHandler) DeleteAlias(w http.ResponseWriter, r *http.Request) {
	aliasIdStr := chi.URLParam(r, "alias_id")
	aliasID, err := strconv.ParseInt(aliasIdStr, 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid alias ID format"})
		return
	}

	err = database.DeleteAlias(ph.DB, aliasID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "Alias not found"})
		} else {
			log.Printf("Error deleting alias %d: %v", aliasID, err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to delete alias"})
		}
		return
	}

	writeJSON(w, http.StatusNoContent, nil)
}
