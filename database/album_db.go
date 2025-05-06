package database

import (
	"database/sql"
	"fmt"
	"log"
	"path/filepath"
	"strings"
	"time"

	sq "github.com/Masterminds/squirrel"
)

type Album struct {
	ID                 int64   `json:"id"`
	Name               string  `json:"name"`
	Slug               string  `json:"slug"`
	Description        string  `json:"description,omitempty"`
	FolderPath         string  `json:"folder_path"`
	BannerImagePath    *string `json:"banner_image_path,omitempty"`
	SortOrder          string  `json:"sort_order"`
	ZipPath            *string `json:"zip_path,omitempty"`
	ZipSize            *int64  `json:"zip_size,omitempty"`
	ZipStatus          string  `json:"zip_status"`
	ZipError           *string `json:"zip_error,omitempty"`
	ZipLastGeneratedAt *int64  `json:"zip_last_generated_at,omitempty"`
	ZipLastRequestedAt *int64  `json:"zip_last_requested_at,omitempty"`

	CreatedAt int64 `json:"created_at"`
	UpdatedAt int64 `json:"updated_at"`
}

func CreateAlbum(db *sql.DB, name, slug, description, folderPath string) (int64, error) {
	now := time.Now().Unix()
	folderPath = filepath.ToSlash(folderPath)

	queryBuilder := psql.Insert("albums").
		Columns("name", "slug", "description", "folder_path", "created_at", "updated_at").
		Values(name, slug, description, folderPath, now, now).
		Suffix("RETURNING id")

	sqlStr, args, err := queryBuilder.ToSql()
	if err != nil {
		return 0, fmt.Errorf("failed to build SQL for CreateAlbum: %w", err)
	}

	var albumID int64
	err = db.QueryRow(sqlStr, args...).Scan(&albumID)
	if err != nil {
		return 0, fmt.Errorf("failed to execute CreateAlbum query for %s (slug %s): %w", name, slug, err)
	}

	return albumID, nil
}

func ListAlbums(db *sql.DB) ([]Album, error) {
	queryBuilder := psql.Select("id", "name", "slug", "description", "folder_path",
		"banner_image_path", "sort_order",
		"zip_path", "zip_size", "zip_status", "zip_error", "zip_last_generated_at", "zip_last_requested_at",
		"created_at", "updated_at").
		From("albums").
		OrderBy("name ASC")
	sqlStr, args, err := queryBuilder.ToSql()

	if err != nil {
		return nil, fmt.Errorf("failed to build SQL for ListAlbums: %w", err)
	}

	rows, err := db.Query(sqlStr, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to execute ListAlbums query: %w", err)
	}

	defer rows.Close()
	albums := []Album{}
	for rows.Next() {
		var a Album
		err := rows.Scan(&a.ID, &a.Name, &a.Slug, &a.Description, &a.FolderPath,
			&a.BannerImagePath, &a.SortOrder,
			&a.ZipPath, &a.ZipSize, &a.ZipStatus, &a.ZipError, &a.ZipLastGeneratedAt, &a.ZipLastRequestedAt,
			&a.CreatedAt, &a.UpdatedAt)
		if err != nil {
			log.Printf("Error scanning album row: %v", err)
			continue
		}
		albums = append(albums, a)
	}

	if err = rows.Err(); err != nil {
		return albums, fmt.Errorf("error iterating album rows: %w", err)
	}

	return albums, nil
}

// scanAlbumRow is a helper to scan a single album row
func scanAlbumRow(scanner interface {
	Scan(dest ...interface{}) error
}) (Album, error) {
	var a Album
	err := scanner.Scan(&a.ID, &a.Name, &a.Slug, &a.Description, &a.FolderPath,
		&a.BannerImagePath, &a.SortOrder,
		&a.ZipPath, &a.ZipSize, &a.ZipStatus, &a.ZipError, &a.ZipLastGeneratedAt, &a.ZipLastRequestedAt,
		&a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return Album{}, sql.ErrNoRows
		}
		return Album{}, fmt.Errorf("failed to scan album row: %w", err)
	}
	return a, nil
}

