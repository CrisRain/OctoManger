package main

import (
	"context"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"go.uber.org/zap"

	"octomanger/backend/config"
	"octomanger/backend/internal/daemon"
	"octomanger/backend/internal/repository"
	"octomanger/backend/internal/service"
	"octomanger/backend/pkg/database"
	"octomanger/backend/pkg/logger"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		panic(err)
	}

	log, err := logger.Init(cfg.Logging)
	if err != nil {
		fallback, _ := zap.NewProduction()
		fallback.Fatal("failed to init logger", zap.Error(err))
	}
	defer func() { _ = log.Sync() }()

	db, err := database.Init(cfg.Database)
	if err != nil {
		log.Fatal("failed to init database", zap.Error(err))
	}
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatal("failed to get sql.DB", zap.Error(err))
	}
	defer func() { _ = sqlDB.Close() }()

	apiKeyRepo := repository.NewApiKeyRepository(db)
	systemConfigRepo := repository.NewSystemConfigRepository(db)
	internalToken, tokenErr := service.EnsureInternalKey(context.Background(), apiKeyRepo, systemConfigRepo)
	if tokenErr != nil {
		log.Warn("could not ensure internal api key", zap.Error(tokenErr))
	}

	mgr := daemon.NewManager(db, daemon.Config{
		PythonBin:        cfg.Python.Bin,
		ModuleDir:        strings.TrimSpace(cfg.Paths.OctoModuleDir),
		InternalAPIURL:   internalAPIURL(cfg),
		InternalAPIToken: internalToken,
	}, log)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	log.Info("daemon manager started")
	if err := mgr.Run(ctx); err != nil {
		log.Fatal("daemon manager failed", zap.Error(err))
	}
	log.Info("daemon manager stopped")
}

func internalAPIURL(cfg config.Config) string {
	port := strings.TrimSpace(cfg.Server.Port)
	if port == "" {
		port = "8080"
	}
	port = strings.TrimPrefix(port, ":")
	if cfg.Server.TLS {
		return "https://127.0.0.1:" + port
	}
	return "http://127.0.0.1:" + port
}
