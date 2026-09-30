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
	RoleRepo repository.RoleRepository
}

func NewAdminUserHandler(userRepo repository.UserRepository, roleRepo repository.RoleRepository) *AdminUserHandler {
	return &AdminUserHandler{UserRepo: userRepo, RoleRepo: roleRepo}
}

type UserCreatePayload struct {
	Username          string   `json:"username"`
	Password          string   `json:"password"`
	RoleIDs           []uint   `json:"role_ids"`
	GlobalPermissions []string `json:"global_permissions"`
	FirstName         string   `json:"first_name"`
	LastName          string   `json:"last_name"`
}

type UserUpdatePayload struct {
	Username          *string   `json:"username,omitempty"`
	Password          *string   `json:"password,omitempty"`
	RoleIDs           *[]uint   `json:"role_ids,omitempty"`
	GlobalPermissions *[]string `json:"global_permissions,omitempty"`
	FirstName         *string   `json:"first_name,omitempty"`
	LastName          *string   `json:"last_name,omitempty"`
}

// UserResponseDTO is a simplified User model for API responses
type UserResponseDTO struct {
	ID                uint                         `json:"id"`
	Username          string                       `json:"username"`
	FirstName         string                       `json:"first_name"`
	LastName          string                       `json:"last_name"`
	Roles             []models.Role                `json:"roles"`
	GlobalPermissions []string                     `json:"global_permissions"`
	AlbumPermissions  []models.UserAlbumPermission `json:"album_permissions"`
	CreatedAt         string                       `json:"created_at"`
	UpdatedAt         string                       `json:"updated_at"`
}

func toUserResponseDTO(user *models.User, userAlbumPerms []models.UserAlbumPermission) UserResponseDTO {
	// ensure Roles are loaded if user.Roles is nil but should be populated
	var roles []models.Role
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
		FirstName:         user.FirstName,
		LastName:          user.LastName,
		Roles:             roles,
		GlobalPermissions: user.GlobalPermissions,
		AlbumPermissions:  userAlbumPerms,
		CreatedAt:         user.CreatedAt.Format(http.TimeFormat),
		UpdatedAt:         user.UpdatedAt.Format(http.TimeFormat),
	}
}

