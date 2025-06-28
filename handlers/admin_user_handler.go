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
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type AdminUserHandler struct {
	UserRepo repository.UserRepository
	RoleRepo repository.RoleRepository // For validating role IDs during user creation/update
}

func NewAdminUserHandler(userRepo repository.UserRepository, roleRepo repository.RoleRepository) *AdminUserHandler {
	return &AdminUserHandler{UserRepo: userRepo, RoleRepo: roleRepo}
}

// --- DTOs for User Management ---

type UserCreatePayload struct {
	Username          string   `json:"username"`
	Password          string   `json:"password"`
	RoleIDs           []uint   `json:"role_ids"` // IDs of roles to assign
	GlobalPermissions []string `json:"global_permissions"`
	// AlbumPermissions are more complex and might be handled via separate endpoints
	// e.g., POST /api/admin/users/{id}/album-permissions
}

type UserUpdatePayload struct {
	Username          *string   `json:"username,omitempty"`
	Password          *string   `json:"password,omitempty"` // For password changes
	RoleIDs           *[]uint   `json:"role_ids,omitempty"` // Full set of role IDs to assign
	GlobalPermissions *[]string `json:"global_permissions,omitempty"`
}

// UserResponseDTO is a simplified User model for API responses, excluding sensitive data.
type UserResponseDTO struct {
	ID                uint                         `json:"id"`
	Username          string                       `json:"username"`
	Roles             []models.Role                `json:"roles"` // Or RoleDTOs
	GlobalPermissions []string                     `json:"global_permissions"`
	AlbumPermissions  []models.UserAlbumPermission `json:"album_permissions"` // Direct album perms
	CreatedAt         string                       `json:"created_at"`
	UpdatedAt         string                       `json:"updated_at"`
}

func toUserResponseDTO(user *models.User, userAlbumPerms []models.UserAlbumPermission) UserResponseDTO {
	// Ensure Roles are loaded if user.Roles is nil but should be populated
	// This depends on how user was fetched. For now, assume it's populated.
	roles := []models.Role{}
	if user.Roles != nil {
		for _, r := range user.Roles {
			if r != nil {
				roles = append(roles, *r)
			}
		}
	}

	return UserResponseDTO{
		ID:                user.ID,
		Username:          user.Username,
		Roles:             roles,
		GlobalPermissions: user.GlobalPermissions,
		AlbumPermissions:  userAlbumPerms, // Pass the separately fetched direct album perms
		CreatedAt:         user.CreatedAt.Format(http.TimeFormat),
		UpdatedAt:         user.UpdatedAt.Format(http.TimeFormat),
	}
}

func toUserListResponseDTO(users []models.User) []UserResponseDTO {
	dtos := make([]UserResponseDTO, len(users))
	for i, user := range users {
		// For list view, we might not need to fetch individual album perms for each user
		// or roles in full detail to keep it performant.
		// This is a simplification; a real app might need more optimized queries.
		// For now, assuming user objects in the list have roles preloaded if needed.
		dtos[i] = toUserResponseDTO(&user, nil) // Pass nil for album perms in list view for now
	}
	return dtos
}

// --- Handler Methods ---

