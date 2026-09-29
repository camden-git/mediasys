package models

import (
	"sort"
	"strconv"
	"time"

	"github.com/camden-git/mediasysbackend/permissions"
	"golang.org/x/crypto/bcrypt"
)

// User represents an artist or administrator in the system
type User struct {
	ID                uint     `json:"id" gorm:"primaryKey"`
	Username          string   `json:"username" gorm:"uniqueIndex;not null"`
	FirstName         string   `json:"first_name"`
	LastName          string   `json:"last_name"`
	PasswordHash      string   `json:"-" gorm:"not null"`                                   // "-" means don't include in JSON responses
	GlobalPermissions []string `json:"global_permissions" gorm:"serializer:json;type:text"` // Use JSON serializer
	Roles             []*Role  `json:"roles,omitempty" gorm:"many2many:user_roles;"`        // Roles assigned to the user
	// AlbumPermissions stores permissions specific to certain albums.
	// Key: AlbumID (as string, since GORM might handle complex map keys better as JSON or serialized string)
	// Value: List of permission strings for that album
	// For simplicity with GORM and various DBs, this might be better stored as a separate table
	// or as a JSONB field if the database supports it.
	// Let's start with a separate table approach in mind for DB design,
	// but for the model, we can represent the desired structure.
	// For now, let's assume we'll handle serialization/deserialization if using a single JSON field.
	// A more robust way is a separate UserAlbumPermission table: UserID, AlbumID, Permission
	AlbumPermissionsMap map[string][]string `json:"album_permissions_map" gorm:"-"` // not directly mapped, handled by logic
	CreatedAt           time.Time           `json:"created_at"`
	UpdatedAt           time.Time           `json:"updated_at"`

	EffectivePermissions *UserEffectivePermissions `json:"effective_permissions,omitempty" gorm:"-"` // computed at runtime
}

// AlbumPermissionGrants describes album-scoped permissions, broken down by whether they apply globally or to specific albums.
type AlbumPermissionGrants struct {
	ForAll  []string          `json:"for_all,omitempty"`
	ByAlbum map[uint][]string `json:"by_album,omitempty"`
}

// UserEffectivePermissions captures the deduplicated set of permissions a user has after combining direct assignments and roles.
type UserEffectivePermissions struct {
	Global []string              `json:"global,omitempty"`
	Album  AlbumPermissionGrants `json:"album,omitempty"`
}

