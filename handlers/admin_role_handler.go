package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

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

// RoleAlbumPermissionCreate is used within RoleCreatePayload to define album-specific permissions without needing an existing ID
type RoleAlbumPermissionCreate struct {
	AlbumID     uint     `json:"album_id"`
	Permissions []string `json:"permissions"`
}

type RoleCreatePayload struct {
	Name                   string                      `json:"name"`
	GlobalPermissions      []string                    `json:"global_permissions"`
	GlobalAlbumPermissions []string                    `json:"global_album_permissions"`
	AlbumPermissions       []RoleAlbumPermissionCreate `json:"album_permissions"`
}

// RoleAlbumPermissionInput is used within RoleUpdatePayload; it can include an ID for existing permissions or define new ones
type RoleAlbumPermissionInput struct {
	ID          uint     `json:"id,omitempty"` // ID of existing RoleAlbumPermission to update
	AlbumID     uint     `json:"album_id"`     // required
	Permissions []string `json:"permissions"`  // required
}

type RoleUpdatePayload struct {
	Name                   *string                     `json:"name,omitempty"`
	GlobalPermissions      *[]string                   `json:"global_permissions,omitempty"`
	GlobalAlbumPermissions *[]string                   `json:"global_album_permissions,omitempty"`
	AlbumPermissions       *[]RoleAlbumPermissionInput `json:"album_permissions,omitempty"`
}

// RoleResponseDTO is a simplified Role model for API responses
type RoleResponseDTO struct {
	ID                     uint                         `json:"id"`
	Name                   string                       `json:"name"`
	GlobalPermissions      []string                     `json:"global_permissions"`
	GlobalAlbumPermissions []string                     `json:"global_album_permissions"`
	AlbumPermissions       []models.RoleAlbumPermission `json:"album_permissions"`
	CreatedAt              string                       `json:"created_at"`
	UpdatedAt              string                       `json:"updated_at"`
	Users                  []UserSummaryDTO             `json:"users,omitempty"`
}

// UserSummaryDTO is a very minimal user representation for embedding in other responses
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
	return RoleResponseDTO{
		ID:                     role.ID,
		Name:                   role.Name,
		GlobalPermissions:      role.GlobalPermissions,
		GlobalAlbumPermissions: role.GlobalAlbumPermissions,
		AlbumPermissions:       role.AlbumPermissions,
		CreatedAt:              role.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:              role.UpdatedAt.UTC().Format(time.RFC3339),
	}
}

