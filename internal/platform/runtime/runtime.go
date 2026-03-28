package runtime

import (
	"context"
	"database/sql"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/redis/go-redis/v9"

	accounttypeapp "octomanger/internal/domains/account-types/app"
	accounttypedomain "octomanger/internal/domains/account-types/domain"
	accountapp "octomanger/internal/domains/accounts/app"
	agentapp "octomanger/internal/domains/agents/app"
	emailapp "octomanger/internal/domains/email/app"
	jobapp "octomanger/internal/domains/jobs/app"
	plugins "octomanger/internal/domains/plugins"
	pluginapp "octomanger/internal/domains/plugins/app"
	"octomanger/internal/domains/plugins/grpcclient"
	systemapp "octomanger/internal/domains/system/app"
	triggerapp "octomanger/internal/domains/triggers/app"
	"octomanger/internal/platform/config"
)

type App struct {
	Config       config.Config
	Logger       *zap.Logger
	DB           *gorm.DB
	Redis        *redis.Client
	AccountTypes accounttypeapp.Service
	Accounts     accountapp.Service
	Email        emailapp.Service
	Triggers     triggerapp.Service
	Plugins      plugins.PluginService
	Jobs         jobapp.Service
	Agents       *agentapp.Service
	System       systemapp.Service
	flushLogs    func()
}

// Close releases DB and Redis connections.
func (a *App) Close() {
	if grpcPlugins, ok := a.Plugins.(*grpcclient.Client); ok && grpcPlugins != nil {
		grpcPlugins.Close()
	}
	if a.flushLogs != nil {
		a.flushLogs()
	}
	if sqlDB, err := a.DB.DB(); err == nil {
		if err := closeSQLDB(sqlDB); err != nil {
			a.Logger.Error("close db connection", zap.Error(err))
		}
	}
	if a.Redis != nil {
		if err := closeRedis(a.Redis); err != nil {
			a.Logger.Error("close redis connection", zap.Error(err))
		}
	}
	_ = a.Logger.Sync()
}

var closeSQLDB = func(db *sql.DB) error {
	return db.Close()
}

var closeRedis = func(rdb *redis.Client) error {
	return rdb.Close()
}

func Bootstrap(ctx context.Context) (*App, error) {
	return bootstrapWith(ctx, bootstrapDepsProvider())
}

var bootstrapDepsProvider = defaultBootstrapDeps

type bootstrapDeps struct {
	bootstrapPlatform func(context.Context) (*platformResources, error)
	bootstrapPlugins  func(context.Context, *platformResources) (plugins.PluginService, error)
	bootstrapDomains  func(*platformResources, plugins.PluginService) *domainServices
	enforceRetention  func(context.Context, *App)
	syncAccountTypes  func(context.Context, *App)
}

func defaultBootstrapDeps() bootstrapDeps {
	return bootstrapDeps{
		bootstrapPlatform: bootstrapPlatform,
		bootstrapPlugins:  bootstrapPluginService,
		bootstrapDomains:  bootstrapDomainServices,
		enforceRetention:  enforceLogRetentionOnStartup,
		syncAccountTypes:  syncPluginAccountTypesOnStartup,
	}
}

func bootstrapWith(ctx context.Context, deps bootstrapDeps) (*App, error) {
	resources, err := deps.bootstrapPlatform(ctx)
	if err != nil {
		return nil, err
	}

	pluginSvc, err := deps.bootstrapPlugins(ctx, resources)
	if err != nil {
		return nil, err
	}

	services := deps.bootstrapDomains(resources, pluginSvc)

	app := &App{
		Config:       *resources.cfg,
		Logger:       resources.logger,
		DB:           resources.db,
		Redis:        resources.rdb,
		AccountTypes: services.accountTypes,
		Accounts:     services.accounts,
		Email:        services.email,
		Triggers:     services.triggers,
		Plugins:      services.plugins,
		Jobs:         services.jobs,
		Agents:       services.agents,
		System:       services.system,
		flushLogs:    resources.flushLogs,
	}

	deps.enforceRetention(ctx, app)
	deps.syncAccountTypes(ctx, app)

	return app, nil
}

func (a *App) SyncPluginAccountTypes(ctx context.Context) error {
	return a.Plugins.SyncAccountTypes(ctx, func(ctx context.Context, spec pluginapp.AccountTypeSpec) error {
		_, err := a.AccountTypes.Upsert(ctx, accounttypedomain.CreateInput{
			Key:          spec.Key,
			Name:         spec.Name,
			Category:     spec.Category,
			Schema:       spec.Schema,
			Capabilities: spec.Capabilities,
		})
		return err
	})
}

// toGRPCServiceMap converts the config's PluginServiceEntry map to the type
// expected by grpcclient.NewStaticRegistry.
func toGRPCServiceMap(src map[string]config.PluginServiceEntry) map[string]grpcclient.PluginServiceConfig {
	dst := make(map[string]grpcclient.PluginServiceConfig, len(src))
	for k, v := range src {
		dst[k] = grpcclient.PluginServiceConfig{
			Address:               v.Address,
			AllowInsecure:         v.AllowInsecure,
			TLSServerName:         v.TLSServerName,
			TLSInsecureSkipVerify: v.TLSInsecureSkipVerify,
		}
	}
	return dst
}
