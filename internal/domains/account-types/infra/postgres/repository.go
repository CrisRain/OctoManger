package accounttypepostgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	accounttypedomain "octomanger/internal/domains/account-types/domain"
	"octomanger/internal/platform/database"
	"octomanger/internal/platform/dbutil"
)

var ErrNotFound = errors.New("account type not found")

type Repository struct {
	db *gorm.DB
}

func New(db *gorm.DB) Repository {
	return Repository{db: db}
}

func (r Repository) List(ctx context.Context) ([]accounttypedomain.AccountType, error) {
	items, _, err := r.ListPage(ctx, 0, 0)
	return items, err
}

func (r Repository) ListPage(ctx context.Context, limit int, offset int) ([]accounttypedomain.AccountType, int64, error) {
	var total int64
	if err := r.db.WithContext(ctx).
		Model(&database.AccountTypeModel{}).
		Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count account types: %w", err)
	}

	var records []database.AccountTypeModel
	query := r.db.WithContext(ctx).Order("created_at DESC")
	if limit > 0 {
		query = query.Limit(limit).Offset(offset)
	}
	if err := query.Find(&records).Error; err != nil {
		return nil, 0, fmt.Errorf("list account types: %w", err)
	}

	items := make([]accounttypedomain.AccountType, len(records))
	for i, record := range records {
		items[i] = toDomainAccountType(record)
	}
	return items, total, nil
}

func (r Repository) GetByKey(ctx context.Context, key string) (*accounttypedomain.AccountType, error) {
	var record database.AccountTypeModel
	if err := r.db.WithContext(ctx).
		First(&record, "key = ?", key).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get account type: %w", err)
	}

	item := toDomainAccountType(record)
	return &item, nil
}

func (r Repository) Create(ctx context.Context, input accounttypedomain.CreateInput) (*accounttypedomain.AccountType, error) {
	record, err := newAccountTypeRecord(input)
	if err != nil {
		return nil, err
	}
	if err := r.db.WithContext(ctx).Create(&record).Error; err != nil {
		return nil, fmt.Errorf("create account type: %w", err)
	}

	item := toDomainAccountType(record)
	return &item, nil
}

func (r Repository) Patch(ctx context.Context, key string, input accounttypedomain.PatchInput) (*accounttypedomain.AccountType, error) {
	current, err := r.GetByKey(ctx, key)
	if err != nil {
		return nil, err
	}

	if input.Name != nil {
		current.Name = *input.Name
	}
	if input.Category != nil {
		current.Category = *input.Category
	}
	if input.Schema != nil {
		current.Schema = input.Schema
	}
	if input.Capabilities != nil {
		current.Capabilities = input.Capabilities
	}

	schemaJSON, err := json.Marshal(dbutil.NormalizeMap(current.Schema))
	if err != nil {
		return nil, fmt.Errorf("marshal schema: %w", err)
	}
	capabilitiesJSON, err := json.Marshal(dbutil.NormalizeMap(current.Capabilities))
	if err != nil {
		return nil, fmt.Errorf("marshal capabilities: %w", err)
	}

	result := r.db.WithContext(ctx).
		Model(&database.AccountTypeModel{}).
		Where("key = ?", key).
		Updates(map[string]any{
			"name":              current.Name,
			"category":          current.Category,
			"schema_json":       schemaJSON,
			"capabilities_json": capabilitiesJSON,
		})
	if result.Error != nil {
		return nil, fmt.Errorf("patch account type: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return nil, ErrNotFound
	}

	return r.GetByKey(ctx, key)
}

func (r Repository) Upsert(ctx context.Context, input accounttypedomain.CreateInput) (*accounttypedomain.AccountType, error) {
	record, err := newAccountTypeRecord(input)
	if err != nil {
		return nil, err
	}

	if err := r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "key"}},
			DoUpdates: clause.AssignmentColumns([]string{"name", "category", "schema_json", "capabilities_json", "updated_at"}),
		}).
		Create(&record).Error; err != nil {
		return nil, fmt.Errorf("upsert account type: %w", err)
	}

	return r.GetByKey(ctx, input.Key)
}

func (r Repository) Delete(ctx context.Context, key string) error {
	result := r.db.WithContext(ctx).
		Where("key = ?", key).
		Delete(&database.AccountTypeModel{})
	if result.Error != nil {
		return fmt.Errorf("delete account type: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func newAccountTypeRecord(input accounttypedomain.CreateInput) (database.AccountTypeModel, error) {
	schemaJSON, err := json.Marshal(dbutil.NormalizeMap(input.Schema))
	if err != nil {
		return database.AccountTypeModel{}, fmt.Errorf("marshal schema: %w", err)
	}
	capabilitiesJSON, err := json.Marshal(dbutil.NormalizeMap(input.Capabilities))
	if err != nil {
		return database.AccountTypeModel{}, fmt.Errorf("marshal capabilities: %w", err)
	}

	return database.AccountTypeModel{
		Key:              input.Key,
		Name:             input.Name,
		Category:         input.Category,
		SchemaJSON:       database.JSONBytes(schemaJSON),
		CapabilitiesJSON: database.JSONBytes(capabilitiesJSON),
	}, nil
}

func toDomainAccountType(record database.AccountTypeModel) accounttypedomain.AccountType {
	return accounttypedomain.AccountType{
		ID:           record.ID,
		Key:          record.Key,
		Name:         record.Name,
		Category:     record.Category,
		Schema:       dbutil.DecodeJSONMap([]byte(record.SchemaJSON)),
		Capabilities: dbutil.DecodeJSONMap([]byte(record.CapabilitiesJSON)),
		CreatedAt:    record.CreatedAt,
		UpdatedAt:    record.UpdatedAt,
	}
}
