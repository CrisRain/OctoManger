package accounttypeapp

import (
	"context"

	accounttypedomain "octomanger/internal/domains/account-types/domain"
	accounttypepostgres "octomanger/internal/domains/account-types/infra/postgres"
)

type Service struct {
	repo accounttypepostgres.Repository
}

func New(repo accounttypepostgres.Repository) Service {
	return Service{repo: repo}
}

func (s Service) List(ctx context.Context) ([]accounttypedomain.AccountType, error) {
	return s.repo.List(ctx)
}

func (s Service) ListPage(ctx context.Context, limit int, offset int) ([]accounttypedomain.AccountType, int64, error) {
	return s.repo.ListPage(ctx, limit, offset)
}

func (s Service) GetByKey(ctx context.Context, key string) (*accounttypedomain.AccountType, error) {
	return s.repo.GetByKey(ctx, key)
}

func (s Service) Create(ctx context.Context, input accounttypedomain.CreateInput) (*accounttypedomain.AccountType, error) {
	if input.Category == "" {
		input.Category = "generic"
	}
	return s.repo.Create(ctx, input)
}

func (s Service) Patch(ctx context.Context, key string, input accounttypedomain.PatchInput) (*accounttypedomain.AccountType, error) {
	return s.repo.Patch(ctx, key, input)
}

func (s Service) Upsert(ctx context.Context, input accounttypedomain.CreateInput) (*accounttypedomain.AccountType, error) {
	if input.Category == "" {
		input.Category = "generic"
	}
	return s.repo.Upsert(ctx, input)
}

func (s Service) Delete(ctx context.Context, key string) error {
	return s.repo.Delete(ctx, key)
}
