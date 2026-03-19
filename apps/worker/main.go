package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"

	platformruntime "octomanger/internal/platform/runtime"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	application, err := platformruntime.Bootstrap(ctx)
	if err != nil {
		panic(err)
	}
	defer application.Close()

	go func() {
		if err := application.Agents.RunSupervisor(ctx, application.Config.Worker.AgentScanInterval); err != nil && ctx.Err() == nil {
			application.Logger.Error("agent supervisor stopped unexpectedly", zap.Error(err))
		}
	}()

	ticker := time.NewTicker(application.Config.Worker.PollInterval)
	defer ticker.Stop()

	application.Logger.Info("worker started", zap.String("worker_id", application.Config.Worker.ID))
	for {
		if _, err := application.Jobs.TickSchedules(ctx, application.Config.Worker.SchedulePollLimit); err != nil {
			application.Logger.Error("tick schedules failed", zap.Error(err))
		}

		for i := 0; i < application.Config.Worker.ExecutionPollLimit; i++ {
			processed, err := application.Jobs.ProcessNextExecution(ctx)
			if err != nil {
				application.Logger.Error("process execution failed", zap.Error(err))
				break
			}
			if !processed {
				break
			}
		}

		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
