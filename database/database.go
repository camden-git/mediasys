package database

import (
	"database/sql"
	"fmt"
	"log"

	sq "github.com/Masterminds/squirrel"
	_ "github.com/mattn/go-sqlite3"
)

var psql = sq.StatementBuilder.PlaceholderFormat(sq.Question)

type ThumbnailInfo struct {
	ThumbnailPath string
	LastModified  int64
}

func InitDB(dataSourceName string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", dataSourceName)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// enable write-ahead Logging for better concurrency
	_, err = db.Exec("PRAGMA journal_mode=WAL;")
	if err != nil {
		log.Printf("warning: failed to set WAL mode: %v", err)
	}

	sqlStmt := `
	CREATE TABLE IF NOT EXISTS thumbnails (
		original_path TEXT PRIMARY KEY,
		thumbnail_path TEXT NOT NULL,
		last_modified INTEGER NOT NULL
	);
	`
	_, err = db.Exec(sqlStmt)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create thumbnails table: %w", err)
	}

	log.Println("database initialized successfully at", dataSourceName)
	return db, nil
}

// GetThumbnailInfo retrieves thumbnail path and last modified time
func GetThumbnailInfo(db *sql.DB, originalPath string) (ThumbnailInfo, error) {
	var info ThumbnailInfo

	queryBuilder := psql.Select("thumbnail_path", "last_modified").
		From("thumbnails").
		Where(sq.Eq{"original_path": originalPath}).
		Limit(1)

	sqlStr, args, err := queryBuilder.ToSql()
	if err != nil {
		return ThumbnailInfo{}, fmt.Errorf("failed to build SQL query for GetThumbnailInfo: %w", err)
	}

	err = db.QueryRow(sqlStr, args...).Scan(&info.ThumbnailPath, &info.LastModified)
	if err != nil {
		if err == sql.ErrNoRows {
			return ThumbnailInfo{}, sql.ErrNoRows // Explicitly return ErrNoRows
		}
		return ThumbnailInfo{}, fmt.Errorf("failed to query or scan thumbnail info for %s: %w", originalPath, err)
	}
	return info, nil
}

// SetThumbnailInfo inserts or updates thumbnail information
func SetThumbnailInfo(db *sql.DB, originalPath, thumbnailPath string, lastModified int64) error {

	queryBuilder := psql.Insert("thumbnails").
		Columns("original_path", "thumbnail_path", "last_modified").
		Values(originalPath, thumbnailPath, lastModified).
		Suffix("ON CONFLICT(original_path) DO UPDATE SET").
		Suffix("thumbnail_path = excluded.thumbnail_path,").
		Suffix("last_modified = excluded.last_modified")

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
