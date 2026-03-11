package service

import (
	"context"
	"encoding/json"

	"octomanger/backend/internal/dto"
	"octomanger/backend/internal/model"
	"octomanger/backend/internal/repository"
)

type ApiKeyService interface {
	List(ctx context.Context) ([]dto.ApiKeyResponse, error)
	Create(ctx context.Context, req *dto.CreateApiKeyRequest) (*dto.CreateApiKeyResponse, error)
	SetEnabled(ctx context.Context, id uint64, enabled bool) (*dto.ApiKeyResponse, error)
	Delete(ctx context.Context, id uint64) error
	ValidateKey(ctx context.Context, rawKey string) (*dto.ApiKeyResponse, error)
	ValidateAdminKey(ctx context.Context, rawKey string) (*dto.ApiKeyResponse, error)
	ValidateWebhookKey(ctx context.Context, rawKey string, slug string) (*dto.ApiKeyResponse, error)
	HasAnyAdminKey(ctx context.Context) (bool, error)
	HasAnyKey(ctx context.Context) (bool, error)
}

type apiKeyService struct {
	repo repository.ApiKeyRepository
}

func NewApiKeyService(repo repository.ApiKeyRepository) ApiKeyService {
	return &apiKeyService{repo: repo}
}

func (s *apiKeyService) List(ctx context.Context) ([]dto.ApiKeyResponse, error) {
	items, err := s.repo.List(ctx)
	if err != nil {
		return nil, internalError("failed to list api keys", err)
	}
	responses := make([]dto.ApiKeyResponse, 0, len(items))
	for i := range items {
		responses = append(responses, apiKeyToResponse(&items[i]))
	}
	return responses, nil
}

func (s *apiKeyService) Create(ctx context.Context, req *dto.CreateApiKeyRequest) (*dto.CreateApiKeyResponse, error) {
	if req == nil {
		return nil, invalidInput("payload is required")
	}
	name := trim(req.Name)
	if name == "" {
		return nil, invalidInput("name is required")
	}

	role := trim(req.Role)
	if role == "" {
		role = model.ApiKeyRoleWebhook
	}
	if role != model.ApiKeyRoleAdmin && role != model.ApiKeyRoleWebhook {
		return nil, invalidInput("role must be 'admin' or 'webhook'")
	}

	// Admin keys may only be created during initial setup (when no admin key exists yet).
	if role == model.ApiKeyRoleAdmin {
		hasAdmin, err := s.HasAnyAdminKey(ctx)
		if err != nil {
			return nil, internalError("failed to check admin keys", err)
		}
		if hasAdmin {
			return nil, conflict("admin key can only be created during initial setup")
		}
	}

	scope := trim(req.WebhookScope)
	if role == model.ApiKeyRoleWebhook && scope == "" {
		scope = model.ApiKeyWebhookScopeAll
	}
	if role == model.ApiKeyRoleAdmin {
		scope = ""
	}

	raw, err := generateSecureToken(32)
	if err != nil {
		return nil, internalError("failed to generate api key", err)
	}
	hash := hashToken(raw)
	prefix := raw[:8]

	item := &model.ApiKey{
		Name:         name,
		KeyHash:      hash,
		KeyPrefix:    prefix,
		Role:         role,
		WebhookScope: scope,
		Enabled:      true,
	}
	if err := s.repo.Create(ctx, item); err != nil {
		return nil, internalError("failed to create api key", err)
	}

	resp := apiKeyToResponse(item)
	return &dto.CreateApiKeyResponse{ApiKey: resp, RawKey: raw}, nil
}

func (s *apiKeyService) SetEnabled(ctx context.Context, id uint64, enabled bool) (*dto.ApiKeyResponse, error) {
	if id == 0 {
		return nil, invalidInput("api key id is required")
	}
	item, err := s.repo.UpdateEnabled(ctx, id, enabled)
	if err != nil {
		return nil, wrapRepoError(err, "api key not found")
	}
	response := apiKeyToResponse(item)
	return &response, nil
}

func (s *apiKeyService) Delete(ctx context.Context, id uint64) error {
	if id == 0 {
		return invalidInput("api key id is required")
	}
	if _, err := s.repo.GetByID(ctx, id); err != nil {
		return wrapRepoError(err, "api key not found")
	}
	if err := s.repo.Delete(ctx, id); err != nil {
		return internalError("failed to delete api key", err)
	}
	return nil
}

func (s *apiKeyService) ValidateKey(ctx context.Context, rawKey string) (*dto.ApiKeyResponse, error) {
	trimmed := trim(rawKey)
	if trimmed == "" {
		return nil, unauthorized("invalid api key")
	}
	hash := hashToken(trimmed)
	item, err := s.repo.GetByHash(ctx, hash)
	if err != nil {
		if isNotFound(err) {
			return nil, unauthorized("invalid api key")
		}
		return nil, internalError("failed to validate api key", err)
	}
	if !item.Enabled {
		return nil, unauthorized("api key is disabled")
	}
	_ = s.repo.UpdateLastUsed(ctx, item.ID)
	response := apiKeyToResponse(item)
	return &response, nil
}

