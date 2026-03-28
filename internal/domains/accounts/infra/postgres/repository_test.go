package accountpostgres

import (
	"context"
	"errors"
	"math"
	"testing"

	"gorm.io/gorm"

	accountdomain "octomanger/internal/domains/accounts/domain"
	"octomanger/internal/platform/database"
	"octomanger/internal/testutil"
)

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

func TestRepositoryCRUD(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := New(db)
	ctx := context.Background()

	accountTypeID := seedAccountType(t, db, "demo")

	created, err := repo.Create(ctx, accountdomain.CreateInput{
		AccountTypeID: accountTypeID,
		Identifier:    "acct-1",
		Status:        "active",
		Tags:          []string{"a"},
		Spec:          map[string]any{"x": 1},
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if created.AccountTypeKey != "demo" {
		t.Fatalf("expected account type key demo, got %q", created.AccountTypeKey)
	}

	fetched, err := repo.Get(ctx, created.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if fetched.Identifier != "acct-1" {
		t.Fatalf("unexpected identifier %q", fetched.Identifier)
	}

	byKey, err := repo.GetByTypeKeyAndIdentifier(ctx, "demo", "acct-1")
	if err != nil {
		t.Fatalf("get by type/identifier: %v", err)
	}
	if byKey.ID != created.ID {
		t.Fatalf("unexpected account id %d", byKey.ID)
	}

	items, total, err := repo.ListPage(ctx, 10, 0)
	if err != nil {
		t.Fatalf("list page: %v", err)
	}
	if total != 1 || len(items) != 1 {
		t.Fatalf("unexpected list result total=%d len=%d", total, len(items))
	}

	newStatus := "inactive"
	updated, err := repo.Patch(ctx, created.ID, accountdomain.PatchInput{Status: &newStatus, Tags: []string{"b"}, Spec: map[string]any{"y": 2}})
	if err != nil {
		t.Fatalf("patch: %v", err)
	}
	if updated.Status != "inactive" || len(updated.Tags) != 1 || updated.Tags[0] != "b" {
		t.Fatalf("unexpected patch result %#v", updated)
	}

	statusUpdated, err := repo.UpdateStatus(ctx, created.ID, "pending")
	if err != nil {
		t.Fatalf("update status: %v", err)
	}
	if statusUpdated.Status != "pending" {
		t.Fatalf("unexpected status %q", statusUpdated.Status)
	}

	if err := repo.Delete(ctx, created.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if err := repo.Delete(ctx, created.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected not found delete, got %v", err)
	}
}

func TestRepositoryErrorPaths(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := New(db)
	ctx := context.Background()

	if _, err := repo.Get(ctx, 999); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected not found, got %v", err)
	}
	if _, err := repo.GetByTypeKeyAndIdentifier(ctx, "demo", "missing"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected not found, got %v", err)
	}
	if _, err := repo.Patch(ctx, 999, accountdomain.PatchInput{}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected not found patch, got %v", err)
	}
	if _, err := repo.UpdateStatus(ctx, 999, "active"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected not found update, got %v", err)
	}
	if _, err := repo.Create(ctx, accountdomain.CreateInput{Identifier: "bad", Spec: map[string]any{"x": math.Inf(1)}}); err == nil {
		t.Fatalf("expected marshal error")
	}
}

func TestRepositoryCreateWithNoAccountTypeID(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := New(db)
	ctx := context.Background()

	created, err := repo.Create(ctx, accountdomain.CreateInput{Identifier: "acct-2", Status: "active"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if created.AccountTypeID != nil {
		t.Fatalf("expected nil account type id")
	}
}

func TestDecodeJSONStringArrayAndNormalizeStrings(t *testing.T) {
	if got := decodeJSONStringArray(nil); len(got) != 0 {
		t.Fatalf("expected empty slice")
	}
	if got := normalizeStrings(nil); len(got) != 0 {
		t.Fatalf("expected empty slice")
	}
}
