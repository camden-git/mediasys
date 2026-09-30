package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/camden-git/mediasysbackend/models"
	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"
)

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
		WriteAPIError(w, http.StatusBadRequest, "InvalidRoleID", "Invalid role ID format")
		return
	}

	if _, err := h.RoleRepo.GetByID(uint(roleID)); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			WriteAPIError(w, http.StatusNotFound, "RoleNotFound", "Role not found")
		} else {
			WriteAPIError(w, http.StatusInternalServerError, "RoleFetchError", "Failed to verify role existence: "+err.Error())
		}
		return
	}

	users, err := h.RoleRepo.FindUsersByRoleID(uint(roleID))
	if err != nil {
		WriteAPIError(w, http.StatusInternalServerError, "RoleUserListError", "Failed to retrieve users for role: "+err.Error())
		return
	}

	params := ParsePaginationParams(r)
	pagedUsers, meta := PaginateSlice(users, params)
	WriteAPIPaginated(w, http.StatusOK, toUserSummaryListDTO(pagedUsers), meta)
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
		WriteAPIError(w, http.StatusBadRequest, "InvalidRoleID", "Invalid role ID format")
		return
	}

	var payload AddUserToRolePayload
	if err := decodeJSONBody(w, r, &payload); err != nil {
		if isBodyTooLarge(err) {
			WriteAPIError(w, http.StatusRequestEntityTooLarge, "PayloadTooLarge", "Request body is too large")
			return
		}
		WriteAPIError(w, http.StatusBadRequest, "InvalidPayload", "Invalid request payload: "+err.Error())
		return
	}

	if payload.UserID == 0 {
		WriteAPIError(w, http.StatusBadRequest, "ValidationError", "User ID is required")
		return
	}

	role, err := h.RoleRepo.GetByID(uint(roleID))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			WriteAPIError(w, http.StatusNotFound, "RoleNotFound", "Role not found")
		} else {
			WriteAPIError(w, http.StatusInternalServerError, "RoleFetchError", "Failed to retrieve role: "+err.Error())
		}
		return
	}
	if role.Name == models.SuperAdminRoleName {
		WriteAPIError(w, http.StatusForbidden, "ForbiddenRoleAssignment", "The Super Administrator role cannot be manually assigned.")
		return
	}

	caller, ok := requestUser(w, r)
	if !ok {
		return
	}
	if msg := roleGrantDenial(caller, role); msg != "" {
		WriteAPIError(w, http.StatusForbidden, "ForbiddenRoleAssignment", msg)
		return
	}

	if err := h.RoleRepo.AddUserToRole(payload.UserID, uint(roleID)); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			WriteAPIError(w, http.StatusNotFound, "UserNotFound", "User not found")
			return
		}
		WriteAPIError(w, http.StatusInternalServerError, "RoleAssignmentError", "Failed to add user to role: "+err.Error())
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
		WriteAPIError(w, http.StatusBadRequest, "InvalidRoleID", "Invalid role ID format")
		return
	}

	userIDStr := chi.URLParam(r, "userID")
	userID, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		WriteAPIError(w, http.StatusBadRequest, "InvalidUserID", "Invalid user ID format")
		return
	}

	role, err := h.RoleRepo.GetByID(uint(roleID))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			WriteAPIError(w, http.StatusNotFound, "RoleNotFound", "Role not found")
		} else {
			WriteAPIError(w, http.StatusInternalServerError, "RoleFetchError", "Failed to retrieve role: "+err.Error())
		}
		return
	}
	if role.Name == models.SuperAdminRoleName {
		WriteAPIError(w, http.StatusForbidden, "ForbiddenRoleAssignment", "Users cannot be removed from the Super Administrator role.")
		return
	}

	caller, ok := requestUser(w, r)
	if !ok {
		return
	}
	if msg := roleGrantDenial(caller, role); msg != "" {
		WriteAPIError(w, http.StatusForbidden, "ForbiddenRoleAssignment", msg)
		return
	}

	if err := h.RoleRepo.RemoveUserFromRole(uint(userID), uint(roleID)); err != nil {
		WriteAPIError(w, http.StatusInternalServerError, "RoleAssignmentError", "Failed to remove user from role: "+err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
