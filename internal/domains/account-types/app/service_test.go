package accounttypeapp

import (
	"context"
	"testing"

	accounttypedomain "octomanger/internal/domains/account-types/domain"
	accounttypepostgres "octomanger/internal/domains/account-types/infra/postgres"
	"octomanger/internal/testutil"
)

func TestServiceDefaultsCategory(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := accounttypepostgres.New(db)
	svc := New(repo)
	ctx := context.Background()

	created, err := svc.Create(ctx, accounttypedomain.CreateInput{Key: "demo", Name: "Demo"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if created.Category != "generic" {
		t.Fatalf("expected default category, got %q", created.Category)
	}

	upserted, err := svc.Upsert(ctx, accounttypedomain.CreateInput{Key: "demo", Name: "Demo 2"})
	if err != nil {
		t.Fatalf("upsert: %v", err)
	}
	if upserted.Category != "generic" {
		t.Fatalf("expected default category, got %q", upserted.Category)
	}
}
