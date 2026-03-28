package accountapp

import (
	"context"
	"errors"
	"strings"

	accountdomain "octomanger/internal/domains/accounts/domain"
	accountpostgres "octomanger/internal/domains/accounts/infra/postgres"
	plugins "octomanger/internal/domains/plugins"
	plugindomain "octomanger/internal/domains/plugins/domain"
)

type Service struct {
	repo    accountpostgres.Repository
	plugins plugins.PluginService
}

var (
	ErrInvalidExecuteAction  = errors.New("action is required")
	ErrMissingAccountTypeKey = errors.New("account has no account_type_key, cannot determine plugin")
	ErrPluginBackendMissing  = errors.New("plugin backend is not configured")
)

type ExecuteActionResult struct {
	Status       string         `json:"status"`
	Result       map[string]any `json:"result,omitempty"`
	ErrorCode    string         `json:"error_code,omitempty"`
	ErrorMessage string         `json:"error_message,omitempty"`
}

func New(repo accountpostgres.Repository, pluginServices ...plugins.PluginService) Service {
	var pluginBackend plugins.PluginService
	if len(pluginServices) > 0 {
		pluginBackend = pluginServices[0]
	}
	return Service{repo: repo, plugins: pluginBackend}
}

func (s Service) List(ctx context.Context) ([]accountdomain.Account, error) {
	return s.repo.List(ctx)
}

func (s Service) ListPage(ctx context.Context, limit int, offset int) ([]accountdomain.Account, int64, error) {
	return s.repo.ListPage(ctx, limit, offset)
}

func (s Service) Get(ctx context.Context, accountID int64) (*accountdomain.Account, error) {
	return s.repo.Get(ctx, accountID)
}

func (s Service) GetByTypeKeyAndIdentifier(ctx context.Context, typeKey string, identifier string) (*accountdomain.Account, error) {
	typeKey = strings.TrimSpace(typeKey)
	identifier = strings.TrimSpace(identifier)
	if typeKey == "" || identifier == "" {
		return nil, accountpostgres.ErrNotFound
	}
	return s.repo.GetByTypeKeyAndIdentifier(ctx, typeKey, identifier)
}

func (s Service) Create(ctx context.Context, input accountdomain.CreateInput) (*accountdomain.Account, error) {
	if strings.TrimSpace(input.Identifier) == "" {
		return nil, errors.New("identifier is required")
	}
	input.Identifier = strings.TrimSpace(input.Identifier)
	input.Status = accountdomain.StatusPending
	return s.repo.Create(ctx, input)
}

func (s Service) Patch(ctx context.Context, accountID int64, input accountdomain.PatchInput) (*accountdomain.Account, error) {
	input.Status = nil
	return s.repo.Patch(ctx, accountID, input)
}

func (s Service) Delete(ctx context.Context, accountID int64) error {
	return s.repo.Delete(ctx, accountID)
}

func (s Service) SetStatus(ctx context.Context, accountID int64, status string) (*accountdomain.Account, error) {
	status = strings.TrimSpace(status)
	if status == "" {
		return s.repo.Get(ctx, accountID)
	}
	return s.repo.UpdateStatus(ctx, accountID, status)
}

func (s Service) ExecuteAction(ctx context.Context, accountID int64, action string, params map[string]any) (*ExecuteActionResult, error) {
	action = strings.TrimSpace(action)
	if action == "" {
		return nil, ErrInvalidExecuteAction
	}
	if s.plugins == nil {
		return nil, ErrPluginBackendMissing
	}

	account, err := s.repo.Get(ctx, accountID)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(account.AccountTypeKey) == "" {
		return nil, ErrMissingAccountTypeKey
	}

	if params == nil {
		params = map[string]any{}
	}

	var (
		resultData   map[string]any
		errorCode    string
		errorMessage string
	)

	execErr := s.plugins.Execute(ctx, account.AccountTypeKey, plugindomain.ExecutionRequest{
		Mode:   "account",
		Action: action,
		Input: map[string]any{
			"account": map[string]any{
				"id":         account.ID,
				"identifier": account.Identifier,
			},
			"params": params,
		},
		Context: map[string]any{
			"source": "account-execute",
		},
	}, func(event plugindomain.ExecutionEvent) {
		switch event.Type {
		case "result":
			resultData = event.Data
		case "error":
			errorCode = event.Error
			if errorCode == "" {
				errorCode = "PLUGIN_ERROR"
			}
			errorMessage = event.Message
		}
	})

	if execErr != nil && errorMessage == "" {
		errorCode = "EXECUTION_FAILED"
		errorMessage = execErr.Error()
	}

	if nextStatus, ok := verificationStatusForAction(action, errorMessage == ""); ok {
		_, _ = s.repo.UpdateStatus(ctx, accountID, nextStatus)
	}

	if errorMessage != "" {
		return &ExecuteActionResult{
			Status:       "error",
			ErrorCode:    errorCode,
			ErrorMessage: errorMessage,
		}, nil
	}

	return &ExecuteActionResult{
		Status: "ok",
		Result: resultData,
	}, nil
}
