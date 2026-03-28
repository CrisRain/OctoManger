package runtime

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"octomanger/internal/platform/config"
	"octomanger/internal/platform/database"
	"octomanger/internal/platform/logging"
	redisclient "octomanger/internal/platform/redis"
)

type platformResources struct {
	cfg       *config.Config
	logger    *zap.Logger
	flushLogs func()
	db        *gorm.DB
	rdb       *redis.Client
}

func bootstrapPlatform(ctx context.Context) (*platformResources, error) {
	deps := platformDeps{
		loadConfig:    config.Load,
		newLogger:     logging.New,
		openDB:        database.Open,
		ensureToken:   ensurePluginInternalAPIToken,
		openRedis:     redisclient.New,
		attachLogSink: logging.AttachSystemLogSink,
	}
	return bootstrapPlatformWith(ctx, deps)
}

type platformDeps struct {
	loadConfig    func() (config.Config, error)
	newLogger     func(config.LoggingConfig) *zap.Logger
	openDB        func(config.DatabaseConfig) (*gorm.DB, error)
	ensureToken   func(context.Context, *gorm.DB, *config.Config) error
	openRedis     func(config.RedisConfig) (*redis.Client, error)
	attachLogSink func(*zap.Logger, *redis.Client, config.LoggingConfig, string) (*zap.Logger, func())
}

func bootstrapPlatformWith(ctx context.Context, deps platformDeps) (*platformResources, error) {
	cfg, err := deps.loadConfig()
	if err != nil {
		return nil, err
	}

	logger := deps.newLogger(cfg.Logging)

	db, err := deps.openDB(cfg.Database)
	if err != nil {
		return nil, err
	}
	if err := deps.ensureToken(ctx, db, &cfg); err != nil {
		return nil, err
	}

	rdb, err := deps.openRedis(cfg.Redis)
	if err != nil {
		logger.Warn("redis unavailable, continuing without cache", zap.Error(err))
		rdb = nil
	}

	logger, flushLogs := deps.attachLogSink(logger, rdb, cfg.Logging, currentProcessName())

	return &platformResources{
		cfg:       &cfg,
		logger:    logger,
		flushLogs: flushLogs,
		db:        db,
		rdb:       rdb,
	}, nil
}

func currentProcessName() string {
	name := strings.TrimSpace(filepath.Base(os.Args[0]))
	if name == "" || name == "." {
		return "octomanger"
	}
	return name
}
