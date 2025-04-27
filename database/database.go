package database

import (
	"database/sql"
	"fmt"
	"log"
	"strings"

	sq "github.com/Masterminds/squirrel"
	_ "github.com/mattn/go-sqlite3"
)

var psql = sq.StatementBuilder.PlaceholderFormat(sq.Question)

func InitDB(dataSourceName string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", dataSourceName)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	_, err = db.Exec("PRAGMA journal_mode=WAL;")
	if err != nil {
		log.Printf("Warning: Failed to set WAL mode: %v", err) // Non-fatal
	}

	_, err = db.Exec("PRAGMA foreign_keys = ON;")
	if err != nil {
		db.Close() // close on critical error
		return nil, fmt.Errorf("failed to enable foreign keys: %w", err)
	}

	tableCreationStmts := []string{
		`CREATE TABLE IF NOT EXISTS thumbnails (
			original_path TEXT PRIMARY KEY,
			thumbnail_path TEXT NOT NULL,
			last_modified INTEGER NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS albums (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL UNIQUE,
			slug TEXT NOT NULL UNIQUE,
			description TEXT,
			folder_path TEXT NOT NULL UNIQUE,
			created_at INTEGER NOT NULL,
			updated_at INTEGER NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS people (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			primary_name TEXT NOT NULL,
			created_at INTEGER NOT NULL,
			updated_at INTEGER NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS aliases (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			person_id INTEGER NOT NULL,
			name TEXT NOT NULL,
			FOREIGN KEY(person_id) REFERENCES people(id) ON DELETE CASCADE,
			UNIQUE(person_id, name)
		);`,
		`CREATE TABLE IF NOT EXISTS faces (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			person_id INTEGER NULL, -- TODO: will we allow untagged faces? for now yes
			image_path TEXT NOT NULL,
			x1 INTEGER NOT NULL,
			y1 INTEGER NOT NULL,
			x2 INTEGER NOT NULL,
			y2 INTEGER NOT NULL,
			created_at INTEGER NOT NULL,
			updated_at INTEGER NOT NULL,
			FOREIGN KEY(person_id) REFERENCES people(id) ON DELETE SET NULL -- untag face if person deleted
		);`,
		`CREATE INDEX IF NOT EXISTS idx_aliases_person_id ON aliases(person_id);`,
		`CREATE INDEX IF NOT EXISTS idx_aliases_name ON aliases(name);`,
		`CREATE INDEX IF NOT EXISTS idx_faces_person_id ON faces(person_id);`,
		`CREATE INDEX IF NOT EXISTS idx_faces_image_path ON faces(image_path);`,
	}

	for i, stmt := range tableCreationStmts {
		if strings.Contains(stmt, "(...)") {
			continue
		}
		if _, err = db.Exec(stmt); err != nil {
			db.Close()
			return nil, fmt.Errorf("failed to execute table creation statement %d: %w\nStatement: %s", i, err, stmt)
		}
	}

	log.Println("Database initialized successfully at", dataSourceName)
	return db, nil
}
