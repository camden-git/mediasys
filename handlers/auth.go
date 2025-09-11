package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

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
}

func NewAuthHandler(userRepo repository.UserRepository, inviteCodeRepo repository.InviteCodeRepository) *AuthHandler {
	return &AuthHandler{UserRepo: userRepo, InviteCodeRepo: inviteCodeRepo}
}

type LoginPayload struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Token     string      `json:"token"`
	User      models.User `json:"user"`
	ExpiresAt time.Time   `json:"expires_at"`
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var payload LoginPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	user, err := h.UserRepo.GetByUsername(payload.Username)
	if err != nil {
		http.Error(w, "Invalid username or password", http.StatusUnauthorized)
		return
	}

	if !user.CheckPassword(payload.Password) {
		http.Error(w, "Invalid username or password", http.StatusUnauthorized)
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
		http.Error(w, "Failed to generate token", http.StatusInternalServerError)
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
		http.Error(w, "Invalid request payload: "+err.Error(), http.StatusBadRequest)
		return
	}

	if payload.Username == "" || payload.Password == "" || payload.InviteCode == "" || payload.FirstName == "" || payload.LastName == "" {
		http.Error(w, "Username, password, first_name, last_name, and invite code are required", http.StatusBadRequest)
		return
	}

	inviteCode, err := h.InviteCodeRepo.GetByCode(payload.InviteCode)
	if err != nil {
		http.Error(w, "Invalid or expired invite code", http.StatusForbidden)
		return
	}

	if !inviteCode.IsValid() {
		http.Error(w, "Invite code is not valid (expired, inactive, or max uses reached)", http.StatusForbidden)
		return
	}

	newUser := &models.User{
		Username:          payload.Username,
		FirstName:         payload.FirstName,
		LastName:          payload.LastName,
		GlobalPermissions: []string{},
	}
	if err := newUser.SetPassword(payload.Password); err != nil {
		http.Error(w, "Failed to hash password: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if err := h.UserRepo.Create(newUser); err != nil {
		http.Error(w, "Failed to create user: "+err.Error(), http.StatusInternalServerError)
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
