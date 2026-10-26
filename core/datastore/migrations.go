// core/datastore/migrations.go

package datastore

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"sort"
	"strings"

	"smegg.me/qr2g.it/common/logger"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

const schemaMigrationsTable = `
CREATE TABLE IF NOT EXISTS schema_migrations (
    name text PRIMARY KEY,
    applied_at datetime NOT NULL DEFAULT (strftime('%Y-%m-%d %H:%M:%f', 'now'))
)`

func runMigrations(ctx context.Context, db *sql.DB) error {
	if _, err := db.ExecContext(ctx, schemaMigrationsTable); err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}

	names, err := listMigrations()
	if err != nil {
		return err
	}

	for _, name := range names {
		applied, err := isMigrationApplied(ctx, db, name)
		if err != nil {
			return fmt.Errorf("check %s: %w", name, err)
		}
		if applied {
			logger.Debug().Str("migration", name).Msg("migration already applied")
			continue
		}
		if err := applyMigration(ctx, db, name); err != nil {
			return err
		}
		logger.Info().Str("migration", name).Msg("migration applied")
	}
	return nil
}

func listMigrations() ([]string, error) {
	entries, err := fs.ReadDir(migrationsFS, "migrations")
	if err != nil {
		return nil, fmt.Errorf("read migrations dir: %w", err)
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if !strings.HasSuffix(e.Name(), ".sql") {
			continue
		}
		names = append(names, e.Name())
	}
	sort.Strings(names)
	if len(names) == 0 {
		return nil, fmt.Errorf("migrations: no embedded SQL files")
	}
	return names, nil
}

func isMigrationApplied(ctx context.Context, db *sql.DB, name string) (bool, error) {
	var exists bool
	err := db.QueryRowContext(ctx,
		`SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE name = ?1)`,
		name,
	).Scan(&exists)
	return exists, err
}

func applyMigration(ctx context.Context, db *sql.DB, name string) error {
	body, err := migrationsFS.ReadFile("migrations/" + name)
	if err != nil {
		return fmt.Errorf("read %s: %w", name, err)
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin %s: %w", name, err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	if _, err := tx.ExecContext(ctx, string(body)); err != nil {
		return fmt.Errorf("apply %s: %w", name, err)
	}
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO schema_migrations (name) VALUES (?1)`,
		name,
	); err != nil {
		return fmt.Errorf("record %s: %w", name, err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit %s: %w", name, err)
	}
	return nil
}
