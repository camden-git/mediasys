package models

import (
	"golang.org/x/crypto/bcrypt"
	"time"
)

// User represents an artist or administrator in the system.
type User struct {
	ID                uint     `json:"id" gorm:"primaryKey"`
	Username          string   `json:"username" gorm:"uniqueIndex;not null"`
	PasswordHash      string   `json:"-" gorm:"not null"`                            // "-" means don't include in JSON responses
	GlobalPermissions []string `json:"global_permissions" gorm:"serializer:json"`    // Use JSON serializer
	Roles             []*Role  `json:"roles,omitempty" gorm:"many2many:user_roles;"` // Roles assigned to the user
	// AlbumPermissions stores permissions specific to certain albums.
	// Key: AlbumID (as string, since GORM might handle complex map keys better as JSON or serialized string)
	// Value: List of permission strings for that album
	// For simplicity with GORM and various DBs, this might be better stored as a separate table
	// or as a JSONB field if the database supports it.
	// Let's start with a separate table approach in mind for DB design,
	// but for the model, we can represent the desired structure.
	// For now, let's assume we'll handle serialization/deserialization if using a single JSON field.
	// A more robust way is a separate UserAlbumPermission table: UserID, AlbumID, Permission
	AlbumPermissionsMap map[string][]string `json:"album_permissions_map" gorm:"-"` // Not directly mapped, handled by logic
	CreatedAt           time.Time           `json:"created_at"`
	UpdatedAt           time.Time           `json:"updated_at"`
}

// UserAlbumPermission defines the relationship and permissions a user has for a specific album.
// This is a more relational way to store per-album permissions.
type UserAlbumPermission struct {
	ID      uint `json:"id" gorm:"primaryKey"`
	UserID  uint `json:"user_id" gorm:"index:idx_user_album,unique"`
	User    User `json:"-" gorm:"foreignKey:UserID"`
	AlbumID uint `json:"album_id" gorm:"index:idx_user_album,unique"` // Assuming Album model will have uint ID
	// Album      Album    `json:"-" gorm:"foreignKey:AlbumID"` // Link to Album model
	Permissions []string  `json:"permissions" gorm:"serializer:json"` // Use JSON serializer
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// SetPassword hashes the given password and sets it on the user model.
func (u *User) SetPassword(password string) error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	u.PasswordHash = string(hashedPassword)
	return nil
}

// CheckPassword verifies if the given password matches the user's hashed password.
func (u *User) CheckPassword(password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password))
	return err == nil
}

// HasGlobalPermission checks if the user has a specific global permission,
// considering both direct permissions and permissions from roles.
func (u *User) HasGlobalPermission(permission string) bool {
	// Check direct global permissions
	for _, p := range u.GlobalPermissions {
		if p == permission {
			return true
		}
	}

	// Check global permissions from roles
	// Assumes u.Roles is preloaded
	for _, role := range u.Roles {
		if role == nil { // Defensive check
			continue
		}
		for _, p := range role.GlobalPermissions {
			if p == permission {
				return true
			}
		}
	}
	return false
}

// getAllAlbumPermissionsSet collects all unique album-specific permissions for a user
// from direct assignments and all assigned roles for a specific album.
// Assumes u.AlbumPermissionsMap is populated for direct permissions,
// and u.Roles with their respective Role.AlbumPermissions are preloaded.
func (u *User) getAllAlbumPermissionsSet(albumID uint) map[string]struct{} {
	allPerms := make(map[string]struct{})

	// 1. Add direct user album permissions
	// AlbumPermissionsMap uses string keys for albumID
	if directPerms, ok := u.AlbumPermissionsMap[string(albumID)]; ok {
		for _, p := range directPerms {
			allPerms[p] = struct{}{}
		}
	}

	// 2. Add role-based permissions
	for _, role := range u.Roles {
		if role == nil { // Defensive check
			continue
		}
		// 2a. Add role-based global album permissions
		for _, p := range role.GlobalAlbumPermissions {
			allPerms[p] = struct{}{}
		}

		// 2b. Add role-based album-specific permissions
		// Assumes role.AlbumPermissions is preloaded
		for _, rap := range role.AlbumPermissions {
			if rap.AlbumID == albumID {
				for _, p := range rap.Permissions {
					allPerms[p] = struct{}{}
				}
			}
		}
	}
	return allPerms
}

// GetAlbumPermissions returns a slice of unique permissions for a specific album,
// considering both direct user permissions and permissions from roles.
func (u *User) GetAlbumPermissions(albumID uint) []string {
	permSet := u.getAllAlbumPermissionsSet(albumID)
	if len(permSet) == 0 {
		return []string{}
	}
	permissions := make([]string, 0, len(permSet))
	for p := range permSet {
		permissions = append(permissions, p)
	}
	return permissions
}

// HasAlbumPermission checks if the user has a specific permission for a given album,
// considering both direct user permissions and permissions from roles.
func (u *User) HasAlbumPermission(albumID uint, permission string) bool {
	permSet := u.getAllAlbumPermissionsSet(albumID)
	_, ok := permSet[permission]
	return ok
}
