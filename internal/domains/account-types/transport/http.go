package accounttypestransport

import (
	"context"
	"errors"
	"net/http"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/route"

	accounttypeapp "octomanger/internal/domains/account-types/app"
	accounttypedomain "octomanger/internal/domains/account-types/domain"
	accounttypepostgres "octomanger/internal/domains/account-types/infra/postgres"
	"octomanger/internal/platform/httpx"
)

type Handler struct {
	service accounttypeapp.Service
}

func NewHandler(service accounttypeapp.Service) Handler {
	return Handler{service: service}
}

func (h Handler) Register(r *route.RouterGroup) {
	r.GET("/account-types", h.list)
	r.POST("/account-types", h.create)
	r.GET("/account-types/:key", h.get)
	r.PATCH("/account-types/:key", h.patch)
	r.DELETE("/account-types/:key", h.delete)
}

func (h Handler) list(ctx context.Context, c *app.RequestContext) {
	page, err := httpx.ParsePageRequest(c)
	if err != nil {
		httpx.BadRequest(ctx, c, err.Error())
		return
	}
	items, total, err := h.service.ListPage(ctx, page.Limit, page.Offset)
	if err != nil {
		httpx.InternalServerError(ctx, c, err.Error())
		return
	}
	c.JSON(http.StatusOK, map[string]any{
		"items":      items,
		"pagination": httpx.BuildPageMeta(page, total),
	})
}

func (h Handler) get(ctx context.Context, c *app.RequestContext) {
	item, err := h.service.GetByKey(ctx, c.Param("key"))
	if err != nil {
		if errors.Is(err, accounttypepostgres.ErrNotFound) {
			httpx.NotFound(ctx, c, "account type not found")
			return
		}
		httpx.InternalServerError(ctx, c, err.Error())
		return
	}
	c.JSON(http.StatusOK, item)
}

func (h Handler) create(ctx context.Context, c *app.RequestContext) {
	var input accounttypedomain.CreateInput
	if err := httpx.DecodeJSON(c, &input); err != nil {
		httpx.BadRequest(ctx, c, err.Error())
		return
	}
	item, err := h.service.Create(ctx, input)
	if err != nil {
		httpx.BadRequest(ctx, c, err.Error())
		return
	}
	c.JSON(http.StatusCreated, item)
}

func (h Handler) patch(ctx context.Context, c *app.RequestContext) {
	var input accounttypedomain.PatchInput
	if err := httpx.DecodeJSON(c, &input); err != nil {
		httpx.BadRequest(ctx, c, err.Error())
		return
	}
	item, err := h.service.Patch(ctx, c.Param("key"), input)
	if err != nil {
		if errors.Is(err, accounttypepostgres.ErrNotFound) {
			httpx.NotFound(ctx, c, "account type not found")
			return
		}
		httpx.BadRequest(ctx, c, err.Error())
		return
	}
	c.JSON(http.StatusOK, item)
}

func (h Handler) delete(ctx context.Context, c *app.RequestContext) {
	if err := h.service.Delete(ctx, c.Param("key")); err != nil {
		if errors.Is(err, accounttypepostgres.ErrNotFound) {
			httpx.NotFound(ctx, c, "account type not found")
			return
		}
		httpx.InternalServerError(ctx, c, err.Error())
		return
	}
	c.JSON(http.StatusOK, map[string]any{"deleted": true})
}
