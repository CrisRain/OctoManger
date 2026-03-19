package httpx

import (
	"context"
	"fmt"
	"net/http"

	"github.com/cloudwego/hertz/pkg/app"
)

type Problem struct {
	Type     string         `json:"type"`
	Title    string         `json:"title"`
	Status   int            `json:"status"`
	Detail   string         `json:"detail,omitempty"`
	Instance string         `json:"instance,omitempty"`
	Errors   map[string]any `json:"errors,omitempty"`
}

func WriteProblem(ctx context.Context, c *app.RequestContext, status int, title, detail string) {
	problem := Problem{
		Type:   fmt.Sprintf("https://docs.octomanger.dev/problems/%d", status),
		Title:  title,
		Status: status,
		Detail: detail,
	}
	c.JSON(status, problem)
}

func NotFound(ctx context.Context, c *app.RequestContext, detail string) {
	WriteProblem(ctx, c, http.StatusNotFound, "not_found", detail)
}

func BadRequest(ctx context.Context, c *app.RequestContext, detail string) {
	WriteProblem(ctx, c, http.StatusBadRequest, "bad_request", detail)
}

func Unauthorized(ctx context.Context, c *app.RequestContext, detail string) {
	WriteProblem(ctx, c, http.StatusUnauthorized, "unauthorized", detail)
}

func InternalServerError(ctx context.Context, c *app.RequestContext, detail string) {
	WriteProblem(ctx, c, http.StatusInternalServerError, "internal_server_error", detail)
}
