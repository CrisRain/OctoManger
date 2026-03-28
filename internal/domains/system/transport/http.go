package systemtransport

import (
	"context"
	"net/http"
	"strconv"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/route"

	systemapp "octomanger/internal/domains/system/app"
	"octomanger/internal/platform/httpx"
)

type Handler struct {
	service systemapp.Service
}

func NewHandler(service systemapp.Service) Handler {
	return Handler{service: service}
}

// Register registers healthz on root and system routes on v2 group.
func (h Handler) Register(root *route.RouterGroup, v2 *route.RouterGroup) {
	root.GET("/healthz", h.healthz)
	root.GET("/api/v2/setup/status", h.setupStatus)
	root.POST("/api/v2/setup/initialize", h.setupInitialize)

	v2.GET("/system/status", h.status)
	v2.GET("/system/logs", h.logs)
	v2.GET("/dashboard", h.dashboard)
	v2.GET("/config", h.getConfig)
	v2.PUT("/config", h.putConfig)
}

func (h Handler) healthz(ctx context.Context, c *app.RequestContext) {
	c.JSON(http.StatusOK, map[string]any{"status": "ok"})
}

func (h Handler) setupStatus(ctx context.Context, c *app.RequestContext) {
	item, err := h.service.SetupStatus(ctx)
	if err != nil {
		httpx.InternalServerError(ctx, c, err.Error())
		return
	}
	c.JSON(http.StatusOK, item)
}

func (h Handler) dashboard(ctx context.Context, c *app.RequestContext) {
	summary, err := h.service.DashboardSummary(ctx)
	if err != nil {
		httpx.InternalServerError(ctx, c, err.Error())
		return
	}
	c.JSON(http.StatusOK, summary)
}

func (h Handler) status(ctx context.Context, c *app.RequestContext) {
	status, err := h.service.Status(ctx)
	if err != nil {
		httpx.InternalServerError(ctx, c, err.Error())
		return
	}
	c.JSON(http.StatusOK, status)
}

func (h Handler) logs(ctx context.Context, c *app.RequestContext) {
	limit := 200
	if raw, ok := c.GetQuery("limit"); ok {
		if parsed, err := strconv.Atoi(string(raw)); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	items, err := h.service.ListLogs(ctx, limit)
	if err != nil {
		httpx.InternalServerError(ctx, c, err.Error())
		return
	}
	c.JSON(http.StatusOK, map[string]any{"items": items})
}
