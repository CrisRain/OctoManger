package testutil

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"octomanger/internal/platform/database"
)

// NewTestDB returns an in-memory sqlite DB with the OctoManger schema migrated.
func NewTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared&_foreign_keys=1", sanitizeName(t.Name()))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get sql db: %v", err)
	}
	t.Cleanup(func() {
		_ = sqlDB.Close()
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := database.AutoMigrate(ctx, db); err != nil {
		t.Fatalf("auto migrate schema: %v", err)
	}

	return db
}

func sanitizeName(name string) string {
	replacer := strings.NewReplacer("/", "_", "\\", "_", ":", "_", " ", "_")
	return replacer.Replace(name)
}
