package accountstransport

import (
	"context"
	"testing"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/route/param"
	"gorm.io/gorm"

	accountapp "octomanger/internal/domains/accounts/app"
	accountdomain "octomanger/internal/domains/accounts/domain"
	accountpostgres "octomanger/internal/domains/accounts/infra/postgres"
	pluginapp "octomanger/internal/domains/plugins/app"
	plugindomain "octomanger/internal/domains/plugins/domain"
	"octomanger/internal/platform/database"
	"octomanger/internal/testutil"
)

type stubPluginService struct {
	execFn func(context.Context, string, plugindomain.ExecutionRequest, func(plugindomain.ExecutionEvent)) error
}

func (s stubPluginService) Execute(ctx context.Context, key string, req plugindomain.ExecutionRequest, onEvent func(plugindomain.ExecutionEvent)) error {
	if s.execFn != nil {
		return s.execFn(ctx, key, req, onEvent)
	}
	return nil
}
func (s stubPluginService) List(context.Context) ([]plugindomain.Plugin, error) { return nil, nil }
func (s stubPluginService) Get(context.Context, string) (*plugindomain.Plugin, error) {
	return nil, nil
}
func (s stubPluginService) SyncAccountTypes(context.Context, pluginapp.SyncAccountTypeFunc) error {
	return nil
}

func seedAccountType(t *testing.T, db *gorm.DB) int64 {
	t.Helper()
	record := database.AccountTypeModel{
		Key:              "demo",
		Name:             "demo",
		Category:         "generic",
		SchemaJSON:       database.JSONBytes([]byte("{}")),
		CapabilitiesJSON: database.JSONBytes([]byte("{}")),
	}
	if err := db.Create(&record).Error; err != nil {
		t.Fatalf("seed account type: %v", err)
	}
	return record.ID
}

func TestAccountHandlerCreateListGetDelete(t *testing.T) {
	db := testutil.NewTestDB(t)
	accountTypeID := seedAccountType(t, db)
	svc := accountapp.New(accountpostgres.New(db), stubPluginService{})
	h := NewHandler(svc)

	ctx := &app.RequestContext{}
	ctx.Request.SetBody([]byte("not-json"))
	ctx.Request.Header.SetContentTypeBytes([]byte("application/json"))
	h.create(context.Background(), ctx)
	if ctx.Response.StatusCode() != 400 {
		t.Fatalf("expected 400, got %d", ctx.Response.StatusCode())
	}

	createCtx := testutil.NewJSONRequestContext("POST", "/accounts", accountdomain.CreateInput{
		AccountTypeID: accountTypeID,
		Identifier:    "acct-1",
		Status:        "active",
		Tags:          []string{},
		Spec:          map[string]any{},
	})
	h.create(context.Background(), createCtx)
	if createCtx.Response.StatusCode() != 201 {
		t.Fatalf("expected 201, got %d", createCtx.Response.StatusCode())
	}

	listCtx := &app.RequestContext{}
	listCtx.Request.SetRequestURI("/accounts?limit=10")
	h.list(context.Background(), listCtx)
	if listCtx.Response.StatusCode() != 200 {
		t.Fatalf("expected 200, got %d", listCtx.Response.StatusCode())
	}

	getCtx := &app.RequestContext{}
	getCtx.Params = append(getCtx.Params, param.Param{Key: "id", Value: "bad"})
	h.get(context.Background(), getCtx)
	if getCtx.Response.StatusCode() != 400 {
		t.Fatalf("expected 400, got %d", getCtx.Response.StatusCode())
	}

	missingCtx := &app.RequestContext{}
	missingCtx.Params = append(missingCtx.Params, param.Param{Key: "id", Value: "999"})
	h.get(context.Background(), missingCtx)
	if missingCtx.Response.StatusCode() != 404 {
		t.Fatalf("expected 404, got %d", missingCtx.Response.StatusCode())
	}
}

