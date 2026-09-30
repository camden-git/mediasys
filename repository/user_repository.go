package repository

import (
	"errors"
	"fmt"
	"time"

	"github.com/camden-git/mediasysbackend/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ErrSetupAlreadyCompleted is returned by CreateFirstAdmin when a user
// already exists, meaning initial setup has already been completed.
var ErrSetupAlreadyCompleted = errors.New("setup already completed")

// ErrInviteCodeInvalid is returned by CreateWithInviteCode when the invite code
// does not exist, is inactive or expired, or has no uses left.
var ErrInviteCodeInvalid = errors.New("invite code is invalid, expired, or exhausted")

// firstAdminLockKey is the Postgres advisory lock key that serializes first-admin creation.
const firstAdminLockKey int64 = 0x6d65646961737973 // "mediasys"

type GormUserRepository struct {
	db *gorm.DB
}

func NewGormUserRepository(db *gorm.DB) UserRepository {
	return &GormUserRepository{db: db}
}

func (r *GormUserRepository) withRolePreloads(tx *gorm.DB) *gorm.DB {
	return tx.Preload("Roles").Preload("Roles.AlbumPermissions")
}

func (r *GormUserRepository) hydrateUserWithAlbumPermissions(user *models.User) error {
	if user == nil {
		return nil
	}
	var records []models.UserAlbumPermission
	if err := r.db.Where("user_id = ?", user.ID).Find(&records).Error; err != nil {
		return fmt.Errorf("failed to load user album permissions: %w", err)
	}
	user.AlbumPermissionsMap = make(map[string][]string)
	for _, record := range records {
		user.AlbumPermissionsMap[fmt.Sprint(record.AlbumID)] = append([]string(nil), record.Permissions...)
	}
	user.RefreshEffectivePermissions()
	return nil
}

func (r *GormUserRepository) hydrateUsersWithAlbumPermissions(users []models.User, albumID *uint) (map[uint]models.UserAlbumPermission, error) {
	directMap := make(map[uint]models.UserAlbumPermission)
	if len(users) == 0 {
		return directMap, nil
	}

	userIDs := make([]uint, 0, len(users))
	for _, user := range users {
		userIDs = append(userIDs, user.ID)
	}

	query := r.db.Where("user_id IN ?", userIDs)
	if albumID != nil {
		query = query.Where("album_id = ?", *albumID)
	}

	var records []models.UserAlbumPermission
	if err := query.Find(&records).Error; err != nil {
		return nil, fmt.Errorf("failed to load album permissions for users: %w", err)
	}

	grouped := make(map[uint][]models.UserAlbumPermission)
	for _, record := range records {
		grouped[record.UserID] = append(grouped[record.UserID], record)
		if albumID != nil {
			directMap[record.UserID] = record
		}
	}

	for i := range users {
		users[i].AlbumPermissionsMap = make(map[string][]string)
		for _, record := range grouped[users[i].ID] {
			users[i].AlbumPermissionsMap[fmt.Sprint(record.AlbumID)] = append([]string(nil), record.Permissions...)
		}
		users[i].RefreshEffectivePermissions()
	}

	return directMap, nil
}

func (r *GormUserRepository) Create(user *models.User) error {
	return r.db.Create(user).Error
}

// CreateWithInviteCode claims one use of the invite code and creates the user in a
// single transaction. The claim is a conditional UPDATE, so concurrent registrations
// cannot exceed max_uses. Returns ErrInviteCodeInvalid if the code cannot be claimed.
func (r *GormUserRepository) CreateWithInviteCode(user *models.User, code string) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		res := tx.Model(&models.InviteCode{}).
			Where("code = ? AND is_active = ? AND (expires_at IS NULL OR expires_at > ?) AND (max_uses IS NULL OR uses < max_uses)", code, true, time.Now()).
			UpdateColumn("uses", gorm.Expr("uses + 1"))
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return ErrInviteCodeInvalid
		}
		return tx.Create(user).Error
	})
}

func (r *GormUserRepository) GetByID(id uint) (*models.User, error) {
	var user models.User

	err := r.withRolePreloads(r.db).First(&user, id).Error
	if err != nil {
		return nil, err
	}

	if err := r.hydrateUserWithAlbumPermissions(&user); err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *GormUserRepository) GetByUsername(username string) (*models.User, error) {
	var user models.User
	err := r.withRolePreloads(r.db).Where("username = ?", username).First(&user).Error
	if err != nil {
		return nil, err
	}

	if err := r.hydrateUserWithAlbumPermissions(&user); err != nil {
		return nil, fmt.Errorf("failed to load user album permissions for user %s: %w", username, err)
	}
	return &user, nil
}

