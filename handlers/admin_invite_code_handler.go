package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/camden-git/mediasysbackend/models"
	"github.com/camden-git/mediasysbackend/repository"
	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"
)

type AdminInviteCodeHandler struct {
	InviteCodeRepo repository.InviteCodeRepository
}

func NewAdminInviteCodeHandler(inviteCodeRepo repository.InviteCodeRepository) *AdminInviteCodeHandler {
	return &AdminInviteCodeHandler{InviteCodeRepo: inviteCodeRepo}
}

type InviteCodeCreatePayload struct {
	ExpiresAt *string `json:"expires_at,omitempty"` // ISO 8601 format e.g., "2023-12-31T23:59:59Z" or null
	MaxUses   *int    `json:"max_uses,omitempty"`   // Nullable for unlimited
}

type InviteCodeUpdatePayload struct {
	ExpiresAt *string `json:"expires_at,omitempty"`
	MaxUses   *int    `json:"max_uses,omitempty"`
	IsActive  *bool   `json:"is_active,omitempty"`
}

// InviteCodeResponseDTO for API responses
type InviteCodeResponseDTO struct {
	ID              uint    `json:"id"`
	Code            string  `json:"code"`
	ExpiresAt       *string `json:"expires_at,omitempty"`
	MaxUses         *int    `json:"max_uses,omitempty"`
	Uses            int     `json:"uses"`
	IsActive        bool    `json:"is_active"`
	CreatedByUserID uint    `json:"created_by_user_id"`
	CreatedAt       string  `json:"created_at"`
	UpdatedAt       string  `json:"updated_at"`
}

func toInviteCodeResponseDTO(ic *models.InviteCode) InviteCodeResponseDTO {
	var expiresAtStr *string
	if ic.ExpiresAt != nil {
		s := ic.ExpiresAt.Format(time.RFC3339)
		expiresAtStr = &s
	}
	return InviteCodeResponseDTO{
		ID:              ic.ID,
		Code:            ic.Code,
		ExpiresAt:       expiresAtStr,
		MaxUses:         ic.MaxUses,
		Uses:            ic.Uses,
		IsActive:        ic.IsActive,
		CreatedByUserID: ic.CreatedByUserID,
		CreatedAt:       ic.CreatedAt.Format(http.TimeFormat),
		UpdatedAt:       ic.UpdatedAt.Format(http.TimeFormat),
	}
}

func toInviteCodeListResponseDTO(ics []models.InviteCode) []InviteCodeResponseDTO {
	dtos := make([]InviteCodeResponseDTO, len(ics))
	for i, ic := range ics {
		dtos[i] = toInviteCodeResponseDTO(&ic)
	}
	return dtos
}

func (h *AdminInviteCodeHandler) ListInviteCodes(w http.ResponseWriter, r *http.Request) {
	params := ParsePaginationParams(r)
	codes, err := h.InviteCodeRepo.ListAll()
	if err != nil {
		WriteAPIError(w, http.StatusInternalServerError, "InviteCodeListError", "Failed to retrieve invite codes: "+err.Error())
		return
	}
	pagedCodes, meta := PaginateSlice(codes, params)
	response := toInviteCodeListResponseDTO(pagedCodes)
	WriteAPIPaginated(w, http.StatusOK, response, meta)
}

func (h *AdminInviteCodeHandler) GetInviteCode(w http.ResponseWriter, r *http.Request) {
	codeIDStr := chi.URLParam(r, "id")
	codeID, err := strconv.ParseUint(codeIDStr, 10, 32)
	if err != nil {
		WriteAPIError(w, http.StatusBadRequest, "InvalidInviteCodeID", "Invalid invite code ID format")
		return
	}

	code, err := h.InviteCodeRepo.GetByID(uint(codeID))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			WriteAPIError(w, http.StatusNotFound, "InviteCodeNotFound", "Invite code not found")
		} else {
			WriteAPIError(w, http.StatusInternalServerError, "InviteCodeFetchError", "Failed to retrieve invite code: "+err.Error())
		}
		return
	}
	WriteAPIResponse(w, http.StatusOK, toInviteCodeResponseDTO(code))
}

