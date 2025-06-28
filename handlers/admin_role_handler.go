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

type AdminRoleHandler struct {
	RoleRepo repository.RoleRepository
}

func NewAdminRoleHandler(roleRepo repository.RoleRepository) *AdminRoleHandler {
	return &AdminRoleHandler{RoleRepo: roleRepo}
}

// --- DTOs for Role Management ---

// RoleAlbumPermissionCreate is used within RoleCreatePayload
// to define album-specific permissions without needing an existing ID.
type RoleAlbumPermissionCreate struct {
	AlbumID     uint     `json:"album_id"`
	Permissions []string `json:"permissions"`
}

type RoleCreatePayload struct {
	Name                   string                      `json:"name"`
	GlobalPermissions      []string                    `json:"global_permissions"`
	GlobalAlbumPermissions []string                    `json:"global_album_permissions"`
	AlbumPermissions       []RoleAlbumPermissionCreate `json:"album_permissions"` // Simplified for creation
}

// RoleAlbumPermissionInput is used within RoleUpdatePayload
// It can include an ID for existing permissions or define new ones.
type RoleAlbumPermissionInput struct {
	ID          uint     `json:"id,omitempty"` // ID of existing RoleAlbumPermission to update
	AlbumID     uint     `json:"album_id"`     // Required
	Permissions []string `json:"permissions"`  // Required
}

type RoleUpdatePayload struct {
	Name                   *string                     `json:"name,omitempty"`
	GlobalPermissions      *[]string                   `json:"global_permissions,omitempty"`
	GlobalAlbumPermissions *[]string                   `json:"global_album_permissions,omitempty"`
	AlbumPermissions       *[]RoleAlbumPermissionInput `json:"album_permissions,omitempty"` // For full replacement
}

// RoleResponseDTO is a simplified Role model for API responses.
type RoleResponseDTO struct {
	ID                     uint                         `json:"id"`
	Name                   string                       `json:"name"`
	GlobalPermissions      []string                     `json:"global_permissions"`
	GlobalAlbumPermissions []string                     `json:"global_album_permissions"`
	AlbumPermissions       []models.RoleAlbumPermission `json:"album_permissions"` // Full RoleAlbumPermission objects
	CreatedAt              string                       `json:"created_at"`
	UpdatedAt              string                       `json:"updated_at"`
	Users                  []UserSummaryDTO             `json:"users,omitempty"` // Optional list of users in the role
}

// UserSummaryDTO is a very minimal user representation for embedding in other responses.
type UserSummaryDTO struct {
	ID       uint   `json:"id"`
	Username string `json:"username"`
}

func toUserSummaryDTO(user models.User) UserSummaryDTO {
	return UserSummaryDTO{
		ID:       user.ID,
		Username: user.Username,
	}
}

func toUserSummaryListDTO(users []models.User) []UserSummaryDTO {
	dtos := make([]UserSummaryDTO, len(users))
	for i, user := range users {
		dtos[i] = toUserSummaryDTO(user)
	}
	return dtos
}

func toRoleResponseDTO(role *models.Role) RoleResponseDTO {
	// AlbumPermissions should be preloaded by RoleRepo methods
	return RoleResponseDTO{
		ID:                     role.ID,
		Name:                   role.Name,
		GlobalPermissions:      role.GlobalPermissions,
		GlobalAlbumPermissions: role.GlobalAlbumPermissions,
		AlbumPermissions:       role.AlbumPermissions,
		CreatedAt:              role.CreatedAt.Format(http.TimeFormat),
		UpdatedAt:              role.UpdatedAt.Format(http.TimeFormat),
		// Users are not populated by default, only on specific requests
	}
}

func toRoleListResponseDTO(roles []models.Role) []RoleResponseDTO {
	dtos := make([]RoleResponseDTO, len(roles))
	for i, role := range roles {
		dtos[i] = toRoleResponseDTO(&role)
	}
	return dtos
}

// --- Handler Methods ---

