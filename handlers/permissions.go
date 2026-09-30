package handlers

import (
	"net/http"

	"github.com/camden-git/mediasysbackend/permissions"
)

// PermissionsHandler serves the static permission definitions.
type PermissionsHandler struct{}

func NewPermissionsHandler() *PermissionsHandler {
	return &PermissionsHandler{}
}

// ListDefinedPermissions serves the statically defined permission groups and their permissions.
// This endpoint can be used by a UI to understand available permissions for assignment.
func (h *PermissionsHandler) ListDefinedPermissions(w http.ResponseWriter, r *http.Request) {
	WriteAPIResponse(w, http.StatusOK, permissions.DefinedPermissionGroups)
}

// ListDefinedPermissionKeys serves just the keys of all defined permissions.
func (h *PermissionsHandler) ListDefinedPermissionKeys(w http.ResponseWriter, r *http.Request) {
	WriteAPIResponse(w, http.StatusOK, permissions.GetAllPermissionKeys())
}
