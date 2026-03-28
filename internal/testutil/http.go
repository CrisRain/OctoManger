package testutil

import (
	"encoding/json"
	"testing"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/route/param"
)

// NewJSONRequestContext builds a Hertz RequestContext with a JSON body.
func NewJSONRequestContext(method, path string, body any) *app.RequestContext {
	ctx := &app.RequestContext{}
	ctx.Request.SetMethod(method)
	ctx.Request.SetRequestURI(path)
	if body != nil {
		payload, _ := json.Marshal(body)
		ctx.Request.SetBody(payload)
		ctx.Request.Header.SetContentTypeBytes([]byte("application/json"))
	}
	return ctx
}

// SetPathParam sets a path parameter on the request context.
func SetPathParam(ctx *app.RequestContext, key, value string) {
	ctx.Params = append(ctx.Params, param.Param{Key: key, Value: value})
}

// DecodeJSONResponse decodes the JSON response body into target.
func DecodeJSONResponse(t *testing.T, ctx *app.RequestContext, target any) {
	t.Helper()
	if err := json.Unmarshal(ctx.Response.Body(), target); err != nil {
		t.Fatalf("decode response body: %v", err)
	}
}
