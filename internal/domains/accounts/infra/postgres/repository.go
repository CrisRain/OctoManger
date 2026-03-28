package accountpostgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"gorm.io/gorm"

	accountdomain "octomanger/internal/domains/accounts/domain"
	"octomanger/internal/platform/database"
	"octomanger/internal/platform/dbutil"
)

var ErrNotFound = errors.New("account not found")

type Repository struct {
	db *gorm.DB
}

func New(db *gorm.DB) Repository {
	return Repository{db: db}
}

func (r Repository) List(ctx context.Context) ([]accountdomain.Account, error) {
	items, _, err := r.ListPage(ctx, 0, 0)
	return items, err
}

func (r Repository) ListPage(ctx context.Context, limit int, offset int) ([]accountdomain.Account, int64, error) {
	var total int64
	if err := r.db.WithContext(ctx).
		Model(&database.AccountModel{}).
		Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count accounts: %w", err)
	}

	var records []database.AccountModel
	query := r.db.WithContext(ctx).Order("created_at DESC")
	if limit > 0 {
		query = query.Limit(limit).Offset(offset)
	}
	if err := query.Find(&records).Error; err != nil {
		return nil, 0, fmt.Errorf("list accounts: %w", err)
	}

	typeKeys, err := r.accountTypeKeysByID(ctx, records)
	if err != nil {
		return nil, 0, err
	}

	items := make([]accountdomain.Account, len(records))
	for i, record := range records {
		items[i] = toDomainAccount(record, typeKeys)
	}
	return items, total, nil
}

func (r Repository) Get(ctx context.Context, accountID int64) (*accountdomain.Account, error) {
	var record database.AccountModel
	if err := r.db.WithContext(ctx).
		First(&record, "id = ?", accountID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get account: %w", err)
	}

	typeKeys, err := r.accountTypeKeysByID(ctx, []database.AccountModel{record})
	if err != nil {
		return nil, err
	}

	item := toDomainAccount(record, typeKeys)
	return &item, nil
}

func (r Repository) GetByTypeKeyAndIdentifier(ctx context.Context, typeKey string, identifier string) (*accountdomain.Account, error) {
	var record database.AccountModel
	if err := r.db.WithContext(ctx).
		Table(database.AccountModel{}.TableName()).
		Select("accounts.*").
		Joins("JOIN account_types ON account_types.id = accounts.account_type_id").
		Where("account_types.key = ? AND accounts.identifier = ?", typeKey, identifier).
		First(&record).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get account by identifier: %w", err)
	}

	typeKeys, err := r.accountTypeKeysByID(ctx, []database.AccountModel{record})
	if err != nil {
		return nil, err
	}

	item := toDomainAccount(record, typeKeys)
	return &item, nil
}

func (r Repository) Create(ctx context.Context, input accountdomain.CreateInput) (*accountdomain.Account, error) {
	specJSON, err := json.Marshal(dbutil.NormalizeMap(input.Spec))
	if err != nil {
		return nil, fmt.Errorf("marshal account spec: %w", err)
	}
	tagsJSON, err := json.Marshal(normalizeStrings(input.Tags))
	if err != nil {
		return nil, fmt.Errorf("marshal account tags: %w", err)
	}

	record := database.AccountModel{
		Identifier: input.Identifier,
		Status:     input.Status,
		TagsJSON:   database.JSONBytes(tagsJSON),
		SpecJSON:   database.JSONBytes(specJSON),
	}
	// Only set the FK when a real account type is specified; a zero value means
	// "no type" and should be stored as NULL, not as account_type_id = 0 which
	// would violate the foreign-key constraint.
	if input.AccountTypeID != 0 {
		v := input.AccountTypeID
		record.AccountTypeID = &v
	}

	if err := r.db.WithContext(ctx).Create(&record).Error; err != nil {
		return nil, fmt.Errorf("create account: %w", err)
	}

	return r.Get(ctx, record.ID)
}

func (r Repository) Patch(ctx context.Context, accountID int64, input accountdomain.PatchInput) (*accountdomain.Account, error) {
	current, err := r.Get(ctx, accountID)
	if err != nil {
		return nil, err
	}

	if input.Status != nil {
		current.Status = *input.Status
	}
	if input.Tags != nil {
		current.Tags = input.Tags
	}
	if input.Spec != nil {
		current.Spec = input.Spec
	}

	specJSON, err := json.Marshal(dbutil.NormalizeMap(current.Spec))
	if err != nil {
		return nil, fmt.Errorf("marshal account spec: %w", err)
	}
	tagsJSON, err := json.Marshal(normalizeStrings(current.Tags))
	if err != nil {
		return nil, fmt.Errorf("marshal account tags: %w", err)
	}

	result := r.db.WithContext(ctx).
		Model(&database.AccountModel{}).
		Where("id = ?", accountID).
		Updates(map[string]any{
			"status":    current.Status,
			"tags_json": tagsJSON,
			"spec_json": specJSON,
		})
	if result.Error != nil {
		return nil, fmt.Errorf("patch account: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return nil, ErrNotFound
	}

	return r.Get(ctx, accountID)
}

func (r Repository) UpdateStatus(ctx context.Context, accountID int64, status string) (*accountdomain.Account, error) {
	result := r.db.WithContext(ctx).
		Model(&database.AccountModel{}).
		Where("id = ?", accountID).
		Update("status", status)
	if result.Error != nil {
		return nil, fmt.Errorf("update account status: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return nil, ErrNotFound
	}

	return r.Get(ctx, accountID)
}

func (r Repository) Delete(ctx context.Context, accountID int64) error {
	result := r.db.WithContext(ctx).Delete(&database.AccountModel{}, accountID)
	if result.Error != nil {
		return fmt.Errorf("delete account: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (r Repository) accountTypeKeysByID(ctx context.Context, records []database.AccountModel) (map[int64]string, error) {
	ids := make([]int64, 0)
	seen := map[int64]struct{}{}
	for _, record := range records {
		if record.AccountTypeID == nil {
			continue
		}
		if _, ok := seen[*record.AccountTypeID]; ok {
			continue
		}
		seen[*record.AccountTypeID] = struct{}{}
		ids = append(ids, *record.AccountTypeID)
	}
	if len(ids) == 0 {
		return map[int64]string{}, nil
	}

	var types []database.AccountTypeModel
	if err := r.db.WithContext(ctx).
		Where("id IN ?", ids).
		Find(&types).Error; err != nil {
		return nil, fmt.Errorf("load account types: %w", err)
	}

	items := make(map[int64]string, len(types))
	for _, item := range types {
		items[item.ID] = item.Key
	}
	return items, nil
}

func toDomainAccount(record database.AccountModel, typeKeys map[int64]string) accountdomain.Account {
	item := accountdomain.Account{
		ID:         record.ID,
		Identifier: record.Identifier,
		Status:     record.Status,
		Tags:       decodeJSONStringArray([]byte(record.TagsJSON)),
		Spec:       dbutil.DecodeJSONMap([]byte(record.SpecJSON)),
		CreatedAt:  record.CreatedAt,
		UpdatedAt:  record.UpdatedAt,
	}
	if record.AccountTypeID != nil {
		v := *record.AccountTypeID
		item.AccountTypeID = &v
		item.AccountTypeKey = typeKeys[v]
	}
	return item
}

func decodeJSONStringArray(raw []byte) []string {
	if len(raw) == 0 {
		return []string{}
	}
	v := []string{}
	_ = json.Unmarshal(raw, &v)
	return v
}

func normalizeStrings(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
}