// ListUsers godoc
// @Summary List all users
// @Description Get a list of all users
// @Tags admin-users
// @Produce json
// @Success 200 {array} UserResponseDTO
// @Failure 500 {object} map[string]string
// @Router /api/admin/users [get]
// @Security BearerAuth
func (h *AdminUserHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	users, err := h.UserRepo.ListAll() // Uses the new ListAll method
	if err != nil {
		http.Error(w, "Failed to retrieve users: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// The toUserListResponseDTO already handles converting []models.User to []UserResponseDTO.
	// It also correctly passes nil for album permissions in the list view for now.
	// If detailed album permissions per user are needed in the list, toUserListResponseDTO
	// and UserRepo.ListAll() would need adjustment for preloading or N+1 fetching.
	responseDTOs := toUserListResponseDTO(users)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(responseDTOs); err != nil {
		fmt.Printf("Error encoding JSON response for ListUsers: %v\n", err)
	}
}

// GetUser godoc
// @Summary Get a single user by ID
// @Description Get details of a specific user by their ID
// @Tags admin-users
// @Produce json
// @Param id path int true "User ID"
// @Success 200 {object} UserResponseDTO
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/admin/users/{id} [get]
// @Security BearerAuth
func (h *AdminUserHandler) GetUser(w http.ResponseWriter, r *http.Request) {
	userIDStr := chi.URLParam(r, "id")
	userID, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		http.Error(w, "Invalid user ID format", http.StatusBadRequest)
		return
	}

	user, err := h.UserRepo.GetByID(uint(userID))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			http.Error(w, "User not found", http.StatusNotFound)
		} else {
			http.Error(w, "Failed to retrieve user: "+err.Error(), http.StatusInternalServerError)
		}
		return
	}
	userAlbumPerms, _ := h.UserRepo.GetUserAlbumPermissions(user.ID) // Fetch direct album perms

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(toUserResponseDTO(user, userAlbumPerms)); err != nil {
		fmt.Printf("Error encoding JSON response for GetUser: %v\n", err)
	}
}

