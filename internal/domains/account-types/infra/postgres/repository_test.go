package accounttypepostgres

import (
	"context"
	"errors"
	"math"
	"testing"

	accounttypedomain "octomanger/internal/domains/account-types/domain"
	"octomanger/internal/testutil"
)

func TestRepositoryCRUD(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := New(db)
	ctx := context.Background()

	created, err := repo.Create(ctx, accounttypedomain.CreateInput{
		Key:          "demo",
		Name:         "Demo",
		Category:     "generic",
		Schema:       map[string]any{"a": 1},
		Capabilities: map[string]any{"cap": true},
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if created.Key != "demo" {
		t.Fatalf("unexpected key %q", created.Key)
	}

	item, err := repo.GetByKey(ctx, "demo")
	if err != nil {
		t.Fatalf("get by key: %v", err)
	}
	if item.Name != "Demo" {
		t.Fatalf("unexpected name %q", item.Name)
	}

	items, total, err := repo.ListPage(ctx, 10, 0)
	if err != nil {
		t.Fatalf("list page: %v", err)
	}
	if total != 1 || len(items) != 1 {
		t.Fatalf("unexpected list result: total=%d len=%d", total, len(items))
	}

	newName := "Updated"
	newCategory := "special"
	updated, err := repo.Patch(ctx, "demo", accounttypedomain.PatchInput{
		Name:     &newName,
		Category: &newCategory,
	})
	if err != nil {
		t.Fatalf("patch: %v", err)
	}
	if updated.Name != "Updated" || updated.Category != "special" {
		t.Fatalf("unexpected updated record %#v", updated)
	}

	upserted, err := repo.Upsert(ctx, accounttypedomain.CreateInput{Key: "demo", Name: "Upserted", Category: "generic"})
	if err != nil {
		t.Fatalf("upsert: %v", err)
	}
	if upserted.Name != "Upserted" {
		t.Fatalf("unexpected upsert name %q", upserted.Name)
	}

	if err := repo.Delete(ctx, "demo"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if err := repo.Delete(ctx, "missing"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected not found delete error, got %v", err)
	}
}

func TestRepositoryErrors(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := New(db)
	ctx := context.Background()

	if _, err := repo.GetByKey(ctx, "missing"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected not found, got %v", err)
	}
	if _, err := repo.Patch(ctx, "missing", accounttypedomain.PatchInput{}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected not found patch, got %v", err)
	}
	if _, err := repo.Create(ctx, accounttypedomain.CreateInput{
		Key:      "bad",
		Name:     "Bad",
		Category: "generic",
		Schema:   map[string]any{"bad": math.Inf(1)},
	}); err == nil {
		t.Fatalf("expected marshal error")
	}
}