// Update saves the user's own columns only; associations such as roles are left untouched.
func (r *GormUserRepository) Update(user *models.User) error {
	return r.db.Omit(clause.Associations).Save(user).Error
}

// UpdateWithRoles saves the user's own columns and replaces the user's roles with roleIDs in one transaction.
func (r *GormUserRepository) UpdateWithRoles(user *models.User, roleIDs []uint) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Omit(clause.Associations).Save(user).Error; err != nil {
			return err
		}
		if err := tx.Where("user_id = ?", user.ID).Delete(&models.UserRole{}).Error; err != nil {
			return err
		}
		for _, roleID := range roleIDs {
			userRole := models.UserRole{UserID: user.ID, RoleID: roleID}
			if err := tx.Omit(clause.Associations).Clauses(clause.OnConflict{DoNothing: true}).Create(&userRole).Error; err != nil {
				return err
			}
		}
		return nil
	})
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

	err := r.withRolePreloads(r.db).Find(&users).Error
	if err != nil {
		return nil, err
	}
	if _, err := r.hydrateUsersWithAlbumPermissions(users, nil); err != nil {
		return nil, err
	}
	return users, nil
}

func (r *GormUserRepository) CountAll() (int64, error) {
	var count int64
	err := r.db.Model(&models.User{}).Count(&count).Error
	return count, err
}

// CreateFirstAdmin atomically creates the given user and assigns them the
// named role, failing with an error if any user already exists. Intended for
// bootstrapping the very first administrator during initial setup.
func (r *GormUserRepository) CreateFirstAdmin(user *models.User, roleName string) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// serialize concurrent setups so the count check below cannot race
		if err := tx.Exec("SELECT pg_advisory_xact_lock(?)", firstAdminLockKey).Error; err != nil {
			return fmt.Errorf("failed to acquire setup lock: %w", err)
		}

		var count int64
		if err := tx.Model(&models.User{}).Count(&count).Error; err != nil {
			return fmt.Errorf("failed to count existing users: %w", err)
		}
		if count > 0 {
			return ErrSetupAlreadyCompleted
		}

		var role models.Role
		if err := tx.Where("name = ?", roleName).First(&role).Error; err != nil {
			return fmt.Errorf("could not find the '%s' role, which should have been auto-generated: %w", roleName, err)
		}

		if err := tx.Create(user).Error; err != nil {
			return fmt.Errorf("failed to create admin user: %w", err)
		}

		userRole := models.UserRole{UserID: user.ID, RoleID: role.ID}
		if err := tx.Create(&userRole).Error; err != nil {
			return fmt.Errorf("failed to assign role to user: %w", err)
		}

		return nil
	})
}

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

// GetUsersWithAlbumPermissions returns all users who have direct album permissions for a specific album
func (r *GormUserRepository) GetUsersWithAlbumPermissions(albumID uint) ([]models.User, map[uint]models.UserAlbumPermission, error) {
	var users []models.User

	// get users with direct album permissions
	err := r.withRolePreloads(r.db.Joins("JOIN user_album_permissions ON users.id = user_album_permissions.user_id")).
		Where("user_album_permissions.album_id = ?", albumID).
		Find(&users).Error

	if err != nil {
		return nil, nil, err
	}

	directMap, err := r.hydrateUsersWithAlbumPermissions(users, &albumID)
	if err != nil {
		return nil, nil, err
	}

	return users, directMap, nil
}

// GetUsersWithoutAlbumPermissions returns all users who don't have direct album permissions for a specific album
func (r *GormUserRepository) GetUsersWithoutAlbumPermissions(albumID uint) ([]models.User, error) {
	var users []models.User

	// get users who don't have direct album permissions for this album
	err := r.withRolePreloads(r.db.Where("id NOT IN (SELECT user_id FROM user_album_permissions WHERE album_id = ?)", albumID)).
		Find(&users).Error

	if err != nil {
		return nil, err
	}

	if _, err := r.hydrateUsersWithAlbumPermissions(users, nil); err != nil {
		return nil, err
	}

	return users, err
}
