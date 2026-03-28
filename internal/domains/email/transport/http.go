package emailtransport

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/route"

	emailapp "octomanger/internal/domains/email/app"
	emaildomain "octomanger/internal/domains/email/domain"
	emailpostgres "octomanger/internal/domains/email/infra/postgres"
	"octomanger/internal/platform/httpx"
)

type Handler struct {
	service emailapp.Service
}

func NewHandler(service emailapp.Service) Handler {
	return Handler{service: service}
}

func (h Handler) Register(r *route.RouterGroup) {
	r.GET("/email/accounts", h.list)
	r.POST("/email/accounts/bulk-import", h.bulkImport)
	r.POST("/email/accounts", h.create)
	r.GET("/email/accounts/:id", h.get)
	r.PATCH("/email/accounts/:id", h.patch)
	r.DELETE("/email/accounts/:id", h.delete)
	r.POST("/email/accounts/:id/outlook/authorize-url", h.buildAuthorizeURL)
	r.POST("/email/accounts/:id/outlook/exchange-code", h.exchangeOutlookCode)
	r.GET("/email/accounts/:id/mailboxes", h.listMailboxes)
	r.GET("/email/accounts/:id/messages", h.listMessages)
	r.GET("/email/accounts/:id/messages/latest", h.getLatestMessage)
	r.GET("/email/accounts/:id/messages/:message_id", h.getMessage)
	r.POST("/email/preview/mailboxes", h.previewMailboxes)
	r.POST("/email/preview/messages/latest", h.previewLatestMessage)
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
		httpx.BadRequest(ctx, c, "invalid email account id")
		return
	}
	item, err := h.service.Get(ctx, id)
	if err != nil {
		if errors.Is(err, emailpostgres.ErrNotFound) {
			httpx.NotFound(ctx, c, "email account not found")
			return
		}
		httpx.InternalServerError(ctx, c, err.Error())
		return
	}
	c.JSON(http.StatusOK, item)
}

func (h Handler) create(ctx context.Context, c *app.RequestContext) {
	var input emaildomain.CreateInput
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
		httpx.BadRequest(ctx, c, "invalid email account id")
		return
	}
	var input emaildomain.PatchInput
	if err := httpx.DecodeJSON(c, &input); err != nil {
		httpx.BadRequest(ctx, c, err.Error())
		return
	}
	item, err := h.service.Patch(ctx, id, input)
	if err != nil {
		if errors.Is(err, emailpostgres.ErrNotFound) {
			httpx.NotFound(ctx, c, "email account not found")
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
		httpx.BadRequest(ctx, c, "invalid email account id")
		return
	}
	if err := h.service.Delete(ctx, id); err != nil {
		if errors.Is(err, emailpostgres.ErrNotFound) {
			httpx.NotFound(ctx, c, "email account not found")
			return
		}
		httpx.InternalServerError(ctx, c, err.Error())
		return
	}
	c.JSON(http.StatusOK, map[string]any{"deleted": true})
}