func toUserListResponseDTO(users []models.User) []UserResponseDTO {
	dtos := make([]UserResponseDTO, len(users))
	for i, user := range users {
		dtos[i] = toUserResponseDTO(&user, nil) // TODO: pass nil for album perms in list view for now
	}
	return dtos
}

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
	params := ParsePaginationParams(r)
	users, err := h.UserRepo.ListAll()
	if err != nil {
		WriteAPIError(w, http.StatusInternalServerError, "UserListError", "Failed to retrieve users: "+err.Error())
		return
	}

	pagedUsers, meta := PaginateSlice(users, params)
	responseDTOs := toUserListResponseDTO(pagedUsers)
	WriteAPIPaginated(w, http.StatusOK, responseDTOs, meta)
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
		WriteAPIError(w, http.StatusBadRequest, "InvalidUserID", "Invalid user ID format")
		return
	}

	user, err := h.UserRepo.GetByID(uint(userID))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			WriteAPIError(w, http.StatusNotFound, "UserNotFound", "User not found")
		} else {
			WriteAPIError(w, http.StatusInternalServerError, "UserFetchError", "Failed to retrieve user: "+err.Error())
		}
		return
	}
	userAlbumPerms, _ := h.UserRepo.GetUserAlbumPermissions(user.ID)

	WriteAPIResponse(w, http.StatusOK, toUserResponseDTO(user, userAlbumPerms))
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
// @Failure 403 {object} map[string]string "Caller lacks a permission or role being granted"
// @Failure 500 {object} map[string]string
// @Router /api/admin/users [post]
// @Security BearerAuth
func (h *AdminUserHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	var payload UserCreatePayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		WriteAPIError(w, http.StatusBadRequest, "InvalidPayload", "Invalid request payload: "+err.Error())
		return
	}

	if payload.Username == "" || payload.Password == "" || payload.FirstName == "" || payload.LastName == "" {
		WriteAPIError(w, http.StatusBadRequest, "ValidationError", "Username, password, first_name, and last_name are required")
		return
	}

	for _, pKey := range payload.GlobalPermissions {
		if !permissions.IsValidPermissionKey(pKey) {
			WriteAPIError(w, http.StatusBadRequest, "ValidationError", fmt.Sprintf("Invalid global permission key: %s", pKey))
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

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(payload.Password), bcrypt.DefaultCost)
	if err != nil {
		WriteAPIError(w, http.StatusInternalServerError, "HashingError", "Failed to hash password: "+err.Error())
		return
	}

	user := &models.User{
		Username:          payload.Username,
		PasswordHash:      string(hashedPassword),
		GlobalPermissions: payload.GlobalPermissions,
		FirstName:         payload.FirstName,
		LastName:          payload.LastName,
	}

	if len(payload.RoleIDs) > 0 {
		user.Roles = make([]*models.Role, 0, len(payload.RoleIDs))
		for _, roleID := range payload.RoleIDs {
			role, err := h.RoleRepo.GetByID(roleID)
			if err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					WriteAPIError(w, http.StatusBadRequest, "RoleNotFound", fmt.Sprintf("Role with ID %d not found", roleID))
				} else {
					WriteAPIError(w, http.StatusInternalServerError, "RoleFetchError", fmt.Sprintf("Failed to retrieve role %d: %s", roleID, err.Error()))
				}
				return
			}
			if msg := roleGrantDenial(caller, role); msg != "" {
				WriteAPIError(w, http.StatusForbidden, "ForbiddenRoleAssignment", msg)
				return
			}
			user.Roles = append(user.Roles, role)
		}
	}

	if err := h.UserRepo.Create(user); err != nil {
		WriteAPIError(w, http.StatusInternalServerError, "UserCreateError", "Failed to create user: "+err.Error())
		return
	}

	createdUser, err := h.UserRepo.GetByUsername(user.Username)
	if err != nil {
		WriteAPIError(w, http.StatusInternalServerError, "UserFetchError", "Failed to retrieve newly created user: "+err.Error())
		return
	}
	userAlbumPerms, _ := h.UserRepo.GetUserAlbumPermissions(createdUser.ID)

	WriteAPIResponse(w, http.StatusCreated, toUserResponseDTO(createdUser, userAlbumPerms))
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
// @Failure 403 {object} map[string]string "Caller lacks a permission or role being granted, or target is a Super Administrator"
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/admin/users/{id} [put]
// @Security BearerAuth
func (h *AdminUserHandler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	userIDStr := chi.URLParam(r, "id")
	userID, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		WriteAPIError(w, http.StatusBadRequest, "InvalidUserID", "Invalid user ID format")
		return
	}

	var payload UserUpdatePayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		WriteAPIError(w, http.StatusBadRequest, "InvalidPayload", "Invalid request payload: "+err.Error())
		return
	}

	user, err := h.UserRepo.GetByID(uint(userID))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			WriteAPIError(w, http.StatusNotFound, "UserNotFound", "User not found")
		} else {
			WriteAPIError(w, http.StatusInternalServerError, "UserFetchError", "Failed to retrieve user for update: "+err.Error())
		}
		return
	}

	caller, ok := requestUser(w, r)
	if !ok {
		return
	}
	if isSuperAdmin(user) && !isSuperAdmin(caller) {
		WriteAPIError(w, http.StatusForbidden, "ForbiddenUserUpdate", "Only a Super Administrator can edit a Super Administrator")
		return
	}

	if payload.Username != nil {
		user.Username = *payload.Username
	}
	if payload.Password != nil && *payload.Password != "" {
		if err := user.SetPassword(*payload.Password); err != nil {
			WriteAPIError(w, http.StatusInternalServerError, "HashingError", "Failed to set new password: "+err.Error())
			return
		}
		user.TokenVersion++
	}
	if payload.GlobalPermissions != nil {
		for _, pKey := range *payload.GlobalPermissions {
			if !permissions.IsValidPermissionKey(pKey) {
				WriteAPIError(w, http.StatusBadRequest, "ValidationError", fmt.Sprintf("Invalid global permission key: %s", pKey))
				return
			}
		}
		if msg := globalGrantDenial(caller, *payload.GlobalPermissions, user.GlobalPermissions); msg != "" {
			WriteAPIError(w, http.StatusForbidden, "ForbiddenPermissionGrant", msg)
			return
		}
		user.GlobalPermissions = *payload.GlobalPermissions
	}

	var newRoleIDs []uint
	if payload.RoleIDs != nil {
		newRoles := make([]*models.Role, 0, len(*payload.RoleIDs))
		for _, roleID := range *payload.RoleIDs {
			role, err := h.RoleRepo.GetByID(roleID)
			if err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					WriteAPIError(w, http.StatusBadRequest, "RoleNotFound", fmt.Sprintf("Role with ID %d not found for update", roleID))
				} else {
					WriteAPIError(w, http.StatusInternalServerError, "RoleFetchError", fmt.Sprintf("Failed to retrieve role %d for update: %s", roleID, err.Error()))
				}
				return
			}
			newRoles = append(newRoles, role)
		}
		// every role being added or removed must be one the caller may hand out
		changedRoles := append(roleSetDifference(newRoles, user.Roles), roleSetDifference(user.Roles, newRoles)...)
		for _, role := range changedRoles {
			if msg := roleGrantDenial(caller, role); msg != "" {
				WriteAPIError(w, http.StatusForbidden, "ForbiddenRoleAssignment", msg)
				return
			}
		}
		if isSuperAdmin(user) && !hasSuperAdminRole(newRoles) {
			last, err := h.isLastSuperAdmin()
			if err != nil {
				WriteAPIError(w, http.StatusInternalServerError, "RoleFetchError", "Failed to check Super Administrators: "+err.Error())
				return
			}
			if last {
				WriteAPIError(w, http.StatusForbidden, "ForbiddenLastSuperAdmin", "The last Super Administrator cannot lose the Super Administrator role")
				return
			}
		}
		user.Roles = newRoles
		newRoleIDs = make([]uint, 0, len(newRoles))
		for _, role := range newRoles {
			newRoleIDs = append(newRoleIDs, role.ID)
		}
	}

	if payload.FirstName != nil {
		user.FirstName = *payload.FirstName
	}
	if payload.LastName != nil {
		user.LastName = *payload.LastName
	}

	var updateErr error
	if payload.RoleIDs != nil {
		updateErr = h.UserRepo.UpdateWithRoles(user, newRoleIDs)
	} else {
		updateErr = h.UserRepo.Update(user)
	}
	if updateErr != nil {
		WriteAPIError(w, http.StatusInternalServerError, "UserUpdateError", "Failed to update user: "+updateErr.Error())
		return
	}

	// reload user to get updated fields and associations
	updatedUser, err := h.UserRepo.GetByID(user.ID)
	if err != nil {
		WriteAPIError(w, http.StatusInternalServerError, "UserFetchError", "Failed to retrieve updated user: "+err.Error())
		return
	}
	userAlbumPerms, _ := h.UserRepo.GetUserAlbumPermissions(updatedUser.ID)

	WriteAPIResponse(w, http.StatusOK, toUserResponseDTO(updatedUser, userAlbumPerms))
}

