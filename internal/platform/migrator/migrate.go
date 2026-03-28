package migrator

import (
	"context"

	"octomanger/internal/platform/config"
	"octomanger/internal/platform/database"
)

var (
	loadConfig  = config.Load
	openDB      = database.Open
	runMigrate  = database.Migrate
	runRollback = database.RollbackLastMigration
)

func Run(ctx context.Context) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	db, err := openDB(cfg.Database)
	if err != nil {
		return err
	}
	defer func() {
		if sqlDB, err := db.DB(); err == nil {
			_ = sqlDB.Close()
		}
	}()

	return runMigrate(ctx, db, &cfg)
}

func RollbackLast(ctx context.Context) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	db, err := openDB(cfg.Database)
	if err != nil {
		return err
	}
	defer func() {
		if sqlDB, err := db.DB(); err == nil {
			_ = sqlDB.Close()
		}
	}()

	return runRollback(ctx, db, &cfg)
}