// CreateUser godoc
// @Summary Create a new user
// @Description Add a new user to the system
// @Tags admin-users
// @Accept json
// @Produce json
// @Param user body UserCreatePayload true "User creation payload"
// @Success 201 {object} UserResponseDTO
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/admin/users [post]
// @Security BearerAuth
func (h *AdminUserHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	var payload UserCreatePayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "Invalid request payload: "+err.Error(), http.StatusBadRequest)
		return
	}

	if payload.Username == "" || payload.Password == "" {
		http.Error(w, "Username and password are required", http.StatusBadRequest)
		return
	}

	// Validate global permissions
	for _, pKey := range payload.GlobalPermissions {
		if !permissions.IsValidPermissionKey(pKey) {
			http.Error(w, fmt.Sprintf("Invalid global permission key: %s", pKey), http.StatusBadRequest)
			return
		}
		// Further check if scope is global (optional, as HasGlobalPermission handles it)
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(payload.Password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "Failed to hash password: "+err.Error(), http.StatusInternalServerError)
		return
	}

	user := &models.User{
		Username:          payload.Username,
		PasswordHash:      string(hashedPassword),
		GlobalPermissions: payload.GlobalPermissions,
	}

	// Validate and fetch roles
	if len(payload.RoleIDs) > 0 {
		user.Roles = make([]*models.Role, 0, len(payload.RoleIDs))
		for _, roleID := range payload.RoleIDs {
			role, err := h.RoleRepo.GetByID(roleID)
			if err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					http.Error(w, fmt.Sprintf("Role with ID %d not found", roleID), http.StatusBadRequest)
				} else {
					http.Error(w, fmt.Sprintf("Failed to retrieve role %d: %s", roleID, err.Error()), http.StatusInternalServerError)
				}
				return
			}
			user.Roles = append(user.Roles, role)
		}
	}

	if err := h.UserRepo.Create(user); err != nil {
		// Could be a unique constraint violation for username
		http.Error(w, "Failed to create user: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Reload user to get ID and populated fields like CreatedAt, UpdatedAt, and preloaded roles
	createdUser, err := h.UserRepo.GetByUsername(user.Username)
	if err != nil {
		http.Error(w, "Failed to retrieve newly created user: "+err.Error(), http.StatusInternalServerError)
		return
	}
	userAlbumPerms, _ := h.UserRepo.GetUserAlbumPermissions(createdUser.ID)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(toUserResponseDTO(createdUser, userAlbumPerms)); err != nil {
		fmt.Printf("Error encoding JSON response for CreateUser: %v\n", err)
	}
}

// UpdateUser godoc
// @Summary Update an existing user
// @Description Update details of an existing user
// @Tags admin-users
// @Accept json
// @Produce json
// @Param id path int true "User ID"
// @Param user body UserUpdatePayload true "User update payload"
// @Success 200 {object} UserResponseDTO
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/admin/users/{id} [put]
// @Security BearerAuth
func (h *AdminUserHandler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	userIDStr := chi.URLParam(r, "id")
	userID, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		http.Error(w, "Invalid user ID format", http.StatusBadRequest)
		return
	}

	var payload UserUpdatePayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "Invalid request payload: "+err.Error(), http.StatusBadRequest)
		return
	}

	user, err := h.UserRepo.GetByID(uint(userID))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			http.Error(w, "User not found", http.StatusNotFound)
		} else {
			http.Error(w, "Failed to retrieve user for update: "+err.Error(), http.StatusInternalServerError)
		}
		return
	}

	if payload.Username != nil {
		user.Username = *payload.Username
	}
	if payload.Password != nil && *payload.Password != "" {
		if err := user.SetPassword(*payload.Password); err != nil {
			http.Error(w, "Failed to set new password: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}
	if payload.GlobalPermissions != nil {
		// Validate global permissions
		for _, pKey := range *payload.GlobalPermissions {
			if !permissions.IsValidPermissionKey(pKey) {
				http.Error(w, fmt.Sprintf("Invalid global permission key: %s", pKey), http.StatusBadRequest)
				return
			}
		}
		user.GlobalPermissions = *payload.GlobalPermissions
	}

	// Handle RoleIDs update: Replace all existing roles with the new set
	if payload.RoleIDs != nil {
		newRoles := make([]*models.Role, 0, len(*payload.RoleIDs))
		for _, roleID := range *payload.RoleIDs {
			role, err := h.RoleRepo.GetByID(roleID)
			if err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					http.Error(w, fmt.Sprintf("Role with ID %d not found for update", roleID), http.StatusBadRequest)
				} else {
					http.Error(w, fmt.Sprintf("Failed to retrieve role %d for update: %s", roleID, err.Error()), http.StatusInternalServerError)
				}
				return
			}
			newRoles = append(newRoles, role)
		}
		user.Roles = newRoles // This will trigger GORM to update the associations
	}

	if err := h.UserRepo.Update(user); err != nil {
		http.Error(w, "Failed to update user: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Reload user to get updated fields and associations
	updatedUser, err := h.UserRepo.GetByID(user.ID)
	if err != nil {
		http.Error(w, "Failed to retrieve updated user: "+err.Error(), http.StatusInternalServerError)
		return
	}
	userAlbumPerms, _ := h.UserRepo.GetUserAlbumPermissions(updatedUser.ID)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(toUserResponseDTO(updatedUser, userAlbumPerms)); err != nil {
		fmt.Printf("Error encoding JSON response for UpdateUser: %v\n", err)
	}
}

// DeleteUser godoc
// @Summary Delete a user
// @Description Remove a user from the system
// @Tags admin-users
// @Param id path int true "User ID"
// @Success 204 "No Content"
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/admin/users/{id} [delete]
// @Security BearerAuth
func (h *AdminUserHandler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	userIDStr := chi.URLParam(r, "id")
	userID, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		http.Error(w, "Invalid user ID format", http.StatusBadRequest)
		return
	}

	// Optional: Check if user exists before attempting delete, though Delete might handle it.
	_, err = h.UserRepo.GetByID(uint(userID))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Failed to check user before delete: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if err := h.UserRepo.Delete(uint(userID)); err != nil {
		http.Error(w, "Failed to delete user: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// TODO: Add handlers for managing user's album-specific permissions
// e.g., POST /api/admin/users/{id}/album-permissions
// Body: { "album_id": 123, "permissions": ["album.photo.upload", "album.photo.delete"] }
// This would use UserRepo.CreateUserAlbumPermission or UpdateUserAlbumPermission

// TODO: Add handlers for managing user's roles more granularly if needed
// e.g., POST /api/admin/users/{id}/roles/{role_id} (Add role)
// DELETE /api/admin/users/{id}/roles/{role_id} (Remove role)
// The current UpdateUser replaces all roles, which is often simpler for UIs.
