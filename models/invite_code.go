package models

import (
	"crypto/rand"
	"math/big"
	"time"

	"gorm.io/gorm"
)

// InviteCode represents a single-use or multi-use invitation for registration.
type InviteCode struct {
	ID              uint       `json:"id" gorm:"primaryKey"`
	Code            string     `json:"code" gorm:"size:32;not null;unique"`
	ExpiresAt       *time.Time `json:"expires_at,omitempty"`
	MaxUses         *int       `json:"max_uses,omitempty"`
	Uses            int        `json:"uses"`
	IsActive        bool       `json:"is_active" gorm:"not null;default:true"`
	CreatedByUserID uint       `json:"created_by_user_id" gorm:"index;not null"`
	CreatedAt       time.Time  `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt       time.Time  `json:"updated_at" gorm:"autoUpdateTime"`
}

// BeforeCreate ensures new invite codes have a generated code and sane defaults.
func (ic *InviteCode) BeforeCreate(tx *gorm.DB) error {
	if ic.Code == "" {
		code, err := generateInviteCode(12)
		if err != nil {
			return err
		}
		ic.Code = code
	}
	if !ic.IsActive {
		ic.IsActive = true
	}
	return nil
}

// IsValid reports whether the invite code can still be used.
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

var inviteAlphabet = []rune("ABCDEFGHJKLMNPQRSTUVWXYZ23456789")

func generateInviteCode(length int) (string, error) {
	if length <= 0 {
		length = 12
	}
	runes := make([]rune, length)
	max := big.NewInt(int64(len(inviteAlphabet)))

	for i := 0; i < length; i++ {
		n, err := rand.Int(rand.Reader, max)
		if err != nil {
			return "", err
		}
		runes[i] = inviteAlphabet[n.Int64()]
	}

	return string(runes), nil
}
