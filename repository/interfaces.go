package repository

import (
	"github.com/camden-git/mediasysbackend/media"
	"github.com/camden-git/mediasysbackend/models"
)

// AlbumRepositoryInterface defines the methods for album data operations
type AlbumRepositoryInterface interface {
	Create(album *models.Album) error
	ListAll() ([]models.Album, error)
	ListAllAdmin() ([]models.Album, error)
	GetByID(id uint) (*models.Album, error)
	GetBySlug(slug string) (*models.Album, error)
	Update(albumID uint, name string, description *string, isHidden *bool, location *string) error
	RequestZip(albumID uint) error
	MarkZipProcessing(albumID uint) error
	SetZipResult(albumID uint, zipPath *string, zipSize *int64, taskErr error) error
	UpdateBannerPath(albumID uint, bannerPath *string) error
	UpdateSortOrder(albumID uint, sortOrder string) error
	Delete(id uint) error
}

// PersonRepositoryInterface defines the methods for person data operations
type PersonRepositoryInterface interface {
	Create(person *models.Person) error
	GetByID(id uint) (*models.Person, error)
	ListAll() ([]models.Person, error)
	Update(person *models.Person) error
	Delete(id uint) error
	AddAlias(alias *models.Alias) error
	ListAliasesByPersonID(personID uint) ([]models.Alias, error)
	DeleteAlias(aliasID uint) error
	FindPersonIDsByNameOrAlias(query string) ([]uint, error)
	FindImagesByPersonIDs(personIDs []uint) ([]string, error)
}

// ImageRepositoryInterface defines the methods for image data operations
type ImageRepositoryInterface interface {
	GetByPath(originalPath string) (*models.Image, error)
	EnsureExists(originalPath string, modTime int64) (bool, error)
	MarkTaskProcessing(originalPath, taskStatusColumn string) error
	UpdateThumbnailResult(originalPath string, thumbPath *string, modTime int64, taskErr error) error
	UpdateMetadataResult(originalPath string, meta *media.Metadata, modTime int64, taskErr error) error
	UpdateDetectionResult(originalPath string, detections []media.DetectionResult, modTime int64, taskErr error) error
	Delete(originalPath string) error
	GetImagesRequiringProcessing() ([]models.Image, error)
	GetImagesByPaths(originalPaths []string) ([]models.Image, error)
}

// FaceRepositoryInterface defines the methods for face data operations
type FaceRepositoryInterface interface {
	Create(face *models.Face) error
	GetByID(id uint) (*models.Face, error)
	ListByImagePath(imagePath string) ([]models.Face, error)
	Update(faceID uint, personID *uint, x1, y1, x2, y2 *int) error
	Delete(id uint) error
	DeleteUntaggedByImagePath(imagePath string) (int64, error)
	TagFace(faceID uint, personID uint) error
	UntagFace(faceID uint) error
}

// UserRepository defines the methods for user data operations
type UserRepository interface {
	Create(user *models.User) error
	GetByID(id uint) (*models.User, error)
	GetByUsername(username string) (*models.User, error)
	Update(user *models.User) error
	Delete(id uint) error
	ListAll() ([]models.User, error)

	// role management for a user
	AddRoleToUser(userID uint, roleID uint) error
	RemoveRoleFromUser(userID uint, roleID uint) error
	GetUserRoles(userID uint) ([]models.Role, error)

	// direct global permission management for a user
	SetUserGlobalPermissions(userID uint, permissions []string) error

	// direct album-specific permission management for a user
	CreateUserAlbumPermission(uap *models.UserAlbumPermission) error
	GetUserAlbumPermission(userID, albumID uint) (*models.UserAlbumPermission, error)
	UpdateUserAlbumPermission(uap *models.UserAlbumPermission) error
	DeleteUserAlbumPermission(userID, albumID uint) error
	GetUserAlbumPermissions(userID uint) ([]models.UserAlbumPermission, error)

	// album-specific user management
	GetUsersWithAlbumPermissions(albumID uint) ([]models.User, error)    // get all users who have permissions for a specific album
	GetUsersWithoutAlbumPermissions(albumID uint) ([]models.User, error) // get all users who don't have permissions for a specific album
}

// RoleRepository defines the methods for role data operations
type RoleRepository interface {
	Create(role *models.Role) error
	GetByID(id uint) (*models.Role, error)
	GetByName(name string) (*models.Role, error)
	ListAll() ([]models.Role, error)
	Update(role *models.Role) error // General update
	Delete(id uint) error

	// global permission management for a role
	SetRoleGlobalPermissions(roleID uint, permissions []string) error

	// album-specific permission management for a role
	CreateRoleAlbumPermission(rap *models.RoleAlbumPermission) error
	GetRoleAlbumPermission(roleID, albumID uint) (*models.RoleAlbumPermission, error)
	UpdateRoleAlbumPermission(rap *models.RoleAlbumPermission) error
	DeleteRoleAlbumPermission(roleID, albumID uint) error
	GetRoleAlbumPermissions(roleID uint) ([]models.RoleAlbumPermission, error)

	// user-Role Management
	FindUsersByRoleID(roleID uint) ([]models.User, error)
	AddUserToRole(userID, roleID uint) error
	RemoveUserFromRole(userID, roleID uint) error
}

// InviteCodeRepository defines the methods for invite code data operations
type InviteCodeRepository interface {
	Create(inviteCode *models.InviteCode) error
	GetByCode(code string) (*models.InviteCode, error)
	GetByID(id uint) (*models.InviteCode, error)
	Update(inviteCode *models.InviteCode) error
	IncrementUses(id uint) error
	ListAll() ([]models.InviteCode, error)
	Delete(id uint) error
}
