package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/camden-git/mediasysbackend/config"
	"github.com/camden-git/mediasysbackend/database"
	"github.com/camden-git/mediasysbackend/media"
	"github.com/camden-git/mediasysbackend/models"
	"github.com/camden-git/mediasysbackend/repository"
	"github.com/camden-git/mediasysbackend/workers"
	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"
)

type AlbumHandler struct {
	AlbumRepo      repository.AlbumRepositoryInterface
	ImageRepo      repository.ImageRepositoryInterface
	UserRepo       repository.UserRepository
	TagRepo        repository.ImageTagRepositoryInterface
	Cfg            config.Config
	ThumbGen       *workers.ImageProcessor
	MediaProcessor *media.Processor
}

func (ah *AlbumHandler) getAlbumByIdentifier(identifier string) (*models.Album, error) {
	// try parsing as ID
	if albumID, err := strconv.ParseUint(identifier, 10, 64); err == nil {
		album, err := ah.AlbumRepo.GetByID(uint(albumID))
		if err == nil {
			return album, nil // found by ID
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("error fetching album by ID %d: %w", albumID, err)
		}
		// If not found by ID, continue to try by slug
	}

	// not a valid ID or not found by ID, try fetching by slug
	album, err := ah.AlbumRepo.GetBySlug(identifier)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, gorm.ErrRecordNotFound // not found by slug either
		}
		return nil, fmt.Errorf("error fetching album by slug '%s': %w", identifier, err)
	}
	return album, nil
}

func (ah *AlbumHandler) CreateAlbum(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name        string  `json:"name"`
		Slug        string  `json:"slug"`
		FolderPath  string  `json:"folder_path"`
		Description *string `json:"description"`
		IsHidden    *bool   `json:"is_hidden"`
		Location    *string `json:"location"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request body: " + err.Error()})
		return
	}

	if req.Name == "" || req.FolderPath == "" || req.Slug == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Missing required fields: name, slug, and folder_path"})
		return
	}

	if strings.ContainsAny(req.Slug, " /\\?%*:|\"<>") || strings.TrimSpace(req.Slug) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid slug format. Use URL-safe characters without spaces."})
		return
	}

	cleanRelativePath := filepath.Clean(req.FolderPath)
	if filepath.IsAbs(cleanRelativePath) || strings.HasPrefix(cleanRelativePath, "..") {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "folder_path must be relative and cannot use '..'"})
		return
	}
	folderPathForDB := filepath.ToSlash(cleanRelativePath)
	fullPath := filepath.Join(ah.Cfg.RootDirectory, folderPathForDB)
	stat, err := os.Stat(fullPath)
	if os.IsNotExist(err) {
		// create the directory if it doesn't exist
		err = os.MkdirAll(fullPath, 0755)
		if err != nil {
			log.Printf("Error creating folder path %s during album creation: %v", fullPath, err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Could not create folder_path"})
			return
		}
		log.Printf("Created folder path: %s", fullPath)
	} else if err != nil {
		log.Printf("Error stating folder path %s during album creation: %v", fullPath, err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Could not verify folder_path"})
		return
	} else if !stat.IsDir() {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "folder_path is not a directory: " + folderPathForDB})
		return
	}

	newAlbumGorm := models.Album{
		Name:        req.Name,
		Slug:        req.Slug,
		Description: req.Description,
		FolderPath:  folderPathForDB,
	}
	if req.IsHidden != nil {
		newAlbumGorm.IsHidden = *req.IsHidden
	}
	if req.Location != nil {
		newAlbumGorm.Location = req.Location
	}

	err = ah.AlbumRepo.Create(&newAlbumGorm)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			writeJSON(w, http.StatusConflict, map[string]string{"error": "Album name, slug, or folder path already exists"})
		} else {
			log.Printf("Error creating album '%s' (slug '%s'): %v", req.Name, req.Slug, err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to create album"})
		}
		return
	}

	writeJSON(w, http.StatusCreated, newAlbumGorm)
}

func (ah *AlbumHandler) ListAlbums(w http.ResponseWriter, r *http.Request) {
	albums, err := ah.AlbumRepo.ListAll()
	if err != nil {
		log.Printf("Error listing albums: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to retrieve albums"})
		return
	}
	if albums == nil {
		albums = []models.Album{} // ensure an empty array instead of null for JSON
	}
	writeJSON(w, http.StatusOK, albums)
}

func (ah *AlbumHandler) GetAlbum(w http.ResponseWriter, r *http.Request) {
	identifier := chi.URLParam(r, "album_identifier")

	album, err := ah.getAlbumByIdentifier(identifier)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "Album not found"})
		} else {
			log.Printf("Error getting album by identifier '%s': %v", identifier, err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to retrieve album"})
		}
		return
	}

	// Build artists list from uploaders
	var artists []map[string]interface{}
	if ah.ImageRepo != nil && ah.UserRepo != nil {
		if ids, err := ah.ImageRepo.GetDistinctUploaderIDsByFolderPrefix(album.FolderPath); err == nil {
			for _, id := range ids {
				if u, err := ah.UserRepo.GetByID(id); err == nil && u != nil {
					artists = append(artists, map[string]interface{}{
						"id":         u.ID,
						"username":   u.Username,
						"first_name": u.FirstName,
						"last_name":  u.LastName,
					})
				}
			}
		}
	}

	type albumWithArtists struct {
		*models.Album
		Artists []map[string]interface{} `json:"artists,omitempty"`
	}
	writeJSON(w, http.StatusOK, albumWithArtists{Album: album, Artists: artists})
}

func (ah *AlbumHandler) UpdateAlbumSortOrder(w http.ResponseWriter, r *http.Request) {
	identifier := chi.URLParam(r, "album_identifier")

	album, err := ah.getAlbumByIdentifier(identifier)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "Album not found"})
		} else {
			log.Printf("Error finding album '%s' for sort update: %v", identifier, err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to find album"})
		}
		return
	}

	var req struct {
		SortOrder string `json:"sort_order"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request body: " + err.Error()})
		return
	}

	if !database.IsValidSortOrder(req.SortOrder) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid sort_order value provided"})
		return
	}

	err = ah.AlbumRepo.UpdateSortOrder(album.ID, req.SortOrder)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "Album not found during update"})
		} else {
			log.Printf("Error updating sort order for album %d: %v", album.ID, err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to update sort order"})
		}
		return
	}

	updatedAlbum, err := ah.AlbumRepo.GetByID(album.ID)
	if err != nil {
		log.Printf("Error fetching updated album %d after sort update: %v", album.ID, err)
		writeJSON(w, http.StatusOK, map[string]string{"message": "Sort order updated successfully"})
		return
	}
	writeJSON(w, http.StatusOK, updatedAlbum)
}

