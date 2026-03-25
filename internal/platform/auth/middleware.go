package auth

import (
	"context"
	"net/http"
	"strings"

	"github.com/cloudwego/hertz/pkg/app"

	"octomanger/internal/platform/httpx"
)

// RequireAdmin returns a Hertz HandlerFunc that validates the admin key.
// If adminKey is empty, the middleware is a no-op (dev mode).
func RequireAdmin(adminKey string) app.HandlerFunc {
	trimmedKey := strings.TrimSpace(adminKey)
	if trimmedKey == "" {
		return func(ctx context.Context, c *app.RequestContext) {
			c.Next(ctx)
		}
	}

	return func(ctx context.Context, c *app.RequestContext) {
		providedKey := strings.TrimSpace(string(c.GetHeader("X-Admin-Key")))
		if providedKey == "" {
			authHeader := strings.TrimSpace(string(c.GetHeader("Authorization")))
			providedKey = strings.TrimPrefix(authHeader, "Bearer ")
			providedKey = strings.TrimSpace(providedKey)
		}

		if providedKey != trimmedKey {
			httpx.Unauthorized(ctx, c, "missing or invalid admin key")
			c.Abort()
			return
		}

		c.Next(ctx)
	}
}

// RequireAdminForRouter returns Hertz middleware suitable for router.Use().
func RequireAdminForRouter(adminKey string) app.HandlerFunc {
	return RequireAdmin(adminKey)
}

// StatusUnauthorized is kept as a named constant for clarity.
const StatusUnauthorized = http.StatusUnauthorized