// ListRoles godoc
// @Summary List all roles
// @Description Get a list of all roles with their permissions
// @Tags admin-roles
// @Produce json
// @Success 200 {array} RoleResponseDTO
// @Failure 500 {object} map[string]string
// @Router /api/admin/roles [get]
// @Security BearerAuth
func (h *AdminRoleHandler) ListRoles(w http.ResponseWriter, r *http.Request) {
	roles, err := h.RoleRepo.ListAll() // Assumes ListAll preloads AlbumPermissions
	if err != nil {
		http.Error(w, "Failed to retrieve roles: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(toRoleListResponseDTO(roles)); err != nil {
		fmt.Printf("Error encoding JSON response for ListRoles: %v\n", err)
	}
}

// GetRole godoc
// @Summary Get a single role by ID
// @Description Get details of a specific role by its ID, including all its permissions
// @Tags admin-roles
// @Produce json
// @Param id path int true "Role ID"
// @Success 200 {object} RoleResponseDTO
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/admin/roles/{id} [get]
// @Security BearerAuth
func (h *AdminRoleHandler) GetRole(w http.ResponseWriter, r *http.Request) {
	roleIDStr := chi.URLParam(r, "roleID")
	roleID, err := strconv.ParseUint(roleIDStr, 10, 32)
	if err != nil {
		http.Error(w, "Invalid role ID format", http.StatusBadRequest)
		return
	}

	role, err := h.RoleRepo.GetByID(uint(roleID)) // Assumes GetByID preloads AlbumPermissions
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			http.Error(w, "Role not found", http.StatusNotFound)
		} else {
			http.Error(w, "Failed to retrieve role: "+err.Error(), http.StatusInternalServerError)
		}
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(toRoleResponseDTO(role)); err != nil {
		fmt.Printf("Error encoding JSON response for GetRole: %v\n", err)
	}
}

