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
var jwtKey = []byte("your_super_secret_key_that_should_be_in_config") // Replace with a strong, configured secret
const jwtExpirationHours = 24

type AuthHandler struct {
	UserRepo       repository.UserRepository
	InviteCodeRepo repository.InviteCodeRepository
	// Add other dependencies like config if needed
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
	User      models.User `json:"user"` // Or a DTO to not expose everything
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
		// Consider logging the error internally but returning a generic message
		http.Error(w, "Invalid username or password", http.StatusUnauthorized)
		return
	}

	if !user.CheckPassword(payload.Password) {
		http.Error(w, "Invalid username or password", http.StatusUnauthorized)
		return
	}

	// --- JWT Generation ---
	expirationTime := time.Now().Add(jwtExpirationHours * time.Hour)
	claims := &jwt.RegisteredClaims{
		Subject:   fmt.Sprint(user.ID), // Using user ID as subject
		ExpiresAt: jwt.NewNumericDate(expirationTime),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		Issuer:    "mediasysbackend", // Optional: identify the issuer
		// Can add custom claims here if needed, e.g. roles, username
		// CustomClaims: map[string]string{"username": user.Username},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(jwtKey)
	if err != nil {
		http.Error(w, "Failed to generate token", http.StatusInternalServerError)
		return
	}
	// --- End JWT Generation ---

	// Prepare user data for response, potentially filtering sensitive info
	// For now, sending the full user model but excluding PasswordHash (already done by JSON tag)
	// and potentially large associations if not needed for login response.
	// A UserDTO might be better here.
	userForResponse := *user          // Create a copy
	userForResponse.PasswordHash = "" // Ensure it's not sent, though "-" tag should handle it
	// Clear roles and album permissions map if they are large and not needed immediately after login
	// userForResponse.Roles = nil
	// userForResponse.AlbumPermissionsMap = nil

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
}

// RegisterHandler handles new user registration using an invite code.
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var payload RegisterPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "Invalid request payload: "+err.Error(), http.StatusBadRequest)
		return
	}

	if payload.Username == "" || payload.Password == "" || payload.InviteCode == "" {
		http.Error(w, "Username, password, and invite code are required", http.StatusBadRequest)
		return
	}

	// 1. Validate Invite Code
	inviteCode, err := h.InviteCodeRepo.GetByCode(payload.InviteCode)
	if err != nil {
		// Could be gorm.ErrRecordNotFound or other DB error
		http.Error(w, "Invalid or expired invite code", http.StatusForbidden)
		return
	}

	if !inviteCode.IsValid() {
		http.Error(w, "Invite code is not valid (expired, inactive, or max uses reached)", http.StatusForbidden)
		return
	}

	// 2. Create User
	newUser := &models.User{
		Username: payload.Username,
		// By default, new users from invite codes might have no roles or minimal global permissions.
		// This can be configured or decided later. For now, no default roles/permissions.
		GlobalPermissions: []string{},
	}
	if err := newUser.SetPassword(payload.Password); err != nil {
		http.Error(w, "Failed to hash password: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// It's good practice to wrap user creation and invite code update in a transaction
	// For now, proceeding sequentially. A transaction would ensure atomicity.
	if err := h.UserRepo.Create(newUser); err != nil {
		// Handle potential errors, e.g., username already exists
		// GORM might return a specific error for unique constraint violations.
		http.Error(w, "Failed to create user: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// 3. Increment Invite Code Uses
	if err := h.InviteCodeRepo.IncrementUses(inviteCode.ID); err != nil {
		// Log this error, as user is created but invite code update failed.
		// This is where a transaction would be beneficial.
		fmt.Printf("CRITICAL: User %s created but failed to increment uses for invite code %s (ID: %d): %v\n", newUser.Username, inviteCode.Code, inviteCode.ID, err)
		// Continue, as user creation was the primary goal for the registrant.
	}

	// Optionally, deactivate invite code if it reached max uses after this increment
	// This requires fetching the code again to check current uses vs max_uses.
	// For simplicity, this check is omitted here but can be added.

	// Return a success message. Could also return the created user DTO (without password).
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"message": "User registered successfully. Please log in."})
}

// LogoutHandler (Placeholder - JWT logout is typically client-side by discarding the token)
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	// For JWT, logout is primarily client-side (deleting the token).
	// Server-side might involve token blocklisting if using a more complex setup.
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Logged out successfully. Please discard your token."})
}

// CurrentUserHandler retrieves the authenticated user from the request context.
// This handler should be protected by the AuthMiddleware.
func (h *AuthHandler) CurrentUser(w http.ResponseWriter, r *http.Request) {
	user, ok := r.Context().Value(UserContextKey).(*models.User)
	if !ok || user == nil {
		// This should ideally not happen if AuthMiddleware is correctly applied
		http.Error(w, "Could not retrieve user from context", http.StatusInternalServerError)
		return
	}

	// Prepare user data for response, similar to Login response
	userForResponse := *user
	userForResponse.PasswordHash = "" // Ensure password hash is not sent

	// Decide if Roles and AlbumPermissionsMap should be included.
	// For a "current user" endpoint, it's often useful to include this information
	// so the frontend can know the user's permissions.
	// The GetByID method in UserRepository should already preload these.

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(userForResponse)
}