func toRoleListResponseDTO(roles []models.Role) []RoleResponseDTO {
	dtos := make([]RoleResponseDTO, len(roles))
	for i, role := range roles {
		dtos[i] = toRoleResponseDTO(&role)
	}
	return dtos
}

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
	params := ParsePaginationParams(r)
	roles, err := h.RoleRepo.ListAll()
	if err != nil {
		WriteAPIError(w, http.StatusInternalServerError, "RoleListError", "Failed to retrieve roles: "+err.Error())
		return
	}
	pagedRoles, meta := PaginateSlice(roles, params)
	WriteAPIPaginated(w, http.StatusOK, toRoleListResponseDTO(pagedRoles), meta)
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
		WriteAPIError(w, http.StatusBadRequest, "InvalidRoleID", "Invalid role ID format")
		return
	}

	role, err := h.RoleRepo.GetByID(uint(roleID)) // Assumes GetByID preloads AlbumPermissions
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			WriteAPIError(w, http.StatusNotFound, "RoleNotFound", "Role not found")
		} else {
			WriteAPIError(w, http.StatusInternalServerError, "RoleFetchError", "Failed to retrieve role: "+err.Error())
		}
		return
	}
	WriteAPIResponse(w, http.StatusOK, toRoleResponseDTO(role))
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
// @Failure 403 {object} map[string]string "Caller lacks a permission being granted"
// @Failure 500 {object} map[string]string
// @Router /api/admin/roles [post]
// @Security BearerAuth
func (h *AdminRoleHandler) CreateRole(w http.ResponseWriter, r *http.Request) {
	var payload RoleCreatePayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		WriteAPIError(w, http.StatusBadRequest, "InvalidPayload", "Invalid request payload: "+err.Error())
		return
	}

	if payload.Name == "" {
		WriteAPIError(w, http.StatusBadRequest, "ValidationError", "Role name is required")
		return
	}

	if payload.Name == models.SuperAdminRoleName {
		WriteAPIError(w, http.StatusBadRequest, "ValidationError", fmt.Sprintf("Role name '%s' is reserved.", models.SuperAdminRoleName))
		return
	}

	for _, pKey := range payload.GlobalPermissions {
		permDef, ok := permissions.GetPermissionDefinition(pKey)
		if !ok {
			WriteAPIError(w, http.StatusBadRequest, "ValidationError", fmt.Sprintf("Invalid global permission key: %s", pKey))
			return
		}
		if permDef.Scope != permissions.ScopeGlobal {
			WriteAPIError(w, http.StatusBadRequest, "ValidationError", fmt.Sprintf("Permission '%s' is not a global permission", pKey))
			return
		}
	}

	for _, pKey := range payload.GlobalAlbumPermissions {
		permDef, ok := permissions.GetPermissionDefinition(pKey)
		if !ok {
			WriteAPIError(w, http.StatusBadRequest, "ValidationError", fmt.Sprintf("Invalid album permission key: %s", pKey))
			return
		}
		if permDef.Scope != permissions.ScopeAlbum {
			WriteAPIError(w, http.StatusBadRequest, "ValidationError", fmt.Sprintf("Permission '%s' is not an album-scoped permission", pKey))
			return
		}
	}

	caller, ok := requestUser(w, r)
	if !ok {
		return
	}
	if msg := globalGrantDenial(caller, payload.GlobalPermissions, nil); msg != "" {
		WriteAPIError(w, http.StatusForbidden, "ForbiddenPermissionGrant", msg)
		return
	}
	if msg := albumForAllGrantDenial(caller, payload.GlobalAlbumPermissions, nil); msg != "" {
		WriteAPIError(w, http.StatusForbidden, "ForbiddenPermissionGrant", msg)
		return
	}
	albumPerms := make([]models.RoleAlbumPermission, 0, len(payload.AlbumPermissions))
	for _, apPayload := range payload.AlbumPermissions {
		for _, pKey := range apPayload.Permissions {
			permDef, ok := permissions.GetPermissionDefinition(pKey)
			if !ok {
				WriteAPIError(w, http.StatusBadRequest, "ValidationError", fmt.Sprintf("Invalid album permission key: %s for album %d", pKey, apPayload.AlbumID))
				return
			}
			if permDef.Scope != permissions.ScopeAlbum {
				WriteAPIError(w, http.StatusBadRequest, "ValidationError", fmt.Sprintf("Permission %s is not an album-specific permission for album %d", pKey, apPayload.AlbumID))
				return
			}
		}
		if msg := albumGrantDenial(caller, apPayload.AlbumID, apPayload.Permissions, nil); msg != "" {
			WriteAPIError(w, http.StatusForbidden, "ForbiddenPermissionGrant", msg)
			return
		}
		albumPerms = append(albumPerms, models.RoleAlbumPermission{AlbumID: apPayload.AlbumID, Permissions: apPayload.Permissions})
	}

	role := &models.Role{
		Name:                   payload.Name,
		GlobalPermissions:      payload.GlobalPermissions,
		GlobalAlbumPermissions: payload.GlobalAlbumPermissions,
	}

	if err := h.RoleRepo.CreateWithAlbumPermissions(role, albumPerms); err != nil {
		WriteAPIError(w, http.StatusInternalServerError, "RoleCreateError", "Failed to create role: "+err.Error())
		return
	}

	reloadedRole, err := h.RoleRepo.GetByID(role.ID)
	if err != nil {
		WriteAPIError(w, http.StatusInternalServerError, "RoleFetchError", "Failed to retrieve newly created role with associations: "+err.Error())
		return
	}

	WriteAPIResponse(w, http.StatusCreated, toRoleResponseDTO(reloadedRole))
}

