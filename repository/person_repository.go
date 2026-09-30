package repository

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/camden-git/mediasysbackend/database"
	"github.com/camden-git/mediasysbackend/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// PersonRepository handles database operations for Person and related Alias entities
type PersonRepository struct {
	DB *gorm.DB
}

// NewPersonRepository creates a new instance of PersonRepository
func NewPersonRepository(db *gorm.DB) *PersonRepository {
	return &PersonRepository{DB: db}
}

// Create creates a new person record in the database
func (r *PersonRepository) Create(person *models.Person) error {
	now := time.Now().Unix()
	if person.CreatedAt == 0 {
		person.CreatedAt = now
	}
	if person.UpdatedAt == 0 {
		person.UpdatedAt = now
	}

	err := r.DB.Create(person).Error
	if err != nil {
		return fmt.Errorf("failed to create person %s: %w", person.PrimaryName, err)
	}
	return nil
}

// escapeLike escapes LIKE/ILIKE wildcards so user input matches literally.
func escapeLike(s string) string {
	return strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(s)
}

// CreateWithAliases creates a person and their initial aliases in one transaction.
func (r *PersonRepository) CreateWithAliases(person *models.Person, aliases []string) error {
	return r.DB.Transaction(func(tx *gorm.DB) error {
		now := time.Now().Unix()
		if person.CreatedAt == 0 {
			person.CreatedAt = now
		}
		if person.UpdatedAt == 0 {
			person.UpdatedAt = now
		}
		if err := tx.Create(person).Error; err != nil {
			return fmt.Errorf("failed to create person %s: %w", person.PrimaryName, err)
		}
		for _, name := range aliases {
			alias := models.Alias{PersonID: person.ID, Name: name}
			if err := tx.Create(&alias).Error; err != nil {
				return fmt.Errorf("failed to add alias '%s' for person ID %d: %w", name, person.ID, err)
			}
		}
		return nil
	})
}

// GetBasic retrieves a person's own columns only (no aliases or faces); use it as a
// cheap existence check.
func (r *PersonRepository) GetBasic(id uint) (*models.Person, error) {
	var person models.Person
	if err := r.DB.First(&person, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
		return nil, fmt.Errorf("failed to get person by ID %d: %w", id, err)
	}
	return &person, nil
}

// GetByID retrieves a person by their ID, preloading Aliases and Faces
func (r *PersonRepository) GetByID(id uint) (*models.Person, error) {
	var person models.Person
	err := r.DB.Preload("Aliases").Preload("Faces").First(&person, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
		return nil, fmt.Errorf("failed to get person by ID %d: %w", id, err)
	}
	return &person, nil
}

// visibleImagePathsSQL selects the paths of images that are not soft-deleted and do not
// belong to a hidden album, i.e. images that may appear in public listings.
const visibleImagePathsSQL = "SELECT i.original_path FROM images i JOIN albums a ON a.id = i.album_id " +
	"WHERE i.deleted_at IS NULL AND a.deleted_at IS NULL AND a.is_hidden = false"

// GetPublicByID retrieves a person like GetByID, but only preloads faces whose images are
// not in hidden albums so the result is safe for public responses.
func (r *PersonRepository) GetPublicByID(id uint) (*models.Person, error) {
	var person models.Person
	err := r.DB.Preload("Aliases").
		Preload("Faces", "image_path IN ("+visibleImagePathsSQL+")").
		First(&person, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
		return nil, fmt.Errorf("failed to get person by ID %d: %w", id, err)
	}
	return &person, nil
}

// ListAll retrieves all people, ordered by primary_name, preloading Aliases
func (r *PersonRepository) ListAll() ([]models.Person, error) {
	var people []models.Person
	err := r.DB.Preload("Aliases").Order("primary_name ASC").Find(&people).Error
	if err != nil {
		return nil, fmt.Errorf("failed to list people: %w", err)
	}
	return people, nil
}

