package runtime

import (
	"context"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"octomanger/internal/platform/config"
	"octomanger/internal/platform/database"
	"octomanger/internal/testutil"
)

func TestEnsurePluginInternalAPITokenSkipsWhenNoTable(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get sql db: %v", err)
	}
	defer func() { _ = sqlDB.Close() }()

	cfg := &config.Config{}
	if err := ensurePluginInternalAPIToken(context.Background(), db, cfg); err != nil {
		t.Fatalf("ensure token: %v", err)
	}
	if cfg.Auth.PluginInternalAPIToken != "" {
		t.Fatalf("expected token to remain empty")
	}
}

func TestEnsurePluginInternalAPITokenUsesConfigured(t *testing.T) {
	db := testutil.NewTestDB(t)
	cfg := &config.Config{}
	cfg.Auth.PluginInternalAPIToken = "configured"

	if err := ensurePluginInternalAPIToken(context.Background(), db, cfg); err != nil {
		t.Fatalf("ensure token: %v", err)
	}
	if cfg.Auth.PluginInternalAPIToken != "configured" {
		t.Fatalf("expected configured token")
	}

	stored, err := loadPluginInternalAPIToken(context.Background(), db)
	if err != nil {
		t.Fatalf("load token: %v", err)
	}
	if stored != "configured" {
		t.Fatalf("expected stored token %q", stored)
	}
}

func TestEnsurePluginInternalAPITokenLoadsExisting(t *testing.T) {
	db := testutil.NewTestDB(t)
	ctx := context.Background()

	if err := upsertPluginInternalAPIToken(ctx, db, "stored", true); err != nil {
		t.Fatalf("seed token: %v", err)
	}

	cfg := &config.Config{}
	if err := ensurePluginInternalAPIToken(ctx, db, cfg); err != nil {
		t.Fatalf("ensure token: %v", err)
	}
	if cfg.Auth.PluginInternalAPIToken != "stored" {
		t.Fatalf("expected stored token, got %q", cfg.Auth.PluginInternalAPIToken)
	}
}

func TestEnsurePluginInternalAPITokenGenerates(t *testing.T) {
	db := testutil.NewTestDB(t)
	ctx := context.Background()

	cfg := &config.Config{}
	if err := ensurePluginInternalAPIToken(ctx, db, cfg); err != nil {
		t.Fatalf("ensure token: %v", err)
	}
	if cfg.Auth.PluginInternalAPIToken == "" {
		t.Fatalf("expected generated token")
	}
}

func TestLoadAndUpsertPluginInternalAPITokenEdges(t *testing.T) {
	db := testutil.NewTestDB(t)
	ctx := context.Background()

	if err := upsertPluginInternalAPIToken(ctx, db, "", false); err != nil {
		t.Fatalf("expected empty token to be ignored: %v", err)
	}

	if token, err := loadPluginInternalAPIToken(ctx, db); err != nil {
		t.Fatalf("load token: %v", err)
	} else if token != "" {
		t.Fatalf("expected empty token to remain unchanged, got %q", token)
	}
}

func TestNowExpr(t *testing.T) {
	if expr := nowExpr(nil); expr.SQL != "NOW()" {
		t.Fatalf("expected default now expr, got %q", expr.SQL)
	}
	db := testutil.NewTestDB(t)
	if expr := nowExpr(db); expr.SQL != "CURRENT_TIMESTAMP" {
		t.Fatalf("expected sqlite now expr, got %q", expr.SQL)
	}
}

func TestLoadPluginInternalAPITokenMissingRow(t *testing.T) {
	db := testutil.NewTestDB(t)
	ctx := context.Background()

	// Delete the seeded row to hit not found branch.
	if err := db.WithContext(ctx).Exec("DELETE FROM system_settings").Error; err != nil {
		t.Fatalf("delete system settings: %v", err)
	}

	value, err := loadPluginInternalAPIToken(ctx, db)
	if err != nil {
		t.Fatalf("load token: %v", err)
	}
	if value != "" {
		t.Fatalf("expected empty token for missing row")
	}
}

func TestEnsurePluginInternalAPITokenWhenColumnMissing(t *testing.T) {
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
	if err := db.WithContext(ctx).AutoMigrate(&database.SystemSettingsModel{}); err != nil {
		t.Fatalf("migrate system settings: %v", err)
	}
	if err := db.Migrator().DropColumn(&database.SystemSettingsModel{}, "PluginInternalAPIToken"); err != nil {
		t.Fatalf("drop column: %v", err)
	}

	cfg := &config.Config{}
	if err := ensurePluginInternalAPIToken(ctx, db, cfg); err != nil {
		t.Fatalf("ensure token: %v", err)
	}
}
