package runtime

import (
	"context"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/redis/go-redis/v9"

	accounttypeapp "octomanger/internal/domains/account-types/app"
	accounttypedomain "octomanger/internal/domains/account-types/domain"
	accounttypepostgres "octomanger/internal/domains/account-types/infra/postgres"
	accountapp "octomanger/internal/domains/accounts/app"
	accountpostgres "octomanger/internal/domains/accounts/infra/postgres"
	agentapp "octomanger/internal/domains/agents/app"
	agentpostgres "octomanger/internal/domains/agents/infra/postgres"
	emailapp "octomanger/internal/domains/email/app"
	emailpostgres "octomanger/internal/domains/email/infra/postgres"
	jobapp "octomanger/internal/domains/jobs/app"
	jobpostgres "octomanger/internal/domains/jobs/infra/postgres"
	pluginapp "octomanger/internal/domains/plugins/app"
	"octomanger/internal/domains/plugins/infra/fsrepo"
	systemapp "octomanger/internal/domains/system/app"
	triggerapp "octomanger/internal/domains/triggers/app"
	triggerpostgres "octomanger/internal/domains/triggers/infra/postgres"
	"octomanger/internal/platform/config"
	"octomanger/internal/platform/database"
	"octomanger/internal/platform/logging"
	redisclient "octomanger/internal/platform/redis"
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
	Plugins      pluginapp.Service
	Jobs         jobapp.Service
	Agents       *agentapp.Service
	System       systemapp.Service
}

// Close releases DB and Redis connections.
func (a *App) Close() {
	if sqlDB, err := a.DB.DB(); err == nil {
		sqlDB.Close()
	}
	if a.Redis != nil {
		a.Redis.Close()
	}
	_ = a.Logger.Sync()
}

func Bootstrap(ctx context.Context) (*App, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}

	logger := logging.New(cfg.Logging)

	db, err := database.Open(cfg.Database)
	if err != nil {
		return nil, err
	}

	rdb, err := redisclient.New(cfg.Redis)
	if err != nil {
		logger.Warn("redis unavailable, continuing without cache", zap.Error(err))
		rdb = nil
	}

	pluginRepo := fsrepo.New(cfg.Plugins.ModulesDir)
	plugins := pluginapp.New(pluginRepo, cfg.Plugins.PythonBin, cfg.Plugins.SDKDir)

	accountTypeRepo := accounttypepostgres.New(db)
	accountTypes := accounttypeapp.New(accountTypeRepo)

	accountRepo := accountpostgres.New(db)
	accounts := accountapp.New(accountRepo)

	emailRepo := emailpostgres.New(db)
	email := emailapp.New(emailRepo)

	jobRepo := jobpostgres.New(db)
	jobs := jobapp.New(logger, jobRepo, plugins, cfg.Worker.ID)

	triggerRepo := triggerpostgres.New(db)
	triggers := triggerapp.New(triggerRepo, jobs)

	agentRepo := agentpostgres.New(db)
	agents := agentapp.New(
		logger,
		agentRepo,
		plugins,
		rdb,
		cfg.Worker.ID,
		cfg.Worker.AgentLoopInterval,
		cfg.Worker.AgentErrorBackoff,
	)

	system := systemapp.New(db, plugins)

	// Sync account types from plugin account_type.{key}.json files on every startup.
	if syncErr := plugins.SyncAccountTypes(ctx, func(ctx context.Context, spec pluginapp.AccountTypeSpec) error {
		_, err := accountTypes.Upsert(ctx, accounttypedomain.CreateInput{
			Key:          spec.Key,
			Name:         spec.Name,
			Category:     spec.Category,
			Schema:       spec.Schema,
			Capabilities: spec.Capabilities,
		})
		return err
	}); syncErr != nil {
		logger.Warn("plugin account type sync failed", zap.Error(syncErr))
	} else {
		logger.Info("plugin account types synced")
	}

	return &App{
		Config:       cfg,
		Logger:       logger,
		DB:           db,
		Redis:        rdb,
		AccountTypes: accountTypes,
		Accounts:     accounts,
		Email:        email,
		Triggers:     triggers,
		Plugins:      plugins,
		Jobs:         jobs,
		Agents:       agents,
		System:       system,
	}, nil
}
