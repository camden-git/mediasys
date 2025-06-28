package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/camden-git/mediasysbackend/permissions"
)

type PermissionsHandler struct {
	// No dependencies needed for now, as it serves static data
}

func NewPermissionsHandler() *PermissionsHandler {
	return &PermissionsHandler{}
}

// ListDefinedPermissions serves the statically defined permission groups and their permissions.
// This endpoint can be used by a UI to understand available permissions for assignment.
func (h *PermissionsHandler) ListDefinedPermissions(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(permissions.DefinedPermissionGroups); err != nil {
		// Log the error internally
		// log.Printf("Error encoding defined permissions: %v", err)
		http.Error(w, "Failed to serve permission definitions", http.StatusInternalServerError)
	}
}

// ListDefinedPermissionKeys serves just the keys of all defined permissions.
func (h *PermissionsHandler) ListDefinedPermissionKeys(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	keys := permissions.GetAllPermissionKeys()
	if err := json.NewEncoder(w).Encode(keys); err != nil {
		http.Error(w, "Failed to serve permission keys", http.StatusInternalServerError)
	}
}
