package database

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"log"

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
	if err := baselineLegacySchema(ctx, sqlDB); err != nil {
		return err
	}

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

// baselineAdvisoryLockID guards baselineLegacySchema against concurrent instances.
// It is distinct from goose's own advisory lock id.
const baselineAdvisoryLockID = 5087605

// baselineLegacySchema handles databases created by GORM AutoMigrate before goose
// was introduced. Such a database already contains the application tables but has
// no goose version table, so running migration 00001 (CREATE TABLE ...) against it
// would fail with "already exists". When the version table is absent and the app
// schema is present, it creates the version table and records version 1 as applied
// so goose only runs the later migrations, which reconcile any differences between
// the old AutoMigrate schema and 00001 idempotently. It is a no-op for empty
// databases and for databases goose already manages.
func baselineLegacySchema(ctx context.Context, sqlDB *sql.DB) error {
	tx, err := sqlDB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin baseline transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, "SELECT pg_advisory_xact_lock($1)", baselineAdvisoryLockID); err != nil {
		return fmt.Errorf("failed to take baseline advisory lock: %w", err)
	}

	var hasVersionTable, hasAppSchema bool
	if err := tx.QueryRowContext(ctx,
		"SELECT to_regclass('goose_db_version') IS NOT NULL, to_regclass('albums') IS NOT NULL",
	).Scan(&hasVersionTable, &hasAppSchema); err != nil {
		return fmt.Errorf("failed to inspect existing schema: %w", err)
	}
	if hasVersionTable || !hasAppSchema {
		return nil
	}

	log.Println("database: found pre-goose (AutoMigrate) schema, baselining at migration 1")
	// Same layout goose creates for its version table on Postgres.
	stmts := []string{
		`CREATE TABLE goose_db_version (
			id serial NOT NULL PRIMARY KEY,
			version_id bigint NOT NULL,
			is_applied boolean NOT NULL,
			tstamp timestamp NOT NULL DEFAULT now()
		)`,
		"INSERT INTO goose_db_version (version_id, is_applied) VALUES (0, true), (1, true)",
	}
	for _, stmt := range stmts {
		if _, err := tx.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("failed to baseline legacy schema: %w", err)
		}
	}
	return tx.Commit()
}
