package emailapp

import (
	"context"
	"fmt"
	"strings"

	emaildomain "octomanger/internal/domains/email/domain"
	emailpostgres "octomanger/internal/domains/email/infra/postgres"
	"octomanger/internal/domains/email/providers/outlook"
)

const outlookDefaultScopes = "offline_access https://graph.microsoft.com/Mail.Read https://graph.microsoft.com/Mail.ReadWrite https://graph.microsoft.com/User.Read"

type Service struct {
	repo emailpostgres.Repository
}

func New(repo emailpostgres.Repository) Service {
	return Service{repo: repo}
}

func (s Service) List(ctx context.Context) ([]emaildomain.Account, error) {
	return s.repo.List(ctx)
}

func (s Service) Get(ctx context.Context, emailID int64) (*emaildomain.Account, error) {
	return s.repo.Get(ctx, emailID)
}

func (s Service) Create(ctx context.Context, input emaildomain.CreateInput) (*emaildomain.Account, error) {
	if strings.TrimSpace(input.Address) == "" {
		return nil, fmt.Errorf("email address is required")
	}
	if strings.TrimSpace(input.Provider) == "" {
		input.Provider = "outlook"
	}
	if input.Status == "" {
		input.Status = "active"
	}
	return s.repo.Create(ctx, input)
}

func (s Service) Patch(ctx context.Context, emailID int64, input emaildomain.PatchInput) (*emaildomain.Account, error) {
	if emailID <= 0 {
		return nil, fmt.Errorf("email account id is required")
	}
	return s.repo.Patch(ctx, emailID, input)
}

func (s Service) Delete(ctx context.Context, emailID int64) error {
	if emailID <= 0 {
		return fmt.Errorf("email account id is required")
	}
	return s.repo.Delete(ctx, emailID)
}

func (s Service) BulkImport(ctx context.Context, input emaildomain.BulkImportInput) (*emaildomain.BulkImportResult, error) {
	result := &emaildomain.BulkImportResult{
		Total: len(input.Lines),
		Items: make([]emaildomain.BulkImportLineResult, 0, len(input.Lines)),
	}
	for _, raw := range input.Lines {
		line := strings.TrimSpace(raw)
		if line == "" {
			continue
		}
		item := s.importOutlookLine(ctx, line)
		result.Items = append(result.Items, item)
		if item.OK {
			result.Success++
		} else {
			result.Failed++
		}
	}
	result.Total = len(result.Items)
	return result, nil
}

func (s Service) importOutlookLine(ctx context.Context, line string) emaildomain.BulkImportLineResult {
	parts := strings.Split(line, "----")
	if len(parts) != 4 {
		return emaildomain.BulkImportLineResult{Line: line, OK: false, Error: "格式错误，需要: 邮箱----密码----clientid----refresh_token"}
	}
	address := strings.TrimSpace(parts[0])
	password := strings.TrimSpace(parts[1])
	clientID := strings.TrimSpace(parts[2])
	refreshToken := strings.TrimSpace(parts[3])
	if address == "" || clientID == "" || refreshToken == "" {
		return emaildomain.BulkImportLineResult{Line: line, Address: address, OK: false, Error: "邮箱、clientid 和 refresh_token 不能为空"}
	}
	cfg := map[string]any{
		"client_id":     clientID,
		"refresh_token": refreshToken,
		"tenant":        "common",
		"mailbox":       "Inbox",
		"scope":         outlookDefaultScopes,
	}
	if password != "" {
		cfg["password"] = password
	}
	created, err := s.Create(ctx, emaildomain.CreateInput{
		Address:  address,
		Provider: "outlook",
		Status:   "active",
		Config:   cfg,
	})
	if err != nil {
		return emaildomain.BulkImportLineResult{Line: line, Address: address, OK: false, Error: err.Error()}
	}
	return emaildomain.BulkImportLineResult{Line: line, Address: address, OK: true, ID: &created.ID}
}

func (s Service) BuildOutlookAuthorizeURL(ctx context.Context, emailID int64) (*emaildomain.OutlookAuthorizeURLResult, error) {
	account, config, err := s.loadOutlookAccount(ctx, emailID)
	if err != nil {
		return nil, err
	}

	authorizeURL, err := config.BuildAuthorizeURL(account.Address)
	if err != nil {
		return nil, err
	}

	return &emaildomain.OutlookAuthorizeURLResult{
		AuthorizeURL: authorizeURL,
	}, nil
}

func (s Service) ExchangeOutlookCode(ctx context.Context, emailID int64, input emaildomain.OutlookExchangeCodeInput) (*emaildomain.Account, error) {
	account, config, err := s.loadOutlookAccount(ctx, emailID)
	if err != nil {
		return nil, err
	}

	nextConfig, err := config.ExchangeCode(ctx, input.Code, account.Address)
	if err != nil {
		return nil, err
	}

	return s.repo.Patch(ctx, emailID, emaildomain.PatchInput{
		Config: nextConfig.RawMap(),
	})
}

