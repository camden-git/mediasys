package database

import (
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"

	sq "github.com/Masterminds/squirrel"
)

type Person struct {
	ID          int64   `json:"id"`
	PrimaryName string  `json:"primary_name"`
	Aliases     []Alias `json:"aliases,omitempty"` // populated by handler
	CreatedAt   int64   `json:"created_at"`
	UpdatedAt   int64   `json:"updated_at"`
}

type Alias struct {
	ID       int64  `json:"id"`
	PersonID int64  `json:"person_id"`
	Name     string `json:"name"`
}

func CreatePerson(db *sql.DB, primaryName string) (int64, error) {
	now := time.Now().Unix()
	queryBuilder := psql.Insert("people").
		Columns("primary_name", "created_at", "updated_at").
		Values(primaryName, now, now).
		Suffix("RETURNING id")
	sqlStr, args, err := queryBuilder.ToSql()
	if err != nil {
		return 0, fmt.Errorf("failed to build SQL for CreatePerson: %w", err)
	}
	var personID int64
	err = db.QueryRow(sqlStr, args...).Scan(&personID)
	if err != nil {
		return 0, fmt.Errorf("failed to execute CreatePerson query for %s: %w", primaryName, err)
	}
	return personID, nil
}

func GetPersonByID(db *sql.DB, personID int64) (Person, error) {
	var p Person
	queryBuilder := psql.Select("id", "primary_name", "created_at", "updated_at").
		From("people").
		Where(sq.Eq{"id": personID}).
		Limit(1)
	sqlStr, args, err := queryBuilder.ToSql()
	if err != nil {
		return Person{}, fmt.Errorf("failed to build SQL for GetPersonByID: %w", err)
	}
	err = db.QueryRow(sqlStr, args...).Scan(&p.ID, &p.PrimaryName, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return Person{}, sql.ErrNoRows
		}
		return Person{}, fmt.Errorf("failed to query or scan person with ID %d: %w", personID, err)
	}
	return p, nil
}

func ListPeople(db *sql.DB) ([]Person, error) {
	queryBuilder := psql.Select("id", "primary_name", "created_at", "updated_at").
		From("people").
		OrderBy("primary_name ASC")
	sqlStr, args, err := queryBuilder.ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build SQL for ListPeople: %w", err)
	}
	rows, err := db.Query(sqlStr, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to execute ListPeople query: %w", err)
	}
	defer rows.Close()
	people := []Person{}
	for rows.Next() {
		var p Person
		err := rows.Scan(&p.ID, &p.PrimaryName, &p.CreatedAt, &p.UpdatedAt)
		if err != nil {
			log.Printf("Error scanning person row: %v", err)
			continue
		}
		people = append(people, p)
	}
	if err = rows.Err(); err != nil {
		return people, fmt.Errorf("error iterating people rows: %w", err)
	}
	return people, nil
}

func UpdatePerson(db *sql.DB, personID int64, primaryName string) error {
	now := time.Now().Unix()
	queryBuilder := psql.Update("people").
		Set("primary_name", primaryName).
		Set("updated_at", now).
		Where(sq.Eq{"id": personID})
	sqlStr, args, err := queryBuilder.ToSql()
	if err != nil {
		return fmt.Errorf("failed to build SQL for UpdatePerson: %w", err)
	}
	result, err := db.Exec(sqlStr, args...)
	if err != nil {
		return fmt.Errorf("failed to execute UpdatePerson for ID %d: %w", personID, err)
	}
	rowsAffected, err := result.RowsAffected()
	if err == nil && rowsAffected == 0 {
		return sql.ErrNoRows
	}
	if err != nil {
		log.Printf("Warning: Could not get RowsAffected for UpdatePerson ID %d: %v", personID, err)
	}
	return nil
}

func DeletePerson(db *sql.DB, personID int64) error {
	queryBuilder := psql.Delete("people").Where(sq.Eq{"id": personID})
	sqlStr, args, err := queryBuilder.ToSql()
	if err != nil {
		return fmt.Errorf("failed to build SQL for DeletePerson: %w", err)
	}
	result, err := db.Exec(sqlStr, args...)
	if err != nil {
		return fmt.Errorf("failed to execute DeletePerson for ID %d: %w", personID, err)
	}
	rowsAffected, err := result.RowsAffected()
	if err == nil && rowsAffected == 0 {
		return sql.ErrNoRows
	}
	if err != nil {
		log.Printf("Warning: Could not get RowsAffected for DeletePerson ID %d: %v", personID, err)
	}
	return nil
}

