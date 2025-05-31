package models

import (
	"golang.org/x/crypto/bcrypt"
	"time"
)

// User represents an artist or administrator in the system.
type User struct {
	ID                uint     `json:"id" gorm:"primaryKey"`
	Username          string   `json:"username" gorm:"uniqueIndex;not null"`
	PasswordHash      string   `json:"-" gorm:"not null"`                     // "-" means don't include in JSON responses
	GlobalPermissions []string `json:"global_permissions" gorm:"type:text[]"` // Storing as array of strings
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
	Permissions []string  `json:"permissions" gorm:"type:text[]"`
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

// HasGlobalPermission checks if the user has a specific global permission.
func (u *User) HasGlobalPermission(permission string) bool {
	if u.GlobalPermissions == nil {
		return false
	}
	for _, p := range u.GlobalPermissions {
		if p == permission {
			return true
		}
	}
	return false
}

// GetAlbumPermissions returns the list of permissions for a specific album.
// This would typically be populated by a DB query joining UserAlbumPermission.
func (u *User) GetAlbumPermissions(albumID uint) []string {
	// This is a placeholder. In a real scenario, this data would be loaded
	// from the UserAlbumPermission table or the AlbumPermissionsMap if populated.
	if perms, ok := u.AlbumPermissionsMap[string(albumID)]; ok { // Convert albumID to string for map key
		return perms
	}
	return []string{}
}

// HasAlbumPermission checks if the user has a specific permission for a given album.
// This would also be populated from UserAlbumPermission table.
func (u *User) HasAlbumPermission(albumID uint, permission string) bool {
	albumPerms := u.GetAlbumPermissions(albumID)
	for _, p := range albumPerms {
		if p == permission {
			return true
		}
	}
	return false
}