func (s Service) ListMailboxes(ctx context.Context, emailID int64, input emaildomain.ListMailboxesInput) (*emaildomain.ListMailboxesResult, error) {
	_, graphConfig, err := s.loadGraphConfig(ctx, emailID)
	if err != nil {
		return nil, err
	}

	mailboxes, err := outlook.ListMailboxes(ctx, graphConfig, 200)
	if err != nil {
		return nil, err
	}

	items := make([]emaildomain.Mailbox, 0, len(mailboxes))
	for _, mailbox := range mailboxes {
		if !matchMailboxPattern(mailbox.Name, input.Pattern) {
			continue
		}
		items = append(items, emaildomain.Mailbox{
			ID:   mailbox.ID,
			Name: mailbox.Name,
		})
	}

	return &emaildomain.ListMailboxesResult{
		Pattern: strings.TrimSpace(input.Pattern),
		Items:   items,
	}, nil
}

func (s Service) ListMessages(ctx context.Context, emailID int64, input emaildomain.ListMessagesInput) (*emaildomain.ListMessagesResult, error) {
	account, graphConfig, err := s.loadGraphConfig(ctx, emailID)
	if err != nil {
		return nil, err
	}

	mailbox := strings.TrimSpace(input.Mailbox)
	if mailbox == "" {
		config := outlook.ParseAccountConfig(account.Config)
		mailbox = config.Mailbox
	}

	mailboxID, mailboxName, err := outlook.ResolveMailbox(ctx, graphConfig, mailbox)
	if err != nil {
		return nil, err
	}

	messages, total, err := outlook.ListMessages(ctx, graphConfig, mailboxID, input.Limit, input.Offset)
	if err != nil {
		return nil, err
	}

	items := make([]emaildomain.MessageSummary, 0, len(messages))
	for _, message := range messages {
		items = append(items, emaildomain.MessageSummary{
			ID:      message.ID,
			Subject: message.Subject,
			From:    message.From,
			To:      message.To,
			Date:    message.Date,
			Size:    message.Size,
			Flags:   message.Flags,
		})
	}

	limit := input.Limit
	if limit <= 0 {
		limit = 20
	}
	if limit > 200 {
		limit = 200
	}

	offset := input.Offset
	if offset < 0 {
		offset = 0
	}

	return &emaildomain.ListMessagesResult{
		Mailbox: mailboxName,
		Limit:   limit,
		Offset:  offset,
		Total:   total,
		Items:   items,
	}, nil
}

func (s Service) GetLatestMessage(ctx context.Context, emailID int64, input emaildomain.ListMessagesInput) (*emaildomain.LatestMessageResult, error) {
	result, err := s.ListMessages(ctx, emailID, emaildomain.ListMessagesInput{
		Mailbox: input.Mailbox,
		Limit:   1,
		Offset:  0,
	})
	if err != nil {
		return nil, err
	}

	if len(result.Items) == 0 {
		return &emaildomain.LatestMessageResult{
			Mailbox: result.Mailbox,
			Found:   false,
		}, nil
	}

	item, err := s.GetMessage(ctx, emailID, result.Items[0].ID)
	if err != nil {
		return nil, err
	}

	return &emaildomain.LatestMessageResult{
		Mailbox: result.Mailbox,
		Found:   true,
		Item:    item,
	}, nil
}

func (s Service) GetMessage(ctx context.Context, emailID int64, messageID string) (*emaildomain.MessageDetail, error) {
	if strings.TrimSpace(messageID) == "" {
		return nil, fmt.Errorf("message id is required")
	}

	_, graphConfig, err := s.loadGraphConfig(ctx, emailID)
	if err != nil {
		return nil, err
	}

	message, err := outlook.GetMessage(ctx, graphConfig, messageID)
	if err != nil {
		return nil, err
	}

	return &emaildomain.MessageDetail{
		ID:       message.ID,
		Subject:  message.Subject,
		From:     message.From,
		To:       message.To,
		Cc:       message.Cc,
		Date:     message.Date,
		Size:     message.Size,
		Flags:    message.Flags,
		Headers:  message.Headers,
		TextBody: message.TextBody,
		HTMLBody: message.HTMLBody,
	}, nil
}

func (s Service) PreviewMailboxes(ctx context.Context, input emaildomain.PreviewInput) (*emaildomain.ListMailboxesResult, error) {
	graphConfig, _, err := buildPreviewGraphConfig(ctx, input.Config)
	if err != nil {
		return nil, err
	}

	mailboxes, err := outlook.ListMailboxes(ctx, graphConfig, 200)
	if err != nil {
		return nil, err
	}

	items := make([]emaildomain.Mailbox, 0, len(mailboxes))
	for _, mailbox := range mailboxes {
		if !matchMailboxPattern(mailbox.Name, input.Pattern) {
			continue
		}
		items = append(items, emaildomain.Mailbox{
			ID:   mailbox.ID,
			Name: mailbox.Name,
		})
	}

	return &emaildomain.ListMailboxesResult{
		Pattern: strings.TrimSpace(input.Pattern),
		Items:   items,
	}, nil
}

