package handlers

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"slices"
	"strconv"
	"strings"
)

const (
	defaultPage    = 1
	defaultPerPage = 25
	maxPerPage     = 100
)

// PaginationParams represents pagination parameters supplied via the query string.
type PaginationParams struct {
	Page    int
	PerPage int
}

// Offset returns the calculated offset for database queries.
func (p PaginationParams) Offset() int {
	if p.Page <= 1 {
		return 0
	}
	if p.PerPage > 0 && p.Page-1 > math.MaxInt/p.PerPage {
		return math.MaxInt
	}
	return (p.Page - 1) * p.PerPage
}

// PaginationMeta describes pagination information returned alongside a list response.
type PaginationMeta struct {
	Total       int `json:"total"`
	Count       int `json:"count"`
	PerPage     int `json:"per_page"`
	CurrentPage int `json:"current_page"`
	TotalPages  int `json:"total_pages"`
}

// APIMeta contains optional metadata for API responses.
type APIMeta struct {
	Pagination *PaginationMeta `json:"pagination,omitempty"`
}

// APIResponse standardizes JSON response bodies.
type APIResponse[T any] struct {
	Data T        `json:"data"`
	Meta *APIMeta `json:"meta,omitempty"`
}

// WriteAPIResponse writes a standardized JSON response.
func WriteAPIResponse[T any](w http.ResponseWriter, status int, data T) {
	writeJSON(w, status, APIResponse[T]{Data: data})
}

// WriteAPIPaginated writes a standardized JSON response including pagination metadata.
func WriteAPIPaginated[T any](w http.ResponseWriter, status int, data []T, pagination PaginationMeta) {
	meta := &APIMeta{Pagination: &pagination}
	writeJSON(w, status, APIResponse[[]T]{Data: data, Meta: meta})
}

// ParsePaginationParams reads pagination query parameters from the request.
func ParsePaginationParams(r *http.Request) PaginationParams {
	query := r.URL.Query()

	page := clampToMinimum(parseInt(query.Get("page"), defaultPage), 1)
	perPage := clampPerPage(parseInt(query.Get("per_page"), defaultPerPage))

	return PaginationParams{
		Page:    page,
		PerPage: perPage,
	}
}

// NewPaginationMeta builds a PaginationMeta struct based on the total number of items.
func NewPaginationMeta(total int, params PaginationParams, count int) PaginationMeta {
	perPage := params.PerPage
	if perPage <= 0 {
		perPage = defaultPerPage
	}

	totalPages := 0
	if total > 0 && perPage > 0 {
		totalPages = int(math.Ceil(float64(total) / float64(perPage)))
	}

	currentPage := params.Page
	if currentPage < 1 {
		currentPage = 1
	}

	return PaginationMeta{
		Total:       total,
		Count:       count,
		PerPage:     perPage,
		CurrentPage: currentPage,
		TotalPages:  totalPages,
	}
}

// PaginateSlice paginates an in-memory slice using provided params.
func PaginateSlice[T any](items []T, params PaginationParams) ([]T, PaginationMeta) {
	total := len(items)
	if params.PerPage <= 0 {
		params.PerPage = defaultPerPage
	}

	start := params.Offset()
	if start > total {
		start = total
	}

	end := total
	if params.PerPage < total-start {
		end = start + params.PerPage
	}

	pageItems := slices.Clone(items[start:end])
	meta := NewPaginationMeta(total, params, len(pageItems))

	return pageItems, meta
}

func parseInt(value string, fallback int) int {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func clampToMinimum(value, min int) int {
	if value < min {
		return min
	}
	return value
}

func clampPerPage(perPage int) int {
	perPage = clampToMinimum(perPage, 1)
	if perPage > maxPerPage {
		return maxPerPage
	}
	return perPage
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

// setCacheHeaders sets public Cache-Control headers with stale-while-revalidate support.
func setCacheHeaders(w http.ResponseWriter, maxAge int) {
	swr := maxAge * 5
	w.Header().Set("Cache-Control", fmt.Sprintf("public, max-age=%d, stale-while-revalidate=%d", maxAge, swr))
}
