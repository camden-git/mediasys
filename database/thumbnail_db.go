package database

import (
	"database/sql"
	"fmt"
	"path/filepath"

	sq "github.com/Masterminds/squirrel"
)

type ThumbnailInfo struct {
	ThumbnailPath  string
	LastModified   int64
	OriginalWidth  *int
	OriginalHeight *int
}

func GetThumbnailInfo(db *sql.DB, originalPath string) (ThumbnailInfo, error) {
	var info ThumbnailInfo
	queryBuilder := psql.Select(
		"thumbnail_path",
		"last_modified",
		"original_width",
		"original_height",
	).From("thumbnails").
		Where(sq.Eq{"original_path": originalPath}).
		Limit(1)

	sqlStr, args, err := queryBuilder.ToSql()
	if err != nil {
		return ThumbnailInfo{}, fmt.Errorf("failed to build SQL query for GetThumbnailInfo: %w", err)
	}

	err = db.QueryRow(sqlStr, args...).Scan(
		&info.ThumbnailPath,
		&info.LastModified,
		&info.OriginalWidth,
		&info.OriginalHeight,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return ThumbnailInfo{}, sql.ErrNoRows
		}
		return ThumbnailInfo{}, fmt.Errorf("failed to query or scan thumbnail info for %s: %w", originalPath, err)
	}
	return info, nil
}

// SetThumbnailInfo inserts or updates thumbnail information including dimensions
func SetThumbnailInfo(db *sql.DB, originalPath, thumbnailPath string, lastModified int64, originalWidth, originalHeight *int) error {
	originalPath = filepath.ToSlash(originalPath)
	queryBuilder := psql.Insert("thumbnails").
		Columns(
			"original_path",
			"thumbnail_path",
			"last_modified",
			"original_width",
			"original_height",
		).
		Values(
			originalPath,
			thumbnailPath,
			lastModified,
			originalWidth,
			originalHeight,
		).
		Suffix("ON CONFLICT(original_path) DO UPDATE SET").
		Suffix("thumbnail_path = excluded.thumbnail_path,").
		Suffix("last_modified = excluded.last_modified,").
		Suffix("original_width = excluded.original_width,").
		Suffix("original_height = excluded.original_height")

	sqlStr, args, err := queryBuilder.ToSql()
	if err != nil {
		return fmt.Errorf("failed to build SQL query for SetThumbnailInfo: %w", err)
	}
	_, err = db.Exec(sqlStr, args...)
	if err != nil {
		return fmt.Errorf("failed to execute set thumbnail info for %s: %w", originalPath, err)
	}
	return nil
}
