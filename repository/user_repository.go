package repository

import (
	"errors"
	"fmt"

	"github.com/camden-git/mediasysbackend/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type GormUserRepository struct {
	db *gorm.DB
}

func NewGormUserRepository(db *gorm.DB) UserRepository {
	return &GormUserRepository{db: db}
}

func (r *GormUserRepository) Create(user *models.User) error {
	return r.db.Create(user).Error
}

func (r *GormUserRepository) GetByID(id uint) (*models.User, error) {
	var user models.User

	err := r.db.Preload("Roles.AlbumPermissions").Preload("Roles").First(&user, id).Error
	if err != nil {
		return nil, err
	}

	var userAlbumPerms []models.UserAlbumPermission
	if err := r.db.Where("user_id = ?", id).Find(&userAlbumPerms).Error; err == nil {
		user.AlbumPermissionsMap = make(map[string][]string)
		for _, uap := range userAlbumPerms {
			user.AlbumPermissionsMap[fmt.Sprint(uap.AlbumID)] = uap.Permissions
		}
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("failed to load user album permissions: %w", err)
	}
	return &user, nil
}

func (r *GormUserRepository) GetByUsername(username string) (*models.User, error) {
	var user models.User
	err := r.db.Preload("Roles.AlbumPermissions").Preload("Roles").Where("username = ?", username).First(&user).Error
	if err != nil {
		return nil, err
	}

	var userAlbumPerms []models.UserAlbumPermission
	if err := r.db.Where("user_id = ?", user.ID).Find(&userAlbumPerms).Error; err == nil {
		user.AlbumPermissionsMap = make(map[string][]string)
		for _, uap := range userAlbumPerms {
			user.AlbumPermissionsMap[fmt.Sprint(uap.AlbumID)] = uap.Permissions
		}
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("failed to load user album permissions for user %s: %w", username, err)
	}
	return &user, nil
}

func (r *GormUserRepository) Update(user *models.User) error {
	return r.db.Session(&gorm.Session{FullSaveAssociations: true}).Save(user).Error
}

func (r *GormUserRepository) Delete(id uint) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("user_id = ?", id).Delete(&models.UserAlbumPermission{}).Error; err != nil {
			return err
		}
		if err := tx.Where("user_id = ?", id).Delete(&models.UserRole{}).Error; err != nil {
			return err
		}
		return tx.Delete(&models.User{}, id).Error
	})
}

func (r *GormUserRepository) ListAll() ([]models.User, error) {
	var users []models.User

	err := r.db.Preload("Roles").Find(&users).Error
	// for i := range users {
	// 	var userAlbumPerms []models.UserAlbumPermission
	// 	if errDb := r.db.Where("user_id = ?", users[i].ID).Find(&userAlbumPerms).Error; errDb == nil {
	// 		users[i].AlbumPermissionsMap = make(map[string][]string)
	// 		for _, uap := range userAlbumPerms {
	// 			users[i].AlbumPermissionsMap[fmt.Sprint(uap.AlbumID)] = uap.Permissions
	// 		}
	// 	} else if !errors.Is(errDb, gorm.ErrRecordNotFound) {
	// 		return nil, fmt.Errorf("failed to load album permissions for user ID %d: %w", users[i].ID, errDb)
	// 	}
	// }
	return users, err
}

func (r *GormUserRepository) AddRoleToUser(userID uint, roleID uint) error {
	userRole := models.UserRole{UserID: userID, RoleID: roleID}
	// avoid error if association already exists
	return r.db.Clauses(clause.OnConflict{DoNothing: true}).Create(&userRole).Error
}

func (r *GormUserRepository) RemoveRoleFromUser(userID uint, roleID uint) error {
	return r.db.Where("user_id = ? AND role_id = ?", userID, roleID).Delete(&models.UserRole{}).Error
}

func (r *GormUserRepository) GetUserRoles(userID uint) ([]models.Role, error) {
	var user models.User
	if err := r.db.Preload("Roles").First(&user, userID).Error; err != nil {
		return nil, err
	}

	var roles []models.Role
	for _, rPtr := range user.Roles {
		if rPtr != nil {
			roles = append(roles, *rPtr)
		}
	}
	return roles, nil
}

func (r *GormUserRepository) SetUserGlobalPermissions(userID uint, permissions []string) error {
	return r.db.Model(&models.User{}).Where("id = ?", userID).Update("global_permissions", permissions).Error
}

// UserAlbumPermission management
func (r *GormUserRepository) CreateUserAlbumPermission(uap *models.UserAlbumPermission) error {
	return r.db.Create(uap).Error
}

func (r *GormUserRepository) GetUserAlbumPermission(userID, albumID uint) (*models.UserAlbumPermission, error) {
	var uap models.UserAlbumPermission
	err := r.db.Where("user_id = ? AND album_id = ?", userID, albumID).First(&uap).Error
	if err != nil {
		return nil, err
	}
	return &uap, nil
}

func (r *GormUserRepository) UpdateUserAlbumPermission(uap *models.UserAlbumPermission) error {
	if uap.ID == 0 {
		var existingUap models.UserAlbumPermission
		err := r.db.Where("user_id = ? AND album_id = ?", uap.UserID, uap.AlbumID).First(&existingUap).Error
		if err != nil {
			return fmt.Errorf("cannot update UserAlbumPermission, record not found for user %d, album %d: %w", uap.UserID, uap.AlbumID, err)
		}
		uap.ID = existingUap.ID
	}
	return r.db.Save(uap).Error
}

func (r *GormUserRepository) DeleteUserAlbumPermission(userID, albumID uint) error {
	return r.db.Where("user_id = ? AND album_id = ?", userID, albumID).Delete(&models.UserAlbumPermission{}).Error
}

func (r *GormUserRepository) GetUserAlbumPermissions(userID uint) ([]models.UserAlbumPermission, error) {
	var permissions []models.UserAlbumPermission
	err := r.db.Where("user_id = ?", userID).Find(&permissions).Error
	return permissions, err
}
