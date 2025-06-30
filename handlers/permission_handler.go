package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/camden-git/mediasysbackend/permissions"
)

type PermissionHandler struct {
	// No dependencies needed for now, as it uses a package-level variable
}

// ListPermissionDefinitions serves the statically defined permission groups.
func (h *PermissionHandler) ListPermissionDefinitions(w http.ResponseWriter, r *http.Request) {
	definitions := permissions.DefinedPermissionGroups

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(definitions); err != nil {
		// Log error, but response header is already sent.
		// Consider logging to a more persistent store or system logger
		// fmt.Printf("Error encoding JSON response for ListPermissionDefinitions: %v\n", err)
		http.Error(w, "Failed to encode permission definitions", http.StatusInternalServerError)
	}
}