// UpdateKeyPhoto sets or clears the key_photo_face_id for a person.
// Pass nil to clear the key photo.
func (r *PersonRepository) UpdateKeyPhoto(personID uint, faceID *uint) error {
	updates := map[string]interface{}{
		"key_photo_face_id": faceID,
		"updated_at":        time.Now().Unix(),
	}
	result := r.DB.Model(&models.Person{}).Where("id = ?", personID).Updates(updates)
	if result.Error != nil {
		return fmt.Errorf("failed to update key photo for person ID %d: %w", personID, result.Error)
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// Update updates an existing person's details
func (r *PersonRepository) Update(person *models.Person) error {
	person.UpdatedAt = time.Now().Unix()
	result := r.DB.Model(&models.Person{ID: person.ID}).Updates(models.Person{
		PrimaryName: person.PrimaryName,
		UpdatedAt:   person.UpdatedAt,
	})

	if result.Error != nil {
		return fmt.Errorf("failed to update person ID %d: %w", person.ID, result.Error)
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// Delete removes a person by their ID. In one transaction it unassigns the person's faces
// (so they return to the untagged queue), deletes their aliases, then deletes the person.
func (r *PersonRepository) Delete(id uint) error {
	return r.DB.Transaction(func(tx *gorm.DB) error {
		// lock the person row so concurrent tagging can't race the delete
		var person models.Person
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Select("id").First(&person, id).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
			return fmt.Errorf("failed to load person ID %d for delete: %w", id, err)
		}

		err := tx.Model(&models.Face{}).Where("person_id = ?", id).Updates(map[string]interface{}{
			"person_id":  gorm.Expr("NULL"),
			"confirmed":  false,
			"updated_at": time.Now().Unix(),
		}).Error
		if err != nil {
			return fmt.Errorf("failed to unassign faces of person ID %d: %w", id, err)
		}
		if err := tx.Where("person_id = ?", id).Delete(&models.Alias{}).Error; err != nil {
			return fmt.Errorf("failed to delete aliases of person ID %d: %w", id, err)
		}
		if err := tx.Delete(&models.Person{}, id).Error; err != nil {
			return fmt.Errorf("failed to delete person ID %d: %w", id, err)
		}
		return nil
	})
}

// AddAlias adds a new alias for a person
func (r *PersonRepository) AddAlias(alias *models.Alias) error {
	err := r.DB.Create(alias).Error
	if err != nil {
		return fmt.Errorf("failed to add alias '%s' for person ID %d: %w", alias.Name, alias.PersonID, err)
	}
	return nil
}

// ListAliasesByPersonID retrieves all aliases for a given person ID
func (r *PersonRepository) ListAliasesByPersonID(personID uint) ([]models.Alias, error) {
	var aliases []models.Alias
	err := r.DB.Where("person_id = ?", personID).Order("name ASC").Find(&aliases).Error
	if err != nil {
		return nil, fmt.Errorf("failed to list aliases for person ID %d: %w", personID, err)
	}
	return aliases, nil
}

// DeleteAlias removes an alias by its ID.
func (r *PersonRepository) DeleteAlias(aliasID uint) error {
	result := r.DB.Delete(&models.Alias{}, aliasID) // Assumes models.Alias has gorm.Model for soft/hard delete behavior
	if result.Error != nil {
		return fmt.Errorf("failed to delete alias ID %d: %w", aliasID, result.Error)
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// FindPersonIDsByNameOrAlias searches for people by primary name or alias name
// Returns a slice of unique person IDs
func (r *PersonRepository) FindPersonIDsByNameOrAlias(query string) ([]uint, error) {
	var ids []uint
	likeQuery := "%" + escapeLike(query) + "%"

	err := r.DB.Model(&models.Person{}).Where("primary_name ILIKE ?", likeQuery).Pluck("id", &ids).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("error searching people by primary name for '%s': %w", query, err)
	}

	var aliasPersonIDs []uint
	err = r.DB.Model(&models.Alias{}).Where("name ILIKE ?", likeQuery).Pluck("person_id", &aliasPersonIDs).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("error searching aliases by name for '%s': %w", query, err)
	}

	idMap := make(map[uint]bool)
	for _, id := range ids {
		idMap[id] = true
	}
	for _, id := range aliasPersonIDs {
		idMap[id] = true
	}

	uniqueIDs := make([]uint, 0, len(idMap))
	for id := range idMap {
		uniqueIDs = append(uniqueIDs, id)
	}

	return uniqueIDs, nil
}

// PersonImageResult is an image associated with a person, along with its thumbnail (when
// available) and dimensions so callers can render a grid without falling back to
// full-size previews.
type PersonImageResult struct {
	ImagePath     string  `json:"image_path"`
	ThumbnailPath *string `json:"thumbnail_path,omitempty"`
	Width         *int    `json:"width,omitempty"`
	Height        *int    `json:"height,omitempty"`
}

// FindImagesByPersonIDs retrieves one page of the distinct visible (not in a hidden album)
// images that contain any of the given people, ordered by path, along with the total
// number of matching images.
func (r *PersonRepository) FindImagesByPersonIDs(personIDs []uint, offset, limit int) ([]PersonImageResult, int64, error) {
	if len(personIDs) == 0 {
		return []PersonImageResult{}, 0, nil
	}
	base := func() *gorm.DB {
		return r.DB.Table("faces AS f").
			Joins("JOIN images i ON i.original_path = f.image_path AND i.deleted_at IS NULL").
			Joins("JOIN albums a ON a.id = i.album_id AND a.deleted_at IS NULL AND a.is_hidden = false").
			Where("f.deleted_at IS NULL AND f.person_id IN ?", personIDs)
	}

	var total int64
	if err := base().Select("COUNT(DISTINCT i.original_path)").Scan(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count images by person IDs: %w", err)
	}
	if total == 0 {
		return []PersonImageResult{}, 0, nil
	}

	var rows []struct {
		OriginalPath    string
		ThumbnailPath   *string
		ThumbnailStatus string
		Width           *int
		Height          *int
	}
	q := base().
		Select("i.original_path, i.thumbnail_path, i.thumbnail_status, i.width, i.height").
		Group("i.original_path").
		Order("i.original_path ASC").
		Limit(limit).
		Offset(offset)
	if err := q.Scan(&rows).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to find images by person IDs: %w", err)
	}

	results := make([]PersonImageResult, 0, len(rows))
	for _, row := range rows {
		res := PersonImageResult{ImagePath: row.OriginalPath, Width: row.Width, Height: row.Height}
		if row.ThumbnailPath != nil && row.ThumbnailStatus == database.StatusDone {
			u := "/" + *row.ThumbnailPath
			res.ThumbnailPath = &u
		}
		results = append(results, res)
	}
	return results, total, nil
}

// SearchByNameOrAlias searches for people by primary name or alias, returning up to limit results.
func (r *PersonRepository) SearchByNameOrAlias(query string, limit int) ([]models.Person, error) {
	var people []models.Person
	likeQuery := "%" + escapeLike(query) + "%"
	err := r.DB.Preload("Aliases").
		Joins("LEFT JOIN aliases ON aliases.person_id = people.id").
		Where("people.primary_name ILIKE ? OR aliases.name ILIKE ?", likeQuery, likeQuery).
		Group("people.id").
		Order("people.primary_name ASC").
		Limit(limit).
		Find(&people).Error
	if err != nil {
		return nil, fmt.Errorf("failed to search people for '%s': %w", query, err)
	}
	return people, nil
}
