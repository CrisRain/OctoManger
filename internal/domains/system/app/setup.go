package systemapp

import (
	"context"
	"errors"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"octomanger/internal/platform/apikey"
	"octomanger/internal/platform/database"
)

var ErrAlreadyInitialized = errors.New("system is already initialized")

type SetupStatus struct {
	NeedsSetup bool `json:"needs_setup"`
}

type SetupInitializeResult struct {
	APIKey string `json:"api_key"`
	Config Config `json:"config"`
}

func (s Service) SetupStatus(ctx context.Context) (SetupStatus, error) {
	configured, err := s.isAdminKeyConfigured(ctx)
	if err != nil {
		return SetupStatus{}, err
	}
	return SetupStatus{NeedsSetup: !configured}, nil
}

// VerifyAdminKey returns whether an API key is configured and whether the
// provided key matches the configured key.
func (s Service) VerifyAdminKey(ctx context.Context, providedKey string) (bool, bool, error) {
	storedHash, err := s.adminKeyHash(ctx)
	if err != nil {
		return false, false, err
	}
	if storedHash == "" {
		return false, false, nil
	}
	return true, apikey.Match(providedKey, storedHash), nil
}

func (s Service) Initialize(ctx context.Context, item Config) (SetupInitializeResult, error) {
	if s.db == nil {
		return SetupInitializeResult{}, errors.New("database is not configured")
	}

	item = normalizeConfig(item)
	if item.JobDefaultTimeoutMinutes < 0 {
		return SetupInitializeResult{}, errors.New("job_default_timeout_minutes must be >= 0")
	}
	if item.JobMaxConcurrency < 0 {
		return SetupInitializeResult{}, errors.New("job_max_concurrency must be >= 0")
	}

	generatedKey, err := apikey.Generate()
	if err != nil {
		return SetupInitializeResult{}, err
	}
	generatedHash := apikey.Hash(generatedKey)

	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Acquire a session-level advisory lock to serialize concurrent
		// initialization requests.  The lock is released automatically when
		// the transaction ends, so there is no risk of forgetting to release it.
		if err := tx.Exec("SELECT pg_advisory_xact_lock(1)").Error; err != nil {
			return err
		}

		var existing database.SystemSettingsModel
		err := tx.
			First(&existing, "id = ?", systemSettingsSingletonID).Error
		switch {
		case err == nil:
			if strings.TrimSpace(existing.AdminKeyHash) != "" {
				return ErrAlreadyInitialized
			}
		case errors.Is(err, gorm.ErrRecordNotFound):
			// no-op
		default:
			return err
		}

		record := database.SystemSettingsModel{
			ID:                       systemSettingsSingletonID,
			AppName:                  item.AppName,
			JobDefaultTimeoutMinutes: item.JobDefaultTimeoutMinutes,
			JobMaxConcurrency:        item.JobMaxConcurrency,
			AdminKeyHash:             generatedHash,
		}
		return tx.
			Clauses(clause.OnConflict{
				Columns: []clause.Column{{Name: "id"}},
				DoUpdates: clause.AssignmentColumns([]string{
					"app_name",
					"job_default_timeout_minutes",
					"job_max_concurrency",
					"admin_key_hash",
					"updated_at",
				}),
			}).
			Create(&record).Error
	})
	if err != nil {
		return SetupInitializeResult{}, err
	}

	return SetupInitializeResult{
		APIKey: generatedKey,
		Config: item,
	}, nil
}

func (s Service) isAdminKeyConfigured(ctx context.Context) (bool, error) {
	hash, err := s.adminKeyHash(ctx)
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(hash) != "", nil
}

func (s Service) adminKeyHash(ctx context.Context) (string, error) {
	if s.db == nil {
		return "", nil
	}

	var record database.SystemSettingsModel
	if err := s.db.WithContext(ctx).
		Select("admin_key_hash").
		First(&record, "id = ?", systemSettingsSingletonID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", nil
		}
		return "", err
	}
	return strings.TrimSpace(record.AdminKeyHash), nil
}