func (h *AdminInviteCodeHandler) CreateInviteCode(w http.ResponseWriter, r *http.Request) {
	var payload InviteCodeCreatePayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		WriteAPIError(w, http.StatusBadRequest, "InvalidPayload", "Invalid request payload: "+err.Error())
		return
	}

	// get authenticated user ID from context (set by AuthMiddleware)
	currentUser, ok := r.Context().Value(UserContextKey).(*models.User)
	if !ok || currentUser == nil {
		WriteAPIError(w, http.StatusInternalServerError, "ContextUserError", "User not found in context (authentication error)")
		return
	}

	inviteCode := &models.InviteCode{
		CreatedByUserID: currentUser.ID,
		MaxUses:         payload.MaxUses,
	}

	if payload.ExpiresAt != nil && *payload.ExpiresAt != "" {
		t, err := time.Parse(time.RFC3339, *payload.ExpiresAt)
		if err != nil {
			WriteAPIError(w, http.StatusBadRequest, "ValidationError", "Invalid expires_at format (must be RFC3339): "+err.Error())
			return
		}
		inviteCode.ExpiresAt = &t
	}

	if err := h.InviteCodeRepo.Create(inviteCode); err != nil {
		WriteAPIError(w, http.StatusInternalServerError, "InviteCodeCreateError", "Failed to create invite code: "+err.Error())
		return
	}

	reloadedCode, err := h.InviteCodeRepo.GetByID(inviteCode.ID)
	if err != nil {
		WriteAPIError(w, http.StatusInternalServerError, "InviteCodeFetchError", "Failed to retrieve newly created invite code: "+err.Error())
		return
	}

	WriteAPIResponse(w, http.StatusCreated, toInviteCodeResponseDTO(reloadedCode))
}

func (h *AdminInviteCodeHandler) UpdateInviteCode(w http.ResponseWriter, r *http.Request) {
	codeIDStr := chi.URLParam(r, "id")
	codeID, err := strconv.ParseUint(codeIDStr, 10, 32)
	if err != nil {
		WriteAPIError(w, http.StatusBadRequest, "InvalidInviteCodeID", "Invalid invite code ID format")
		return
	}

	var payload InviteCodeUpdatePayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		WriteAPIError(w, http.StatusBadRequest, "InvalidPayload", "Invalid request payload: "+err.Error())
		return
	}

	inviteCode, err := h.InviteCodeRepo.GetByID(uint(codeID))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			WriteAPIError(w, http.StatusNotFound, "InviteCodeNotFound", "Invite code not found")
		} else {
			WriteAPIError(w, http.StatusInternalServerError, "InviteCodeFetchError", "Failed to retrieve invite code for update: "+err.Error())
		}
		return
	}

	if payload.ExpiresAt != nil {
		if *payload.ExpiresAt == "" {
			inviteCode.ExpiresAt = nil
		} else {
			t, err := time.Parse(time.RFC3339, *payload.ExpiresAt)
			if err != nil {
				WriteAPIError(w, http.StatusBadRequest, "ValidationError", "Invalid expires_at format (must be RFC3339): "+err.Error())
				return
			}
			inviteCode.ExpiresAt = &t
		}
	}
	if payload.MaxUses != nil {
		inviteCode.MaxUses = payload.MaxUses
	}
	if payload.IsActive != nil {
		inviteCode.IsActive = *payload.IsActive
	}

	if err := h.InviteCodeRepo.Update(inviteCode); err != nil {
		WriteAPIError(w, http.StatusInternalServerError, "InviteCodeUpdateError", "Failed to update invite code: "+err.Error())
		return
	}
	WriteAPIResponse(w, http.StatusOK, toInviteCodeResponseDTO(inviteCode))
}

func (h *AdminInviteCodeHandler) DeleteInviteCode(w http.ResponseWriter, r *http.Request) {
	codeIDStr := chi.URLParam(r, "id")
	codeID, err := strconv.ParseUint(codeIDStr, 10, 32)
	if err != nil {
		WriteAPIError(w, http.StatusBadRequest, "InvalidInviteCodeID", "Invalid invite code ID format")
		return
	}

	_, err = h.InviteCodeRepo.GetByID(uint(codeID))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			WriteAPIError(w, http.StatusNotFound, "InviteCodeNotFound", "Invite code not found")
			return
		}
		WriteAPIError(w, http.StatusInternalServerError, "InviteCodeFetchError", "Failed to check invite code before delete: "+err.Error())
		return
	}

	if err := h.InviteCodeRepo.Delete(uint(codeID)); err != nil {
		WriteAPIError(w, http.StatusInternalServerError, "InviteCodeDeleteError", "Failed to delete invite code: "+err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
