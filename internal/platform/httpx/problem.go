package httpx

import (
	"context"
	"net/http"
	"strings"

	"github.com/cloudwego/hertz/pkg/app"
)

type Problem struct {
	Title    string         `json:"title"`
	Status   int            `json:"status"`
	Detail   string         `json:"detail,omitempty"`
	Instance string         `json:"instance,omitempty"`
	Errors   map[string]any `json:"errors,omitempty"`
}

func WriteProblem(ctx context.Context, c *app.RequestContext, status int, title, detail string) {
	problem := Problem{
		Title:  title,
		Status: status,
		Detail: detail,
	}
	c.JSON(status, problem)
}

func NotFound(ctx context.Context, c *app.RequestContext, detail string) {
	WriteProblem(ctx, c, http.StatusNotFound, "not_found", sanitizeClientDetail(detail))
}

func BadRequest(ctx context.Context, c *app.RequestContext, detail string) {
	WriteProblem(ctx, c, http.StatusBadRequest, "bad_request", sanitizeClientDetail(detail))
}

func Unauthorized(ctx context.Context, c *app.RequestContext, detail string) {
	WriteProblem(ctx, c, http.StatusUnauthorized, "unauthorized", detail)
}

func TooManyRequests(ctx context.Context, c *app.RequestContext, detail string) {
	WriteProblem(ctx, c, http.StatusTooManyRequests, "too_many_requests", detail)
}

func InternalServerError(ctx context.Context, c *app.RequestContext, detail string) {
	_ = detail
	WriteProblem(ctx, c, http.StatusInternalServerError, "internal_server_error", "internal server error")
}

func sanitizeClientDetail(detail string) string {
	detail = strings.TrimSpace(detail)
	if detail == "" {
		return "invalid request"
	}

	lower := strings.ToLower(detail)
	sensitivePatterns := []string{
		"select ",
		"insert ",
		"update ",
		"delete ",
		"drop table",
		"gorm",
		"postgres",
		"sqlstate",
		"constraint",
		"duplicate key",
		"connection refused",
		"dial tcp",
	}
	for _, pattern := range sensitivePatterns {
		if strings.Contains(lower, pattern) {
			return "invalid request"
		}
	}

	if len(detail) > 280 {
		return "invalid request"
	}
	return detail
}
