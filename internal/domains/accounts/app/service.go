package accountapp

import (
	"context"

	accountdomain "octomanger/internal/domains/accounts/domain"
	accountpostgres "octomanger/internal/domains/accounts/infra/postgres"
)

type Service struct {
	repo accountpostgres.Repository
}

func New(repo accountpostgres.Repository) Service {
	return Service{repo: repo}
}

func (s Service) List(ctx context.Context) ([]accountdomain.Account, error) {
	return s.repo.List(ctx)
}

func (s Service) Get(ctx context.Context, accountID int64) (*accountdomain.Account, error) {
	return s.repo.Get(ctx, accountID)
}

func (s Service) Create(ctx context.Context, input accountdomain.CreateInput) (*accountdomain.Account, error) {
	if input.Status == "" {
		input.Status = "active"
	}
	return s.repo.Create(ctx, input)
}

func (s Service) Patch(ctx context.Context, accountID int64, input accountdomain.PatchInput) (*accountdomain.Account, error) {
	return s.repo.Patch(ctx, accountID, input)
}

func (s Service) Delete(ctx context.Context, accountID int64) error {
	return s.repo.Delete(ctx, accountID)
}
