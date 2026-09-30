package handlers

import (
	"net/http"

	"github.com/camden-git/mediasysbackend/config"
)

// serverMeta is the public, non-sensitive server configuration the frontend needs.
type serverMeta struct {
	FaceRecognitionEnabled bool `json:"face_recognition_enabled"`
}

// MetaHandler serves GET /api/meta: public feature flags so the UI can tell users
// when a capability (e.g. face recognition) is turned off on this server.
func MetaHandler(cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		WriteAPIResponse(w, http.StatusOK, serverMeta{FaceRecognitionEnabled: cfg.FaceRecognitionEnabled})
	}
}
