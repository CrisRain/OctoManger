package emailpostgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"gorm.io/gorm"

	emaildomain "octomanger/internal/domains/email/domain"
	"octomanger/internal/platform/database"
)

var ErrNotFound = errors.New("email account not found")

type Repository struct {
	db *gorm.DB
}

func New(db *gorm.DB) Repository {
	return Repository{db: db}
}

func (r Repository) List(ctx context.Context) ([]emaildomain.Account, error) {
	items, _, err := r.ListPage(ctx, 0, 0)
	return items, err
}

func (r Repository) ListPage(ctx context.Context, limit int, offset int) ([]emaildomain.Account, int64, error) {
	var total int64
	if err := r.db.WithContext(ctx).
		Model(&database.EmailAccountModel{}).
		Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count email accounts: %w", err)
	}

	var records []database.EmailAccountModel
	query := r.db.WithContext(ctx).Order("created_at DESC")
	if limit > 0 {
		query = query.Limit(limit).Offset(offset)
	}
	if err := query.Find(&records).Error; err != nil {
		return nil, 0, fmt.Errorf("list email accounts: %w", err)
	}

	items := make([]emaildomain.Account, len(records))
	for i, record := range records {
		items[i] = toDomainEmailAccount(record)
	}
	return items, total, nil
}

func (r Repository) Get(ctx context.Context, emailID int64) (*emaildomain.Account, error) {
	var record database.EmailAccountModel
	if err := r.db.WithContext(ctx).
		First(&record, "id = ?", emailID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get email account: %w", err)
	}

	item := toDomainEmailAccount(record)
	return &item, nil
}

func (r Repository) Create(ctx context.Context, input emaildomain.CreateInput) (*emaildomain.Account, error) {
	configJSON, err := json.Marshal(normalizeMap(input.Config))
	if err != nil {
		return nil, fmt.Errorf("marshal email config: %w", err)
	}

	record := database.EmailAccountModel{
		Address:    input.Address,
		Provider:   input.Provider,
		Status:     input.Status,
		ConfigJSON: database.JSONBytes(configJSON),
	}
	if err := r.db.WithContext(ctx).Create(&record).Error; err != nil {
		return nil, fmt.Errorf("create email account: %w", err)
	}

	item := toDomainEmailAccount(record)
	return &item, nil
}

func (r Repository) Patch(ctx context.Context, emailID int64, input emaildomain.PatchInput) (*emaildomain.Account, error) {
	current, err := r.Get(ctx, emailID)
	if err != nil {
		return nil, err
	}

	if input.Provider != nil {
		current.Provider = *input.Provider
	}
	if input.Status != nil {
		current.Status = *input.Status
	}
	if input.Config != nil {
		current.Config = input.Config
	}

	configJSON, err := json.Marshal(normalizeMap(current.Config))
	if err != nil {
		return nil, fmt.Errorf("marshal email config: %w", err)
	}

	result := r.db.WithContext(ctx).
		Model(&database.EmailAccountModel{}).
		Where("id = ?", emailID).
		Updates(map[string]any{
			"provider":    current.Provider,
			"status":      current.Status,
			"config_json": configJSON,
		})
	if result.Error != nil {
		return nil, fmt.Errorf("patch email account: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return nil, ErrNotFound
	}

	return r.Get(ctx, emailID)
}

func (r Repository) UpdateStatus(ctx context.Context, emailID int64, status string) (*emaildomain.Account, error) {
	result := r.db.WithContext(ctx).
		Model(&database.EmailAccountModel{}).
		Where("id = ?", emailID).
		Update("status", status)
	if result.Error != nil {
		return nil, fmt.Errorf("update email account status: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return nil, ErrNotFound
	}

	return r.Get(ctx, emailID)
}

func (r Repository) Delete(ctx context.Context, emailID int64) error {
	result := r.db.WithContext(ctx).Delete(&database.EmailAccountModel{}, emailID)
	if result.Error != nil {
		return fmt.Errorf("delete email account: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func toDomainEmailAccount(record database.EmailAccountModel) emaildomain.Account {
	return emaildomain.Account{
		ID:        record.ID,
		Address:   record.Address,
		Provider:  record.Provider,
		Status:    record.Status,
		Config:    decodeJSONMap([]byte(record.ConfigJSON)),
		CreatedAt: record.CreatedAt,
		UpdatedAt: record.UpdatedAt,
	}
}

func decodeJSONMap(raw []byte) map[string]any {
	if len(raw) == 0 {
		return map[string]any{}
	}
	v := map[string]any{}
	_ = json.Unmarshal(raw, &v)
	return v
}

func normalizeMap(value map[string]any) map[string]any {
	if value == nil {
		return map[string]any{}
	}
	return value
}
