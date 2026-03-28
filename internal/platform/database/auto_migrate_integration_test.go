package database

import (
	"context"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	platformconfig "octomanger/internal/platform/config"
)

func TestAutoMigrateMigratesLegacyConfigs(t *testing.T) {
	db := openSQLite(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.Exec(`CREATE TABLE system_configs (key TEXT, value_json JSON)`).Error; err != nil {
		t.Fatalf("create legacy table: %v", err)
	}
	legacyRows := []struct {
		key   string
		value string
	}{
		{"app.name", `"Legacy App"`},
		{"job.default_timeout_minutes", `45`},
		{"job.max_concurrency", `12`},
		{"plugins.grpc_services", `{"octo_demo": {"address": "127.0.0.1:60051"}}`},
		{"plugin_settings:octo-demo", `{"foo":"bar"}`},
		{"plugin_settings:invalid", `not-json`},
	}
	for _, row := range legacyRows {
		if err := db.Exec(`INSERT INTO system_configs (key, value_json) VALUES (?, ?)`, row.key, row.value).Error; err != nil {
			t.Fatalf("insert legacy row %s: %v", row.key, err)
		}
	}

	cfg := &platformconfig.Config{}
	if err := AutoMigrate(ctx, db, cfg); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}

	var settings SystemSettingsModel
	if err := db.First(&settings, "id = ?", systemSettingsSingletonID).Error; err != nil {
		t.Fatalf("load system settings: %v", err)
	}
	if settings.AppName != "Legacy App" {
		t.Fatalf("expected legacy app name, got %q", settings.AppName)
	}
	if settings.JobDefaultTimeoutMinutes != 45 || settings.JobMaxConcurrency != 12 {
		t.Fatalf("unexpected legacy settings: %#v", settings)
	}

	var grpcConfig PluginServiceConfigModel
	if err := db.First(&grpcConfig, "plugin_key = ?", "octo_demo").Error; err != nil {
		t.Fatalf("load plugin service config: %v", err)
	}
	if grpcConfig.GRPCAddress != "127.0.0.1:60051" {
		t.Fatalf("unexpected grpc address %q", grpcConfig.GRPCAddress)
	}

	var pluginSettings PluginSettingsModel
	if err := db.First(&pluginSettings, "plugin_key = ?", "octo_demo").Error; err != nil {
		t.Fatalf("load plugin settings: %v", err)
	}
	if string(pluginSettings.SettingsJSON) != `{"foo":"bar"}` {
		t.Fatalf("unexpected plugin settings json %s", string(pluginSettings.SettingsJSON))
	}

	var invalidSettings PluginSettingsModel
	if err := db.First(&invalidSettings, "plugin_key = ?", "invalid").Error; err != nil {
		t.Fatalf("load invalid plugin settings: %v", err)
	}
	if string(invalidSettings.SettingsJSON) != `{}` {
		t.Fatalf("expected default empty settings for invalid json, got %s", string(invalidSettings.SettingsJSON))
	}
}

func TestAutoMigrateWithoutLegacyTables(t *testing.T) {
	db := openSQLite(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := AutoMigrate(ctx, db); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
}

func openSQLite(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get sql db: %v", err)
	}
	t.Cleanup(func() {
		_ = sqlDB.Close()
	})
	return db
}
