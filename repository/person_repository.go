package repository

import (
	"errors"
	"fmt"
	"time"

	"github.com/camden-git/mediasysbackend/models"
	"gorm.io/gorm"
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

// ListAll retrieves all people, ordered by primary_name, preloading Aliases
func (r *PersonRepository) ListAll() ([]models.Person, error) {
	var people []models.Person
	err := r.DB.Preload("Aliases").Order("primary_name ASC").Find(&people).Error
	if err != nil {
		return nil, fmt.Errorf("failed to list people: %w", err)
	}
	return people, nil
}

// Update updates an existing person's details
func (r *PersonRepository) Update(person *models.Person) error {
	person.UpdatedAt = time.Now().Unix()
	// result := r.DB.Model(&models.Person{ID: person.ID}).Updates(map[string]interface{}{
	// 	"PrimaryName": person.PrimaryName,
	// 	"UpdatedAt":   person.UpdatedAt,
	// })
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

// Delete removes a person by their ID
func (r *PersonRepository) Delete(id uint) error {
	// result := r.DB.Unscoped().Delete(&models.Person{}, id)

	// result := r.DB.Delete(&models.Person{}, id)

	result := r.DB.Delete(&models.Person{}, id)

	if result.Error != nil {
		return fmt.Errorf("failed to delete person ID %d: %w", id, result.Error)
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
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
	likeQuery := "%" + query + "%"

	err := r.DB.Model(&models.Person{}).Where("primary_name LIKE ?", likeQuery).Pluck("id", &ids).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("error searching people by primary name for '%s': %w", query, err)
	}

	var aliasPersonIDs []uint
	err = r.DB.Model(&models.Alias{}).Where("name LIKE ?", likeQuery).Pluck("person_id", &aliasPersonIDs).Error
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

// FindImagesByPersonIDs retrieves distinct image paths associated with a list of person IDs
func (r *PersonRepository) FindImagesByPersonIDs(personIDs []uint) ([]string, error) {
	if len(personIDs) == 0 {
		return []string{}, nil
	}
	var imagePaths []string
	err := r.DB.Model(&models.Face{}).
		Where("person_id IN ?", personIDs).
		Order("image_path ASC").
		Distinct().
		Pluck("image_path", &imagePaths).Error

	if err != nil {
		return nil, fmt.Errorf("failed to find images by person IDs: %w", err)
	}
	return imagePaths, nil
}

// GetPersonWithAliases retrieves a person and their aliases
func (r *PersonRepository) GetPersonWithAliases(personID uint) (*models.Person, error) {
	var person models.Person
	err := r.DB.Preload("Aliases").First(&person, personID).Error
	if err != nil {
		return nil, err
	}
	return &person, nil
}
