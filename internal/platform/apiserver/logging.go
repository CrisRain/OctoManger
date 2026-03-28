package apiserver

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"go.uber.org/zap"
)

func RequestLoggingMiddleware(logger *zap.Logger) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		startedAt := time.Now()
		c.Next(ctx)

		method := string(c.Method())
		path := string(c.Path())
		if path == "/healthz" || path == "/api/v2/system/logs" {
			return
		}

		summary := fmt.Sprintf(
			"%s %s -> %d (%.2fms)",
			method,
			path,
			c.Response.StatusCode(),
			float64(time.Since(startedAt).Microseconds())/1000.0,
		)

		switch status := c.Response.StatusCode(); {
		case status >= 500:
			logger.Error(summary)
		case status >= 400:
			logger.Warn(summary)
		case strings.HasPrefix(path, "/api/v1/octo-modules/internal/"):
			logger.Debug(summary)
		case isReadOnlyMethod(method):
			logger.Debug(summary)
		default:
			logger.Info(summary)
		}
	}
}

func isReadOnlyMethod(method string) bool {
	switch strings.ToUpper(strings.TrimSpace(method)) {
	case "GET", "HEAD", "OPTIONS":
		return true
	default:
		return false
	}
}
