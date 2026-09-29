package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
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

// RolePermissionContribution describes album permissions inherited from a role.
type RolePermissionContribution struct {
	RoleID        uint     `json:"role_id"`
	RoleName      string   `json:"role_name"`
	ForAllAlbums  []string `json:"for_all_albums,omitempty"`
	AlbumSpecific []string `json:"album_specific,omitempty"`
}

// AlbumUserPermissionResponse represents a user with their album permissions
type AlbumUserPermissionResponse struct {
	User                 models.User                  `json:"user"`
	Permissions          []string                     `json:"permissions"`
	DirectPermissions    []string                     `json:"direct_permissions,omitempty"`
	InheritedPermissions []string                     `json:"inherited_permissions,omitempty"`
	RoleContributions    []RolePermissionContribution `json:"role_contributions,omitempty"`
	UserAlbumPermission  *models.UserAlbumPermission  `json:"user_album_permission,omitempty"`
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

func uniqueSortedStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	set := make(map[string]struct{}, len(values))
	for _, value := range values {
		if value == "" {
			continue
		}
		set[value] = struct{}{}
	}
	if len(set) == 0 {
		return nil
	}
	result := make([]string, 0, len(set))
	for value := range set {
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func computeInheritedPermissions(effective []string, direct []string) []string {
	if len(effective) == 0 {
		return nil
	}
	if len(direct) == 0 {
		return effective
	}
	directSet := make(map[string]struct{}, len(direct))
	for _, perm := range direct {
		directSet[perm] = struct{}{}
	}
	inherited := make([]string, 0, len(effective))
	for _, perm := range effective {
		if _, exists := directSet[perm]; !exists {
			inherited = append(inherited, perm)
		}
	}
	if len(inherited) == 0 {
		return nil
	}
	return inherited
}

func buildRoleContributions(user models.User, albumID uint) []RolePermissionContribution {
	if len(user.Roles) == 0 {
		return nil
	}
	contributions := make([]RolePermissionContribution, 0, len(user.Roles))
	for _, role := range user.Roles {
		if role == nil {
			continue
		}
		forAll := uniqueSortedStrings(role.GlobalAlbumPermissions)
		var albumSpecific []string
		for _, rap := range role.AlbumPermissions {
			if rap.AlbumID == albumID {
				albumSpecific = append(albumSpecific, rap.Permissions...)
			}
		}
		albumSpecific = uniqueSortedStrings(albumSpecific)
		if len(forAll) == 0 && len(albumSpecific) == 0 {
			continue
		}
		contributions = append(contributions, RolePermissionContribution{
			RoleID:        role.ID,
			RoleName:      role.Name,
			ForAllAlbums:  forAll,
			AlbumSpecific: albumSpecific,
		})
	}
	if len(contributions) == 0 {
		return nil
	}
	return contributions
}

// GetAlbumUsers returns all users who have permissions for a specific album
func (h *AdminAlbumUserHandler) GetAlbumUsers(w http.ResponseWriter, r *http.Request) {
	albumIDStr := chi.URLParam(r, "id")
	albumID, err := strconv.ParseUint(albumIDStr, 10, 64)
	if err != nil {
		WriteAPIError(w, http.StatusBadRequest, "InvalidID", "Invalid album ID")
		return
	}

	if _, err := h.AlbumRepo.GetByID(uint(albumID)); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			WriteAPIError(w, http.StatusNotFound, "AlbumNotFound", "Album not found")
		} else {
			WriteAPIError(w, http.StatusInternalServerError, "AlbumFetchError", "Failed to verify album")
		}
		return
	}

	users, directPermissionsByUser, err := h.UserRepo.GetUsersWithAlbumPermissions(uint(albumID))
	if err != nil {
		WriteAPIError(w, http.StatusInternalServerError, "AlbumUserListError", "Failed to retrieve album users: "+err.Error())
		return
	}

	// build response with user permissions
	response := make([]AlbumUserPermissionResponse, 0, len(users))
	seenUserIDs := make(map[uint]struct{}, len(users))
	for _, user := range users {
		seenUserIDs[user.ID] = struct{}{}
		var (
			userAlbumPerm   *models.UserAlbumPermission
			directPermSlice []string
		)

		if direct, ok := directPermissionsByUser[user.ID]; ok {
			directCopy := direct
			userAlbumPerm = &directCopy
			directPermSlice = append(directPermSlice, direct.Permissions...)
		}

		effectivePerms := user.GetAlbumPermissions(uint(albumID))
		directPermSlice = uniqueSortedStrings(directPermSlice)
		inheritedPerms := computeInheritedPermissions(effectivePerms, directPermSlice)

		response = append(response, AlbumUserPermissionResponse{
			User:                 user,
			Permissions:          effectivePerms,
			DirectPermissions:    directPermSlice,
			InheritedPermissions: inheritedPerms,
			RoleContributions:    buildRoleContributions(user, uint(albumID)),
			UserAlbumPermission:  userAlbumPerm,
		})
	}

	// include users who inherit album access exclusively via roles or global album permissions
	allUsers, err := h.UserRepo.ListAll()
	if err == nil {
		for _, user := range allUsers {
			if _, alreadyAdded := seenUserIDs[user.ID]; alreadyAdded {
				continue
			}
			effectivePerms := user.GetAlbumPermissions(uint(albumID))
			if len(effectivePerms) == 0 {
				continue
			}
			response = append(response, AlbumUserPermissionResponse{
				User:                 user,
				Permissions:          effectivePerms,
				InheritedPermissions: computeInheritedPermissions(effectivePerms, nil),
				RoleContributions:    buildRoleContributions(user, uint(albumID)),
			})
		}
	}

	writeJSON(w, http.StatusOK, response)
}

