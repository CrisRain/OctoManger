package database

import (
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	platformconfig "octomanger/internal/platform/config"
)

func TestOpenWithNilDialector(t *testing.T) {
	_, err := OpenWith(platformconfig.DatabaseConfig{}, nil)
	if err == nil {
		t.Fatalf("expected error for nil dialector")
	}
}

func TestOpenWithSQLite(t *testing.T) {
	cfg := platformconfig.DatabaseConfig{
		MaxConnections: 4,
		QueryTimeout:   2 * time.Second,
	}
	db, err := OpenWith(cfg, sqlite.Open("file::memory:?cache=shared"))
	if err != nil {
		t.Fatalf("open with sqlite: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get sql db: %v", err)
	}
	if sqlDB.Stats().MaxOpenConnections != 4 {
		t.Fatalf("unexpected max open connections %d", sqlDB.Stats().MaxOpenConnections)
	}
}

func TestOpenUsesDialectorOverride(t *testing.T) {
	original := dialectorForConfig
	dialectorForConfig = func(cfg platformconfig.DatabaseConfig) gorm.Dialector {
		return sqlite.Open("file::memory:?cache=shared")
	}
	t.Cleanup(func() {
		dialectorForConfig = original
	})

	cfg := platformconfig.DatabaseConfig{MaxConnections: 2}
	db, err := Open(cfg)
	if err != nil {
		t.Fatalf("open override: %v", err)
	}
	if db == nil {
		t.Fatalf("expected db")
	}
}
