package config

import (
	"testing"
	"time"
)

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
	if cfg.Database.MigrationMode != "versioned" {
		t.Fatalf("unexpected migration mode %q", cfg.Database.MigrationMode)
	}
}

func TestLoadProvidesDefaultPluginServices(t *testing.T) {
	t.Setenv("DATABASE_DSN", "postgres://octo:octo@localhost:5432/octomanger?sslmode=disable")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	service, ok := cfg.Plugins.Services["octo_demo"]
	if !ok {
		t.Fatalf("expected built-in octo_demo service default")
	}
	if service.Address != "127.0.0.1:50051" {
		t.Fatalf("unexpected default address %q", service.Address)
	}
}

func TestLoadPluginServicesFromEnv(t *testing.T) {
	t.Setenv("DATABASE_DSN", "postgres://octo:octo@localhost:5432/octomanger?sslmode=disable")
	t.Setenv("PLUGIN_GRPC_OCTO_DEMO_ADDR", "127.0.0.1:50051")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	service, ok := cfg.Plugins.Services["octo_demo"]
	if !ok {
		t.Fatalf("expected octo_demo service to be loaded from env")
	}
	if service.Address != "127.0.0.1:50051" {
		t.Fatalf("unexpected service address %q", service.Address)
	}
}

func TestLoadUsesAdminKey(t *testing.T) {
	t.Setenv("DATABASE_DSN", "postgres://octo:octo@localhost:5432/octomanger?sslmode=disable")
	t.Setenv("ADMIN_KEY", "primary-admin-key")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if cfg.Auth.AdminKey != "primary-admin-key" {
		t.Fatalf("unexpected admin key %q", cfg.Auth.AdminKey)
	}
	if cfg.Auth.PluginInternalAPIToken != "primary-admin-key" {
		t.Fatalf("unexpected internal api token %q", cfg.Auth.PluginInternalAPIToken)
	}
}

func TestLoadUsesAdminKeysOverSingleAdminKey(t *testing.T) {
	t.Setenv("DATABASE_DSN", "postgres://octo:octo@localhost:5432/octomanger?sslmode=disable")
	t.Setenv("ADMIN_KEY", "legacy-admin-key")
	t.Setenv("ADMIN_KEYS", "primary,secondary")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if cfg.Auth.AdminKey != "primary,secondary" {
		t.Fatalf("unexpected admin keys %q", cfg.Auth.AdminKey)
	}
	if cfg.Auth.PluginInternalAPIToken != "primary" {
		t.Fatalf("unexpected internal api token %q", cfg.Auth.PluginInternalAPIToken)
	}
}

func TestLoadUsesExplicitPluginInternalToken(t *testing.T) {
	t.Setenv("DATABASE_DSN", "postgres://octo:octo@localhost:5432/octomanger?sslmode=disable")
	t.Setenv("ADMIN_KEYS", "primary,secondary")
	t.Setenv("PLUGIN_INTERNAL_API_TOKEN", "plugin-internal-token")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if cfg.Auth.PluginInternalAPIToken != "plugin-internal-token" {
		t.Fatalf("unexpected internal api token %q", cfg.Auth.PluginInternalAPIToken)
	}
}

func TestLoadProvidesDefaultServerTimeouts(t *testing.T) {
	t.Setenv("DATABASE_DSN", "postgres://octo:octo@localhost:5432/octomanger?sslmode=disable")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if cfg.Server.ReadTimeout != 15*time.Second {
		t.Fatalf("unexpected read timeout %s", cfg.Server.ReadTimeout)
	}
	if cfg.Server.IdleTimeout != time.Minute {
		t.Fatalf("unexpected idle timeout %s", cfg.Server.IdleTimeout)
	}
}

func TestLoadReadsCORSAndPluginGRPCSecurityFromEnv(t *testing.T) {
	t.Setenv("DATABASE_DSN", "postgres://octo:octo@localhost:5432/octomanger?sslmode=disable")
	t.Setenv("CORS_ALLOWED_ORIGINS", "https://admin.example.com, http://localhost:5173")
	t.Setenv("PLUGIN_GRPC_ALLOW_INSECURE_REMOTE", "true")
	t.Setenv("PLUGIN_GRPC_INSECURE_SKIP_VERIFY", "1")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if len(cfg.Server.CORS.AllowedOrigins) != 2 {
		t.Fatalf("unexpected cors origin count %#v", cfg.Server.CORS.AllowedOrigins)
	}
	if cfg.Server.CORS.AllowedOrigins[0] != "https://admin.example.com" {
		t.Fatalf("unexpected first origin %q", cfg.Server.CORS.AllowedOrigins[0])
	}
	if !cfg.Plugins.GRPC.AllowInsecureRemote {
		t.Fatalf("expected allow_insecure_remote to be true")
	}
	if !cfg.Plugins.GRPC.InsecureSkipVerify {
		t.Fatalf("expected insecure_skip_verify to be true")
	}
}
