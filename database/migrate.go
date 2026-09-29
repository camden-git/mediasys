package database

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"

	"github.com/pressly/goose/v3"
	"github.com/pressly/goose/v3/lock"
)

// migrationsFS embeds the versioned SQL migrations that define the database
// schema. AutoMigrate previously drove the schema at startup (see git history for
// database/gorm_db.go); that has been replaced by these migrations, which are the
// only source of truth for the schema going forward. GORM model tags are kept as
// documentation of the Go <-> SQL mapping, but no longer drive migrations.
//
//go:embed migrations/*.sql
var migrationsFS embed.FS

// RunMigrations applies all pending goose migrations embedded in the binary to the
// database reachable via sqlDB, and is safe to call from multiple instances
// starting up concurrently: it takes a Postgres advisory session lock for the
// duration of the migration run, so only one instance actually runs migrations at
// a time while the others wait.
func RunMigrations(ctx context.Context, sqlDB *sql.DB) error {
	locker, err := lock.NewPostgresSessionLocker()
	if err != nil {
		return fmt.Errorf("failed to create migration advisory locker: %w", err)
	}

	migrations, err := fs.Sub(migrationsFS, "migrations")
	if err != nil {
		return fmt.Errorf("failed to open embedded migrations: %w", err)
	}

	provider, err := goose.NewProvider(goose.DialectPostgres, sqlDB, migrations, goose.WithSessionLocker(locker))
	if err != nil {
		return fmt.Errorf("failed to create migration provider: %w", err)
	}

	results, err := provider.Up(ctx)
	if err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	for _, result := range results {
		if result.Error != nil {
			return fmt.Errorf("migration %s failed: %w", result.Source.Path, result.Error)
		}
	}

	return nil
}
