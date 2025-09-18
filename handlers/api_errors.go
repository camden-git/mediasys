package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
)

// APIErrorDetail represents a single error in the standardized error response.
type APIErrorDetail struct {
	Code   string `json:"code"`
	Status string `json:"status"`
	Detail string `json:"detail"`
}

// APIErrorResponse represents the standardized error response body.
type APIErrorResponse struct {
	Errors []APIErrorDetail `json:"errors"`
}

// WriteAPIError writes a standardized error response with the given HTTP status, code, and detail.
func WriteAPIError(w http.ResponseWriter, httpStatus int, code string, detail string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpStatus)

	resp := APIErrorResponse{
		Errors: []APIErrorDetail{
			{
				Code:   code,
				Status: strconv.Itoa(httpStatus),
				Detail: detail,
			},
		},
	}

	_ = json.NewEncoder(w).Encode(resp)
}
