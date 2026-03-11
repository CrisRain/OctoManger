package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"go.uber.org/zap"

	"octomanger/backend/config"
	"octomanger/backend/internal/octomodsvc"
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
	defer func() {
		_ = log.Sync()
	}()

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	if runErr := octomodsvc.Run(ctx, cfg, log); runErr != nil {
		log.Fatal("octomodule service failed", zap.Error(runErr))
	}
}
