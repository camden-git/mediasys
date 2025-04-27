package handlers

import (
	"database/sql"
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
	"github.com/camden-git/mediasysbackend/workers"
	"github.com/go-chi/chi/v5"
)

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if data != nil {
		if err := json.NewEncoder(w).Encode(data); err != nil {
			log.Printf("Error encoding JSON response: %v", err)
		}
	}
}

type AlbumHandler struct {
	DB       *sql.DB
	Cfg      config.Config
	ThumbGen *workers.ThumbnailGenerator
}

func (ah *AlbumHandler) getAlbumByIdentifier(identifier string) (database.Album, error) {
	// try parsing as ID
	if albumID, err := strconv.ParseInt(identifier, 10, 64); err == nil {
		album, err := database.GetAlbumByID(ah.DB, albumID)
		if err == nil {
			return album, nil // Found by ID
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return database.Album{}, fmt.Errorf("error fetching album by ID %d: %w", albumID, err)
		}
	}

	// not a valid ID or not found by ID, try fetching by slug
	album, err := database.GetAlbumBySlug(ah.DB, identifier)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return database.Album{}, sql.ErrNoRows // Not found by slug either
		}
		return database.Album{}, fmt.Errorf("error fetching album by slug '%s': %w", identifier, err)
	}
	return album, nil
}

func (ah *AlbumHandler) CreateAlbum(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name        string `json:"name"`
		Slug        string `json:"slug"`
		FolderPath  string `json:"folder_path"`
		Description string `json:"description"`
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
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "folder_path does not exist: " + folderPathForDB})
		return
	}
	if err != nil {
		log.Printf("Error stating folder path %s during album creation: %v", fullPath, err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Could not verify folder_path"})
		return
	}
	if !stat.IsDir() {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "folder_path is not a directory: " + folderPathForDB})
		return
	}

	albumID, err := database.CreateAlbum(ah.DB, req.Name, req.Slug, req.Description, folderPathForDB)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			writeJSON(w, http.StatusConflict, map[string]string{"error": "Album name, slug, or folder path already exists"})
		} else {
			log.Printf("Error creating album '%s' (slug '%s'): %v", req.Name, req.Slug, err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to create album"})
		}
		return
	}

	newAlbum, err := database.GetAlbumByID(ah.DB, albumID)
	if err != nil {
		log.Printf("Error fetching newly created album %d: %v", albumID, err)
		writeJSON(w, http.StatusCreated, map[string]interface{}{"message": "Album created successfully", "id": albumID})
		return
	}
	writeJSON(w, http.StatusCreated, newAlbum)
}

func (ah *AlbumHandler) ListAlbums(w http.ResponseWriter, r *http.Request) {
	albums, err := database.ListAlbums(ah.DB)
	if err != nil {
		log.Printf("Error listing albums: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to retrieve albums"})
		return
	}
	if albums == nil {
		albums = []database.Album{}
	}
	writeJSON(w, http.StatusOK, albums)
}

func (ah *AlbumHandler) GetAlbum(w http.ResponseWriter, r *http.Request) {
	identifier := chi.URLParam(r, "album_identifier")

	album, err := ah.getAlbumByIdentifier(identifier)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "Album not found"})
		} else {
			log.Printf("Error getting album by identifier '%s': %v", identifier, err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to retrieve album"})
		}
		return
	}
	writeJSON(w, http.StatusOK, album)
}

func (ah *AlbumHandler) GetAlbumContents(w http.ResponseWriter, r *http.Request) {
	identifier := chi.URLParam(r, "album_identifier")

	album, err := ah.getAlbumByIdentifier(identifier)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "Album not found"})
		} else {
			log.Printf("Error getting album '%s' for contents: %v", identifier, err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to retrieve album information"})
		}
		return
	}

	albumFullPath := filepath.Join(ah.Cfg.RootDirectory, album.FolderPath)
	albumFullPath = filepath.Clean(albumFullPath)
	if !strings.HasPrefix(albumFullPath, ah.Cfg.RootDirectory) {
		log.Printf("CRITICAL: Album ID %d (slug %s) folder path '%s' resolved outside root directory ('%s'). Aborting.", album.ID, album.Slug, album.FolderPath, albumFullPath)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Album configuration error"})
		return
	}

	fileInfos, err := listDirectoryContents(albumFullPath, "/"+album.FolderPath, ah.Cfg, ah.DB, ah.ThumbGen)
	if err != nil {
		if os.IsNotExist(err) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "Album folder not found on disk: " + album.FolderPath})
		} else if os.IsPermission(err) {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "Permission denied accessing album folder"})
		} else {
			log.Printf("Error listing contents for album %d/%s (path %s): %v", album.ID, album.Slug, albumFullPath, err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to list album contents"})
		}
		return
	}

	listing := DirectoryListing{
		Path:  "/" + album.FolderPath,
		Files: fileInfos,
		// Parent: "/api/albums",
	}
	writeJSON(w, http.StatusOK, listing)
}

func (ah *AlbumHandler) UpdateAlbum(w http.ResponseWriter, r *http.Request) {
	identifier := chi.URLParam(r, "album_identifier")

	album, err := ah.getAlbumByIdentifier(identifier)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "Album not found"})
		} else {
			log.Printf("Error finding album '%s' for update: %v", identifier, err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to find album for update"})
		}
		return
	}

	var req struct {
		Name        *string `json:"name"`
		Description *string `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request body: " + err.Error()})
		return
	}

	nameUpdate := album.Name
	descUpdate := album.Description
	updateRequested := false
	if req.Name != nil {
		nameUpdate = *req.Name
		updateRequested = true
	}
	if req.Description != nil {
		descUpdate = *req.Description
		updateRequested = true
	}
	if !updateRequested {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "No fields provided for update"})
		return
	}

	err = database.UpdateAlbum(ah.DB, album.ID, nameUpdate, descUpdate)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// should not happen if we found it above, but handle defensively
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "Album not found during update"})
		} else if strings.Contains(err.Error(), "album name conflict") {
			writeJSON(w, http.StatusConflict, map[string]string{"error": "Album name already exists"})
		} else {
			log.Printf("Error updating album %d/%s: %v", album.ID, album.Slug, err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to update album"})
		}
		return
	}

	updatedAlbum, err := database.GetAlbumByID(ah.DB, album.ID)
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
		if errors.Is(err, sql.ErrNoRows) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "Album not found"})
		} else {
			log.Printf("Error finding album '%s' for delete: %v", identifier, err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to find album for delete"})
		}
		return
	}

	err = database.DeleteAlbum(ah.DB, album.ID)
	if err != nil {
		log.Printf("Error deleting album %d/%s: %v", album.ID, album.Slug, err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to delete album"})
		return
	}

	writeJSON(w, http.StatusNoContent, nil)
}
