package apiserver

import (
	"context"
	"crypto/subtle"
	"errors"
	"net/http"
	"strings"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/route"

	accountapp "octomanger/internal/domains/accounts/app"
	accountdomain "octomanger/internal/domains/accounts/domain"
	accountpostgres "octomanger/internal/domains/accounts/infra/postgres"
	emailapp "octomanger/internal/domains/email/app"
	emaildomain "octomanger/internal/domains/email/domain"
	emailpostgres "octomanger/internal/domains/email/infra/postgres"
	"octomanger/internal/platform/httpx"
)

type pluginInternalHandler struct {
	token    string
	accounts accountapp.Service
	email    emailapp.Service
}

func registerPluginInternalAPI(
	root *route.RouterGroup,
	token string,
	accounts accountapp.Service,
	email emailapp.Service,
) {
	handler := pluginInternalHandler{
		token:    strings.TrimSpace(token),
		accounts: accounts,
		email:    email,
	}

	v1 := root.Group("/api/v1/octo-modules/internal")
	guard := handler.requireAccess()
	v1.GET("/accounts/:id", guard, handler.getAccount)
	v1.GET("/accounts/by-identifier", guard, handler.getAccountByIdentifier)
	v1.PATCH("/accounts/:id/spec", guard, handler.patchAccountSpec)
	v1.GET("/email/accounts/:id/messages/latest", guard, handler.getLatestEmailMessage)
}

func (h pluginInternalHandler) requireAccess() app.HandlerFunc {
	if h.token == "" {
		return func(ctx context.Context, c *app.RequestContext) {
			writePluginInternalError(c, http.StatusUnauthorized, "internal api key is not configured")
			c.Abort()
		}
	}

	return func(ctx context.Context, c *app.RequestContext) {
		provided := strings.TrimSpace(string(c.GetHeader("X-Api-Key")))
		if provided == "" {
			provided = strings.TrimSpace(string(c.GetHeader("X-Admin-Key")))
		}
		if provided == "" {
			authHeader := strings.TrimSpace(string(c.GetHeader("Authorization")))
			provided = strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))
		}

		if subtle.ConstantTimeCompare([]byte(provided), []byte(h.token)) != 1 {
			writePluginInternalError(c, http.StatusUnauthorized, "unauthorized")
			c.Abort()
			return
		}

		c.Next(ctx)
	}
}

func (h pluginInternalHandler) getAccount(ctx context.Context, c *app.RequestContext) {
	id, err := httpx.PathInt64(c, "id")
	if err != nil {
		writePluginInternalError(c, http.StatusBadRequest, "invalid account id")
		return
	}

	account, err := h.accounts.Get(ctx, id)
	if err != nil {
		if errors.Is(err, accountpostgres.ErrNotFound) {
			writePluginInternalError(c, http.StatusNotFound, "account not found")
			return
		}
		writePluginInternalError(c, http.StatusInternalServerError, "internal server error")
		return
	}

	writePluginInternalOK(c, account)
}

func (h pluginInternalHandler) getAccountByIdentifier(ctx context.Context, c *app.RequestContext) {
	typeKey, _ := c.GetQuery("type_key")
	identifier, _ := c.GetQuery("identifier")

	account, err := h.accounts.GetByTypeKeyAndIdentifier(ctx, typeKey, identifier)
	if err != nil {
		if errors.Is(err, accountpostgres.ErrNotFound) {
			writePluginInternalError(c, http.StatusNotFound, "account not found")
			return
		}
		writePluginInternalError(c, http.StatusInternalServerError, "internal server error")
		return
	}

	writePluginInternalOK(c, account)
}

func (h pluginInternalHandler) patchAccountSpec(ctx context.Context, c *app.RequestContext) {
	id, err := httpx.PathInt64(c, "id")
	if err != nil {
		writePluginInternalError(c, http.StatusBadRequest, "invalid account id")
		return
	}

	var body struct {
		Spec map[string]any `json:"spec"`
	}
	if err := c.BindJSON(&body); err != nil {
		writePluginInternalError(c, http.StatusBadRequest, "invalid request body")
		return
	}
	if body.Spec == nil {
		body.Spec = map[string]any{}
	}

	account, err := h.accounts.Patch(ctx, id, accountdomain.PatchInput{
		Spec: body.Spec,
	})
	if err != nil {
		if errors.Is(err, accountpostgres.ErrNotFound) {
			writePluginInternalError(c, http.StatusNotFound, "account not found")
			return
		}
		writePluginInternalError(c, http.StatusInternalServerError, "internal server error")
		return
	}

	writePluginInternalOK(c, account)
}

func (h pluginInternalHandler) getLatestEmailMessage(ctx context.Context, c *app.RequestContext) {
	id, err := httpx.PathInt64(c, "id")
	if err != nil {
		writePluginInternalError(c, http.StatusBadRequest, "invalid email account id")
		return
	}

	mailbox, _ := c.GetQuery("mailbox")
	result, err := h.email.GetLatestMessage(ctx, id, emaildomain.ListMessagesInput{
		Mailbox: mailbox,
	})
	if err != nil {
		if errors.Is(err, emailpostgres.ErrNotFound) {
			writePluginInternalError(c, http.StatusNotFound, "email account not found")
			return
		}
		writePluginInternalError(c, http.StatusInternalServerError, "internal server error")
		return
	}

	writePluginInternalOK(c, result)
}

func writePluginInternalOK(c *app.RequestContext, data any) {
	c.JSON(http.StatusOK, map[string]any{
		"code":    0,
		"message": "ok",
		"data":    data,
	})
}

func writePluginInternalError(c *app.RequestContext, status int, message string) {
	c.JSON(status, map[string]any{
		"code":    status,
		"message": strings.TrimSpace(message),
	})
}