func GetAlbumByID(db *sql.DB, id int64) (Album, error) {
	queryBuilder := psql.Select("id", "name", "slug", "description", "folder_path",
		"banner_image_path", "sort_order",
		"zip_path", "zip_size", "zip_status", "zip_error", "zip_last_generated_at", "zip_last_requested_at",
		"created_at", "updated_at").
		From("albums").
		Where(sq.Eq{"id": id}).
		Limit(1)
	sqlStr, args, err := queryBuilder.ToSql()
	if err != nil {
		return Album{}, fmt.Errorf("failed to build SQL for GetAlbumByID: %w", err)
	}
	row := db.QueryRow(sqlStr, args...)
	album, err := scanAlbumRow(row)
	if err != nil {
		return Album{}, fmt.Errorf("GetAlbumByID failed for ID %d: %w", id, err)
	}
	return album, nil
}

func GetAlbumBySlug(db *sql.DB, slug string) (Album, error) {
	queryBuilder := psql.Select("id", "name", "slug", "description", "folder_path",
		"banner_image_path", "sort_order",
		"zip_path", "zip_size", "zip_status", "zip_error", "zip_last_generated_at", "zip_last_requested_at",
		"created_at", "updated_at").
		From("albums").
		Where(sq.Eq{"slug": slug}).
		Limit(1)
	sqlStr, args, err := queryBuilder.ToSql()
	if err != nil {
		return Album{}, fmt.Errorf("failed to build SQL for GetAlbumBySlug: %w", err)
	}
	row := db.QueryRow(sqlStr, args...)
	album, err := scanAlbumRow(row)
	if err != nil {
		return Album{}, fmt.Errorf("GetAlbumBySlug failed for slug %s: %w", slug, err)
	}
	return album, nil
}

func UpdateAlbum(db *sql.DB, id int64, name, description string) error {
	now := time.Now().Unix()
	updateBuilder := psql.Update("albums").Where(sq.Eq{"id": id})
	hasUpdates := false
	if name != "" {
		updateBuilder = updateBuilder.Set("name", name)
		hasUpdates = true
	}
	updateBuilder = updateBuilder.Set("description", description)
	hasUpdates = true // assume empty string means set description to empty
	if !hasUpdates {
		return nil
	}
	updateBuilder = updateBuilder.Set("updated_at", now)
	sqlStr, args, err := updateBuilder.ToSql()
	if err != nil {
		return fmt.Errorf("failed to build SQL for UpdateAlbum: %w", err)
	}
	result, err := db.Exec(sqlStr, args...)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed: albums.name") {
			return fmt.Errorf("album name conflict: %w", err)
		}
		return fmt.Errorf("failed to execute UpdateAlbum for ID %d: %w", id, err)
	}
	rowsAffected, err := result.RowsAffected()
	if err == nil && rowsAffected == 0 {
		return sql.ErrNoRows
	}
	if err != nil {
		log.Printf("Warning: Could not get RowsAffected for UpdateAlbum ID %d: %v", id, err)
	}
	return nil
}

