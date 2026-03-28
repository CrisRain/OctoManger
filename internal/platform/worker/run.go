package worker

import (
	"context"
	"time"

	"go.uber.org/zap"

	agentapp "octomanger/internal/domains/agents/app"
	"octomanger/internal/domains/plugins/grpclauncher"
	platformconfig "octomanger/internal/platform/config"
	platformruntime "octomanger/internal/platform/runtime"
)

func Run(ctx context.Context) error {
	application, err := bootstrapApp(ctx)
	if err != nil {
		return err
	}
	defer application.Close()

	return RunWithApp(ctx, application)
}

func RunWithApp(ctx context.Context, application *platformruntime.App) error {
	logger := application.Logger.Named("worker")

	startPluginLauncherFn(ctx, application, logger)
	startPluginAccountTypeSyncFn(ctx, application, logger, 5*time.Second)
	startAgentSupervisorFn(ctx, application, logger)
	runJobLoopFn(ctx, application, logger)

	return nil
}

var (
	bootstrapApp                 = platformruntime.Bootstrap
	startPluginLauncherFn        = startPluginLauncher
	startPluginAccountTypeSyncFn = startPluginAccountTypeSync
	startAgentSupervisorFn       = startAgentSupervisor
	runJobLoopFn                 = runJobLoop
	syncPluginAccountTypesFn     = syncPluginAccountTypes
	discoverLaunchesFn           = grpclauncher.Discover
	newLauncherFn                = func(logger *zap.Logger, pythonBin, sdkDir string, launches []grpclauncher.ProcessConfig) launcherRunner {
		return grpclauncher.New(logger, pythonBin, sdkDir, launches)
	}
	runAgentSupervisorFn = func(ctx context.Context, svc *agentapp.Service, interval time.Duration) error {
		return svc.RunSupervisor(ctx, interval)
	}
)

type launcherRunner interface {
	Run(context.Context) error
}

func startPluginLauncher(ctx context.Context, application *platformruntime.App, logger *zap.Logger) {
	launches := discoverLaunchesFn(application.Config.Plugins.ModulesDir, toAddressMap(application.Config.Plugins.Services))
	go func() {
		manager := newLauncherFn(
			logger.Named("plugin-launcher"),
			application.Config.Plugins.PythonBin,
			application.Config.Plugins.SDKDir,
			launches,
		)
		if err := manager.Run(ctx); err != nil && ctx.Err() == nil {
			logger.Error("plugin launcher stopped unexpectedly", zap.Error(err))
		}
	}()
}

func toAddressMap(services map[string]platformconfig.PluginServiceEntry) map[string]string {
	result := make(map[string]string, len(services))
	for key, service := range services {
		result[key] = service.Address
	}
	return result
}

func startPluginAccountTypeSync(ctx context.Context, application *platformruntime.App, logger *zap.Logger, interval time.Duration) {
	if len(application.Config.Plugins.Services) == 0 {
		return
	}

	go syncPluginAccountTypesFn(ctx, application, logger, interval)
}

func syncPluginAccountTypes(ctx context.Context, application *platformruntime.App, logger *zap.Logger, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	synced := make(map[string]struct{})

	trySync := func() bool {
		plugins, err := application.Plugins.List(ctx)
		if err != nil {
			logger.Debug("list plugins before account type sync failed", zap.Error(err))
			return false
		}

		var healthy int
		var pending int
		for _, plugin := range plugins {
			if plugin.Healthy {
				healthy++
				if _, ok := synced[plugin.Manifest.Key]; !ok {
					pending++
				}
			}
		}
		if healthy == 0 {
			return false
		}
		if pending > 0 {
			if err := application.SyncPluginAccountTypes(ctx); err != nil {
				logger.Warn("plugin account type sync after launcher startup failed", zap.Error(err))
				return false
			}

			for _, plugin := range plugins {
				if plugin.Healthy {
					synced[plugin.Manifest.Key] = struct{}{}
				}
			}

			logger.Info(
				"plugin account types synced after launcher startup",
				zap.Int("healthy_plugins", healthy),
				zap.Int("synced_plugins", len(synced)),
				zap.Int("registered_plugins", len(plugins)),
			)
		}
		return len(plugins) > 0 && len(synced) == len(plugins)
	}

	if trySync() {
		return
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if trySync() {
				return
			}
		}
	}
}

func startAgentSupervisor(ctx context.Context, application *platformruntime.App, logger *zap.Logger) {
	go func() {
		if err := runAgentSupervisorFn(ctx, application.Agents, application.Config.Worker.AgentScanInterval); err != nil && ctx.Err() == nil {
			logger.Error("agent supervisor stopped unexpectedly", zap.Error(err))
		}
	}()
}

// dbOpTimeout caps each per-tick database operation so a stalled connection
// cannot block the worker indefinitely or prevent a clean shutdown.
const dbOpTimeout = 30 * time.Second

func runJobLoop(ctx context.Context, application *platformruntime.App, logger *zap.Logger) {
	ticker := time.NewTicker(application.Config.Worker.PollInterval)
	defer ticker.Stop()

	logger.Info("worker started", zap.String("worker_id", application.Config.Worker.ID))

	for {
		processingCtx, cancel := context.WithTimeout(context.Background(), dbOpTimeout)
		runWorkerTick(ctx, processingCtx, application, logger)
		cancel()

		select {
		case <-ctx.Done():
			logger.Info("worker shutdown requested; current job execution drained")
			return
		case <-ticker.C:
		}
	}
}

func runWorkerTick(rootCtx context.Context, processingCtx context.Context, application *platformruntime.App, logger *zap.Logger) {
	if _, err := application.Jobs.TickSchedules(processingCtx, application.Config.Worker.SchedulePollLimit); err != nil {
		logger.Error("tick schedules failed", zap.Error(err))
	}

	processQueuedExecutions(rootCtx, processingCtx, application, logger)
}

func processQueuedExecutions(rootCtx context.Context, processingCtx context.Context, application *platformruntime.App, logger *zap.Logger) {
	for i := 0; i < application.Config.Worker.ExecutionPollLimit; i++ {
		if rootCtx.Err() != nil {
			return
		}
		processed, err := application.Jobs.ProcessNextExecution(processingCtx)
		if err != nil {
			logger.Error("process execution failed", zap.Error(err))
			return
		}
		if !processed {
			return
		}
	}
}
