package repository

import (
	"errors"
	"fmt"

	"github.com/camden-git/mediasysbackend/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type GormRoleRepository struct {
	db *gorm.DB
}

func NewGormRoleRepository(db *gorm.DB) RoleRepository {
	return &GormRoleRepository{db: db}
}

func (r *GormRoleRepository) Create(role *models.Role) error {
	return r.db.Create(role).Error
}

func (r *GormRoleRepository) GetByID(id uint) (*models.Role, error) {
	var role models.Role

	err := r.db.Preload("AlbumPermissions").First(&role, id).Error
	return &role, err
}

func (r *GormRoleRepository) GetByName(name string) (*models.Role, error) {
	var role models.Role
	err := r.db.Preload("AlbumPermissions").Where("name = ?", name).First(&role).Error
	return &role, err
}

func (r *GormRoleRepository) ListAll() ([]models.Role, error) {
	var roles []models.Role
	// Preload AlbumPermissions for all roles listed
	err := r.db.Preload("AlbumPermissions").Find(&roles).Error
	return roles, err
}

func (r *GormRoleRepository) Update(role *models.Role) error {
	return r.db.Session(&gorm.Session{FullSaveAssociations: true}).Save(role).Error
}

func (r *GormRoleRepository) Delete(id uint) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// delete associated RoleAlbumPermission entries
		if err := tx.Where("role_id = ?", id).Delete(&models.RoleAlbumPermission{}).Error; err != nil {
			return err
		}
		// delete associated UserRole entries (assignments of this role to users)
		if err := tx.Where("role_id = ?", id).Delete(&models.UserRole{}).Error; err != nil {
			return err
		}
		// delete the role itself
		return tx.Delete(&models.Role{}, id).Error
	})
}

func (r *GormRoleRepository) SetRoleGlobalPermissions(roleID uint, permissions []string) error {
	return r.db.Model(&models.Role{}).Where("id = ?", roleID).Update("global_permissions", permissions).Error
}

// RoleAlbumPermission management
func (r *GormRoleRepository) CreateRoleAlbumPermission(rap *models.RoleAlbumPermission) error {
	return r.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "role_id"}, {Name: "album_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"permissions"}),
	}).Create(rap).Error
}

func (r *GormRoleRepository) GetRoleAlbumPermission(roleID, albumID uint) (*models.RoleAlbumPermission, error) {
	var rap models.RoleAlbumPermission
	err := r.db.Where("role_id = ? AND album_id = ?", roleID, albumID).First(&rap).Error
	if err != nil {
		return nil, err
	}
	return &rap, nil
}

func (r *GormRoleRepository) UpdateRoleAlbumPermission(rap *models.RoleAlbumPermission) error {
	if rap.ID == 0 {
		var existingRap models.RoleAlbumPermission
		err := r.db.Where("role_id = ? AND album_id = ?", rap.RoleID, rap.AlbumID).First(&existingRap).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				// If not found, it means we should create it instead of updating
				return r.CreateRoleAlbumPermission(rap)
			}
			return fmt.Errorf("cannot find existing RoleAlbumPermission to update for role %d, album %d: %w", rap.RoleID, rap.AlbumID, err)
		}
		rap.ID = existingRap.ID
	}
	return r.db.Save(rap).Error
}

func (r *GormRoleRepository) DeleteRoleAlbumPermission(roleID, albumID uint) error {
	return r.db.Where("role_id = ? AND album_id = ?", roleID, albumID).Delete(&models.RoleAlbumPermission{}).Error
}

func (r *GormRoleRepository) GetRoleAlbumPermissions(roleID uint) ([]models.RoleAlbumPermission, error) {
	var permissions []models.RoleAlbumPermission
	err := r.db.Where("role_id = ?", roleID).Find(&permissions).Error
	return permissions, err
}

func (r *GormRoleRepository) FindUsersByRoleID(roleID uint) ([]models.User, error) {
	var role models.Role

	err := r.db.Preload("Users").First(&role, roleID).Error
	if err != nil {
		return nil, err
	}

	users := make([]models.User, len(role.Users))
	for i, userPtr := range role.Users {
		if userPtr != nil {
			users[i] = *userPtr
		}
	}
	return users, nil
}

func (r *GormRoleRepository) AddUserToRole(userID, roleID uint) error {
	userRole := models.UserRole{
		UserID: userID,
		RoleID: roleID,
	}

	return r.db.Clauses(clause.OnConflict{DoNothing: true}).Create(&userRole).Error
}

func (r *GormRoleRepository) RemoveUserFromRole(userID, roleID uint) error {
	return r.db.Where("user_id = ? AND role_id = ?", userID, roleID).Delete(&models.UserRole{}).Error
}
