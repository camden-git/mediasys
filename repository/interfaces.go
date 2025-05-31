package repository

import (
	"github.com/camden-git/mediasysbackend/media"
	"github.com/camden-git/mediasysbackend/models"
)

// AlbumRepositoryInterface defines the methods for album data operations
type AlbumRepositoryInterface interface {
	Create(album *models.Album) error
	ListAll() ([]models.Album, error)
	GetByID(id uint) (*models.Album, error)
	GetBySlug(slug string) (*models.Album, error)
	Update(albumID uint, name string, description *string) error
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