// CreateRole godoc
// @Summary Create a new role
// @Description Add a new role to the system with specified global and album permissions
// @Tags admin-roles
// @Accept json
// @Produce json
// @Param role body RoleCreatePayload true "Role creation payload"
// @Success 201 {object} RoleResponseDTO
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/admin/roles [post]
// @Security BearerAuth
func (h *AdminRoleHandler) CreateRole(w http.ResponseWriter, r *http.Request) {
	var payload RoleCreatePayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "Invalid request payload: "+err.Error(), http.StatusBadRequest)
		return
	}

	if payload.Name == "" {
		http.Error(w, "Role name is required", http.StatusBadRequest)
		return
	}
	// Prevent creating another role with the Super Admin name
	if payload.Name == models.SuperAdminRoleName {
		http.Error(w, fmt.Sprintf("Role name '%s' is reserved.", models.SuperAdminRoleName), http.StatusBadRequest)
		return
	}

	// Validate global permissions
	for _, pKey := range payload.GlobalPermissions {
		permDef, ok := permissions.GetPermissionDefinition(pKey)
		if !ok {
			http.Error(w, fmt.Sprintf("Invalid global permission key: %s", pKey), http.StatusBadRequest)
			return
		}
		if permDef.Scope != permissions.ScopeGlobal {
			http.Error(w, fmt.Sprintf("Permission '%s' is not a global permission", pKey), http.StatusBadRequest)
			return
		}
	}

	// Validate global album permissions
	for _, pKey := range payload.GlobalAlbumPermissions {
		permDef, ok := permissions.GetPermissionDefinition(pKey)
		if !ok {
			http.Error(w, fmt.Sprintf("Invalid album permission key: %s", pKey), http.StatusBadRequest)
			return
		}
		if permDef.Scope != permissions.ScopeAlbum {
			http.Error(w, fmt.Sprintf("Permission '%s' is not an album-scoped permission", pKey), http.StatusBadRequest)
			return
		}
	}

	role := &models.Role{
		Name:                   payload.Name,
		GlobalPermissions:      payload.GlobalPermissions,
		GlobalAlbumPermissions: payload.GlobalAlbumPermissions,
	}

	// Create the role first to get an ID
	if err := h.RoleRepo.Create(role); err != nil {
		http.Error(w, "Failed to create role: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Now handle album permissions
	createdAlbumPermissions := []models.RoleAlbumPermission{}
	for _, apPayload := range payload.AlbumPermissions {
		// Validate album permissions
		for _, pKey := range apPayload.Permissions {
			permDef, ok := permissions.GetPermissionDefinition(pKey)
			if !ok {
				http.Error(w, fmt.Sprintf("Invalid album permission key: %s for album %d", pKey, apPayload.AlbumID), http.StatusBadRequest)
				return
			}
			if permDef.Scope != permissions.ScopeAlbum {
				http.Error(w, fmt.Sprintf("Permission %s is not an album-specific permission for album %d", pKey, apPayload.AlbumID), http.StatusBadRequest)
				return
			}
		}

		rap := &models.RoleAlbumPermission{
			RoleID:      role.ID,
			AlbumID:     apPayload.AlbumID,
			Permissions: apPayload.Permissions,
		}
		if err := h.RoleRepo.CreateRoleAlbumPermission(rap); err != nil {
			// Attempt to clean up the created role if subsequent album perm creation fails
			_ = h.RoleRepo.Delete(role.ID)
			http.Error(w, fmt.Sprintf("Failed to create album permission for album %d: %s", apPayload.AlbumID, err.Error()), http.StatusInternalServerError)
			return
		}
		createdAlbumPermissions = append(createdAlbumPermissions, *rap)
	}
	role.AlbumPermissions = createdAlbumPermissions

	// Reload the role to get all associations correctly populated by GORM
	reloadedRole, err := h.RoleRepo.GetByID(role.ID)
	if err != nil {
		http.Error(w, "Failed to retrieve newly created role with associations: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(toRoleResponseDTO(reloadedRole)); err != nil {
		fmt.Printf("Error encoding JSON response for CreateRole: %v\n", err)
	}
}

// UpdateRole godoc
// @Summary Update an existing role
// @Description Update details of an existing role, including its global and album-specific permissions.
// @Description Album permissions are fully replaced by the provided set. The Super Administrator role cannot be modified.
// @Tags admin-roles
// @Accept json
// @Produce json
// @Param id path int true "Role ID"
// @Param role body RoleUpdatePayload true "Role update payload"
// @Success 200 {object} RoleResponseDTO
// @Failure 400 {object} map[string]string
// @Failure 403 {object} map[string]string "Forbidden to modify Super Administrator role"
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/admin/roles/{id} [put]
// @Security BearerAuth
func (h *AdminRoleHandler) UpdateRole(w http.ResponseWriter, r *http.Request) {
	roleIDStr := chi.URLParam(r, "roleID")
	roleID, err := strconv.ParseUint(roleIDStr, 10, 32)
	if err != nil {
		http.Error(w, "Invalid role ID format", http.StatusBadRequest)
		return
	}

	var payload RoleUpdatePayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "Invalid request payload: "+err.Error(), http.StatusBadRequest)
		return
	}

	role, err := h.RoleRepo.GetByID(uint(roleID))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			http.Error(w, "Role not found", http.StatusNotFound)
		} else {
			http.Error(w, "Failed to retrieve role for update: "+err.Error(), http.StatusInternalServerError)
		}
		return
	}

	// Prevent any modification of the Super Administrator role
	if role.Name == models.SuperAdminRoleName {
		http.Error(w, "The Super Administrator role cannot be modified.", http.StatusForbidden)
		return
	}

	// Proceed with updates for other roles
	if payload.Name != nil {
		// Prevent renaming another role to "Super Administrator"
		if *payload.Name == models.SuperAdminRoleName && role.Name != models.SuperAdminRoleName {
			http.Error(w, fmt.Sprintf("Role name '%s' is reserved.", models.SuperAdminRoleName), http.StatusBadRequest)
			return
		}
		role.Name = *payload.Name
	}

	if payload.GlobalPermissions != nil {
		// Validate global permissions
		for _, pKey := range *payload.GlobalPermissions {
			permDef, ok := permissions.GetPermissionDefinition(pKey)
			if !ok {
				http.Error(w, fmt.Sprintf("Invalid global permission key: %s", pKey), http.StatusBadRequest)
				return
			}
			if permDef.Scope != permissions.ScopeGlobal {
				http.Error(w, fmt.Sprintf("Permission '%s' is not a global permission", pKey), http.StatusBadRequest)
				return
			}
		}
		role.GlobalPermissions = *payload.GlobalPermissions
	}

	if payload.GlobalAlbumPermissions != nil {
		// Validate global album permissions
		for _, pKey := range *payload.GlobalAlbumPermissions {
			permDef, ok := permissions.GetPermissionDefinition(pKey)
			if !ok {
				http.Error(w, fmt.Sprintf("Invalid album permission key: %s", pKey), http.StatusBadRequest)
				return
			}
			if permDef.Scope != permissions.ScopeAlbum {
				http.Error(w, fmt.Sprintf("Permission '%s' is not an album-scoped permission", pKey), http.StatusBadRequest)
				return
			}
		}
		role.GlobalAlbumPermissions = *payload.GlobalAlbumPermissions
	}

	// Handle AlbumPermissions update: Full replacement
	if payload.AlbumPermissions != nil {
		// 1. Delete all existing RoleAlbumPermissions for this role
		existingRaps, err := h.RoleRepo.GetRoleAlbumPermissions(role.ID)
		if err != nil {
			http.Error(w, "Failed to retrieve existing album permissions for update: "+err.Error(), http.StatusInternalServerError)
			return
		}
		for _, existingRap := range existingRaps {
			if err := h.RoleRepo.DeleteRoleAlbumPermission(role.ID, existingRap.AlbumID); err != nil {
				http.Error(w, fmt.Sprintf("Failed to delete existing album permission for album %d: %s", existingRap.AlbumID, err.Error()), http.StatusInternalServerError)
				return
			}
		}

		// 2. Add new RoleAlbumPermissions from payload
		newAlbumPermissions := []models.RoleAlbumPermission{}
		for _, apInput := range *payload.AlbumPermissions {
			// Validate album permissions
			for _, pKey := range apInput.Permissions {
				permDef, ok := permissions.GetPermissionDefinition(pKey)
				if !ok {
					http.Error(w, fmt.Sprintf("Invalid album permission key: %s for album %d", pKey, apInput.AlbumID), http.StatusBadRequest)
					return
				}
				if permDef.Scope != permissions.ScopeAlbum {
					http.Error(w, fmt.Sprintf("Permission %s is not an album-specific permission for album %d", pKey, apInput.AlbumID), http.StatusBadRequest)
					return
				}
			}
			rap := &models.RoleAlbumPermission{
				RoleID:      role.ID,
				AlbumID:     apInput.AlbumID,
				Permissions: apInput.Permissions,
			}
			// Use CreateRoleAlbumPermission which handles conflicts (upsert-like)
			if err := h.RoleRepo.CreateRoleAlbumPermission(rap); err != nil {
				http.Error(w, fmt.Sprintf("Failed to create/update album permission for album %d: %s", apInput.AlbumID, err.Error()), http.StatusInternalServerError)
				return
			}
			createdRap, _ := h.RoleRepo.GetRoleAlbumPermission(role.ID, apInput.AlbumID)
			if createdRap != nil {
				newAlbumPermissions = append(newAlbumPermissions, *createdRap)
			}
		}
		role.AlbumPermissions = newAlbumPermissions
	}

	// Update the role's direct fields (Name, GlobalPermissions, etc.)
	if err := h.RoleRepo.Update(role); err != nil {
		http.Error(w, "Failed to update role: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Reload the role to get all associations correctly populated
	updatedRole, err := h.RoleRepo.GetByID(role.ID)
	if err != nil {
		http.Error(w, "Failed to retrieve updated role with associations: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(toRoleResponseDTO(updatedRole)); err != nil {
		fmt.Printf("Error encoding JSON response for UpdateRole: %v\n", err)
	}
}

// DeleteRole godoc
// @Summary Delete a role
// @Description Remove a role from the system. This also removes its assignments to users. The Super Administrator role cannot be deleted.
// @Tags admin-roles
// @Param id path int true "Role ID"
// @Success 204 "No Content"
// @Failure 400 {object} map[string]string
// @Failure 403 {object} map[string]string "Forbidden to delete Super Administrator role"
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/admin/roles/{id} [delete]
// @Security BearerAuth
func (h *AdminRoleHandler) DeleteRole(w http.ResponseWriter, r *http.Request) {
	roleIDStr := chi.URLParam(r, "roleID")
	roleID, err := strconv.ParseUint(roleIDStr, 10, 32)
	if err != nil {
		http.Error(w, "Invalid role ID format", http.StatusBadRequest)
		return
	}

	// Check if role exists
	role, err := h.RoleRepo.GetByID(uint(roleID))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			http.Error(w, "Role not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Failed to check role before delete: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Prevent deletion of the Super Administrator role
	if role.Name == models.SuperAdminRoleName {
		http.Error(w, "The Super Administrator role cannot be deleted.", http.StatusForbidden)
		return
	}

	if err := h.RoleRepo.Delete(uint(roleID)); err != nil {
		http.Error(w, "Failed to delete role: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// --- User-Role Association Handlers ---

// GetRoleUsers godoc
// @Summary Get users assigned to a role
// @Description Get a list of all users who are assigned to a specific role
// @Tags admin-roles
// @Produce json
// @Param id path int true "Role ID"
// @Success 200 {array} UserSummaryDTO
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/admin/roles/{id}/users [get]
// @Security BearerAuth
func (h *AdminRoleHandler) GetRoleUsers(w http.ResponseWriter, r *http.Request) {
	roleIDStr := chi.URLParam(r, "roleID")
	roleID, err := strconv.ParseUint(roleIDStr, 10, 32)
	if err != nil {
		http.Error(w, "Invalid role ID format", http.StatusBadRequest)
		return
	}

	// Check if role exists first
	if _, err := h.RoleRepo.GetByID(uint(roleID)); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			http.Error(w, "Role not found", http.StatusNotFound)
		} else {
			http.Error(w, "Failed to verify role existence: "+err.Error(), http.StatusInternalServerError)
		}
		return
	}

	users, err := h.RoleRepo.FindUsersByRoleID(uint(roleID))
	if err != nil {
		http.Error(w, "Failed to retrieve users for role: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(toUserSummaryListDTO(users)); err != nil {
		fmt.Printf("Error encoding JSON response for GetRoleUsers: %v\n", err)
	}
}

type AddUserToRolePayload struct {
	UserID uint `json:"user_id"`
}

// AddUserToRole godoc
// @Summary Assign a user to a role
// @Description Assign a user to a role. The Super Administrator role cannot be assigned.
// @Tags admin-roles
// @Accept json
// @Produce json
// @Param id path int true "Role ID"
// @Param payload body AddUserToRolePayload true "User ID to add"
// @Success 204 "No Content"
// @Failure 400 {object} map[string]string
// @Failure 403 {object} map[string]string "Forbidden to modify Super Administrator role"
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/admin/roles/{id}/users [post]
// @Security BearerAuth
func (h *AdminRoleHandler) AddUserToRole(w http.ResponseWriter, r *http.Request) {
	roleIDStr := chi.URLParam(r, "roleID")
	roleID, err := strconv.ParseUint(roleIDStr, 10, 32)
	if err != nil {
		http.Error(w, "Invalid role ID format", http.StatusBadRequest)
		return
	}

	var payload AddUserToRolePayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "Invalid request payload: "+err.Error(), http.StatusBadRequest)
		return
	}

	if payload.UserID == 0 {
		http.Error(w, "User ID is required", http.StatusBadRequest)
		return
	}

	// Prevent assignment to the Super Administrator role
	role, err := h.RoleRepo.GetByID(uint(roleID))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			http.Error(w, "Role not found", http.StatusNotFound)
		} else {
			http.Error(w, "Failed to retrieve role: "+err.Error(), http.StatusInternalServerError)
		}
		return
	}
	if role.Name == models.SuperAdminRoleName {
		http.Error(w, "The Super Administrator role cannot be manually assigned.", http.StatusForbidden)
		return
	}

	// TODO: Check if user exists before adding. This requires a UserRepository.
	// For now, we assume the frontend sends a valid user ID.

	if err := h.RoleRepo.AddUserToRole(payload.UserID, uint(roleID)); err != nil {
		http.Error(w, "Failed to add user to role: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// RemoveUserFromRole godoc
// @Summary Remove a user from a role
// @Description Remove a user's assignment from a role. The Super Administrator role cannot be modified.
// @Tags admin-roles
// @Param roleID path int true "Role ID"
// @Param userID path int true "User ID"
// @Success 204 "No Content"
// @Failure 400 {object} map[string]string
// @Failure 403 {object} map[string]string "Forbidden to modify Super Administrator role"
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/admin/roles/{roleID}/users/{userID} [delete]
// @Security BearerAuth
func (h *AdminRoleHandler) RemoveUserFromRole(w http.ResponseWriter, r *http.Request) {
	roleIDStr := chi.URLParam(r, "roleID")
	roleID, err := strconv.ParseUint(roleIDStr, 10, 32)
	if err != nil {
		http.Error(w, "Invalid role ID format", http.StatusBadRequest)
		return
	}

	userIDStr := chi.URLParam(r, "userID")
	userID, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		http.Error(w, "Invalid user ID format", http.StatusBadRequest)
		return
	}

	// Prevent modification of the Super Administrator role
	role, err := h.RoleRepo.GetByID(uint(roleID))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			http.Error(w, "Role not found", http.StatusNotFound)
		} else {
			http.Error(w, "Failed to retrieve role: "+err.Error(), http.StatusInternalServerError)
		}
		return
	}
	if role.Name == models.SuperAdminRoleName {
		http.Error(w, "Users cannot be removed from the Super Administrator role.", http.StatusForbidden)
		return
	}

	if err := h.RoleRepo.RemoveUserFromRole(uint(userID), uint(roleID)); err != nil {
		http.Error(w, "Failed to remove user from role: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