// GetAvailableUsers returns all users who don't have permissions for a specific album
func (h *AdminAlbumUserHandler) GetAvailableUsers(w http.ResponseWriter, r *http.Request) {
	albumIDStr := chi.URLParam(r, "id")
	albumID, err := strconv.ParseUint(albumIDStr, 10, 64)
	if err != nil {
		WriteAPIError(w, http.StatusBadRequest, "InvalidID", "Invalid album ID")
		return
	}

	if _, err := h.AlbumRepo.GetByID(uint(albumID)); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			WriteAPIError(w, http.StatusNotFound, "AlbumNotFound", "Album not found")
		} else {
			WriteAPIError(w, http.StatusInternalServerError, "AlbumFetchError", "Failed to verify album")
		}
		return
	}

	users, err := h.UserRepo.GetUsersWithoutAlbumPermissions(uint(albumID))
	if err != nil {
		WriteAPIError(w, http.StatusInternalServerError, "UserListError", "Failed to retrieve available users: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, users)
}

// AddUserToAlbum adds a user to an album with specific permissions
func (h *AdminAlbumUserHandler) AddUserToAlbum(w http.ResponseWriter, r *http.Request) {
	albumIDStr := chi.URLParam(r, "id")
	albumID, err := strconv.ParseUint(albumIDStr, 10, 64)
	if err != nil {
		WriteAPIError(w, http.StatusBadRequest, "InvalidID", "Invalid album ID")
		return
	}

	var payload AddUserToAlbumPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		WriteAPIError(w, http.StatusBadRequest, "InvalidPayload", "Invalid request payload: "+err.Error())
		return
	}

	if _, err := h.AlbumRepo.GetByID(uint(albumID)); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			WriteAPIError(w, http.StatusNotFound, "AlbumNotFound", "Album not found")
		} else {
			WriteAPIError(w, http.StatusInternalServerError, "AlbumFetchError", "Failed to verify album")
		}
		return
	}

	if _, err := h.UserRepo.GetByID(payload.UserID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			WriteAPIError(w, http.StatusNotFound, "UserNotFound", "User not found")
		} else {
			WriteAPIError(w, http.StatusInternalServerError, "UserFetchError", "Failed to verify user")
		}
		return
	}

	for _, perm := range payload.Permissions {
		permDef, ok := permissions.GetPermissionDefinition(perm)
		if !ok {
			WriteAPIError(w, http.StatusBadRequest, "InvalidPermission", fmt.Sprintf("Invalid permission: %s", perm))
			return
		}
		if permDef.Scope != permissions.ScopeAlbum {
			WriteAPIError(w, http.StatusBadRequest, "InvalidPermission", fmt.Sprintf("Permission %s is not album-scoped", perm))
			return
		}
	}

	existingPerm, err := h.UserRepo.GetUserAlbumPermission(payload.UserID, uint(albumID))
	if err == nil && existingPerm != nil {
		WriteAPIError(w, http.StatusConflict, "AlbumUserConflict", "User already has permissions for this album")
		return
	}

	userAlbumPerm := &models.UserAlbumPermission{
		UserID:      payload.UserID,
		AlbumID:     uint(albumID),
		Permissions: payload.Permissions,
	}

	if err := h.UserRepo.CreateUserAlbumPermission(userAlbumPerm); err != nil {
		WriteAPIError(w, http.StatusInternalServerError, "AlbumUserCreateError", "Failed to add user to album: "+err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, userAlbumPerm)
}

