package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strconv"
)

const (
	defaultListLimit = 120
	maxListLimit     = 500

	// maxBannerUploadSize caps a banner upload request body.
	maxBannerUploadSize = 20 << 20

	// maxJSONBodySize caps the body of JSON endpoints, which only carry small
	// payloads. The reverse proxy allows unbounded bodies for uploads.
	maxJSONBodySize = 64 << 10
)

// parseListWindow reads the offset/limit query parameters of a public listing.
// The limit defaults to defaultListLimit and is clamped to maxListLimit.
func parseListWindow(q url.Values) (offset, limit int) {
	limit = defaultListLimit
	if o := q.Get("offset"); o != "" {
		if v, err := strconv.Atoi(o); err == nil && v >= 0 {
			offset = v
		}
	}
	if l := q.Get("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil && v > 0 {
			limit = min(v, maxListLimit)
		}
	}
	return offset, limit
}

// parseBannerForm caps the request body and parses the multipart form of a
// banner upload. It writes the error response itself and returns false on failure.
func parseBannerForm(w http.ResponseWriter, r *http.Request) bool {
	r.Body = http.MaxBytesReader(w, r.Body, maxBannerUploadSize)
	if err := r.ParseMultipartForm(maxBannerUploadSize); err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			WriteAPIError(w, http.StatusRequestEntityTooLarge, "PayloadTooLarge", "Banner image is too large")
		} else {
			WriteAPIError(w, http.StatusBadRequest, "InvalidForm", "Invalid form data: "+err.Error())
		}
		return false
	}
	return true
}

// decodeJSONBody caps the request body at maxJSONBodySize and decodes it into
// dst. On failure callers should check isBodyTooLarge on the error so oversized
// bodies get a 413 instead of a generic 400.
func decodeJSONBody(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxJSONBodySize)
	return json.NewDecoder(r.Body).Decode(dst)
}

// isBodyTooLarge reports whether err came from exceeding a MaxBytesReader limit.
func isBodyTooLarge(err error) bool {
	var tooLarge *http.MaxBytesError
	return errors.As(err, &tooLarge)
}
