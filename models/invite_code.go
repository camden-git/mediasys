package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// InviteCode represents an invitation code for user registration.
type InviteCode struct {
	ID              uint       `json:"id" gorm:"primaryKey"`
	Code            string     `json:"code" gorm:"uniqueIndex;not null"`
	ExpiresAt       *time.Time `json:"expires_at,omitempty" gorm:"index"` // Nullable for no expiration
	MaxUses         *int       `json:"max_uses,omitempty"`                // Nullable for unlimited uses
	Uses            int        `json:"uses" gorm:"default:0"`
	IsActive        bool       `json:"is_active" gorm:"default:true"`
	CreatedByUserID uint       `json:"created_by_user_id"` // ID of the admin user who created the code
	CreatedByUser   User       `json:"-" gorm:"foreignKey:CreatedByUserID"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

// BeforeCreate generates a unique code if not provided.
func (ic *InviteCode) BeforeCreate(tx *gorm.DB) (err error) {
	if ic.Code == "" {
		ic.Code = uuid.New().String() // Generate a UUID as the invite code
	}
	// GORM's default:true for IsActive handles this now.
	// if ic.IsActive == false && ic.ID == 0 { // Default to true on creation if not set
	// 	ic.IsActive = true
	// }
	return
}

// IsValid checks if the invite code can still be used.
func (ic *InviteCode) IsValid() bool {
	if !ic.IsActive {
		return false
	}
	if ic.ExpiresAt != nil && time.Now().After(*ic.ExpiresAt) {
		return false
	}
	if ic.MaxUses != nil && ic.Uses >= *ic.MaxUses {
		return false
	}
	return true
}
