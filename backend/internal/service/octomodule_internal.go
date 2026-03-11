package service

import (
	"context"
	"encoding/json"
	"errors"

	"octomanger/backend/internal/dto"
	"octomanger/backend/internal/repository"
)

type OctoModuleInternalService interface {
	GetAccount(ctx context.Context, id uint64) (*dto.AccountResponse, error)
	GetAccountByIdentifier(ctx context.Context, typeKey string, identifier string) (*dto.AccountResponse, error)
	PatchAccountSpec(ctx context.Context, id uint64, req *dto.OctoModuleInternalPatchAccountSpecRequest) (*dto.AccountResponse, error)
	GetLatestEmail(ctx context.Context, emailAccountID uint64, mailbox string) (*dto.LatestEmailMessageResponse, error)
}

type octoModuleInternalService struct {
	accountRepo repository.AccountRepository
	emailSvc    EmailAccountService
}

func NewOctoModuleInternalService(
	accountRepo repository.AccountRepository,
	emailSvc EmailAccountService,
) OctoModuleInternalService {
	return &octoModuleInternalService{
		accountRepo: accountRepo,
		emailSvc:    emailSvc,
	}
}

func (s *octoModuleInternalService) GetAccount(ctx context.Context, id uint64) (*dto.AccountResponse, error) {
	if id == 0 {
		return nil, invalidInput("account id is required")
	}
	if s.accountRepo == nil {
		return nil, internalError("account repository is not configured", errors.New("missing account repository"))
	}
	item, err := s.accountRepo.GetByID(ctx, id)
	if err != nil {
		return nil, wrapRepoError(err, "account not found")
	}
	response := accountToResponse(item)
	return &response, nil
}

func (s *octoModuleInternalService) GetAccountByIdentifier(ctx context.Context, typeKey string, identifier string) (*dto.AccountResponse, error) {
	trimmedTypeKey := trim(typeKey)
	trimmedIdentifier := trim(identifier)
	if trimmedTypeKey == "" {
		return nil, invalidInput("type_key is required")
	}
	if trimmedIdentifier == "" {
		return nil, invalidInput("identifier is required")
	}
	if s.accountRepo == nil {
		return nil, internalError("account repository is not configured", errors.New("missing account repository"))
	}
	item, err := s.accountRepo.GetByTypeKeyAndIdentifier(ctx, trimmedTypeKey, trimmedIdentifier)
	if err != nil {
		return nil, wrapRepoError(err, "account not found")
	}
	response := accountToResponse(item)
	return &response, nil
}

func (s *octoModuleInternalService) PatchAccountSpec(
	ctx context.Context,
	id uint64,
	req *dto.OctoModuleInternalPatchAccountSpecRequest,
) (*dto.AccountResponse, error) {
	if id == 0 {
		return nil, invalidInput("account id is required")
	}
	if req == nil {
		return nil, invalidInput("payload is required")
	}
	if s.accountRepo == nil {
		return nil, internalError("account repository is not configured", errors.New("missing account repository"))
	}

	rawSpec, marshalErr := json.Marshal(req.Spec)
	if marshalErr != nil || !isJSONObject(rawSpec) {
		return nil, invalidInput("spec must be a valid JSON object")
	}

	item, err := s.accountRepo.GetByID(ctx, id)
	if err != nil {
		return nil, wrapRepoError(err, "account not found")
	}
	item.Spec = normalizeJSON(rawSpec, "{}")
	if err := s.accountRepo.Update(ctx, item); err != nil {
		return nil, internalError("failed to update account spec", err)
	}
	response := accountToResponse(item)
	return &response, nil
}

func (s *octoModuleInternalService) GetLatestEmail(
	ctx context.Context,
	emailAccountID uint64,
	mailbox string,
) (*dto.LatestEmailMessageResponse, error) {
	if emailAccountID == 0 {
		return nil, invalidInput("email account id is required")
	}
	if s.emailSvc == nil {
		return nil, internalError("email service is not configured", errors.New("missing email service"))
	}
	return s.emailSvc.GetLatestMessage(ctx, emailAccountID, &dto.ListEmailMessagesQuery{
		Mailbox: trim(mailbox),
		Limit:   1,
		Offset:  0,
	})
}

var _ OctoModuleInternalService = (*octoModuleInternalService)(nil)
