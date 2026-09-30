package repository

import (
	"github.com/camden-git/mediasysbackend/media"
	"github.com/camden-git/mediasysbackend/models"
)

// AlbumGroupRepositoryInterface defines the methods for album group data operations
type AlbumGroupRepositoryInterface interface {
	Create(group *models.AlbumGroup) error
	GetByID(id uint) (*models.AlbumGroup, error)
	GetBySlug(slug string) (*models.AlbumGroup, error)
	ListAll() ([]models.AlbumGroup, error)
	ListAllAdmin() ([]models.AlbumGroup, error)
	Update(groupID uint, name, slug string, description *string, isHidden bool) error
	// Delete soft-deletes the group, detaches its albums and returns its banner object key.
	Delete(id uint) (*string, error)
	SetBannerPath(groupID uint, bannerPath *string) error
	// SetAlbumGroup returns gorm.ErrRecordNotFound for a missing album and ErrGroupNotFound for a missing group.
	SetAlbumGroup(albumID uint, groupID *uint) error
}

// AlbumRepositoryInterface defines the methods for album data operations
type AlbumRepositoryInterface interface {
	Create(album *models.Album) error
	ListAll() ([]models.Album, error)
	ListAllAdmin() ([]models.Album, error)
	GetByID(id uint) (*models.Album, error)
	GetBySlug(slug string) (*models.Album, error)
	FolderPathConflicts(folder string) (bool, error)
	Update(albumID uint, name string, description *string, isHidden *bool, location *string, sortOrder *string) error
	RequestZip(albumID uint) error
	InvalidateZip(albumID uint) (*string, error)
	MarkZipProcessing(albumID uint) error
	SetZipResult(albumID uint, zipPath *string, zipSize *int64, taskErr error) error
	ListPendingZips() ([]models.Album, error)
	UpdateSortOrder(albumID uint, sortOrder string) error
	// DeleteCascade removes the album with its images, banners, tags and permissions in one
	// transaction and returns the object keys to delete from storage.
	DeleteCascade(albumID uint) ([]string, error)
	GetBanners(albumID uint) ([]models.AlbumBanner, error)
	AddBanner(banner *models.AlbumBanner) error
	DeleteBanner(bannerID uint, albumID uint) error
	ReorderBanners(albumID uint, orderedIDs []uint) error
}

// PersonRepositoryInterface defines the methods for person data operations
type PersonRepositoryInterface interface {
	Create(person *models.Person) error
	CreateWithAliases(person *models.Person, aliases []string) error
	GetBasic(id uint) (*models.Person, error)
	GetByID(id uint) (*models.Person, error)
	GetPublicByID(id uint) (*models.Person, error)
	ListAll() ([]models.Person, error)
	Update(person *models.Person) error
	UpdateKeyPhoto(personID uint, faceID *uint) error
	Delete(id uint) error
	AddAlias(alias *models.Alias) error
	ListAliasesByPersonID(personID uint) ([]models.Alias, error)
	DeleteAlias(aliasID uint) error
	FindPersonIDsByNameOrAlias(query string) ([]uint, error)
	FindImagesByPersonIDs(personIDs []uint, offset, limit int) ([]PersonImageResult, int64, error)
	SearchByNameOrAlias(query string, limit int) ([]models.Person, error)
}

// ImageRepositoryInterface defines the methods for image data operations
type ImageRepositoryInterface interface {
	GetByPath(originalPath string) (*models.Image, error)
	Upsert(img *models.Image) (*models.Image, error)
	MarkTaskProcessing(originalPath, taskStatusColumn string) error
	UpdateThumbnailResult(originalPath string, thumbKey *string, taskErr error) error
	UpdatePreviewResult(originalPath string, previewKey *string, taskErr error) error
	UpdateMetadataResult(originalPath string, meta *media.Metadata, taskErr error) error
	UpdateDetectionResult(originalPath string, detections []media.DetectionResult, taskErr error) error
	RequeueTask(originalPath, taskStatusColumn string) error
	ResetInterruptedTasks() error
	ListPendingProcessing(limit int) ([]models.Image, error)
	ListPendingDetection(limit int) ([]models.Image, error)
	ListByAlbum(albumID uint, minRating *int) ([]models.Image, error)
	// ListByAlbumPaged returns a sorted, paginated page of images in an album (sorting
	// and paging performed in SQL), plus the total number of matching images.
	ListByAlbumPaged(albumID uint, minRating *int, sortOrder string, offset, limit int) ([]models.Image, int, error)
	GetImagesByPaths(originalPaths []string) ([]models.Image, error)
	GetImagesByAlbumIDs(albumIDs []uint, minRating *int, offset, limit int) ([]models.Image, int, error)
	ListUploadersByAlbum(albumID uint) ([]models.User, error)
	DeleteImages(paths []string) ([]string, error)
}

// FaceRepositoryInterface defines the methods for face data operations
type FaceRepositoryInterface interface {
	Create(face *models.Face) error
	GetByID(id uint) (*models.Face, error)
	ListByImagePath(imagePath string) ([]models.Face, error)
	Update(faceID uint, personID *uint, x1, y1, x2, y2 *int) error
	Delete(id uint) error
	TagFace(faceID uint, personID uint, confirmed bool) error
	UntagFace(faceID uint) error
}

// UntaggedFaceFilter holds filter and sort options for querying untagged face embeddings.
type UntaggedFaceFilter struct {
	MinQuality    *float32 // minimum faces.quality_score (0–100)
	MinConfidence *float32 // minimum faces.detection_confidence (0–1)
	SortBy        string   // "quality" | "confidence" | "created_at"
	SortOrder     string   // "asc" | "desc"
	GroupByImage  bool     // keep only the best face per image_path
}

