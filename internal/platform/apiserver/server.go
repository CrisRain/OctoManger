package apiserver

import (
	"context"
	"net/http"
	"time"

	hertzapp "github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/common/hlog"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	"go.uber.org/zap"

	accounttypestransport "octomanger/internal/domains/account-types/transport"
	accountstransport "octomanger/internal/domains/accounts/transport"
	agenttransport "octomanger/internal/domains/agents/transport"
	emailtransport "octomanger/internal/domains/email/transport"
	jobtransport "octomanger/internal/domains/jobs/transport"
	plugintransport "octomanger/internal/domains/plugins/transport"
	systemtransport "octomanger/internal/domains/system/transport"
	triggertransport "octomanger/internal/domains/triggers/transport"
	"octomanger/internal/platform/auth"
	"octomanger/internal/platform/database"
	platformruntime "octomanger/internal/platform/runtime"
	platformwebui "octomanger/internal/platform/webui"
)

func Run(ctx context.Context) error {
	application, err := platformruntime.Bootstrap(ctx)
	if err != nil {
		return err
	}
	defer application.Close()

	return RunWithApp(ctx, application)
}

func RunWithApp(ctx context.Context, application *platformruntime.App) error {
	logger := application.Logger.Named("api")
	hlog.SetLevel(hlog.LevelWarn)

	h := server.New(
		server.WithHostPorts(application.Config.Server.APIAddr),
		server.WithReadTimeout(application.Config.Server.ReadTimeout),
		server.WithIdleTimeout(application.Config.Server.IdleTimeout),
		server.WithExitWaitTime(5*time.Second),
		server.WithDisablePrintRoute(true),
	)
	h.Use(RequestLoggingMiddleware(logger))
	h.Use(CORSMiddleware(application.Config.Server.CORS))

	root := h.Group("/")
	v2 := h.Group("/api/v2")
	v2.Use(auth.RequireAdminForRouterWithVerifier(application.System))
	pluginSettingsStore := database.NewPluginSettingsStore(application.DB)
	pluginServiceConfigStore := database.NewPluginServiceConfigStore(application.DB)

	systemtransport.NewHandler(application.System).Register(root, v2)
	plugintransport.NewHandler(
		application.Plugins,
		application.AccountTypes,
		pluginSettingsStore,
		pluginServiceConfigStore,
	).Register(v2)
	jobtransport.NewHandler(application.Jobs).Register(v2)
	agenttransport.NewHandler(application.Agents).Register(v2)
	accounttypestransport.NewHandler(application.AccountTypes).Register(v2)
	accountstransport.NewHandler(application.Accounts).Register(v2)
	emailtransport.NewHandler(application.Email).Register(v2)
	triggertransport.NewHandler(application.Triggers).Register(v2)
	registerPluginInternalAPI(root, application.Config.Auth.PluginInternalAPIToken, application.Accounts, application.Email)

	if assets, source := platformwebui.Open(); assets != nil {
		logger.Info("web ui assets ready", zap.String("source", source))
		h.NoRoute(func(reqCtx context.Context, c *hertzapp.RequestContext) {
			file, ok := ResolveStaticFile(
				assets,
				string(c.Method()),
				string(c.Path()),
				string(c.GetHeader("Accept")),
				string(c.GetHeader("Accept-Encoding")),
			)
			if ok {
				ApplyStaticFileHeaders(c, file)
				if string(c.Method()) == http.MethodHead {
					c.Status(consts.StatusOK)
					return
				}
				c.Data(consts.StatusOK, file.ContentType, file.Body)
				return
			}

			c.JSON(consts.StatusNotFound, map[string]any{"error": "not found"})
		})
	} else {
		logger.Warn("web ui assets unavailable")
	}

	go func() {
		<-ctx.Done()
		logger.Info("shutting down api server")
		h.Shutdown(context.Background()) //nolint:errcheck
	}()

	logger.Info("api server starting", zap.String("addr", application.Config.Server.APIAddr))
	h.Spin()
	return nil
}
