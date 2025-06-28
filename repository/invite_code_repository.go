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
	err := r.db.Where("code = ?", code).First(&inviteCode).Error
	return &inviteCode, err
}

func (r *GormInviteCodeRepository) GetByID(id uint) (*models.InviteCode, error) {
	var inviteCode models.InviteCode
	err := r.db.First(&inviteCode, id).Error
	return &inviteCode, err
}

func (r *GormInviteCodeRepository) Update(inviteCode *models.InviteCode) error {
	return r.db.Save(inviteCode).Error
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