// UserAlbumPermission defines the relationship and permissions a user has for a specific album
type UserAlbumPermission struct {
	ID      uint `json:"id" gorm:"primaryKey"`
	UserID  uint `json:"user_id" gorm:"index:idx_user_album,unique"`
	User    User `json:"-" gorm:"foreignKey:UserID"`
	AlbumID uint `json:"album_id" gorm:"index:idx_user_album,unique"`
	// Album      Album    `json:"-" gorm:"foreignKey:AlbumID"`
	Permissions []string  `json:"permissions" gorm:"serializer:json;type:text"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// SetPassword hashes the given password and sets it on the user model
func (u *User) SetPassword(password string) error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	u.PasswordHash = string(hashedPassword)
	return nil
}

// CheckPassword verifies if the given password matches the user's hashed password
func (u *User) CheckPassword(password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password))
	return err == nil
}

func (u *User) ensureEffectivePermissions() {
	if u.EffectivePermissions != nil {
		return
	}
	effective := u.computeEffectivePermissions()
	u.EffectivePermissions = &effective
}

// RefreshEffectivePermissions recomputes the effective permissions for the user.
func (u *User) RefreshEffectivePermissions() {
	effective := u.computeEffectivePermissions()
	u.EffectivePermissions = &effective
}

// computeEffectivePermissions merges permissions from four sources in order:
//  1. User direct global permissions (GlobalPermissions field)
//  2. Role global permissions (role.GlobalPermissions for each assigned role)
//  3. Role album permissions — global-album (role.GlobalAlbumPermissions) and per-album (role.AlbumPermissions)
//  4. User direct album permissions (AlbumPermissionsMap, loaded from UserAlbumPermission table)
func (u *User) computeEffectivePermissions() UserEffectivePermissions {
	globalSet := make(map[string]struct{})
	albumForAllSet := make(map[string]struct{})
	albumScopedSet := make(map[uint]map[string]struct{})

	addGlobal := func(perm string) {
		if perm == "" {
			return
		}
		if def, ok := permissions.GetPermissionDefinition(perm); ok && def.Scope == permissions.ScopeGlobal {
			globalSet[perm] = struct{}{}
		}
	}

	addAlbumScoped := func(albumID uint, perm string) {
		if perm == "" {
			return
		}
		if def, ok := permissions.GetPermissionDefinition(perm); ok && def.Scope == permissions.ScopeAlbum {
			if _, exists := albumScopedSet[albumID]; !exists {
				albumScopedSet[albumID] = make(map[string]struct{})
			}
			albumScopedSet[albumID][perm] = struct{}{}
		}
	}

	addAlbumGlobal := func(perm string) {
		if perm == "" {
			return
		}
		if def, ok := permissions.GetPermissionDefinition(perm); ok && def.Scope == permissions.ScopeAlbum {
			albumForAllSet[perm] = struct{}{}
		}
	}

	for _, perm := range u.GlobalPermissions {
		addGlobal(perm)
	}

	for _, role := range u.Roles {
		if role == nil {
			continue
		}
		for _, perm := range role.GlobalPermissions {
			addGlobal(perm)
		}
		for _, perm := range role.GlobalAlbumPermissions {
			addAlbumGlobal(perm)
		}
		for _, rap := range role.AlbumPermissions {
			for _, perm := range rap.Permissions {
				addAlbumScoped(rap.AlbumID, perm)
			}
		}
	}

	for albumStr, perms := range u.AlbumPermissionsMap {
		if len(perms) == 0 {
			continue
		}
		albumID64, err := strconv.ParseUint(albumStr, 10, 64)
		if err != nil {
			continue
		}
		albumID := uint(albumID64)
		for _, perm := range perms {
			addAlbumScoped(albumID, perm)
		}
	}

	effective := UserEffectivePermissions{
		Global: toSortedSlice(globalSet),
		Album: AlbumPermissionGrants{
			ForAll:  toSortedSlice(albumForAllSet),
			ByAlbum: make(map[uint][]string),
		},
	}

	for albumID, perms := range albumScopedSet {
		effective.Album.ByAlbum[albumID] = toSortedSlice(perms)
	}

	if len(effective.Album.ByAlbum) == 0 {
		effective.Album.ByAlbum = nil
	}
	if len(effective.Album.ForAll) == 0 {
		effective.Album.ForAll = nil
	}
	if len(effective.Global) == 0 {
		effective.Global = nil
	}

	return effective
}

func toSortedSlice(set map[string]struct{}) []string {
	if len(set) == 0 {
		return nil
	}
	result := make([]string, 0, len(set))
	for perm := range set {
		result = append(result, perm)
	}
	sort.Strings(result)
	return result
}

// HasGlobalPermission checks if the user has a specific global permission, considering both direct permissions and permissions from roles
func (u *User) HasGlobalPermission(permission string) bool {
	u.ensureEffectivePermissions()
	if u.EffectivePermissions == nil {
		return false
	}
	for _, perm := range u.EffectivePermissions.Global {
		if perm == permission {
			return true
		}
	}
	return false
}

// GetAlbumPermissions returns a slice of unique permissions for a specific album, considering both direct user permissions and permissions from roles
func (u *User) GetAlbumPermissions(albumID uint) []string {
	u.ensureEffectivePermissions()
	if u.EffectivePermissions == nil {
		return []string{}
	}
	permSet := make(map[string]struct{})
	for _, perm := range u.EffectivePermissions.Album.ForAll {
		permSet[perm] = struct{}{}
	}
	if scoped, ok := u.EffectivePermissions.Album.ByAlbum[albumID]; ok {
		for _, perm := range scoped {
			permSet[perm] = struct{}{}
		}
	}
	if len(permSet) == 0 {
		return []string{}
	}
	return toSortedSlice(permSet)
}

// HasAlbumPermission checks if the user has a specific permission for a given album, considering both direct user permissions and permissions from roles
func (u *User) HasAlbumPermission(albumID uint, permission string) bool {
	u.ensureEffectivePermissions()
	if u.EffectivePermissions == nil {
		return false
	}
	for _, perm := range u.EffectivePermissions.Album.ForAll {
		if perm == permission {
			return true
		}
	}
	if scoped, ok := u.EffectivePermissions.Album.ByAlbum[albumID]; ok {
		for _, perm := range scoped {
			if perm == permission {
				return true
			}
		}
	}
	return false
}