func RequestAlbumZip(db Querier, albumID int64) error {
	now := time.Now().Unix()
	queryBuilder := psql.Update("albums").
		Set("zip_status", StatusPending).
		Set("zip_last_requested_at", now).
		Set("zip_error", nil).
		Set("updated_at", now).
		Where(sq.Eq{"id": albumID})

	sqlStr, args, err := queryBuilder.ToSql()
	if err != nil {
		return fmt.Errorf("failed to build SQL for RequestAlbumZip: %w", err)
	}

	res, err := db.Exec(sqlStr, args...)
	if err != nil {
		return fmt.Errorf("failed to request album zip for ID %d: %w", albumID, err)
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func MarkAlbumZipProcessing(db Querier, albumID int64) error {
	now := time.Now().Unix()
	queryBuilder := psql.Update("albums").
		Set("zip_status", StatusProcessing).
		Set("updated_at", now).
		Where(sq.Eq{"id": albumID})

	sqlStr, args, err := queryBuilder.ToSql()
	if err != nil {
		return fmt.Errorf("failed to build SQL for MarkAlbumZipProcessing: %w", err)
	}

	res, err := db.Exec(sqlStr, args...)
	if err != nil {
		return fmt.Errorf("failed to mark album zip processing for ID %d: %w", albumID, err)
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func SetAlbumZipResult(db Querier, albumID int64, zipPath *string, zipSize *int64, taskErr error) error {
	now := time.Now().Unix()
	status := StatusDone
	var errStr *string
	if taskErr != nil {
		status = StatusError
		s := taskErr.Error()
		errStr = &s
	}

	updateMap := map[string]interface{}{
		"zip_status": status,
		"zip_error":  errStr,
		"updated_at": now,
	}
	if status == StatusDone {
		updateMap["zip_path"] = zipPath
		updateMap["zip_size"] = zipSize
		updateMap["zip_last_generated_at"] = now
	}

	queryBuilder := psql.Update("albums").
		SetMap(updateMap).
		Where(sq.Eq{"id": albumID})

	sqlStr, args, err := queryBuilder.ToSql()
	if err != nil {
		return fmt.Errorf("failed to build SQL for SetAlbumZipResult: %w", err)
	}

	_, err = db.Exec(sqlStr, args...)
	if err != nil {
		return fmt.Errorf("failed to update album zip result for ID %d: %w", albumID, err)
	}
	return nil
}

func UpdateAlbumBannerPath(db Querier, albumID int64, bannerPath *string) error {
	now := time.Now().Unix()

	queryBuilder := psql.Update("albums").
		Set("banner_image_path", bannerPath).
		Set("updated_at", now).
		Where(sq.Eq{"id": albumID})

	sqlStr, args, err := queryBuilder.ToSql()
	if err != nil {
		return fmt.Errorf("failed to build SQL for UpdateAlbumBannerPath: %w", err)
	}

	result, err := db.Exec(sqlStr, args...)
	if err != nil {
		return fmt.Errorf("failed to execute UpdateAlbumBannerPath for ID %d: %w", albumID, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err == nil && rowsAffected == 0 {
		return sql.ErrNoRows
	}
	if err != nil {
		log.Printf("Warning: Could not get RowsAffected for UpdateAlbumBannerPath ID %d: %v", albumID, err)
	}
	return nil
}

func UpdateAlbumSortOrder(db Querier, albumID int64, sortOrder string) error {
	if !IsValidSortOrder(sortOrder) {
		return fmt.Errorf("invalid sort order value provided: %s", sortOrder)
	}

	now := time.Now().Unix()
	queryBuilder := psql.Update("albums").
		Set("sort_order", sortOrder).
		Set("updated_at", now).
		Where(sq.Eq{"id": albumID})

	sqlStr, args, err := queryBuilder.ToSql()
	if err != nil {
		return fmt.Errorf("failed to build SQL for UpdateAlbumSortOrder: %w", err)
	}

	result, err := db.Exec(sqlStr, args...)
	if err != nil {
		return fmt.Errorf("failed to execute UpdateAlbumSortOrder for ID %d: %w", albumID, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err == nil && rowsAffected == 0 {
		return sql.ErrNoRows
	}
	if err != nil {
		log.Printf("Warning: Could not get RowsAffected for UpdateAlbumSortOrder ID %d: %v", albumID, err)
	}
	return nil
}

func DeleteAlbum(db *sql.DB, id int64) error {
	queryBuilder := psql.Delete("albums").Where(sq.Eq{"id": id})
	sqlStr, args, err := queryBuilder.ToSql()
	if err != nil {
		return fmt.Errorf("failed to build SQL for DeleteAlbum: %w", err)
	}
	result, err := db.Exec(sqlStr, args...)
	if err != nil {
		return fmt.Errorf("failed to execute DeleteAlbum for ID %d: %w", id, err)
	}
	rowsAffected, err := result.RowsAffected()
	if err == nil && rowsAffected == 0 {
		return sql.ErrNoRows
	}
	if err != nil {
		log.Printf("Warning: Could not get RowsAffected for DeleteAlbum ID %d: %v", id, err)
	}
	return nil
}
