package migrator

import (
	"context"
	"errors"
	"testing"

	"gorm.io/gorm"

	"octomanger/internal/platform/config"
	"octomanger/internal/testutil"
)

func TestRunAndRollbackUseDependencies(t *testing.T) {
	origLoad := loadConfig
	origOpen := openDB
	origMigrate := runMigrate
	origRollback := runRollback
	t.Cleanup(func() {
		loadConfig = origLoad
		openDB = origOpen
		runMigrate = origMigrate
		runRollback = origRollback
	})

	loadConfig = func() (config.Config, error) {
		return config.Config{}, nil
	}
	openDB = func(cfg config.DatabaseConfig) (*gorm.DB, error) {
		return testutil.NewTestDB(t), nil
	}

	called := struct{ migrate, rollback bool }{}
	runMigrate = func(ctx context.Context, db *gorm.DB, cfgs ...*config.Config) error {
		called.migrate = true
		return nil
	}
	runRollback = func(ctx context.Context, db *gorm.DB, cfgs ...*config.Config) error {
		called.rollback = true
		return nil
	}

	if err := Run(context.Background()); err != nil {
		t.Fatalf("run migrate: %v", err)
	}
	if err := RollbackLast(context.Background()); err != nil {
		t.Fatalf("run rollback: %v", err)
	}
	if !called.migrate || !called.rollback {
		t.Fatalf("expected migrate and rollback to be called")
	}
}

func TestRunReturnsErrors(t *testing.T) {
	origLoad := loadConfig
	origOpen := openDB
	origMigrate := runMigrate
	t.Cleanup(func() {
		loadConfig = origLoad
		openDB = origOpen
		runMigrate = origMigrate
	})

	loadConfig = func() (config.Config, error) { return config.Config{}, errors.New("load") }
	if err := Run(context.Background()); err == nil {
		t.Fatalf("expected load error")
	}

	loadConfig = func() (config.Config, error) { return config.Config{}, nil }
	openDB = func(cfg config.DatabaseConfig) (*gorm.DB, error) { return nil, errors.New("open") }
	if err := Run(context.Background()); err == nil {
		t.Fatalf("expected open error")
	}
}