// FaceEmbeddingRepositoryInterface defines the methods for face embedding data operations
type FaceEmbeddingRepositoryInterface interface {
	GetByFaceID(faceID uint) (*models.FaceEmbedding, error)
	DeleteByFaceID(faceID uint) error
	ListUntagged(filter UntaggedFaceFilter, albumID *uint, limit int) ([]UntaggedFace, error)
	FindSimilarFaces(targetEmbedding []float32, embeddingModel string, excludeFaceID uint, threshold float32, limit int) ([]SimilarFace, error)
	NeighborsForFaces(faceIDs []uint, threshold float32, perFace int) ([]SimilarFace, error)
}

// UserRepository defines the methods for user data operations
type UserRepository interface {
	Create(user *models.User) error
	CreateWithInviteCode(user *models.User, code string) error // claims an invite code use and creates the user atomically
	GetByID(id uint) (*models.User, error)
	GetByUsername(username string) (*models.User, error)
	Update(user *models.User) error                          // saves the user's own columns only, not roles
	UpdateWithRoles(user *models.User, roleIDs []uint) error // saves the user and replaces its roles atomically
	Delete(id uint) error
	ListAll() ([]models.User, error)
	CountAll() (int64, error)

	// CreateFirstAdmin atomically creates the initial admin user and assigns
	// them the named role, failing if any user already exists.
	CreateFirstAdmin(user *models.User, roleName string) error

	// direct album-specific permission management for a user
	CreateUserAlbumPermission(uap *models.UserAlbumPermission) error
	GetUserAlbumPermission(userID, albumID uint) (*models.UserAlbumPermission, error)
	UpdateUserAlbumPermission(uap *models.UserAlbumPermission) error
	DeleteUserAlbumPermission(userID, albumID uint) error
	GetUserAlbumPermissions(userID uint) ([]models.UserAlbumPermission, error)

	// album-specific user management
	GetUsersWithAlbumPermissions(albumID uint) ([]models.User, map[uint]models.UserAlbumPermission, error) // get all users who have permissions for a specific album
	GetUsersWithoutAlbumPermissions(albumID uint) ([]models.User, error)                                   // get all users who don't have permissions for a specific album
}

// RoleRepository defines the methods for role data operations
type RoleRepository interface {
	Create(role *models.Role) error
	CreateWithAlbumPermissions(role *models.Role, albumPerms []models.RoleAlbumPermission) error // atomic
	GetByID(id uint) (*models.Role, error)
	GetByName(name string) (*models.Role, error)
	ListAll() ([]models.Role, error)
	Update(role *models.Role) error                                                               // General update
	UpdateWithAlbumPermissions(role *models.Role, albumPerms *[]models.RoleAlbumPermission) error // atomic; nil albumPerms leaves them unchanged
	Delete(id uint) error

	// album-specific permissions for a role
	GetRoleAlbumPermissions(roleID uint) ([]models.RoleAlbumPermission, error)

	// user-Role Management
	FindUsersByRoleID(roleID uint) ([]models.User, error)
	AddUserToRole(userID, roleID uint) error
	RemoveUserFromRole(userID, roleID uint) error
}

// ImageTagRepositoryInterface defines the methods for image tag data operations
type ImageTagRepositoryInterface interface {
	GetTagsByImagePath(imagePath string) ([]models.ImageTag, error)
	// SetXMPTags replaces all source="xmp" tags for the given image path within a transaction.
	SetXMPTags(imagePath string, tags []models.ImageTag) error
	AddManualTag(imagePath, key, value string) error
	RemoveManualTag(imagePath, key, value string) error
	// ApplyAlbumDefaultTags copies album_default_tags for the album → image_tags with source="album_default", INSERT OR IGNORE.
	ApplyAlbumDefaultTags(imagePath string, albumID uint) error
	GetAlbumDefaultTags(albumID uint) ([]models.AlbumDefaultTag, error)
	// SetAlbumDefaultTags replaces all default tags for an album in a transaction.
	SetAlbumDefaultTags(albumID uint, tags []models.AlbumDefaultTag) error
}

// CollectionRepositoryInterface defines the methods for collection data operations
type CollectionRepositoryInterface interface {
	Create(collection *models.Collection) error
	GetByID(id uint) (*models.Collection, error)
	GetBySlug(slug string) (*models.Collection, error)
	ListPublic() ([]models.Collection, error)
	ListAll() ([]models.Collection, error)
	Update(collectionID uint, name, slug string, description *string, isPublic bool, filterMatch string, sortOrder string) error
	Delete(id uint) error
	// SetFilters replaces all tag filters for a collection in a transaction.
	SetFilters(collectionID uint, filters []models.CollectionTagFilter) error
	// ListImages returns a sorted, paginated page of images matching the collection's
	// tag filters (sorting and paging performed in SQL via a subquery/join), plus the
	// total number of matching images.
	ListImages(collectionID uint, sortOrder string, offset, limit int) ([]models.Image, int, error)
	GetBanners(collectionID uint) ([]models.CollectionBanner, error)
	AddBanner(banner *models.CollectionBanner) error
	DeleteCollectionBanner(bannerID uint, collectionID uint) error
	ReorderCollectionBanners(collectionID uint, orderedIDs []uint) error
	SetInheritBanners(collectionID uint, inherit bool) error
	GetInheritedBannerPaths(collectionID uint) ([]string, error)
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
