package plugintransport

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/route"

	accounttypeapp "octomanger/internal/domains/account-types/app"
	accounttypedomain "octomanger/internal/domains/account-types/domain"
	pluginapp "octomanger/internal/domains/plugins/app"
	"octomanger/internal/platform/auth"
	"octomanger/internal/platform/httpx"
)

// ConfigStore is the subset of systemapp.Service used for plugin settings persistence.
type ConfigStore interface {
	GetConfig(ctx context.Context, key string) (json.RawMessage, error)
	SetConfig(ctx context.Context, key string, value json.RawMessage) error
}

type Handler struct {
	adminKey     string
	service      pluginapp.Service
	accountTypes accounttypeapp.Service
	configs      ConfigStore
}

func NewHandler(adminKey string, service pluginapp.Service, accountTypes accounttypeapp.Service, configs ConfigStore) Handler {
	return Handler{adminKey: adminKey, service: service, accountTypes: accountTypes, configs: configs}
}

func (h Handler) Register(r *route.RouterGroup) {
	guard := auth.RequireAdmin(h.adminKey)
	r.GET("/plugins", h.list)
	r.GET("/plugins/:key", h.get)
	r.POST("/plugins/sync", guard, h.sync)
	r.GET("/plugins/:key/settings", guard, h.getSettings)
	r.PUT("/plugins/:key/settings", guard, h.putSettings)
}

func (h Handler) list(ctx context.Context, c *app.RequestContext) {
	plugins, err := h.service.List(ctx)
	if err != nil {
		httpx.InternalServerError(ctx, c, err.Error())
		return
	}
	c.JSON(http.StatusOK, map[string]any{"items": plugins})
}

func (h Handler) get(ctx context.Context, c *app.RequestContext) {
	plugin, err := h.service.Get(ctx, c.Param("key"))
	if err != nil {
		httpx.NotFound(ctx, c, err.Error())
		return
	}
	if plugin == nil {
		httpx.NotFound(ctx, c, "plugin not found")
		return
	}
	c.JSON(http.StatusOK, plugin)
}

func (h Handler) sync(ctx context.Context, c *app.RequestContext) {
	var synced, failed int
	var errs []string

	err := h.service.SyncAccountTypes(ctx, func(ctx context.Context, spec pluginapp.AccountTypeSpec) error {
		_, err := h.accountTypes.Upsert(ctx, accounttypedomain.CreateInput{
			Key:          spec.Key,
			Name:         spec.Name,
			Category:     spec.Category,
			Schema:       spec.Schema,
			Capabilities: spec.Capabilities,
		})
		if err != nil {
			failed++
			errs = append(errs, spec.Key+": "+err.Error())
			return nil
		}
		synced++
		return nil
	})
	if err != nil {
		httpx.InternalServerError(ctx, c, err.Error())
		return
	}
	c.JSON(http.StatusOK, map[string]any{
		"synced": synced,
		"failed": failed,
		"errors": errs,
	})
}

func (h Handler) getSettings(ctx context.Context, c *app.RequestContext) {
	key := c.Param("key")
	raw, err := h.configs.GetConfig(ctx, settingsKey(key))
	if err != nil {
		if errors.Is(err, fmt.Errorf("config not found")) || err.Error() == "config not found" {
			c.JSON(http.StatusOK, map[string]any{})
			return
		}
		httpx.InternalServerError(ctx, c, err.Error())
		return
	}
	c.Data(http.StatusOK, "application/json", raw)
}

func (h Handler) putSettings(ctx context.Context, c *app.RequestContext) {
	key := c.Param("key")
	body := c.Request.Body()
	if len(body) == 0 {
		body = []byte("{}")
	}
	if !json.Valid(body) {
		httpx.BadRequest(ctx, c, "body must be valid JSON")
		return
	}
	if err := h.configs.SetConfig(ctx, settingsKey(key), json.RawMessage(body)); err != nil {
		httpx.InternalServerError(ctx, c, err.Error())
		return
	}
	c.JSON(http.StatusOK, map[string]any{"saved": true})
}

func settingsKey(pluginKey string) string {
	return "plugin_settings:" + pluginKey
}