func (s *apiKeyService) ValidateAdminKey(ctx context.Context, rawKey string) (*dto.ApiKeyResponse, error) {
	resp, err := s.ValidateKey(ctx, rawKey)
	if err != nil {
		return nil, err
	}
	if resp.Role != model.ApiKeyRoleAdmin && resp.Role != model.ApiKeyRoleInternal {
		return nil, unauthorized("api key does not have admin role")
	}
	return resp, nil
}

func (s *apiKeyService) ValidateWebhookKey(ctx context.Context, rawKey string, slug string) (*dto.ApiKeyResponse, error) {
	resp, err := s.ValidateKey(ctx, rawKey)
	if err != nil {
		return nil, err
	}
	if resp.Role != model.ApiKeyRoleWebhook && resp.Role != model.ApiKeyRoleAdmin {
		return nil, unauthorized("api key does not have webhook access")
	}
	if resp.Role == model.ApiKeyRoleWebhook {
		if resp.WebhookScope != model.ApiKeyWebhookScopeAll && resp.WebhookScope != slug {
			return nil, unauthorized("api key is not scoped to this webhook")
		}
	}
	return resp, nil
}

func (s *apiKeyService) HasAnyAdminKey(ctx context.Context) (bool, error) {
	items, err := s.repo.List(ctx)
	if err != nil {
		return false, internalError("failed to list api keys", err)
	}
	for i := range items {
		if items[i].Role == model.ApiKeyRoleAdmin && items[i].Enabled {
			return true, nil
		}
	}
	return false, nil
}

func (s *apiKeyService) HasAnyKey(ctx context.Context) (bool, error) {
	items, err := s.repo.List(ctx)
	if err != nil {
		return false, internalError("failed to list api keys", err)
	}
	return len(items) > 0, nil
}

func apiKeyToResponse(item *model.ApiKey) dto.ApiKeyResponse {
	if item == nil {
		return dto.ApiKeyResponse{}
	}
	return dto.ApiKeyResponse{
		ID:           item.ID,
		Name:         item.Name,
		KeyPrefix:    item.KeyPrefix,
		Role:         item.Role,
		WebhookScope: item.WebhookScope,
		Enabled:      item.Enabled,
		LastUsedAt:   item.LastUsedAt,
		CreatedAt:    item.CreatedAt,
		UpdatedAt:    item.UpdatedAt,
	}
}

var _ ApiKeyService = (*apiKeyService)(nil)

const internalKeyConfigKey = "internal_api_token"

// EnsureInternalKey returns a stable internal API token for OctoModule scripts.
// The raw token is persisted in system_config so it survives restarts.
// On first call (or if the stored token is stale) a new key is generated.
func EnsureInternalKey(ctx context.Context, apiKeyRepo repository.ApiKeyRepository, systemConfigRepo repository.SystemConfigRepository) (string, error) {
	// Try to read the cached raw token from system_config.
	if cfg, err := systemConfigRepo.GetByKey(ctx, internalKeyConfigKey); err == nil {
		var rawKey string
		if json.Unmarshal(cfg.Value, &rawKey) == nil && rawKey != "" {
			hash := hashToken(rawKey)
			if key, err := apiKeyRepo.GetByHash(ctx, hash); err == nil && key.Enabled && key.Role == model.ApiKeyRoleInternal {
				return rawKey, nil
			}
		}
	}

	// Purge stale internal keys before creating a new one.
	if items, err := apiKeyRepo.List(ctx); err == nil {
		for _, item := range items {
			if item.Role == model.ApiKeyRoleInternal {
				_ = apiKeyRepo.Delete(ctx, item.ID)
			}
		}
	}

	raw, err := generateSecureToken(32)
	if err != nil {
		return "", internalError("failed to generate internal api key", err)
	}
	hash := hashToken(raw)
	prefix := raw[:8]

	item := &model.ApiKey{
		Name:      "__internal__",
		KeyHash:   hash,
		KeyPrefix: prefix,
		Role:      model.ApiKeyRoleInternal,
		Enabled:   true,
	}
	if err := apiKeyRepo.Create(ctx, item); err != nil {
		return "", internalError("failed to create internal api key", err)
	}

	rawJSON, _ := json.Marshal(raw)
	_ = systemConfigRepo.Upsert(ctx, &model.SystemConfig{
		Key:         internalKeyConfigKey,
		Value:       rawJSON,
		Description: "Internal API token for OctoModule scripts",
	})

	return raw, nil
}
