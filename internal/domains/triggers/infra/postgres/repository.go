package triggerpostgres

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"gorm.io/gorm"

	triggerdomain "octomanger/internal/domains/triggers/domain"
	"octomanger/internal/platform/database"
	"octomanger/internal/platform/dbutil"
)

var ErrNotFound = errors.New("trigger not found")

type Repository struct {
	db *gorm.DB
}

func New(db *gorm.DB) Repository {
	return Repository{db: db}
}

func (r Repository) List(ctx context.Context) ([]triggerdomain.Trigger, error) {
	items, _, err := r.ListPage(ctx, 0, 0)
	return items, err
}

func (r Repository) ListPage(ctx context.Context, limit int, offset int) ([]triggerdomain.Trigger, int64, error) {
	var total int64
	if err := r.db.WithContext(ctx).
		Model(&database.TriggerModel{}).
		Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count triggers: %w", err)
	}

	var records []database.TriggerModel
	query := r.db.WithContext(ctx).Order("created_at DESC")
	if limit > 0 {
		query = query.Limit(limit).Offset(offset)
	}
	if err := query.Find(&records).Error; err != nil {
		return nil, 0, fmt.Errorf("list triggers: %w", err)
	}

	items := make([]triggerdomain.Trigger, len(records))
	for i, record := range records {
		items[i] = toDomainTrigger(record)
	}
	return items, total, nil
}

func (r Repository) GetByID(ctx context.Context, triggerID int64) (*triggerdomain.Trigger, error) {
	var record database.TriggerModel
	if err := r.db.WithContext(ctx).
		First(&record, "id = ?", triggerID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get trigger: %w", err)
	}

	item := toDomainTrigger(record)
	return &item, nil
}

func (r Repository) GetByKey(ctx context.Context, key string) (*triggerdomain.Trigger, string, error) {
	var record database.TriggerModel
	if err := r.db.WithContext(ctx).
		First(&record, "key = ?", key).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, "", ErrNotFound
		}
		return nil, "", fmt.Errorf("get trigger by key: %w", err)
	}

	item := toDomainTrigger(record)
	return &item, record.TokenHash, nil
}

func (r Repository) Create(ctx context.Context, input triggerdomain.CreateInput, token string) (*triggerdomain.Trigger, error) {
	defaultInputJSON, err := json.Marshal(dbutil.NormalizeMap(input.DefaultInput))
	if err != nil {
		return nil, fmt.Errorf("marshal trigger default input: %w", err)
	}

	tokenHash := hashToken(token)
	tokenPrefix := token
	if len(tokenPrefix) > 8 {
		tokenPrefix = tokenPrefix[:8]
	}

	record := database.TriggerModel{
		Key:              input.Key,
		Name:             input.Name,
		JobDefinitionID:  input.JobDefinitionID,
		Mode:             input.Mode,
		DefaultInputJSON: database.JSONBytes(defaultInputJSON),
		TokenHash:        tokenHash,
		TokenPrefix:      tokenPrefix,
		Enabled:          input.Enabled,
	}
	if err := r.db.WithContext(ctx).Create(&record).Error; err != nil {
		return nil, fmt.Errorf("create trigger: %w", err)
	}

	item := toDomainTrigger(record)
	return &item, nil
}

func (r Repository) Patch(ctx context.Context, triggerID int64, input triggerdomain.PatchTriggerInput) (*triggerdomain.Trigger, error) {
	current, err := r.GetByID(ctx, triggerID)
	if err != nil {
		return nil, err
	}
	if input.Name != nil {
		current.Name = *input.Name
	}
	if input.JobDefinitionID != nil {
		current.JobDefinitionID = *input.JobDefinitionID
	}
	if input.Mode != nil {
		current.Mode = *input.Mode
	}
	if input.DefaultInput != nil {
		current.DefaultInput = input.DefaultInput
	}
	if input.Enabled != nil {
		current.Enabled = *input.Enabled
	}

	defaultInputJSON, err := json.Marshal(dbutil.NormalizeMap(current.DefaultInput))
	if err != nil {
		return nil, fmt.Errorf("marshal trigger default input: %w", err)
	}

	result := r.db.WithContext(ctx).
		Model(&database.TriggerModel{}).
		Where("id = ?", triggerID).
		Updates(map[string]any{
			"name":               current.Name,
			"job_definition_id":  current.JobDefinitionID,
			"mode":               current.Mode,
			"default_input_json": defaultInputJSON,
			"enabled":            current.Enabled,
		})
	if result.Error != nil {
		return nil, fmt.Errorf("patch trigger: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return nil, ErrNotFound
	}

	return r.GetByID(ctx, triggerID)
}

func (r Repository) Delete(ctx context.Context, triggerID int64) error {
	result := r.db.WithContext(ctx).Delete(&database.TriggerModel{}, triggerID)
	if result.Error != nil {
		return fmt.Errorf("delete trigger: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func VerifyToken(token, hash string) bool {
	return hashToken(token) == hash
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(token)))
	return hex.EncodeToString(sum[:])
}

func toDomainTrigger(record database.TriggerModel) triggerdomain.Trigger {
	return triggerdomain.Trigger{
		ID:              record.ID,
		Key:             record.Key,
		Name:            record.Name,
		JobDefinitionID: record.JobDefinitionID,
		Mode:            record.Mode,
		DefaultInput:    dbutil.DecodeJSONMap([]byte(record.DefaultInputJSON)),
		TokenPrefix:     record.TokenPrefix,
		Enabled:         record.Enabled,
		CreatedAt:       record.CreatedAt,
		UpdatedAt:       record.UpdatedAt,
	}
}