func (h Handler) buildAuthorizeURL(ctx context.Context, c *app.RequestContext) {
	id, err := httpx.PathInt64(c, "id")
	if err != nil {
		httpx.BadRequest(ctx, c, "invalid email account id")
		return
	}
	result, err := h.service.BuildOutlookAuthorizeURL(ctx, id)
	if err != nil {
		if errors.Is(err, emailpostgres.ErrNotFound) {
			httpx.NotFound(ctx, c, "email account not found")
			return
		}
		httpx.BadRequest(ctx, c, err.Error())
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h Handler) exchangeOutlookCode(ctx context.Context, c *app.RequestContext) {
	id, err := httpx.PathInt64(c, "id")
	if err != nil {
		httpx.BadRequest(ctx, c, "invalid email account id")
		return
	}
	var input emaildomain.OutlookExchangeCodeInput
	if err := httpx.DecodeJSON(c, &input); err != nil {
		httpx.BadRequest(ctx, c, err.Error())
		return
	}
	item, err := h.service.ExchangeOutlookCode(ctx, id, input)
	if err != nil {
		if errors.Is(err, emailpostgres.ErrNotFound) {
			httpx.NotFound(ctx, c, "email account not found")
			return
		}
		httpx.BadRequest(ctx, c, err.Error())
		return
	}
	c.JSON(http.StatusOK, item)
}

func (h Handler) listMailboxes(ctx context.Context, c *app.RequestContext) {
	id, err := httpx.PathInt64(c, "id")
	if err != nil {
		httpx.BadRequest(ctx, c, "invalid email account id")
		return
	}
	result, err := h.service.ListMailboxes(ctx, id, emaildomain.ListMailboxesInput{
		Pattern: func() string { v, _ := c.GetQuery("pattern"); return v }(),
	})
	if err != nil {
		if errors.Is(err, emailpostgres.ErrNotFound) {
			httpx.NotFound(ctx, c, "email account not found")
			return
		}
		httpx.BadRequest(ctx, c, err.Error())
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h Handler) listMessages(ctx context.Context, c *app.RequestContext) {
	id, err := httpx.PathInt64(c, "id")
	if err != nil {
		httpx.BadRequest(ctx, c, "invalid email account id")
		return
	}
	limit := 20
	if raw, ok := c.GetQuery("limit"); ok {
		if v, err := strconv.Atoi(string(raw)); err == nil {
			limit = v
		}
	}
	offset := 0
	if raw, ok := c.GetQuery("offset"); ok {
		if v, err := strconv.Atoi(string(raw)); err == nil {
			offset = v
		}
	}
	result, err := h.service.ListMessages(ctx, id, emaildomain.ListMessagesInput{
		Mailbox: func() string { v, _ := c.GetQuery("mailbox"); return v }(),
		Limit:   limit,
		Offset:  offset,
	})
	if err != nil {
		if errors.Is(err, emailpostgres.ErrNotFound) {
			httpx.NotFound(ctx, c, "email account not found")
			return
		}
		httpx.BadRequest(ctx, c, err.Error())
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h Handler) getLatestMessage(ctx context.Context, c *app.RequestContext) {
	id, err := httpx.PathInt64(c, "id")
	if err != nil {
		httpx.BadRequest(ctx, c, "invalid email account id")
		return
	}
	result, err := h.service.GetLatestMessage(ctx, id, emaildomain.ListMessagesInput{
		Mailbox: func() string { v, _ := c.GetQuery("mailbox"); return v }(),
	})
	if err != nil {
		if errors.Is(err, emailpostgres.ErrNotFound) {
			httpx.NotFound(ctx, c, "email account not found")
			return
		}
		httpx.BadRequest(ctx, c, err.Error())
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h Handler) getMessage(ctx context.Context, c *app.RequestContext) {
	id, err := httpx.PathInt64(c, "id")
	if err != nil {
		httpx.BadRequest(ctx, c, "invalid email account id")
		return
	}
	item, err := h.service.GetMessage(ctx, id, c.Param("message_id"))
	if err != nil {
		if errors.Is(err, emailpostgres.ErrNotFound) {
			httpx.NotFound(ctx, c, "email account not found")
			return
		}
		httpx.BadRequest(ctx, c, err.Error())
		return
	}
	c.JSON(http.StatusOK, item)
}

func (h Handler) previewMailboxes(ctx context.Context, c *app.RequestContext) {
	var input emaildomain.PreviewInput
	if err := httpx.DecodeJSON(c, &input); err != nil {
		httpx.BadRequest(ctx, c, err.Error())
		return
	}
	result, err := h.service.PreviewMailboxes(ctx, input)
	if err != nil {
		httpx.BadRequest(ctx, c, err.Error())
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h Handler) bulkImport(ctx context.Context, c *app.RequestContext) {
	var input emaildomain.BulkImportInput
	if err := httpx.DecodeJSON(c, &input); err != nil {
		httpx.BadRequest(ctx, c, err.Error())
		return
	}
	result, err := h.service.BulkImport(ctx, input)
	if err != nil {
		httpx.InternalServerError(ctx, c, err.Error())
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h Handler) previewLatestMessage(ctx context.Context, c *app.RequestContext) {
	var input emaildomain.PreviewInput
	if err := httpx.DecodeJSON(c, &input); err != nil {
		httpx.BadRequest(ctx, c, err.Error())
		return
	}
	result, err := h.service.PreviewLatestMessage(ctx, input)
	if err != nil {
		httpx.BadRequest(ctx, c, err.Error())
		return
	}
	c.JSON(http.StatusOK, result)
}
