package database

import (
	"fmt"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"octomanger/internal/platform/config"
)

// Open opens a GORM *gorm.DB backed by PostgreSQL.
// The caller is responsible for closing the underlying sql.DB when done:
//
//	sqlDB, _ := db.DB()
//	sqlDB.Close()
func Open(cfg config.DatabaseConfig) (*gorm.DB, error) {
	return OpenWith(cfg, dialectorForConfig(cfg))
}

// OpenWith opens a GORM *gorm.DB with a caller-provided dialector.
// This is primarily intended for testing with in-memory databases.
func OpenWith(cfg config.DatabaseConfig, dialector gorm.Dialector) (*gorm.DB, error) {
	if dialector == nil {
		return nil, fmt.Errorf("open database: missing dialector")
	}
	gormCfg := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	}

	db, err := gorm.Open(dialector, gormCfg)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("get underlying sql.DB: %w", err)
	}

	sqlDB.SetMaxOpenConns(cfg.MaxConnections)
	sqlDB.SetMaxIdleConns(cfg.MaxConnections / 2)
	connMaxLifetime := 30 * time.Minute
	if tuned := cfg.QueryTimeout * 3; tuned > connMaxLifetime {
		connMaxLifetime = tuned
	}
	sqlDB.SetConnMaxLifetime(connMaxLifetime)

	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("ping database: %w", err)
	}

	return db, nil
}

var dialectorForConfig = func(cfg config.DatabaseConfig) gorm.Dialector {
	return postgres.Open(cfg.DSN)
}
