package models

import "time"

// Role defines a set of permissions that can be assigned to users
type Role struct {
	ID                     uint                  `json:"id" gorm:"primaryKey"`
	Name                   string                `json:"name" gorm:"uniqueIndex;not null"`
	GlobalPermissions      []string              `json:"global_permissions" gorm:"serializer:json"`       // System-wide permissions
	GlobalAlbumPermissions []string              `json:"global_album_permissions" gorm:"serializer:json"` // Album permissions that apply to ALL albums
	CreatedAt              time.Time             `json:"created_at"`
	UpdatedAt              time.Time             `json:"updated_at"`
	Users                  []*User               `json:"-" gorm:"many2many:user_roles;"`                       // Many-to-many relationship with User
	AlbumPermissions       []RoleAlbumPermission `json:"album_permissions,omitempty" gorm:"foreignKey:RoleID"` // Album-specific permissions for this role
}

// UserRole is the join table for the many-to-many relationship between users and roles.
type UserRole struct {
	UserID    uint      `json:"user_id" gorm:"primaryKey"`
	RoleID    uint      `json:"role_id" gorm:"primaryKey"`
	User      User      `json:"-" gorm:"foreignKey:UserID"`
	Role      Role      `json:"-" gorm:"foreignKey:RoleID"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// RoleAlbumPermission defines the permissions a role has for a specific album
type RoleAlbumPermission struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	RoleID      uint      `json:"role_id" gorm:"index:idx_role_album,unique"`
	Role        Role      `json:"-" gorm:"foreignKey:RoleID"`
	AlbumID     uint      `json:"album_id" gorm:"index:idx_role_album,unique"`
	Permissions []string  `json:"permissions" gorm:"serializer:json"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// TableName overrides the table name for UserRole to be `user_roles`
func (UserRole) TableName() string {
	return "user_roles"
}