// UpdateRole godoc
// @Summary Update an existing role
// @Description Update details of an existing role, including its global and album-specific permissions.
// @Description The provided set fully replaces album permissions. The Super Administrator role cannot be modified.
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
		WriteAPIError(w, http.StatusBadRequest, "InvalidRoleID", "Invalid role ID format")
		return
	}

	var payload RoleUpdatePayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		WriteAPIError(w, http.StatusBadRequest, "InvalidPayload", "Invalid request payload: "+err.Error())
		return
	}

	role, err := h.RoleRepo.GetByID(uint(roleID))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			WriteAPIError(w, http.StatusNotFound, "RoleNotFound", "Role not found")
		} else {
			WriteAPIError(w, http.StatusInternalServerError, "RoleFetchError", "Failed to retrieve role for update: "+err.Error())
		}
		return
	}

	if role.Name == models.SuperAdminRoleName {
		WriteAPIError(w, http.StatusForbidden, "ForbiddenRoleUpdate", "The Super Administrator role cannot be modified.")
		return
	}

	caller, ok := requestUser(w, r)
	if !ok {
		return
	}

	if payload.Name != nil {
		if *payload.Name == models.SuperAdminRoleName && role.Name != models.SuperAdminRoleName {
			WriteAPIError(w, http.StatusBadRequest, "ValidationError", fmt.Sprintf("Role name '%s' is reserved.", models.SuperAdminRoleName))
			return
		}
		role.Name = *payload.Name
	}

	if payload.GlobalPermissions != nil {
		for _, pKey := range *payload.GlobalPermissions {
			permDef, ok := permissions.GetPermissionDefinition(pKey)
			if !ok {
				WriteAPIError(w, http.StatusBadRequest, "ValidationError", fmt.Sprintf("Invalid global permission key: %s", pKey))
				return
			}
			if permDef.Scope != permissions.ScopeGlobal {
				WriteAPIError(w, http.StatusBadRequest, "ValidationError", fmt.Sprintf("Permission '%s' is not a global permission", pKey))
				return
			}
		}
		if msg := globalGrantDenial(caller, *payload.GlobalPermissions, role.GlobalPermissions); msg != "" {
			WriteAPIError(w, http.StatusForbidden, "ForbiddenPermissionGrant", msg)
			return
		}
		role.GlobalPermissions = *payload.GlobalPermissions
	}

	if payload.GlobalAlbumPermissions != nil {
		for _, pKey := range *payload.GlobalAlbumPermissions {
			permDef, ok := permissions.GetPermissionDefinition(pKey)
			if !ok {
				WriteAPIError(w, http.StatusBadRequest, "ValidationError", fmt.Sprintf("Invalid album permission key: %s", pKey))
				return
			}
			if permDef.Scope != permissions.ScopeAlbum {
				WriteAPIError(w, http.StatusBadRequest, "ValidationError", fmt.Sprintf("Permission '%s' is not an album-scoped permission", pKey))
				return
			}
		}
		if msg := albumForAllGrantDenial(caller, *payload.GlobalAlbumPermissions, role.GlobalAlbumPermissions); msg != "" {
			WriteAPIError(w, http.StatusForbidden, "ForbiddenPermissionGrant", msg)
			return
		}
		role.GlobalAlbumPermissions = *payload.GlobalAlbumPermissions
	}

	var newAlbumPerms *[]models.RoleAlbumPermission
	if payload.AlbumPermissions != nil {
		existingRaps, err := h.RoleRepo.GetRoleAlbumPermissions(role.ID)
		if err != nil {
			WriteAPIError(w, http.StatusInternalServerError, "RoleAlbumPermissionError", "Failed to retrieve existing album permissions for update: "+err.Error())
			return
		}
		existingByAlbum := make(map[uint][]string, len(existingRaps))
		for _, existingRap := range existingRaps {
			existingByAlbum[existingRap.AlbumID] = existingRap.Permissions
		}

		albumPerms := make([]models.RoleAlbumPermission, 0, len(*payload.AlbumPermissions))
		for _, apInput := range *payload.AlbumPermissions {
			for _, pKey := range apInput.Permissions {
				permDef, ok := permissions.GetPermissionDefinition(pKey)
				if !ok {
					WriteAPIError(w, http.StatusBadRequest, "ValidationError", fmt.Sprintf("Invalid album permission key: %s for album %d", pKey, apInput.AlbumID))
					return
				}
				if permDef.Scope != permissions.ScopeAlbum {
					WriteAPIError(w, http.StatusBadRequest, "ValidationError", fmt.Sprintf("Permission %s is not an album-specific permission for album %d", pKey, apInput.AlbumID))
					return
				}
			}
			if msg := albumGrantDenial(caller, apInput.AlbumID, apInput.Permissions, existingByAlbum[apInput.AlbumID]); msg != "" {
				WriteAPIError(w, http.StatusForbidden, "ForbiddenPermissionGrant", msg)
				return
			}
			albumPerms = append(albumPerms, models.RoleAlbumPermission{AlbumID: apInput.AlbumID, Permissions: apInput.Permissions})
		}
		newAlbumPerms = &albumPerms
	}

	if err := h.RoleRepo.UpdateWithAlbumPermissions(role, newAlbumPerms); err != nil {
		WriteAPIError(w, http.StatusInternalServerError, "RoleUpdateError", "Failed to update role: "+err.Error())
		return
	}

	updatedRole, err := h.RoleRepo.GetByID(role.ID)
	if err != nil {
		WriteAPIError(w, http.StatusInternalServerError, "RoleFetchError", "Failed to retrieve updated role with associations: "+err.Error())
		return
	}

	WriteAPIResponse(w, http.StatusOK, toRoleResponseDTO(updatedRole))
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
		WriteAPIError(w, http.StatusBadRequest, "InvalidRoleID", "Invalid role ID format")
		return
	}

	role, err := h.RoleRepo.GetByID(uint(roleID))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			WriteAPIError(w, http.StatusNotFound, "RoleNotFound", "Role not found")
			return
		}
		WriteAPIError(w, http.StatusInternalServerError, "RoleFetchError", "Failed to check role before delete: "+err.Error())
		return
	}

	if role.Name == models.SuperAdminRoleName {
		WriteAPIError(w, http.StatusForbidden, "ForbiddenRoleDelete", "The Super Administrator role cannot be deleted.")
		return
	}

	if err := h.RoleRepo.Delete(uint(roleID)); err != nil {
		WriteAPIError(w, http.StatusInternalServerError, "RoleDeleteError", "Failed to delete role: "+err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
