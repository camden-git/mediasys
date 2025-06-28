package handlers

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/camden-git/mediasysbackend/models" // Added import
	"github.com/camden-git/mediasysbackend/repository"
	"github.com/golang-jwt/jwt/v5"
)

// ContextKey is a custom type for context keys to avoid collisions.
type ContextKey string

const (
	// UserContextKey is the key used to store the user object in the request context.
	UserContextKey ContextKey = "user"
)

// AuthMiddleware creates a middleware handler for JWT authentication.
// It verifies the token and, if valid, fetches the user and adds them to the request context.
func AuthMiddleware(userRepo repository.UserRepository, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Authorization header required", http.StatusUnauthorized)
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			http.Error(w, "Authorization header format must be Bearer {token}", http.StatusUnauthorized)
			return
		}
		tokenString := parts[1]

		claims := &jwt.RegisteredClaims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return jwtKey, nil // jwtKey is defined in auth.go (ideally from config)
		})

		if err != nil {
			if err == jwt.ErrSignatureInvalid {
				http.Error(w, "Invalid token signature", http.StatusUnauthorized)
				return
			}
			http.Error(w, "Invalid token: "+err.Error(), http.StatusUnauthorized)
			return
		}

		if !token.Valid {
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		userIDStr := claims.Subject
		var userID uint
		// Convert userIDStr (which is fmt.Sprint(user.ID)) back to uint
		if _, err := fmt.Sscan(userIDStr, &userID); err != nil {
			http.Error(w, "Invalid user ID in token", http.StatusUnauthorized)
			// Log this error server-side as it indicates a malformed token subject
			fmt.Printf("Error parsing userID from token subject '%s': %v\n", userIDStr, err)
			return
		}

		user, err := userRepo.GetByID(userID)
		if err != nil {
			// This could happen if the user was deleted after the token was issued.
			http.Error(w, "User not found", http.StatusUnauthorized)
			return
		}

		// Add user to context
		ctx := context.WithValue(r.Context(), UserContextKey, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequireGlobalPermission is a middleware that checks if the authenticated user has
// a specific global permission. It should be used after AuthMiddleware.
func RequireGlobalPermission(requiredPermission string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, ok := r.Context().Value(UserContextKey).(*models.User) // models.User needs to be imported
		if !ok || user == nil {
			// This should not happen if AuthMiddleware ran successfully
			http.Error(w, "User not found in context", http.StatusInternalServerError)
			return
		}

		if !user.HasGlobalPermission(requiredPermission) {
			http.Error(w, fmt.Sprintf("Forbidden: requires global permission '%s'", requiredPermission), http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// RequireAnyGlobalPermission is a middleware that checks if the authenticated user has
// at least one of the specified global permissions. It should be used after AuthMiddleware.
func RequireAnyGlobalPermission(permissions []string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, ok := r.Context().Value(UserContextKey).(*models.User) // models.User needs to be imported
		if !ok || user == nil {
			http.Error(w, "User not found in context", http.StatusInternalServerError)
			return
		}

		hasAtLeastOne := false
		for _, p := range permissions {
			if user.HasGlobalPermission(p) {
				hasAtLeastOne = true
				break
			}
		}

		if !hasAtLeastOne {
			http.Error(w, fmt.Sprintf("Forbidden: requires at least one of the following global permissions: %s", strings.Join(permissions, ", ")), http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}