func AddAlias(db *sql.DB, personID int64, name string) (int64, error) {
	queryBuilder := psql.Insert("aliases").
		Columns("person_id", "name").
		Values(personID, name).
		Suffix("RETURNING id")
	sqlStr, args, err := queryBuilder.ToSql()
	if err != nil {
		return 0, fmt.Errorf("failed to build SQL for AddAlias: %w", err)
	}
	var aliasID int64
	err = db.QueryRow(sqlStr, args...).Scan(&aliasID)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed: aliases.person_id, aliases.name") {
			return 0, fmt.Errorf("alias '%s' already exists for person %d: %w", name, personID, err)
		}
		return 0, fmt.Errorf("failed to execute AddAlias query for person %d, name %s: %w", personID, name, err)
	}
	return aliasID, nil
}

func ListAliasesByPersonID(db *sql.DB, personID int64) ([]Alias, error) {
	queryBuilder := psql.Select("id", "person_id", "name").
		From("aliases").
		Where(sq.Eq{"person_id": personID}).
		OrderBy("name ASC")
	sqlStr, args, err := queryBuilder.ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build SQL for ListAliasesByPersonID: %w", err)
	}
	rows, err := db.Query(sqlStr, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to execute ListAliasesByPersonID query for person %d: %w", personID, err)
	}
	defer rows.Close()
	aliases := []Alias{}
	for rows.Next() {
		var a Alias
		err := rows.Scan(&a.ID, &a.PersonID, &a.Name)
		if err != nil {
			log.Printf("Error scanning alias row: %v", err)
			continue
		}
		aliases = append(aliases, a)
	}
	if err = rows.Err(); err != nil {
		return aliases, fmt.Errorf("error iterating alias rows for person %d: %w", personID, err)
	}
	return aliases, nil
}

func DeleteAlias(db *sql.DB, aliasID int64) error {
	queryBuilder := psql.Delete("aliases").Where(sq.Eq{"id": aliasID})
	sqlStr, args, err := queryBuilder.ToSql()
	if err != nil {
		return fmt.Errorf("failed to build SQL for DeleteAlias: %w", err)
	}
	result, err := db.Exec(sqlStr, args...)
	if err != nil {
		return fmt.Errorf("failed to execute DeleteAlias for ID %d: %w", aliasID, err)
	}
	rowsAffected, err := result.RowsAffected()
	if err == nil && rowsAffected == 0 {
		return sql.ErrNoRows
	}
	if err != nil {
		log.Printf("Warning: Could not get RowsAffected for DeleteAlias ID %d: %v", aliasID, err)
	}
	return nil
}

func FindPersonIDsByNameOrAlias(db *sql.DB, query string) ([]int64, error) {
	peopleQuery := psql.Select("id").From("people").Where(sq.Like{"primary_name": "%" + query + "%"})
	aliasQuery := psql.Select("person_id").From("aliases").Where(sq.Like{"name": "%" + query + "%"})
	sqlStrP, argsP, errP := peopleQuery.ToSql()
	sqlStrA, argsA, errA := aliasQuery.ToSql()
	if errP != nil || errA != nil {
		return nil, fmt.Errorf("failed to build search queries (people: %v, alias: %v)", errP, errA)
	}
	fullSQL := fmt.Sprintf("(%s) UNION (%s)", sqlStrP, sqlStrA)
	allArgs := append(argsP, argsA...)
	rows, err := db.Query(fullSQL, allArgs...)
	if err != nil {
		return nil, fmt.Errorf("failed to execute combined search query for '%s': %w", query, err)
	}
	defer rows.Close()
	ids := []int64{}
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			log.Printf("Error scanning person ID from search result: %v", err)
			continue
		}
		ids = append(ids, id)
	}
	if err = rows.Err(); err != nil {
		return ids, fmt.Errorf("error iterating search results for '%s': %w", query, err)
	}
	return ids, nil
}

func FindImagesByPersonIDs(db *sql.DB, personIDs []int64) ([]string, error) {
	if len(personIDs) == 0 {
		return []string{}, nil
	}
	queryBuilder := psql.Select("DISTINCT image_path").
		From("faces").
		Where(sq.Eq{"person_id": personIDs}).
		OrderBy("image_path ASC")
	sqlStr, args, err := queryBuilder.ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build SQL for FindImagesByPersonIDs: %w", err)
	}
	rows, err := db.Query(sqlStr, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to execute FindImagesByPersonIDs query: %w", err)
	}
	defer rows.Close()
	imagePaths := []string{}
	for rows.Next() {
		var path string
		if err := rows.Scan(&path); err != nil {
			log.Printf("Error scanning image path from face search result: %v", err)
			continue
		}
		imagePaths = append(imagePaths, path)
	}
	if err = rows.Err(); err != nil {
		return imagePaths, fmt.Errorf("error iterating image path results: %w", err)
	}
	return imagePaths, nil
}
