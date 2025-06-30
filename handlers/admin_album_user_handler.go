package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/camden-git/mediasysbackend/models"
	"github.com/camden-git/mediasysbackend/permissions"
	"github.com/camden-git/mediasysbackend/repository"
	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"
)

type AdminAlbumUserHandler struct {
	UserRepo  repository.UserRepository
	AlbumRepo repository.AlbumRepositoryInterface
}

func NewAdminAlbumUserHandler(userRepo repository.UserRepository, albumRepo repository.AlbumRepositoryInterface) *AdminAlbumUserHandler {
	return &AdminAlbumUserHandler{UserRepo: userRepo, AlbumRepo: albumRepo}
}

// AlbumUserPermissionResponse represents a user with their album permissions
type AlbumUserPermissionResponse struct {
	User                models.User                 `json:"user"`
	Permissions         []string                    `json:"permissions"`
	UserAlbumPermission *models.UserAlbumPermission `json:"user_album_permission,omitempty"`
}

// AddUserToAlbumPayload represents the payload for adding a user to an album
type AddUserToAlbumPayload struct {
	UserID      uint     `json:"user_id"`
	Permissions []string `json:"permissions"`
}

// UpdateUserAlbumPermissionsPayload represents the payload for updating user album permissions
type UpdateUserAlbumPermissionsPayload struct {
	Permissions []string `json:"permissions"`
}

// GetAlbumUsers returns all users who have permissions for a specific album
func (h *AdminAlbumUserHandler) GetAlbumUsers(w http.ResponseWriter, r *http.Request) {
	albumIDStr := chi.URLParam(r, "id")
	albumID, err := strconv.ParseUint(albumIDStr, 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid album ID"})
		return
	}

	if _, err := h.AlbumRepo.GetByID(uint(albumID)); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "Album not found"})
		} else {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to verify album"})
		}
		return
	}

	users, err := h.UserRepo.GetUsersWithAlbumPermissions(uint(albumID))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to retrieve album users: " + err.Error()})
		return
	}

	// build response with user permissions
	response := make([]AlbumUserPermissionResponse, 0, len(users))
	for _, user := range users {
		userAlbumPerm, _ := h.UserRepo.GetUserAlbumPermission(user.ID, uint(albumID))

		response = append(response, AlbumUserPermissionResponse{
			User:                user,
			Permissions:         user.GetAlbumPermissions(uint(albumID)),
			UserAlbumPermission: userAlbumPerm,
		})
	}

	writeJSON(w, http.StatusOK, response)
}

// GetAvailableUsers returns all users who don't have permissions for a specific album
func (h *AdminAlbumUserHandler) GetAvailableUsers(w http.ResponseWriter, r *http.Request) {
	albumIDStr := chi.URLParam(r, "id")
	albumID, err := strconv.ParseUint(albumIDStr, 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid album ID"})
		return
	}

	if _, err := h.AlbumRepo.GetByID(uint(albumID)); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "Album not found"})
		} else {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to verify album"})
		}
		return
	}

	users, err := h.UserRepo.GetUsersWithoutAlbumPermissions(uint(albumID))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to retrieve available users: " + err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, users)
}

// AddUserToAlbum adds a user to an album with specific permissions
func (h *AdminAlbumUserHandler) AddUserToAlbum(w http.ResponseWriter, r *http.Request) {
	albumIDStr := chi.URLParam(r, "id")
	albumID, err := strconv.ParseUint(albumIDStr, 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid album ID"})
		return
	}

	var payload AddUserToAlbumPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request payload: " + err.Error()})
		return
	}

	if _, err := h.AlbumRepo.GetByID(uint(albumID)); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "Album not found"})
		} else {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to verify album"})
		}
		return
	}

	if _, err := h.UserRepo.GetByID(payload.UserID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "User not found"})
		} else {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to verify user"})
		}
		return
	}

	for _, perm := range payload.Permissions {
		permDef, ok := permissions.GetPermissionDefinition(perm)
		if !ok {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("Invalid permission: %s", perm)})
			return
		}
		if permDef.Scope != permissions.ScopeAlbum {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("Permission %s is not album-scoped", perm)})
			return
		}
	}

	existingPerm, err := h.UserRepo.GetUserAlbumPermission(payload.UserID, uint(albumID))
	if err == nil && existingPerm != nil {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "User already has permissions for this album"})
		return
	}

	userAlbumPerm := &models.UserAlbumPermission{
		UserID:      payload.UserID,
		AlbumID:     uint(albumID),
		Permissions: payload.Permissions,
	}

	if err := h.UserRepo.CreateUserAlbumPermission(userAlbumPerm); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to add user to album: " + err.Error()})
		return
	}

	writeJSON(w, http.StatusCreated, userAlbumPerm)
}

// UpdateUserAlbumPermissions updates a user's permissions for a specific album
func (h *AdminAlbumUserHandler) UpdateUserAlbumPermissions(w http.ResponseWriter, r *http.Request) {
	albumIDStr := chi.URLParam(r, "id")
	albumID, err := strconv.ParseUint(albumIDStr, 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid album ID"})
		return
	}

	userIDStr := chi.URLParam(r, "userID")
	userID, err := strconv.ParseUint(userIDStr, 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid user ID"})
		return
	}

	var payload UpdateUserAlbumPermissionsPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request payload: " + err.Error()})
		return
	}

	if _, err := h.AlbumRepo.GetByID(uint(albumID)); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "Album not found"})
		} else {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to verify album"})
		}
		return
	}

	if _, err := h.UserRepo.GetByID(uint(userID)); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "User not found"})
		} else {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to verify user"})
		}
		return
	}

	for _, perm := range payload.Permissions {
		permDef, ok := permissions.GetPermissionDefinition(perm)
		if !ok {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("Invalid permission: %s", perm)})
			return
		}
		if permDef.Scope != permissions.ScopeAlbum {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("Permission %s is not album-scoped", perm)})
			return
		}
	}

	userAlbumPerm, err := h.UserRepo.GetUserAlbumPermission(uint(userID), uint(albumID))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "User does not have permissions for this album"})
		} else {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to retrieve user album permissions"})
		}
		return
	}

	userAlbumPerm.Permissions = payload.Permissions

	if err := h.UserRepo.UpdateUserAlbumPermission(userAlbumPerm); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to update user album permissions: " + err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, userAlbumPerm)
}

// RemoveUserFromAlbum removes a user's permissions for a specific album
func (h *AdminAlbumUserHandler) RemoveUserFromAlbum(w http.ResponseWriter, r *http.Request) {
	albumIDStr := chi.URLParam(r, "id")
	albumID, err := strconv.ParseUint(albumIDStr, 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid album ID"})
		return
	}

	userIDStr := chi.URLParam(r, "userID")
	userID, err := strconv.ParseUint(userIDStr, 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid user ID"})
		return
	}

	if _, err := h.AlbumRepo.GetByID(uint(albumID)); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "Album not found"})
		} else {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to verify album"})
		}
		return
	}

	if _, err := h.UserRepo.GetByID(uint(userID)); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "User not found"})
		} else {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to verify user"})
		}
		return
	}

	if err := h.UserRepo.DeleteUserAlbumPermission(uint(userID), uint(albumID)); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to remove user from album: " + err.Error()})
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
