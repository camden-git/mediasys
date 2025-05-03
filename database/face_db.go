package database

import (
	"database/sql"
	"fmt"
	"log"
	"path/filepath"
	"time"

	sq "github.com/Masterminds/squirrel"
)

type Face struct {
	ID         int64   `json:"id"`
	PersonID   *int64  `json:"person_id,omitempty"`
	ImagePath  string  `json:"image_path"`
	X1         int     `json:"x1"`
	Y1         int     `json:"y1"`
	X2         int     `json:"x2"`
	Y2         int     `json:"y2"`
	CreatedAt  int64   `json:"created_at"`
	UpdatedAt  int64   `json:"updated_at"`
	PersonName *string `json:"person_name,omitempty"`
}

func AddFace(db Querier, personID *int64, imagePath string, x1, y1, x2, y2 int) (int64, error) {
	now := time.Now().Unix()
	imagePath = filepath.ToSlash(imagePath)
	queryBuilder := psql.Insert("faces").
		Columns("person_id", "image_path", "x1", "y1", "x2", "y2", "created_at", "updated_at").
		Values(personID, imagePath, x1, y1, x2, y2, now, now).
		Suffix("RETURNING id")
	sqlStr, args, err := queryBuilder.ToSql()
	if err != nil {
		return 0, fmt.Errorf("failed to build SQL for AddFace: %w", err)
	}
	var faceID int64
	err = db.QueryRow(sqlStr, args...).Scan(&faceID)
	if err != nil {
		return 0, fmt.Errorf("failed to execute AddFace query: %w", err)
	}
	return faceID, nil
}

func scanFaceRow(scanner interface {
	Scan(dest ...interface{}) error
}) (Face, error) {
	var f Face
	var nullablePersonID sql.NullInt64
	var nullablePersonName sql.NullString
	err := scanner.Scan(
		&f.ID, &nullablePersonID, &f.ImagePath, &f.X1, &f.Y1, &f.X2, &f.Y2,
		&f.CreatedAt, &f.UpdatedAt, &nullablePersonName,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return Face{}, sql.ErrNoRows
		}
		return Face{}, fmt.Errorf("failed to scan face row: %w", err)
	}
	if nullablePersonID.Valid {
		f.PersonID = &nullablePersonID.Int64
	}
	if nullablePersonName.Valid {
		f.PersonName = &nullablePersonName.String
	}
	return f, nil
}

func GetFaceByID(db Querier, faceID int64) (Face, error) {
	queryBuilder := psql.Select("f.id", "f.person_id", "f.image_path", "f.x1", "f.y1", "f.x2", "f.y2", "f.created_at", "f.updated_at", "p.primary_name").
		From("faces f").
		LeftJoin("people p ON f.person_id = p.id").
		Where(sq.Eq{"f.id": faceID}).
		Limit(1)
	sqlStr, args, err := queryBuilder.ToSql()
	if err != nil {
		return Face{}, fmt.Errorf("failed to build SQL for GetFaceByID: %w", err)
	}
	row := db.QueryRow(sqlStr, args...)
	face, err := scanFaceRow(row)
	if err != nil {
		return Face{}, fmt.Errorf("GetFaceByID failed for ID %d: %w", faceID, err)
	}
	return face, nil
}

func ListFacesByImagePath(db Querier, imagePath string) ([]Face, error) {
	imagePath = filepath.ToSlash(imagePath)
	queryBuilder := psql.Select("f.id", "f.person_id", "f.image_path", "f.x1", "f.y1", "f.x2", "f.y2", "f.created_at", "f.updated_at", "p.primary_name").
		From("faces f").
		LeftJoin("people p ON f.person_id = p.id").
		Where(sq.Eq{"f.image_path": imagePath}).
		OrderBy("f.id ASC")
	sqlStr, args, err := queryBuilder.ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build SQL for ListFacesByImagePath: %w", err)
	}
	rows, err := db.Query(sqlStr, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to execute ListFacesByImagePath query for %s: %w", imagePath, err)
	}
	defer rows.Close()
	faces := []Face{}
	for rows.Next() {
		face, err := scanFaceRow(rows)
		if err != nil {
			log.Printf("Error scanning face row for image %s: %v", imagePath, err)
			continue
		}
		faces = append(faces, face)
	}
	if err = rows.Err(); err != nil {
		return faces, fmt.Errorf("error iterating face rows for %s: %w", imagePath, err)
	}
	return faces, nil
}

