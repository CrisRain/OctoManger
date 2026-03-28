package systemapp

import (
	"context"
	"errors"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"octomanger/internal/platform/database"
)

const systemSettingsSingletonID int64 = 1

type Config struct {
	AppName                  string `json:"app_name"`
	JobDefaultTimeoutMinutes int    `json:"job_default_timeout_minutes"`
	JobMaxConcurrency        int    `json:"job_max_concurrency"`
}

func DefaultConfig() Config {
	return Config{
		AppName:                  "OctoManager",
		JobDefaultTimeoutMinutes: 30,
		JobMaxConcurrency:        10,
	}
}

func (s Service) GetConfig(ctx context.Context) (Config, error) {
	item := DefaultConfig()

	var record database.SystemSettingsModel
	if err := s.db.WithContext(ctx).
		First(&record, "id = ?", systemSettingsSingletonID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return item, nil
		}
		return Config{}, err
	}

	return normalizeConfig(Config{
		AppName:                  record.AppName,
		JobDefaultTimeoutMinutes: record.JobDefaultTimeoutMinutes,
		JobMaxConcurrency:        record.JobMaxConcurrency,
	}), nil
}

func (s Service) SetConfig(ctx context.Context, item Config) (Config, error) {
	item = normalizeConfig(item)
	if item.JobDefaultTimeoutMinutes < 0 {
		return Config{}, errors.New("job_default_timeout_minutes must be >= 0")
	}
	if item.JobMaxConcurrency < 0 {
		return Config{}, errors.New("job_max_concurrency must be >= 0")
	}

	record := database.SystemSettingsModel{
		ID:                       systemSettingsSingletonID,
		AppName:                  item.AppName,
		JobDefaultTimeoutMinutes: item.JobDefaultTimeoutMinutes,
		JobMaxConcurrency:        item.JobMaxConcurrency,
	}
	if err := s.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "id"}},
			DoUpdates: clause.AssignmentColumns([]string{"app_name", "job_default_timeout_minutes", "job_max_concurrency", "updated_at"}),
		}).
		Create(&record).Error; err != nil {
		return Config{}, err
	}

	return item, nil
}

func normalizeConfig(item Config) Config {
	item.AppName = strings.TrimSpace(item.AppName)
	if item.AppName == "" {
		item.AppName = DefaultConfig().AppName
	}
	return item
}
