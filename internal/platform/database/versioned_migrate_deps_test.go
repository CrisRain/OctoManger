package database

import (
	"context"
	"errors"
	"testing"

	"gorm.io/gorm"

	platformconfig "octomanger/internal/platform/config"
)

func TestRunVersionedMigrationsWithDepsSkipsApplied(t *testing.T) {
	applied := map[int64]struct{}{1: {}}
	applyCount := 0

	deps := migrationDeps{
		ensureSchema: func(ctx context.Context, db *gorm.DB) error { return nil },
		loadApplied:  func(ctx context.Context, db *gorm.DB) (map[int64]struct{}, error) { return applied, nil },
		applyMigration: func(ctx context.Context, db *gorm.DB, m versionedMigration) error {
			applyCount++
			if m.Version != 2 {
				t.Fatalf("unexpected migration version %d", m.Version)
			}
			return nil
		},
		migrateLegacy: func(context.Context, *gorm.DB) error { return nil },
		seedSystem:    func(context.Context, *gorm.DB) error { return nil },
		seedServices:  func(context.Context, *gorm.DB, *platformconfig.Config) error { return nil },
	}

	migrations := []versionedMigration{{Version: 1, Name: "one"}, {Version: 2, Name: "two"}}

	if err := runVersionedMigrationsWithDeps(context.Background(), nil, nil, migrations, deps); err != nil {
		t.Fatalf("run migrations: %v", err)
	}
	if applyCount != 1 {
		t.Fatalf("expected 1 migration applied, got %d", applyCount)
	}
}

func TestRunVersionedMigrationsWithDepsErrors(t *testing.T) {
	errBoom := errors.New("boom")
	migrations := []versionedMigration{{Version: 1, Name: "one"}}

	cases := []struct {
		name string
		deps migrationDeps
	}{
		{
			name: "ensure",
			deps: migrationDeps{ensureSchema: func(context.Context, *gorm.DB) error { return errBoom }},
		},
		{
			name: "loadApplied",
			deps: migrationDeps{
				ensureSchema: func(context.Context, *gorm.DB) error { return nil },
				loadApplied:  func(context.Context, *gorm.DB) (map[int64]struct{}, error) { return nil, errBoom },
			},
		},
		{
			name: "apply",
			deps: migrationDeps{
				ensureSchema:   func(context.Context, *gorm.DB) error { return nil },
				loadApplied:    func(context.Context, *gorm.DB) (map[int64]struct{}, error) { return map[int64]struct{}{}, nil },
				applyMigration: func(context.Context, *gorm.DB, versionedMigration) error { return errBoom },
			},
		},
		{
			name: "migrateLegacy",
			deps: migrationDeps{
				ensureSchema:   func(context.Context, *gorm.DB) error { return nil },
				loadApplied:    func(context.Context, *gorm.DB) (map[int64]struct{}, error) { return map[int64]struct{}{}, nil },
				applyMigration: func(context.Context, *gorm.DB, versionedMigration) error { return nil },
				migrateLegacy:  func(context.Context, *gorm.DB) error { return errBoom },
			},
		},
		{
			name: "seedSystem",
			deps: migrationDeps{
				ensureSchema:   func(context.Context, *gorm.DB) error { return nil },
				loadApplied:    func(context.Context, *gorm.DB) (map[int64]struct{}, error) { return map[int64]struct{}{}, nil },
				applyMigration: func(context.Context, *gorm.DB, versionedMigration) error { return nil },
				migrateLegacy:  func(context.Context, *gorm.DB) error { return nil },
				seedSystem:     func(context.Context, *gorm.DB) error { return errBoom },
			},
		},
		{
			name: "seedServices",
			deps: migrationDeps{
				ensureSchema:   func(context.Context, *gorm.DB) error { return nil },
				loadApplied:    func(context.Context, *gorm.DB) (map[int64]struct{}, error) { return map[int64]struct{}{}, nil },
				applyMigration: func(context.Context, *gorm.DB, versionedMigration) error { return nil },
				migrateLegacy:  func(context.Context, *gorm.DB) error { return nil },
				seedSystem:     func(context.Context, *gorm.DB) error { return nil },
				seedServices:   func(context.Context, *gorm.DB, *platformconfig.Config) error { return errBoom },
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := runVersionedMigrationsWithDeps(context.Background(), nil, nil, migrations, tc.deps); err == nil {
				t.Fatalf("expected error")
			}
		})
	}
}
