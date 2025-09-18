package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/camden-git/mediasysbackend/config"
	"github.com/camden-git/mediasysbackend/models"
	"github.com/camden-git/mediasysbackend/repository"
	"github.com/golang-jwt/jwt/v5"
)

// TODO: Move JWT secret and expiration to config
var jwtKey = []byte("your_super_secret_key_that_should_be_in_config")

const jwtExpirationHours = 24

type AuthHandler struct {
	UserRepo       repository.UserRepository
	InviteCodeRepo repository.InviteCodeRepository
	Cfg            config.Config
}

func NewAuthHandler(userRepo repository.UserRepository, inviteCodeRepo repository.InviteCodeRepository, cfg config.Config) *AuthHandler {
	return &AuthHandler{UserRepo: userRepo, InviteCodeRepo: inviteCodeRepo, Cfg: cfg}
}

type LoginPayload struct {
	Username       string `json:"username"`
	Password       string `json:"password"`
	TurnstileToken string `json:"turnstile_token"`
}

type LoginResponse struct {
	Token     string      `json:"token"`
	User      models.User `json:"user"`
	ExpiresAt time.Time   `json:"expires_at"`
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var payload LoginPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		WriteAPIError(w, http.StatusBadRequest, "InvalidPayloadException", "Invalid request payload")
		return
	}

	// If Turnstile is configured, verify the token before proceeding
	if strings.TrimSpace(h.Cfg.TurnstileSecretKey) != "" {
		if strings.TrimSpace(payload.TurnstileToken) == "" {
			WriteAPIError(w, http.StatusBadRequest, "TurnstileVerificationException", "Turnstile verification token is required")
			return
		}

		clientIP := getClientIP(r)
		ok, verr := verifyTurnstile(h.Cfg.TurnstileSecretKey, payload.TurnstileToken, clientIP)
		if verr != nil {
			WriteAPIError(w, http.StatusBadGateway, "TurnstileVerificationException", "Failed to verify Turnstile token")
			return
		}
		if !ok {
			WriteAPIError(w, http.StatusForbidden, "TurnstileVerificationException", "Turnstile verification failed")
			return
		}
	}

	user, err := h.UserRepo.GetByUsername(payload.Username)
	if err != nil {
		WriteAPIError(w, http.StatusUnauthorized, "DisplayException", "No account matching those credentials could be found.")
		return
	}

	if !user.CheckPassword(payload.Password) {
		WriteAPIError(w, http.StatusUnauthorized, "DisplayException", "No account matching those credentials could be found.")
		return
	}

	expirationTime := time.Now().Add(jwtExpirationHours * time.Hour)
	claims := &jwt.RegisteredClaims{
		Subject:   fmt.Sprint(user.ID),
		ExpiresAt: jwt.NewNumericDate(expirationTime),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		Issuer:    "mediasysbackend",
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(jwtKey)
	if err != nil {
		WriteAPIError(w, http.StatusInternalServerError, "TokenGenerationException", "Failed to generate token")
		return
	}

	// a UserDTO might be better here.
	userForResponse := *user
	userForResponse.PasswordHash = "" // i dont think this is needed

	response := LoginResponse{
		Token:     tokenString,
		User:      userForResponse,
		ExpiresAt: expirationTime,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// verifyTurnstile verifies a Cloudflare Turnstile token using the secret key
func verifyTurnstile(secret, responseToken, remoteIP string) (bool, error) {
	form := url.Values{}
	form.Set("secret", secret)
	form.Set("response", responseToken)
	if strings.TrimSpace(remoteIP) != "" {
		form.Set("remoteip", remoteIP)
	}

	resp, err := http.PostForm("https://challenges.cloudflare.com/turnstile/v0/siteverify", form)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, err
	}
	var parsed struct {
		Success bool `json:"success"`
		// ErrorCodes []string `json:"error-codes"` // optional
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return false, err
	}
	return parsed.Success, nil
}

// getClientIP attempts to determine the client's IP address, respecting common proxy headers
func getClientIP(r *http.Request) string {
	// Try CF-Connecting-IP first (Cloudflare)
	if ip := strings.TrimSpace(r.Header.Get("CF-Connecting-IP")); ip != "" {
		return ip
	}
	// Then X-Forwarded-For (first IP)
	if xff := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); xff != "" {
		parts := strings.Split(xff, ",")
		if len(parts) > 0 {
			return strings.TrimSpace(parts[0])
		}
	}
	// Then X-Real-IP
	if ip := strings.TrimSpace(r.Header.Get("X-Real-IP")); ip != "" {
		return ip
	}
	// Fallback to RemoteAddr (strip port if present)
	hostPort := strings.TrimSpace(r.RemoteAddr)
	if idx := strings.LastIndex(hostPort, ":"); idx != -1 {
		return hostPort[:idx]
	}
	return hostPort
}

type RegisterPayload struct {
	Username   string `json:"username"`
	Password   string `json:"password"`
	InviteCode string `json:"invite_code"`
	FirstName  string `json:"first_name"`
	LastName   string `json:"last_name"`
}

// Register handles new user registration using an invitation code
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var payload RegisterPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		WriteAPIError(w, http.StatusBadRequest, "InvalidPayloadException", "Invalid request payload: "+err.Error())
		return
	}

	if payload.Username == "" || payload.Password == "" || payload.InviteCode == "" || payload.FirstName == "" || payload.LastName == "" {
		WriteAPIError(w, http.StatusBadRequest, "ValidationException", "Username, password, first_name, last_name, and invite code are required")
		return
	}

	inviteCode, err := h.InviteCodeRepo.GetByCode(payload.InviteCode)
	if err != nil {
		WriteAPIError(w, http.StatusForbidden, "InviteCodeException", "Invalid or expired invite code")
		return
	}

	if !inviteCode.IsValid() {
		WriteAPIError(w, http.StatusForbidden, "InviteCodeException", "Invite code is not valid (expired, inactive, or max uses reached)")
		return
	}

	newUser := &models.User{
		Username:          payload.Username,
		FirstName:         payload.FirstName,
		LastName:          payload.LastName,
		GlobalPermissions: []string{},
	}
	if err := newUser.SetPassword(payload.Password); err != nil {
		WriteAPIError(w, http.StatusInternalServerError, "HashingException", "Failed to hash password: "+err.Error())
		return
	}

	if err := h.UserRepo.Create(newUser); err != nil {
		WriteAPIError(w, http.StatusInternalServerError, "PersistenceException", "Failed to create user: "+err.Error())
		return
	}

	if err := h.InviteCodeRepo.IncrementUses(inviteCode.ID); err != nil {
		fmt.Printf("CRITICAL: User %s created but failed to increment uses for invite code %s (ID: %d): %v\n", newUser.Username, inviteCode.Code, inviteCode.ID, err)
	}

	// TODO: deactivate invite code if it reached max uses after this increment
	// this requires fetching the code again to check current uses vs max_uses

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"message": "User registered successfully. Please log in."})
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Logged out successfully. Please discard your token."})
}

// CurrentUser retrieves the authenticated user from the request context
func (h *AuthHandler) CurrentUser(w http.ResponseWriter, r *http.Request) {
	user, ok := r.Context().Value(UserContextKey).(*models.User)
	if !ok || user == nil {
		// ideally impossible
		http.Error(w, "Could not retrieve user from context", http.StatusInternalServerError)
		return
	}

	userForResponse := *user
	userForResponse.PasswordHash = ""

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(userForResponse)
}
