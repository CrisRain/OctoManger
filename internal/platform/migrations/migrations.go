package migrations

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"

	"gorm.io/gorm"
)

// Apply runs all pending SQL migration files matching globPattern against db.
// It uses the schema_migrations table to track applied versions.
func Apply(ctx context.Context, db *gorm.DB, globPattern string) error {
	if globPattern == "" {
		globPattern = "db/migrations/*.sql"
	}

	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("get sql.DB: %w", err)
	}

	if _, err := sqlDB.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version    TEXT        PRIMARY KEY,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`); err != nil {
		return fmt.Errorf("ensure schema_migrations table: %w", err)
	}

	paths, err := filepath.Glob(globPattern)
	if err != nil {
		return err
	}
	sort.Strings(paths)

	for _, path := range paths {
		if err := applyFile(ctx, sqlDB, path); err != nil {
			return err
		}
	}

	return nil
}

func applyFile(ctx context.Context, db *sql.DB, path string) error {
	version := filepath.Base(path)

	var applied bool
	if err := db.QueryRowContext(ctx,
		`SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version = $1)`, version,
	).Scan(&applied); err != nil {
		return err
	}
	if applied {
		fmt.Fprintf(os.Stdout, "skip    %s\n", version)
		return nil
	}

	content, err := os.ReadFile(path)
	if err != nil {
		if _, ok := err.(*fs.PathError); ok {
			return fmt.Errorf("read migration %s: %w", path, err)
		}
		return err
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	if _, err := tx.ExecContext(ctx, string(content)); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("exec migration %s: %w", version, err)
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO schema_migrations(version) VALUES ($1)`, version); err != nil {
		_ = tx.Rollback()
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}

	fmt.Fprintf(os.Stdout, "applied %s\n", version)
	return nil
}
