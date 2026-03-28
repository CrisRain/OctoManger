package systemtransport

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/cloudwego/hertz/pkg/app"

	systemapp "octomanger/internal/domains/system/app"
	"octomanger/internal/platform/httpx"
)

func (h Handler) getConfig(ctx context.Context, c *app.RequestContext) {
	item, err := h.service.GetConfig(ctx)
	if err != nil {
		httpx.InternalServerError(ctx, c, err.Error())
		return
	}
	c.JSON(http.StatusOK, item)
}

func (h Handler) putConfig(ctx context.Context, c *app.RequestContext) {
	var req struct {
		AppName                  string `json:"app_name"`
		JobDefaultTimeoutMinutes int    `json:"job_default_timeout_minutes"`
		JobMaxConcurrency        int    `json:"job_max_concurrency"`
	}
	if err := httpx.DecodeJSON(c, &req); err != nil {
		httpx.BadRequest(ctx, c, "invalid JSON body")
		return
	}
	item, err := h.service.SetConfig(ctx, systemapp.Config{
		AppName:                  req.AppName,
		JobDefaultTimeoutMinutes: req.JobDefaultTimeoutMinutes,
		JobMaxConcurrency:        req.JobMaxConcurrency,
	})
	if err != nil {
		if strings.Contains(err.Error(), "must be >=") {
			httpx.BadRequest(ctx, c, err.Error())
			return
		}
		httpx.InternalServerError(ctx, c, err.Error())
		return
	}
	c.JSON(http.StatusOK, item)
}

func (h Handler) setupInitialize(ctx context.Context, c *app.RequestContext) {
	var req struct {
		AppName                  string `json:"app_name"`
		JobDefaultTimeoutMinutes int    `json:"job_default_timeout_minutes"`
		JobMaxConcurrency        int    `json:"job_max_concurrency"`
	}
	if err := httpx.DecodeJSON(c, &req); err != nil {
		httpx.BadRequest(ctx, c, "invalid JSON body")
		return
	}

	result, err := h.service.Initialize(ctx, systemapp.Config{
		AppName:                  req.AppName,
		JobDefaultTimeoutMinutes: req.JobDefaultTimeoutMinutes,
		JobMaxConcurrency:        req.JobMaxConcurrency,
	})
	if err != nil {
		switch {
		case errors.Is(err, systemapp.ErrAlreadyInitialized):
			httpx.WriteProblem(ctx, c, http.StatusConflict, "already_initialized", "system is already initialized")
		case strings.Contains(err.Error(), "must be >="):
			httpx.BadRequest(ctx, c, err.Error())
		default:
			httpx.InternalServerError(ctx, c, err.Error())
		}
		return
	}

	c.JSON(http.StatusCreated, result)
}
