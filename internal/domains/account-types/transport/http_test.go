package accounttypestransport

import (
	"context"
	"testing"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/route/param"

	accounttypeapp "octomanger/internal/domains/account-types/app"
	accounttypedomain "octomanger/internal/domains/account-types/domain"
	accounttypepostgres "octomanger/internal/domains/account-types/infra/postgres"
	"octomanger/internal/testutil"
)

func TestAccountTypeHandlerListAndCreate(t *testing.T) {
	db := testutil.NewTestDB(t)
	svc := accounttypeapp.New(accounttypepostgres.New(db))
	h := NewHandler(svc)

	ctx := testutil.NewJSONRequestContext("POST", "/account-types", accounttypedomain.CreateInput{Key: "demo", Name: "Demo"})
	h.create(context.Background(), ctx)
	if ctx.Response.StatusCode() != 201 {
		t.Fatalf("expected 201, got %d", ctx.Response.StatusCode())
	}

	listCtx := &app.RequestContext{}
	listCtx.Request.SetRequestURI("/account-types?limit=10")
	h.list(context.Background(), listCtx)
	if listCtx.Response.StatusCode() != 200 {
		t.Fatalf("expected 200, got %d", listCtx.Response.StatusCode())
	}
}

func TestAccountTypeHandlerErrors(t *testing.T) {
	db := testutil.NewTestDB(t)
	svc := accounttypeapp.New(accounttypepostgres.New(db))
	h := NewHandler(svc)

	ctx := &app.RequestContext{}
	ctx.Request.SetBody([]byte("not-json"))
	ctx.Request.Header.SetContentTypeBytes([]byte("application/json"))
	h.create(context.Background(), ctx)
	if ctx.Response.StatusCode() != 400 {
		t.Fatalf("expected 400, got %d", ctx.Response.StatusCode())
	}

	getCtx := &app.RequestContext{}
	getCtx.Params = append(getCtx.Params, param.Param{Key: "key", Value: "missing"})
	h.get(context.Background(), getCtx)
	if getCtx.Response.StatusCode() != 404 {
		t.Fatalf("expected 404, got %d", getCtx.Response.StatusCode())
	}

	patchCtx := &app.RequestContext{}
	patchCtx.Request.SetBody([]byte("not-json"))
	patchCtx.Request.Header.SetContentTypeBytes([]byte("application/json"))
	h.patch(context.Background(), patchCtx)
	if patchCtx.Response.StatusCode() != 400 {
		t.Fatalf("expected 400, got %d", patchCtx.Response.StatusCode())
	}

	patchCtx = testutil.NewJSONRequestContext("PATCH", "/account-types/missing", accounttypedomain.PatchInput{})
	patchCtx.Params = append(patchCtx.Params, param.Param{Key: "key", Value: "missing"})
	h.patch(context.Background(), patchCtx)
	if patchCtx.Response.StatusCode() != 404 {
		t.Fatalf("expected 404, got %d", patchCtx.Response.StatusCode())
	}

	deleteCtx := &app.RequestContext{}
	deleteCtx.Params = append(deleteCtx.Params, param.Param{Key: "key", Value: "missing"})
	h.delete(context.Background(), deleteCtx)
	if deleteCtx.Response.StatusCode() != 404 {
		t.Fatalf("expected 404, got %d", deleteCtx.Response.StatusCode())
	}
}