func TestAccountHandlerPatchDeleteExecuteErrors(t *testing.T) {
	db := testutil.NewTestDB(t)
	accountTypeID := seedAccountType(t, db)
	svc := accountapp.New(accountpostgres.New(db), stubPluginService{})
	h := NewHandler(svc)

	patchCtx := &app.RequestContext{}
	patchCtx.Request.SetBody([]byte("not-json"))
	patchCtx.Request.Header.SetContentTypeBytes([]byte("application/json"))
	h.patch(context.Background(), patchCtx)
	if patchCtx.Response.StatusCode() != 400 {
		t.Fatalf("expected 400, got %d", patchCtx.Response.StatusCode())
	}

	patchCtx = testutil.NewJSONRequestContext("PATCH", "/accounts/999", accountdomain.PatchInput{})
	patchCtx.Params = append(patchCtx.Params, param.Param{Key: "id", Value: "999"})
	h.patch(context.Background(), patchCtx)
	if patchCtx.Response.StatusCode() != 404 {
		t.Fatalf("expected 404, got %d", patchCtx.Response.StatusCode())
	}

	deleteCtx := &app.RequestContext{}
	deleteCtx.Params = append(deleteCtx.Params, param.Param{Key: "id", Value: "999"})
	h.delete(context.Background(), deleteCtx)
	if deleteCtx.Response.StatusCode() != 404 {
		t.Fatalf("expected 404, got %d", deleteCtx.Response.StatusCode())
	}

	execCtx := testutil.NewJSONRequestContext("POST", "/accounts/999/execute", map[string]any{})
	execCtx.Params = append(execCtx.Params, param.Param{Key: "id", Value: "999"})
	h.execute(context.Background(), execCtx)
	if execCtx.Response.StatusCode() != 400 {
		t.Fatalf("expected 400 for missing action, got %d", execCtx.Response.StatusCode())
	}

	execCtx = testutil.NewJSONRequestContext("POST", "/accounts/999/execute", map[string]any{"action": "LIST"})
	execCtx.Params = append(execCtx.Params, param.Param{Key: "id", Value: "999"})
	h.execute(context.Background(), execCtx)
	if execCtx.Response.StatusCode() != 404 {
		t.Fatalf("expected 404, got %d", execCtx.Response.StatusCode())
	}

	createCtx := testutil.NewJSONRequestContext("POST", "/accounts", accountdomain.CreateInput{
		AccountTypeID: accountTypeID,
		Identifier:    "acct-2",
		Status:        "active",
	})
	h.create(context.Background(), createCtx)
	if createCtx.Response.StatusCode() != 201 {
		t.Fatalf("expected 201, got %d", createCtx.Response.StatusCode())
	}

	execSvc := accountapp.New(accountpostgres.New(db), stubPluginService{execFn: func(ctx context.Context, key string, req plugindomain.ExecutionRequest, onEvent func(plugindomain.ExecutionEvent)) error {
		if onEvent != nil {
			onEvent(plugindomain.ExecutionEvent{Type: "result", Data: map[string]any{"ok": true}})
		}
		return nil
	}})
	execHandler := NewHandler(execSvc)

	execCtx = testutil.NewJSONRequestContext("POST", "/accounts/1/execute", map[string]any{"action": "LIST"})
	execCtx.Params = append(execCtx.Params, param.Param{Key: "id", Value: "1"})
	execHandler.execute(context.Background(), execCtx)
	if execCtx.Response.StatusCode() != 200 {
		t.Fatalf("expected 200, got %d", execCtx.Response.StatusCode())
	}
}

func TestAccountHandlerPluginBackendMissing(t *testing.T) {
	db := testutil.NewTestDB(t)
	accountTypeID := seedAccountType(t, db)
	repo := accountpostgres.New(db)
	_, err := repo.Create(context.Background(), accountdomain.CreateInput{AccountTypeID: accountTypeID, Identifier: "acct-3", Status: "active"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	svc := accountapp.New(repo)
	h := NewHandler(svc)

	execCtx := testutil.NewJSONRequestContext("POST", "/accounts/1/execute", map[string]any{"action": "LIST"})
	execCtx.Params = append(execCtx.Params, param.Param{Key: "id", Value: "1"})
	h.execute(context.Background(), execCtx)
	if execCtx.Response.StatusCode() != 400 {
		t.Fatalf("expected 400, got %d", execCtx.Response.StatusCode())
	}
}

func TestAccountHandlerInternalError(t *testing.T) {
	// Use a repo backed by a closed DB to force an unexpected error.
	db := testutil.NewTestDB(t)
	sqlDB, _ := db.DB()
	_ = sqlDB.Close()
	repo := accountpostgres.New(db)
	svc := accountapp.New(repo, stubPluginService{})
	h := NewHandler(svc)

	execCtx := testutil.NewJSONRequestContext("POST", "/accounts/1/execute", map[string]any{"action": "LIST"})
	execCtx.Params = append(execCtx.Params, param.Param{Key: "id", Value: "1"})
	h.execute(context.Background(), execCtx)
	if execCtx.Response.StatusCode() != 500 {
		t.Fatalf("expected 500, got %d", execCtx.Response.StatusCode())
	}

	getCtx := &app.RequestContext{}
	getCtx.Params = append(getCtx.Params, param.Param{Key: "id", Value: "1"})
	h.get(context.Background(), getCtx)
	if getCtx.Response.StatusCode() != 500 {
		t.Fatalf("expected 500, got %d", getCtx.Response.StatusCode())
	}

	deleteCtx := &app.RequestContext{}
	deleteCtx.Params = append(deleteCtx.Params, param.Param{Key: "id", Value: "1"})
	h.delete(context.Background(), deleteCtx)
	if deleteCtx.Response.StatusCode() != 500 {
		t.Fatalf("expected 500, got %d", deleteCtx.Response.StatusCode())
	}
}
