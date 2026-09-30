package repository

import (
	"github.com/camden-git/mediasysbackend/models"
	"gorm.io/gorm"
)

type GormInviteCodeRepository struct {
	db *gorm.DB
}

func NewGormInviteCodeRepository(db *gorm.DB) InviteCodeRepository {
	return &GormInviteCodeRepository{db: db}
}

func (r *GormInviteCodeRepository) Create(inviteCode *models.InviteCode) error {
	return r.db.Create(inviteCode).Error
}

func (r *GormInviteCodeRepository) GetByCode(code string) (*models.InviteCode, error) {
	var inviteCode models.InviteCode
	if err := r.db.Where("code = ?", code).First(&inviteCode).Error; err != nil {
		return nil, err
	}
	return &inviteCode, nil
}

func (r *GormInviteCodeRepository) GetByID(id uint) (*models.InviteCode, error) {
	var inviteCode models.InviteCode
	if err := r.db.First(&inviteCode, id).Error; err != nil {
		return nil, err
	}
	return &inviteCode, nil
}

// Update writes only the admin-editable columns, so concurrent changes to uses are not overwritten.
// Nil ExpiresAt/MaxUses are stored as NULL.
func (r *GormInviteCodeRepository) Update(inviteCode *models.InviteCode) error {
	return r.db.Model(&models.InviteCode{}).Where("id = ?", inviteCode.ID).
		Select("expires_at", "max_uses", "is_active").
		Updates(inviteCode).Error
}

func (r *GormInviteCodeRepository) IncrementUses(id uint) error {
	return r.db.Model(&models.InviteCode{}).Where("id = ?", id).UpdateColumn("uses", gorm.Expr("uses + 1")).Error
}

func (r *GormInviteCodeRepository) ListAll() ([]models.InviteCode, error) {
	var inviteCodes []models.InviteCode
	err := r.db.Find(&inviteCodes).Error
	return inviteCodes, err
}

func (r *GormInviteCodeRepository) Delete(id uint) error {
	return r.db.Delete(&models.InviteCode{}, id).Error
}
