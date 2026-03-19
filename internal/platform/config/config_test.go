package config

import "testing"

func TestLoadUsesDatabaseDSN(t *testing.T) {
	t.Setenv("DATABASE_DSN", "postgres://octo:octo@localhost:5432/octomanger?sslmode=disable")
	t.Setenv("DATABASE_MAX_CONNECTIONS", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if cfg.Database.DSN != "postgres://octo:octo@localhost:5432/octomanger?sslmode=disable" {
		t.Fatalf("unexpected database dsn %q", cfg.Database.DSN)
	}
}
