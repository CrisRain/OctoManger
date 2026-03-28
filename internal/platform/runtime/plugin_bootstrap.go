package runtime

import (
	"context"

	"go.uber.org/zap"

	plugins "octomanger/internal/domains/plugins"
	pluginapp "octomanger/internal/domains/plugins/app"
	"octomanger/internal/domains/plugins/grpcclient"
	"octomanger/internal/platform/database"
)

func bootstrapPluginService(ctx context.Context, resources *platformResources) (plugins.PluginService, error) {
	timeouts := pluginapp.ExecutionTimeouts{
		Account: resources.cfg.Plugins.Timeout.Account,
		Job:     resources.cfg.Plugins.Timeout.Job,
		Agent:   resources.cfg.Plugins.Timeout.Agent,
	}

	settingsStore := database.NewPluginSettingsStore(resources.db)
	serviceConfigStore := database.NewPluginServiceConfigStore(resources.db)

	resolvedServices, err := resolvePluginServices(ctx, serviceConfigStore, resources.cfg.Plugins.Services)
	if err != nil {
		return nil, err
	}
	resources.cfg.Plugins.Services = resolvedServices

	resources.logger.Info("plugin backend: gRPC microservices",
		zap.Int("registered_services", len(resources.cfg.Plugins.Services)),
	)

	registry := grpcclient.NewStaticRegistry(toGRPCServiceMap(resources.cfg.Plugins.Services))
	client := grpcclient.New(registry).
		WithExecutionTimeouts(timeouts).
		WithTransportSecurity(grpcclient.TransportSecurityConfig{
			AllowInsecureRemote: resources.cfg.Plugins.GRPC.AllowInsecureRemote,
			InsecureSkipVerify:  resources.cfg.Plugins.GRPC.InsecureSkipVerify,
		}).
		WithInternalAPI(buildPluginInternalAPIConfig(resources.cfg))
	return client.WithSettingsStore(settingsStore), nil
}