// roleSetDifference returns the roles in a whose IDs are not present in b.
func roleSetDifference(a, b []*models.Role) []*models.Role {
	inB := make(map[uint]struct{}, len(b))
	for _, role := range b {
		if role != nil {
			inB[role.ID] = struct{}{}
		}
	}
	var diff []*models.Role
	for _, role := range a {
		if role == nil {
			continue
		}
		if _, ok := inB[role.ID]; !ok {
			diff = append(diff, role)
		}
	}
	return diff
}

func hasSuperAdminRole(roles []*models.Role) bool {
	return isSuperAdmin(&models.User{Roles: roles})
}

// isLastSuperAdmin reports whether at most one user holds the Super Administrator role.
func (h *AdminUserHandler) isLastSuperAdmin() (bool, error) {
	role, err := h.RoleRepo.GetByName(models.SuperAdminRoleName)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, err
	}
	users, err := h.RoleRepo.FindUsersByRoleID(role.ID)
	if err != nil {
		return false, err
	}
	return len(users) <= 1, nil
}

// DeleteUser godoc
// @Summary Delete a user
// @Description Remove a user from the system
// @Tags admin-users
// @Param id path int true "User ID"
// @Success 204 "No Content"
// @Failure 400 {object} map[string]string
// @Failure 403 {object} map[string]string "Cannot delete yourself or a Super Administrator (or the last one)"
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/admin/users/{id} [delete]
// @Security BearerAuth
func (h *AdminUserHandler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	userIDStr := chi.URLParam(r, "id")
	userID, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		WriteAPIError(w, http.StatusBadRequest, "InvalidUserID", "Invalid user ID format")
		return
	}

	target, err := h.UserRepo.GetByID(uint(userID))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			WriteAPIError(w, http.StatusNotFound, "UserNotFound", "User not found")
			return
		}
		WriteAPIError(w, http.StatusInternalServerError, "UserFetchError", "Failed to check user before delete: "+err.Error())
		return
	}

	caller, ok := requestUser(w, r)
	if !ok {
		return
	}
	if caller.ID == target.ID {
		WriteAPIError(w, http.StatusForbidden, "ForbiddenSelfDelete", "You cannot delete your own account")
		return
	}
	if isSuperAdmin(target) {
		if !isSuperAdmin(caller) {
			WriteAPIError(w, http.StatusForbidden, "ForbiddenUserDelete", "Only a Super Administrator can delete a Super Administrator")
			return
		}
		last, err := h.isLastSuperAdmin()
		if err != nil {
			WriteAPIError(w, http.StatusInternalServerError, "RoleFetchError", "Failed to check Super Administrators: "+err.Error())
			return
		}
		if last {
			WriteAPIError(w, http.StatusForbidden, "ForbiddenLastSuperAdmin", "The last Super Administrator cannot be deleted")
			return
		}
	}

	if err := h.UserRepo.Delete(uint(userID)); err != nil {
		WriteAPIError(w, http.StatusInternalServerError, "UserDeleteError", "Failed to delete user: "+err.Error())
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
