package runtime

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"

	accounttypeapp "octomanger/internal/domains/account-types/app"
	accounttypepostgres "octomanger/internal/domains/account-types/infra/postgres"
	plugins "octomanger/internal/domains/plugins"
	pluginapp "octomanger/internal/domains/plugins/app"
	"octomanger/internal/platform/config"
	"octomanger/internal/testutil"
)

func TestBootstrapPlatformWithDeps(t *testing.T) {
	cfg := config.Config{}
	called := struct{ ensure, attach bool }{}

	deps := platformDeps{
		loadConfig:  func() (config.Config, error) { return cfg, nil },
		newLogger:   func(config.LoggingConfig) *zap.Logger { return zap.NewNop() },
		openDB:      func(config.DatabaseConfig) (*gorm.DB, error) { return testutil.NewTestDB(t), nil },
		ensureToken: func(context.Context, *gorm.DB, *config.Config) error { called.ensure = true; return nil },
		openRedis:   func(config.RedisConfig) (*redis.Client, error) { return nil, errors.New("redis down") },
		attachLogSink: func(logger *zap.Logger, rdb *redis.Client, cfg config.LoggingConfig, source string) (*zap.Logger, func()) {
			called.attach = true
			return logger, func() {}
		},
	}

	res, err := bootstrapPlatformWith(context.Background(), deps)
	if err != nil {
		t.Fatalf("bootstrap platform: %v", err)
	}
	if res.db == nil || res.logger == nil {
		t.Fatalf("expected db and logger")
	}
	if res.rdb != nil {
		t.Fatalf("expected redis to be nil on error")
	}
	if !called.ensure || !called.attach {
		t.Fatalf("expected ensure/attach to be called")
	}
}

func TestBootstrapPlatformWithDepsOpenDBError(t *testing.T) {
	deps := platformDeps{
		loadConfig:  func() (config.Config, error) { return config.Config{}, nil },
		newLogger:   func(config.LoggingConfig) *zap.Logger { return zap.NewNop() },
		openDB:      func(config.DatabaseConfig) (*gorm.DB, error) { return nil, errors.New("db") },
		ensureToken: func(context.Context, *gorm.DB, *config.Config) error { return nil },
		openRedis:   func(config.RedisConfig) (*redis.Client, error) { return nil, nil },
		attachLogSink: func(logger *zap.Logger, rdb *redis.Client, cfg config.LoggingConfig, source string) (*zap.Logger, func()) {
			return logger, func() {}
		},
	}

	if _, err := bootstrapPlatformWith(context.Background(), deps); err == nil {
		t.Fatalf("expected error from openDB")
	}
}

func TestBootstrapPluginServiceAndDomains(t *testing.T) {
	resources := &platformResources{
		cfg: &config.Config{
			Worker: config.WorkerConfig{ID: "worker"},
			Plugins: config.PluginsConfig{
				Services: map[string]config.PluginServiceEntry{
					"octo_demo": {Address: "127.0.0.1:50051"},
				},
				Timeout: config.PluginsTimeoutConfig{Account: 30 * time.Second},
			},
		},
		logger: zap.NewNop(),
		db:     testutil.NewTestDB(t),
	}

	pluginSvc, err := bootstrapPluginService(context.Background(), resources)
	if err != nil {
		t.Fatalf("bootstrap plugin service: %v", err)
	}
	if pluginSvc == nil {
		t.Fatalf("expected plugin service")
	}

	services := bootstrapDomainServices(resources, pluginSvc)
	if services == nil || services.agents == nil {
		t.Fatalf("expected domain services")
	}
}

func TestBootstrapWithDeps(t *testing.T) {
	origProvider := bootstrapDepsProvider
	defer func() { bootstrapDepsProvider = origProvider }()

	resources := &platformResources{
		cfg:    &config.Config{Worker: config.WorkerConfig{ID: "worker"}},
		logger: zap.NewNop(),
		db:     testutil.NewTestDB(t),
	}

	called := struct{ retention, sync bool }{}
	bootstrapDepsProvider = func() bootstrapDeps {
		return bootstrapDeps{
			bootstrapPlatform: func(context.Context) (*platformResources, error) { return resources, nil },
			bootstrapPlugins: func(context.Context, *platformResources) (plugins.PluginService, error) {
				return stubPluginService{}, nil
			},
			bootstrapDomains: func(*platformResources, plugins.PluginService) *domainServices { return &domainServices{} },
			enforceRetention: func(context.Context, *App) { called.retention = true },
			syncAccountTypes: func(context.Context, *App) { called.sync = true },
		}
	}

	app, err := Bootstrap(context.Background())
	if err != nil {
		t.Fatalf("bootstrap: %v", err)
	}
	if app == nil || !called.retention || !called.sync {
		t.Fatalf("expected app and startup hooks")
	}
}

func TestAppSyncPluginAccountTypes(t *testing.T) {
	db := testutil.NewTestDB(t)
	service := accounttypeapp.New(accounttypepostgres.New(db))
	app := &App{AccountTypes: service}

	app.Plugins = stubPluginService{
		syncFn: func(ctx context.Context, fn pluginapp.SyncAccountTypeFunc) error {
			return fn(ctx, pluginapp.AccountTypeSpec{
				Key:      "demo",
				Name:     "Demo",
				Category: "generic",
				Schema:   map[string]any{"a": 1},
			})
		},
	}

	if err := app.SyncPluginAccountTypes(context.Background()); err != nil {
		t.Fatalf("sync plugin account types: %v", err)
	}

	item, err := service.GetByKey(context.Background(), "demo")
	if err != nil {
		t.Fatalf("load account type: %v", err)
	}
	if item == nil || item.Key != "demo" {
		t.Fatalf("expected synced account type")
	}
}

var _ plugins.PluginService = stubPluginService{}
