package apiserver

import (
	"context"
	"net/http"
	"testing"

	"github.com/cloudwego/hertz/pkg/app"
)

func TestPluginInternalRequireAccessRejectsWhenTokenMissing(t *testing.T) {
	h := pluginInternalHandler{}
	mw := h.requireAccess()

	c := &app.RequestContext{}
	c.Request.SetRequestURI("/api/v1/octo-modules/internal/accounts/1")

	mw(context.Background(), c)

	if c.Response.StatusCode() != http.StatusUnauthorized {
		t.Fatalf("unexpected status code: %d", c.Response.StatusCode())
	}
}

func TestPluginInternalRequireAccessAcceptsXAPIKey(t *testing.T) {
	h := pluginInternalHandler{token: "internal-token"}
	mw := h.requireAccess()

	c := &app.RequestContext{}
	c.Request.SetRequestURI("/api/v1/octo-modules/internal/accounts/1")
	c.Request.Header.Set("X-Api-Key", "internal-token")

	mw(context.Background(), c)

	if c.Response.StatusCode() != http.StatusOK {
		t.Fatalf("expected successful middleware pass-through, got status=%d", c.Response.StatusCode())
	}
}

func TestPluginInternalRequireAccessAcceptsBearerFallback(t *testing.T) {
	h := pluginInternalHandler{token: "internal-token"}
	mw := h.requireAccess()

	c := &app.RequestContext{}
	c.Request.SetRequestURI("/api/v1/octo-modules/internal/accounts/1")
	c.Request.Header.Set("Authorization", "Bearer internal-token")

	mw(context.Background(), c)

	if c.Response.StatusCode() != http.StatusOK {
		t.Fatalf("expected successful middleware pass-through, got status=%d", c.Response.StatusCode())
	}
}