func (s Service) PreviewLatestMessage(ctx context.Context, input emaildomain.PreviewInput) (*emaildomain.LatestMessageResult, error) {
	graphConfig, config, err := buildPreviewGraphConfig(ctx, input.Config)
	if err != nil {
		return nil, err
	}

	targetMailbox := strings.TrimSpace(input.Mailbox)
	if targetMailbox == "" {
		targetMailbox = config.Mailbox
	}

	mailboxID, mailboxName, err := outlook.ResolveMailbox(ctx, graphConfig, targetMailbox)
	if err != nil {
		return nil, err
	}

	items, _, err := outlook.ListMessages(ctx, graphConfig, mailboxID, 1, 0)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return &emaildomain.LatestMessageResult{
			Mailbox: mailboxName,
			Found:   false,
		}, nil
	}

	message, err := outlook.GetMessage(ctx, graphConfig, items[0].ID)
	if err != nil {
		return nil, err
	}

	return &emaildomain.LatestMessageResult{
		Mailbox: mailboxName,
		Found:   true,
		Item: &emaildomain.MessageDetail{
			ID:       message.ID,
			Subject:  message.Subject,
			From:     message.From,
			To:       message.To,
			Cc:       message.Cc,
			Date:     message.Date,
			Size:     message.Size,
			Flags:    message.Flags,
			Headers:  message.Headers,
			TextBody: message.TextBody,
			HTMLBody: message.HTMLBody,
		},
	}, nil
}

func (s Service) loadOutlookAccount(ctx context.Context, emailID int64) (*emaildomain.Account, outlook.AccountConfig, error) {
	if emailID <= 0 {
		return nil, outlook.AccountConfig{}, fmt.Errorf("email account id is required")
	}

	account, err := s.repo.Get(ctx, emailID)
	if err != nil {
		return nil, outlook.AccountConfig{}, err
	}

	if !strings.EqualFold(strings.TrimSpace(account.Provider), "outlook") {
		return nil, outlook.AccountConfig{}, fmt.Errorf("email provider %q does not support outlook message access", account.Provider)
	}

	return account, outlook.ParseAccountConfig(account.Config), nil
}

func (s Service) loadGraphConfig(ctx context.Context, emailID int64) (*emaildomain.Account, outlook.GraphConfig, error) {
	account, config, err := s.loadOutlookAccount(ctx, emailID)
	if err != nil {
		return nil, outlook.GraphConfig{}, err
	}

	refreshedConfig, changed, err := config.EnsureAccessToken(ctx, account.Address)
	if err != nil {
		return nil, outlook.GraphConfig{}, err
	}
	if changed {
		account, err = s.repo.Patch(ctx, emailID, emaildomain.PatchInput{
			Config: refreshedConfig.RawMap(),
		})
		if err != nil {
			return nil, outlook.GraphConfig{}, err
		}
		config = refreshedConfig
	}

	graphConfig, err := config.GraphConfig()
	if err != nil {
		return nil, outlook.GraphConfig{}, err
	}
	return account, graphConfig, nil
}

func buildPreviewGraphConfig(ctx context.Context, raw map[string]any) (outlook.GraphConfig, outlook.AccountConfig, error) {
	config := outlook.ParseAccountConfig(raw)
	refreshedConfig, changed, err := config.EnsureAccessToken(ctx, config.Username)
	if err != nil {
		return outlook.GraphConfig{}, outlook.AccountConfig{}, err
	}
	if changed {
		config = refreshedConfig
	}

	graphConfig, err := config.GraphConfig()
	if err != nil {
		return outlook.GraphConfig{}, outlook.AccountConfig{}, err
	}
	return graphConfig, config, nil
}

func matchMailboxPattern(name string, pattern string) bool {
	target := strings.ToLower(strings.TrimSpace(name))
	rawPattern := strings.TrimSpace(pattern)
	if rawPattern == "" || rawPattern == "*" {
		return true
	}

	normalized := strings.ToLower(strings.ReplaceAll(rawPattern, "%", "*"))
	switch {
	case strings.HasPrefix(normalized, "*") && strings.HasSuffix(normalized, "*") && len(normalized) >= 2:
		return strings.Contains(target, strings.Trim(normalized, "*"))
	case strings.HasPrefix(normalized, "*"):
		return strings.HasSuffix(target, strings.TrimPrefix(normalized, "*"))
	case strings.HasSuffix(normalized, "*"):
		return strings.HasPrefix(target, strings.TrimSuffix(normalized, "*"))
	default:
		return target == normalized
	}
}
