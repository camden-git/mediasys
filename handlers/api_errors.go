package handlers

import (
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
	resp := APIErrorResponse{
		Errors: []APIErrorDetail{
			{
				Code:   code,
				Status: strconv.Itoa(httpStatus),
				Detail: detail,
			},
		},
	}
	writeJSON(w, httpStatus, resp)
}
