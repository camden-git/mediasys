package models

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"time"

	"gorm.io/gorm"
)

// InviteCode represents an invitation code for user registration
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

// BeforeCreate generates a unique code if not provided
func (ic *InviteCode) BeforeCreate(tx *gorm.DB) (err error) {
	if ic.Code == "" {
		// Attempt to generate a unique 6-digit PIN, retrying a few times in the unlikely event of collisions
		const maxAttempts = 10
		for attempt := 0; attempt < maxAttempts; attempt++ {
			code, genErr := generateSixDigitPIN()
			if genErr != nil {
				return genErr
			}
			var existing InviteCode
			findErr := tx.Where("code = ?", code).Select("id").First(&existing).Error
			if findErr == gorm.ErrRecordNotFound {
				ic.Code = code
				return nil
			}
			if findErr != nil {
				return findErr
			}
			// if found, loop to try another code
		}
		return fmt.Errorf("failed to generate unique invite code after %d attempts", maxAttempts)
	}
	return nil
}

func generateSixDigitPIN() (string, error) {
	// Securely generate a number in [0, 1_000_000)
	n, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}

// IsValid checks if the invite code can still be used
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
