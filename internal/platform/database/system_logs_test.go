package database

import (
	"context"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestEnsureSystemLogSchemaNilDB(t *testing.T) {
	if err := EnsureSystemLogSchema(context.Background(), nil); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
}

func TestEnsureSystemLogSchemaCreatesTable(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get sql db: %v", err)
	}
	defer func() { _ = sqlDB.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := EnsureSystemLogSchema(ctx, db); err != nil {
		t.Fatalf("ensure system log schema: %v", err)
	}
	if !db.Migrator().HasTable(&SystemLogModel{}) {
		t.Fatalf("expected system_logs table to exist")
	}
}