// UpdateFace allows updating coordinates and optionally setting/unsetting personID
func UpdateFace(db Querier, faceID int64, personID *int64, x1, y1, x2, y2 *int) error {
	// deed explicit tracking if personID was meant to be updated (even to NULL) vs not provided
	// the handler logic should decide this and pass personID pointer accordingly
	now := time.Now().Unix()
	updateBuilder := psql.Update("faces").Where(sq.Eq{"id": faceID})
	hasUpdates := false

	// check if the pointer itself is non-nil, indicating the field was in the request
	if personID != nil {
		updateBuilder = updateBuilder.Set("person_id", *personID) // dereference pointer here
		hasUpdates = true
	}
	if x1 != nil {
		updateBuilder = updateBuilder.Set("x1", *x1)
		hasUpdates = true
	}
	if y1 != nil {
		updateBuilder = updateBuilder.Set("y1", *y1)
		hasUpdates = true
	}
	if x2 != nil {
		updateBuilder = updateBuilder.Set("x2", *x2)
		hasUpdates = true
	}
	if y2 != nil {
		updateBuilder = updateBuilder.Set("y2", *y2)
		hasUpdates = true
	}

	if !hasUpdates {
		return nil
	}

	updateBuilder = updateBuilder.Set("updated_at", now)
	sqlStr, args, err := updateBuilder.ToSql()
	if err != nil {
		return fmt.Errorf("failed to build SQL for UpdateFace: %w", err)
	}
	result, err := db.Exec(sqlStr, args...)
	if err != nil {
		return fmt.Errorf("failed to execute UpdateFace for ID %d: %w", faceID, err)
	}
	rowsAffected, err := result.RowsAffected()
	if err == nil && rowsAffected == 0 {
		return sql.ErrNoRows
	}
	if err != nil {
		log.Printf("Warning: Could not get RowsAffected for UpdateFace ID %d: %v", faceID, err)
	}
	return nil
}

func DeleteFace(db Querier, faceID int64) error {
	queryBuilder := psql.Delete("faces").Where(sq.Eq{"id": faceID})
	sqlStr, args, err := queryBuilder.ToSql()
	if err != nil {
		return fmt.Errorf("failed to build SQL for DeleteFace: %w", err)
	}
	result, err := db.Exec(sqlStr, args...)
	if err != nil {
		return fmt.Errorf("failed to execute DeleteFace for ID %d: %w", faceID, err)
	}
	rowsAffected, err := result.RowsAffected()
	if err == nil && rowsAffected == 0 {
		return sql.ErrNoRows
	}
	if err != nil {
		log.Printf("Warning: Could not get RowsAffected for DeleteFace ID %d: %v", faceID, err)
	}
	return nil
}

func DeleteUntaggedFacesByImagePath(db Querier, imagePath string) (int64, error) {
	imagePath = filepath.ToSlash(imagePath)
	queryBuilder := psql.Delete("faces").
		Where(sq.Eq{"image_path": imagePath}).
		Where(sq.Eq{"person_id": nil})

	sqlStr, args, err := queryBuilder.ToSql()
	if err != nil {
		return 0, fmt.Errorf("failed to build SQL for DeleteUntaggedFacesByImagePath: %w", err)
	}

	result, err := db.Exec(sqlStr, args...)
	if err != nil {
		return 0, fmt.Errorf("failed to execute DeleteUntaggedFacesByImagePath for %s: %w", imagePath, err)
	}

	rowsAffected, _ := result.RowsAffected()
	return rowsAffected, nil
}
