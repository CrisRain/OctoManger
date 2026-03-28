package httpx

import (
	"testing"

	"github.com/cloudwego/hertz/pkg/app"
)

func TestProblemHelpers(t *testing.T) {
	ctx := &app.RequestContext{}
	NotFound(nil, ctx, "missing")
	if ctx.Response.StatusCode() != 404 {
		t.Fatalf("expected 404, got %d", ctx.Response.StatusCode())
	}

	ctx = &app.RequestContext{}
	BadRequest(nil, ctx, "bad")
	if ctx.Response.StatusCode() != 400 {
		t.Fatalf("expected 400, got %d", ctx.Response.StatusCode())
	}

	ctx = &app.RequestContext{}
	Unauthorized(nil, ctx, "nope")
	if ctx.Response.StatusCode() != 401 {
		t.Fatalf("expected 401, got %d", ctx.Response.StatusCode())
	}

	ctx = &app.RequestContext{}
	TooManyRequests(nil, ctx, "slow")
	if ctx.Response.StatusCode() != 429 {
		t.Fatalf("expected 429, got %d", ctx.Response.StatusCode())
	}

	ctx = &app.RequestContext{}
	InternalServerError(nil, ctx, "db failure")
	if ctx.Response.StatusCode() != 500 {
		t.Fatalf("expected 500, got %d", ctx.Response.StatusCode())
	}
}