// UpdateUserAlbumPermissions updates a user's permissions for a specific album
func (h *AdminAlbumUserHandler) UpdateUserAlbumPermissions(w http.ResponseWriter, r *http.Request) {
	albumIDStr := chi.URLParam(r, "id")
	albumID, err := strconv.ParseUint(albumIDStr, 10, 64)
	if err != nil {
		WriteAPIError(w, http.StatusBadRequest, "InvalidID", "Invalid album ID")
		return
	}

	userIDStr := chi.URLParam(r, "userID")
	userID, err := strconv.ParseUint(userIDStr, 10, 64)
	if err != nil {
		WriteAPIError(w, http.StatusBadRequest, "InvalidID", "Invalid user ID")
		return
	}

	var payload UpdateUserAlbumPermissionsPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		WriteAPIError(w, http.StatusBadRequest, "InvalidPayload", "Invalid request payload: "+err.Error())
		return
	}

	if _, err := h.AlbumRepo.GetByID(uint(albumID)); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			WriteAPIError(w, http.StatusNotFound, "AlbumNotFound", "Album not found")
		} else {
			WriteAPIError(w, http.StatusInternalServerError, "AlbumFetchError", "Failed to verify album")
		}
		return
	}

	if _, err := h.UserRepo.GetByID(uint(userID)); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			WriteAPIError(w, http.StatusNotFound, "UserNotFound", "User not found")
		} else {
			WriteAPIError(w, http.StatusInternalServerError, "UserFetchError", "Failed to verify user")
		}
		return
	}

	for _, perm := range payload.Permissions {
		permDef, ok := permissions.GetPermissionDefinition(perm)
		if !ok {
			WriteAPIError(w, http.StatusBadRequest, "InvalidPermission", fmt.Sprintf("Invalid permission: %s", perm))
			return
		}
		if permDef.Scope != permissions.ScopeAlbum {
			WriteAPIError(w, http.StatusBadRequest, "InvalidPermission", fmt.Sprintf("Permission %s is not album-scoped", perm))
			return
		}
	}

	userAlbumPerm, err := h.UserRepo.GetUserAlbumPermission(uint(userID), uint(albumID))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			WriteAPIError(w, http.StatusNotFound, "AlbumUserNotFound", "User does not have permissions for this album")
		} else {
			WriteAPIError(w, http.StatusInternalServerError, "AlbumUserFetchError", "Failed to retrieve user album permissions")
		}
		return
	}

	userAlbumPerm.Permissions = payload.Permissions

	if err := h.UserRepo.UpdateUserAlbumPermission(userAlbumPerm); err != nil {
		WriteAPIError(w, http.StatusInternalServerError, "AlbumUserUpdateError", "Failed to update user album permissions: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, userAlbumPerm)
}

// RemoveUserFromAlbum removes a user's permissions for a specific album
func (h *AdminAlbumUserHandler) RemoveUserFromAlbum(w http.ResponseWriter, r *http.Request) {
	albumIDStr := chi.URLParam(r, "id")
	albumID, err := strconv.ParseUint(albumIDStr, 10, 64)
	if err != nil {
		WriteAPIError(w, http.StatusBadRequest, "InvalidID", "Invalid album ID")
		return
	}

	userIDStr := chi.URLParam(r, "userID")
	userID, err := strconv.ParseUint(userIDStr, 10, 64)
	if err != nil {
		WriteAPIError(w, http.StatusBadRequest, "InvalidID", "Invalid user ID")
		return
	}

	if _, err := h.AlbumRepo.GetByID(uint(albumID)); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			WriteAPIError(w, http.StatusNotFound, "AlbumNotFound", "Album not found")
		} else {
			WriteAPIError(w, http.StatusInternalServerError, "AlbumFetchError", "Failed to verify album")
		}
		return
	}

	if _, err := h.UserRepo.GetByID(uint(userID)); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			WriteAPIError(w, http.StatusNotFound, "UserNotFound", "User not found")
		} else {
			WriteAPIError(w, http.StatusInternalServerError, "UserFetchError", "Failed to verify user")
		}
		return
	}

	if err := h.UserRepo.DeleteUserAlbumPermission(uint(userID), uint(albumID)); err != nil {
		WriteAPIError(w, http.StatusInternalServerError, "AlbumUserDeleteError", "Failed to remove user from album: "+err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
