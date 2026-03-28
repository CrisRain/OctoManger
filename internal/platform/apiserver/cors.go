package apiserver

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	"github.com/cloudwego/hertz/pkg/app"

	"octomanger/internal/platform/config"
)

var defaultCORSMethods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"}

func CORSMiddleware(corsCfg config.CORSConfig) app.HandlerFunc {
	allowedOrigins := normalizeOrigins(corsCfg.AllowedOrigins)
	allowedMethods := joinHeaderValues(corsCfg.AllowedMethods, defaultCORSMethods)
	allowedHeaders := joinHeaderValues(corsCfg.AllowedHeaders, nil)
	maxAgeSeconds := int64(corsCfg.MaxAge.Seconds())
	if maxAgeSeconds < 0 {
		maxAgeSeconds = 0
	}

	return func(ctx context.Context, c *app.RequestContext) {
		origin := normalizeOrigin(strings.TrimSpace(string(c.GetHeader("Origin"))))
		if origin == "" {
			c.Next(ctx)
			return
		}

		allowOrigin := matchAllowedOrigin(origin, allowedOrigins)
		if allowOrigin == "" {
			if string(c.Method()) == http.MethodOptions {
				c.Status(http.StatusForbidden)
				c.Abort()
				return
			}
			c.Next(ctx)
			return
		}

		c.Header("Vary", "Origin")
		c.Header("Access-Control-Allow-Origin", allowOrigin)
		if allowedMethods != "" {
			c.Header("Access-Control-Allow-Methods", allowedMethods)
		}
		if allowedHeaders != "" {
			c.Header("Access-Control-Allow-Headers", allowedHeaders)
		}
		if maxAgeSeconds > 0 {
			c.Header("Access-Control-Max-Age", strconv.FormatInt(maxAgeSeconds, 10))
		}

		if string(c.Method()) == http.MethodOptions {
			c.Status(http.StatusNoContent)
			c.Abort()
			return
		}
		c.Next(ctx)
	}
}

func normalizeOrigins(origins []string) []string {
	normalized := make([]string, 0, len(origins))
	seen := make(map[string]struct{}, len(origins))
	for _, origin := range origins {
		item := normalizeOrigin(origin)
		if item == "" {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		normalized = append(normalized, item)
	}
	return normalized
}

func normalizeOrigin(origin string) string {
	trimmed := strings.TrimSpace(origin)
	if trimmed == "" {
		return ""
	}
	return strings.TrimSuffix(trimmed, "/")
}

func matchAllowedOrigin(origin string, allowed []string) string {
	for _, candidate := range allowed {
		if candidate == "*" {
			return "*"
		}
		if strings.EqualFold(candidate, origin) {
			return origin
		}
	}
	return ""
}

func joinHeaderValues(values []string, fallback []string) string {
	selected := values
	if len(selected) == 0 {
		selected = fallback
	}
	if len(selected) == 0 {
		return ""
	}
	normalized := make([]string, 0, len(selected))
	seen := make(map[string]struct{}, len(selected))
	for _, item := range selected {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" {
			continue
		}
		key := strings.ToLower(trimmed)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		normalized = append(normalized, trimmed)
	}
	return strings.Join(normalized, ", ")
}
