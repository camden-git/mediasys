package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/camden-git/mediasysbackend/models" // Added import
	"github.com/camden-git/mediasysbackend/repository"
	"github.com/go-chi/chi/v5"
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
			WriteAPIError(w, http.StatusUnauthorized, "AuthHeaderMissing", "Authorization header required")
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			WriteAPIError(w, http.StatusUnauthorized, "AuthHeaderFormat", "Authorization header format must be Bearer {token}")
			return
		}
		tokenString := parts[1]

		claims := &jwt.RegisteredClaims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return jwtKey, nil // jwtKey is defined in auth.go and set from config.Config.JWTSecret via SetJWTSecret
		})

		if err != nil {
			if errors.Is(err, jwt.ErrSignatureInvalid) {
				WriteAPIError(w, http.StatusUnauthorized, "InvalidTokenSignature", "Invalid token signature")
				return
			}
			WriteAPIError(w, http.StatusUnauthorized, "InvalidToken", "Invalid token: "+err.Error())
			return
		}

		if !token.Valid {
			WriteAPIError(w, http.StatusUnauthorized, "InvalidToken", "Invalid token")
			return
		}

		userIDStr := claims.Subject
		var userID uint
		// Convert userIDStr (which is fmt.Sprint(user.ID)) back to uint
		if _, err := fmt.Sscan(userIDStr, &userID); err != nil {
			WriteAPIError(w, http.StatusUnauthorized, "InvalidTokenSubject", "Invalid user ID in token")
			// Log this error server-side as it indicates a malformed token subject
			fmt.Printf("Error parsing userID from token subject '%s': %v\n", userIDStr, err)
			return
		}

		user, err := userRepo.GetByID(userID)
		if err != nil {
			// This could happen if the user was deleted after the token was issued.
			WriteAPIError(w, http.StatusUnauthorized, "UserNotFound", "User not found")
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
			WriteAPIError(w, http.StatusInternalServerError, "ContextUserMissing", "User not found in context")
			return
		}

		if !user.HasGlobalPermission(requiredPermission) {
			WriteAPIError(w, http.StatusForbidden, "Forbidden", fmt.Sprintf("Forbidden: requires global permission '%s'", requiredPermission))
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
			WriteAPIError(w, http.StatusInternalServerError, "ContextUserMissing", "User not found in context")
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
			WriteAPIError(w, http.StatusForbidden, "Forbidden", fmt.Sprintf("Forbidden: requires at least one of the following global permissions: %s", strings.Join(permissions, ", ")))
			return
		}
		next.ServeHTTP(w, r)
	})
}

// ResolveAlbumSlug rewrites a non-numeric "id" URL parameter to the numeric ID of
// the album with that slug, so the per-album permission checks and handlers nested
// under the route only ever see IDs. Unknown slugs are left untouched and fall
// through to the normal permission/not-found handling.
func ResolveAlbumSlug(albumRepo repository.AlbumRepositoryInterface) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if rctx := chi.RouteContext(r.Context()); rctx != nil {
				for i, key := range rctx.URLParams.Keys {
					if key != "id" {
						continue
					}
					if _, err := strconv.ParseUint(rctx.URLParams.Values[i], 10, 64); err != nil {
						if album, err := albumRepo.GetBySlug(rctx.URLParams.Values[i]); err == nil {
							rctx.URLParams.Values[i] = strconv.FormatUint(uint64(album.ID), 10)
						}
					}
					break
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

// RequireAlbumPermission is a middleware that grants access if the authenticated user has
// the given global permission, OR the given album-scoped permission for the album identified
// by the "id" URL parameter. It should be used after AuthMiddleware, on routes nested under
// a chi route that captures the album ID as "id".
func RequireAlbumPermission(globalPermission, albumPermission string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, ok := r.Context().Value(UserContextKey).(*models.User)
		if !ok || user == nil {
			WriteAPIError(w, http.StatusInternalServerError, "ContextUserMissing", "User not found in context")
			return
		}

		if globalPermission != "" && user.HasGlobalPermission(globalPermission) {
			next.ServeHTTP(w, r)
			return
		}

		if albumPermission != "" {
			if albumIDStr := chi.URLParam(r, "id"); albumIDStr != "" {
				if albumID, err := strconv.ParseUint(albumIDStr, 10, 64); err == nil {
					if user.HasAlbumPermission(uint(albumID), albumPermission) {
						next.ServeHTTP(w, r)
						return
					}
				}
			}
		}

		WriteAPIError(w, http.StatusForbidden, "Forbidden", fmt.Sprintf("Forbidden: requires global permission '%s' or album permission '%s'", globalPermission, albumPermission))
	})
}

// RequireAnyGlobalPermissionOrAlbumAccess is a middleware that grants access if the authenticated
// user has at least one of the given global permissions, OR has at least one album-scoped
// permission for any album (directly or via a role). Handlers guarded by this middleware are
// expected to filter their results to the albums the user actually has access to.
func RequireAnyGlobalPermissionOrAlbumAccess(globalPermissions []string) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, ok := r.Context().Value(UserContextKey).(*models.User)
			if !ok || user == nil {
				WriteAPIError(w, http.StatusInternalServerError, "ContextUserMissing", "User not found in context")
				return
			}

			for _, p := range globalPermissions {
				if user.HasGlobalPermission(p) {
					next.ServeHTTP(w, r)
					return
				}
			}

			if user.HasAnyAlbumPermission() {
				next.ServeHTTP(w, r)
				return
			}

			WriteAPIError(w, http.StatusForbidden, "Forbidden", fmt.Sprintf("Forbidden: requires at least one of the following global permissions: %s", strings.Join(globalPermissions, ", ")))
		})
	}
}
