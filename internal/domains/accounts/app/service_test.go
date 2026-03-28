package accountapp

import (
	"context"
	"errors"
	"testing"

	"gorm.io/gorm"

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

func (s stubPluginService) Execute(ctx context.Context, pluginKey string, req plugindomain.ExecutionRequest, onEvent func(plugindomain.ExecutionEvent)) error {
	if s.execFn != nil {
		return s.execFn(ctx, pluginKey, req, onEvent)
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

func seedAccountType(t *testing.T, db *gorm.DB, key string) int64 {
	t.Helper()
	record := database.AccountTypeModel{
		Key:              key,
		Name:             key,
		Category:         "generic",
		SchemaJSON:       database.JSONBytes([]byte("{}")),
		CapabilitiesJSON: database.JSONBytes([]byte("{}")),
	}
	if err := db.Create(&record).Error; err != nil {
		t.Fatalf("seed account type: %v", err)
	}
	return record.ID
}

func TestServiceCreateAndLookup(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := accountpostgres.New(db)
	svc := New(repo)
	ctx := context.Background()

	if _, err := svc.Create(ctx, accountdomain.CreateInput{Identifier: ""}); err == nil {
		t.Fatalf("expected identifier validation error")
	}

	created, err := svc.Create(ctx, accountdomain.CreateInput{Identifier: "acct-1"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if created.Status != accountdomain.StatusPending {
		t.Fatalf("expected pending status, got %q", created.Status)
	}

	if _, err := svc.GetByTypeKeyAndIdentifier(ctx, "", ""); !errors.Is(err, accountpostgres.ErrNotFound) {
		t.Fatalf("expected not found for empty identifiers")
	}
}

func TestServiceSetStatusEmptyUsesExisting(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := accountpostgres.New(db)
	svc := New(repo)
	ctx := context.Background()

	created, err := svc.Create(ctx, accountdomain.CreateInput{Identifier: "acct-2"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	updated, err := svc.SetStatus(ctx, created.ID, "")
	if err != nil {
		t.Fatalf("set status: %v", err)
	}
	if updated.Status != created.Status {
		t.Fatalf("expected status unchanged")
	}
}

func TestServiceExecuteActionBranches(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := accountpostgres.New(db)
	ctx := context.Background()

	accountTypeID := seedAccountType(t, db, "demo")

	account, err := repo.Create(ctx, accountdomain.CreateInput{AccountTypeID: accountTypeID, Identifier: "acct-3", Status: "active"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	noType, err := repo.Create(ctx, accountdomain.CreateInput{Identifier: "acct-4", Status: "active"})
	if err != nil {
		t.Fatalf("create no type: %v", err)
	}

	svcMissingPlugin := New(repo)
	if _, err := svcMissingPlugin.ExecuteAction(ctx, account.ID, "LIST", nil); !errors.Is(err, ErrPluginBackendMissing) {
		t.Fatalf("expected plugin backend missing, got %v", err)
	}

	svc := New(repo, stubPluginService{})
	if _, err := svc.ExecuteAction(ctx, account.ID, "", nil); !errors.Is(err, ErrInvalidExecuteAction) {
		t.Fatalf("expected invalid action, got %v", err)
	}
	if _, err := svc.ExecuteAction(ctx, 999, "LIST", nil); !errors.Is(err, accountpostgres.ErrNotFound) {
		t.Fatalf("expected not found, got %v", err)
	}
	if _, err := svc.ExecuteAction(ctx, noType.ID, "LIST", nil); !errors.Is(err, ErrMissingAccountTypeKey) {
		t.Fatalf("expected missing account type key, got %v", err)
	}

	svc = New(repo, stubPluginService{execFn: func(ctx context.Context, key string, req plugindomain.ExecutionRequest, onEvent func(plugindomain.ExecutionEvent)) error {
		if onEvent != nil {
			onEvent(plugindomain.ExecutionEvent{Type: "error", Error: "E", Message: "bad"})
		}
		return nil
	}})
	result, err := svc.ExecuteAction(ctx, account.ID, "VERIFY", nil)
	if err != nil {
		t.Fatalf("execute error event: %v", err)
	}
	if result.Status != "error" || result.ErrorCode != "E" {
		t.Fatalf("unexpected error result %#v", result)
	}

	svc = New(repo, stubPluginService{execFn: func(ctx context.Context, key string, req plugindomain.ExecutionRequest, onEvent func(plugindomain.ExecutionEvent)) error {
		return errors.New("boom")
	}})
	result, err = svc.ExecuteAction(ctx, account.ID, "VERIFY", nil)
	if err != nil {
		t.Fatalf("execute error: %v", err)
	}
	if result.ErrorCode != "EXECUTION_FAILED" {
		t.Fatalf("unexpected error code %#v", result)
	}

	svc = New(repo, stubPluginService{execFn: func(ctx context.Context, key string, req plugindomain.ExecutionRequest, onEvent func(plugindomain.ExecutionEvent)) error {
		if onEvent != nil {
			onEvent(plugindomain.ExecutionEvent{Type: "result", Data: map[string]any{"ok": true}})
		}
		return nil
	}})
	result, err = svc.ExecuteAction(ctx, account.ID, "VERIFY", nil)
	if err != nil {
		t.Fatalf("execute success: %v", err)
	}
	if result.Status != "ok" || result.Result["ok"] != true {
		t.Fatalf("unexpected success result %#v", result)
	}
}

func TestVerificationStatusForAction(t *testing.T) {
	if status, ok := verificationStatusForAction("", true); ok || status != "" {
		t.Fatalf("expected no status for empty action")
	}
	if !isVerificationAction("verify") || !isVerificationAction("validate_account") {
		t.Fatalf("expected verification actions")
	}
	if status, ok := verificationStatusForAction("verify", true); !ok || status != accountdomain.StatusActive {
		t.Fatalf("expected active status")
	}
	if status, ok := verificationStatusForAction("verify", false); !ok || status != accountdomain.StatusInactive {
		t.Fatalf("expected inactive status")
	}
}
