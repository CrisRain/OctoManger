package accountstransport

import (
	"context"
	"errors"
	"net/http"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/route"

	accountapp "octomanger/internal/domains/accounts/app"
	accountdomain "octomanger/internal/domains/accounts/domain"
	accountpostgres "octomanger/internal/domains/accounts/infra/postgres"
	"octomanger/internal/platform/httpx"
)

type Handler struct {
	service accountapp.Service
}

func NewHandler(service accountapp.Service) Handler {
	return Handler{service: service}
}

func (h Handler) Register(r *route.RouterGroup) {
	r.GET("/accounts", h.list)
	r.POST("/accounts", h.create)
	r.GET("/accounts/:id", h.get)
	r.PATCH("/accounts/:id", h.patch)
	r.DELETE("/accounts/:id", h.delete)
	r.POST("/accounts/:id/execute", h.execute)
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
	id, err := httpx.PathInt64(c, "id")
	if err != nil {
		httpx.BadRequest(ctx, c, "invalid account id")
		return
	}
	item, err := h.service.Get(ctx, id)
	if err != nil {
		if errors.Is(err, accountpostgres.ErrNotFound) {
			httpx.NotFound(ctx, c, "account not found")
			return
		}
		httpx.InternalServerError(ctx, c, err.Error())
		return
	}
	c.JSON(http.StatusOK, item)
}

func (h Handler) create(ctx context.Context, c *app.RequestContext) {
	var input accountdomain.CreateInput
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
	id, err := httpx.PathInt64(c, "id")
	if err != nil {
		httpx.BadRequest(ctx, c, "invalid account id")
		return
	}
	var input accountdomain.PatchInput
	if err := httpx.DecodeJSON(c, &input); err != nil {
		httpx.BadRequest(ctx, c, err.Error())
		return
	}
	item, err := h.service.Patch(ctx, id, input)
	if err != nil {
		if errors.Is(err, accountpostgres.ErrNotFound) {
			httpx.NotFound(ctx, c, "account not found")
			return
		}
		httpx.BadRequest(ctx, c, err.Error())
		return
	}
	c.JSON(http.StatusOK, item)
}

func (h Handler) delete(ctx context.Context, c *app.RequestContext) {
	id, err := httpx.PathInt64(c, "id")
	if err != nil {
		httpx.BadRequest(ctx, c, "invalid account id")
		return
	}
	if err := h.service.Delete(ctx, id); err != nil {
		if errors.Is(err, accountpostgres.ErrNotFound) {
			httpx.NotFound(ctx, c, "account not found")
			return
		}
		httpx.InternalServerError(ctx, c, err.Error())
		return
	}
	c.JSON(http.StatusOK, map[string]any{"deleted": true})
}

type executeBody struct {
	Action string         `json:"action"`
	Params map[string]any `json:"params"`
}

func (h Handler) execute(ctx context.Context, c *app.RequestContext) {
	id, err := httpx.PathInt64(c, "id")
	if err != nil {
		httpx.BadRequest(ctx, c, "invalid account id")
		return
	}
	var body executeBody
	if err := httpx.DecodeJSON(c, &body); err != nil {
		httpx.BadRequest(ctx, c, err.Error())
		return
	}
	if body.Action == "" {
		httpx.BadRequest(ctx, c, "action is required")
		return
	}

	params := body.Params
	if params == nil {
		params = map[string]any{}
	}

	var (
		result *accountapp.ExecuteActionResult
	)

	result, err = h.service.ExecuteAction(ctx, id, body.Action, params)
	if err != nil {
		if errors.Is(err, accountpostgres.ErrNotFound) {
			httpx.NotFound(ctx, c, "account not found")
			return
		}
		if errors.Is(err, accountapp.ErrInvalidExecuteAction) ||
			errors.Is(err, accountapp.ErrMissingAccountTypeKey) ||
			errors.Is(err, accountapp.ErrPluginBackendMissing) {
			httpx.BadRequest(ctx, c, err.Error())
			return
		}
		httpx.InternalServerError(ctx, c, err.Error())
		return
	}

	c.JSON(http.StatusOK, result)
}
