package database

import (
	"testing"

	platformconfig "octomanger/internal/platform/config"
)

func TestMigrationModeDefaultsToVersioned(t *testing.T) {
	if got := migrationMode(nil); got != MigrationModeVersioned {
		t.Fatalf("unexpected migration mode %q", got)
	}
}

func TestMigrationModeSupportsAutoMigrateAlias(t *testing.T) {
	cfg := &platformconfig.Config{
		Database: platformconfig.DatabaseConfig{
			MigrationMode: "automigrate",
		},
	}
	if got := migrationMode(cfg); got != MigrationModeAuto {
		t.Fatalf("unexpected migration mode %q", got)
	}
}

func TestVersionedMigrationsSorted(t *testing.T) {
	items := versionedMigrations()
	for i := 1; i < len(items); i++ {
		if items[i-1].Version > items[i].Version {
			t.Fatalf("migrations are not sorted by version")
		}
	}
}

func TestVersionedMigrationsIncludePluginInternalTokenMigration(t *testing.T) {
	items := versionedMigrations()
	for _, item := range items {
		if item.Version == 3 && item.Name == "system_settings_plugin_internal_api_token" {
			return
		}
	}
	t.Fatalf("expected migration version 3 for plugin_internal_api_token")
}

func TestParseAppliedVersionsSupportsLegacyFilenameMarkers(t *testing.T) {
	records := []schemaMigrationRawRecord{
		{Version: "0001_v2_core.sql"},
		{Version: "0002_v2_config.sql"},
		{Version: "0003_rename_system_config_value.sql"},
	}

	applied := parseAppliedVersions(records)
	if _, ok := applied[1]; !ok {
		t.Fatalf("expected legacy markers to imply migration version 1 applied")
	}
	if _, ok := applied[2]; ok {
		t.Fatalf("legacy filename markers must not imply numeric migration version 2")
	}
}

func TestParseAppliedVersionsParsesNumericValues(t *testing.T) {
	records := []schemaMigrationRawRecord{
		{Version: int64(1)},
		{Version: "2"},
		{Version: []byte("3")},
	}

	applied := parseAppliedVersions(records)
	if _, ok := applied[1]; !ok {
		t.Fatalf("expected version 1 to be applied")
	}
	if _, ok := applied[2]; !ok {
		t.Fatalf("expected version 2 to be applied")
	}
	if _, ok := applied[3]; !ok {
		t.Fatalf("expected version 3 to be applied")
	}
}