func (ah *AlbumHandler) UpdateAlbum(w http.ResponseWriter, r *http.Request) {
	identifier := chi.URLParam(r, "album_identifier")

	album, err := ah.getAlbumByIdentifier(identifier)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "Album not found"})
		} else {
			log.Printf("Error finding album '%s' for update: %v", identifier, err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to find album for update"})
		}
		return
	}

	var req struct {
		Name        *string `json:"name"` // pointers to distinguish between empty string and not provided
		Description *string `json:"description"`
		IsHidden    *bool   `json:"is_hidden"`
		Location    *string `json:"location"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request body: " + err.Error()})
		return
	}

	var nameUpdate string
	var descUpdate *string // keep as a pointer for repository
	var isHiddenUpdate *bool
	var locationUpdate *string
	updateRequested := false

	if req.Name != nil {
		nameUpdate = *req.Name
		updateRequested = true
	} else {
		nameUpdate = album.Name
	}

	if req.Description != nil {
		descUpdate = req.Description
		updateRequested = true
	} else {
		descUpdate = album.Description
	}

	if req.IsHidden != nil {
		isHiddenUpdate = req.IsHidden
		updateRequested = true
	} else {
		isHiddenUpdate = &album.IsHidden
	}

	if req.Location != nil {
		locationUpdate = req.Location
		updateRequested = true
	} else {
		locationUpdate = album.Location
	}

	if !updateRequested {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "No fields provided for update"})
		return
	}

	err = ah.AlbumRepo.Update(album.ID, nameUpdate, descUpdate, isHiddenUpdate, locationUpdate)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "Album not found during update"})
		} else if strings.Contains(strings.ToLower(err.Error()), "unique") {
			writeJSON(w, http.StatusConflict, map[string]string{"error": "Album name already exists"})
		} else {
			log.Printf("Error updating album %d/%s: %v", album.ID, album.Slug, err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to update album"})
		}
		return
	}

	updatedAlbum, err := ah.AlbumRepo.GetByID(album.ID)
	if err != nil {
		log.Printf("Error fetching updated album %d/%s: %v", album.ID, album.Slug, err)
		writeJSON(w, http.StatusOK, map[string]string{"message": "Album updated successfully"})
		return
	}
	writeJSON(w, http.StatusOK, updatedAlbum)
}

func (ah *AlbumHandler) DeleteAlbum(w http.ResponseWriter, r *http.Request) {
	identifier := chi.URLParam(r, "album_identifier")

	album, err := ah.getAlbumByIdentifier(identifier)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "Album not found"})
		} else {
			log.Printf("Error finding album '%s' for delete: %v", identifier, err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to find album for delete"})
		}
		return
	}

	err = ah.AlbumRepo.Delete(album.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) { // if trying to delete already deleted (by another request)
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "Album not found or already deleted"})
		} else {
			log.Printf("Error deleting album %d/%s: %v", album.ID, album.Slug, err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to delete album"})
		}
		return
	}

	writeJSON(w, http.StatusNoContent, nil)
}
