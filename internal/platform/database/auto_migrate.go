package database

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	platformconfig "octomanger/internal/platform/config"
)

const systemSettingsSingletonID int64 = 1

func AutoMigrate(ctx context.Context, db *gorm.DB, cfgs ...*platformconfig.Config) error {
	if err := db.WithContext(ctx).AutoMigrate(
		&AccountTypeModel{},
		&AccountModel{},
		&EmailAccountModel{},
		&JobDefinitionModel{},
		&ScheduleModel{},
		&JobExecutionModel{},
		&JobLogModel{},
		&TriggerModel{},
		&AgentModel{},
		&AgentLogModel{},
		&SystemSettingsModel{},
		&PluginSettingsModel{},
		&PluginServiceConfigModel{},
	); err != nil {
		return fmt.Errorf("auto migrate database schema: %w", err)
	}

	if err := migrateLegacySystemConfigs(ctx, db); err != nil {
		return err
	}
	if err := seedSystemSettings(ctx, db); err != nil {
		return err
	}
	if err := seedPluginServiceConfigs(ctx, db, firstConfig(cfgs...)); err != nil {
		return err
	}
	return nil
}

func seedSystemSettings(ctx context.Context, db *gorm.DB) error {
	item := defaultSystemSettingsSeed()
	return db.WithContext(ctx).
		Clauses(clause.OnConflict{DoNothing: true}).
		Create(&item).Error
}

func seedPluginServiceConfigs(ctx context.Context, db *gorm.DB, cfg *platformconfig.Config) error {
	for _, item := range defaultPluginServiceSeeds(cfg) {
		if err := db.WithContext(ctx).
			Clauses(clause.OnConflict{DoNothing: true}).
			Create(&item).Error; err != nil {
			return fmt.Errorf("seed plugin service config %s: %w", item.PluginKey, err)
		}
	}
	return nil
}

func migrateLegacySystemConfigs(ctx context.Context, db *gorm.DB) error {
	if !db.Migrator().HasTable("system_configs") {
		return nil
	}

	if err := migrateLegacySystemSettings(ctx, db); err != nil {
		return err
	}
	if err := migrateLegacyPluginServiceConfigs(ctx, db); err != nil {
		return err
	}
	if err := migrateLegacyPluginSettings(ctx, db); err != nil {
		return err
	}
	return nil
}

func migrateLegacySystemSettings(ctx context.Context, db *gorm.DB) error {
	var existing int64
	if err := db.WithContext(ctx).
		Model(&SystemSettingsModel{}).
		Where("id = ?", systemSettingsSingletonID).
		Count(&existing).Error; err != nil {
		return fmt.Errorf("count system settings rows: %w", err)
	}
	if existing > 0 {
		return nil
	}

	item := defaultSystemSettingsSeed()
	var records []legacySystemConfigRecord
	if err := db.WithContext(ctx).
		Table("system_configs").
		Select("key", "value_json").
		Where("key IN ?", []string{"app.name", "job.default_timeout_minutes", "job.max_concurrency"}).
		Find(&records).Error; err != nil {
		return fmt.Errorf("query legacy system settings: %w", err)
	}

	for _, record := range records {
		switch record.Key {
		case "app.name":
			if parsed := decodeLegacyString(record.ValueJSON); parsed != "" {
				item.AppName = parsed
			}
		case "job.default_timeout_minutes":
			if parsed, ok := decodeLegacyInt(record.ValueJSON); ok {
				item.JobDefaultTimeoutMinutes = parsed
			}
		case "job.max_concurrency":
			if parsed, ok := decodeLegacyInt(record.ValueJSON); ok {
				item.JobMaxConcurrency = parsed
			}
		}
	}

	return db.WithContext(ctx).
		Clauses(clause.OnConflict{DoNothing: true}).
		Create(&item).Error
}

func migrateLegacyPluginServiceConfigs(ctx context.Context, db *gorm.DB) error {
	var record legacySystemConfigValueRecord
	if err := db.WithContext(ctx).
		Table("system_configs").
		Select("value_json").
		Where("key = ?", "plugins.grpc_services").
		Take(&record).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return fmt.Errorf("query legacy plugin service config: %w", err)
	}

	for key, address := range decodeLegacyPluginServices(record.ValueJSON) {
		if err := db.WithContext(ctx).
			Clauses(clause.OnConflict{DoNothing: true}).
			Create(&PluginServiceConfigModel{
				PluginKey:   key,
				GRPCAddress: address,
			}).Error; err != nil {
			return fmt.Errorf("migrate legacy plugin service config %s: %w", key, err)
		}
	}
	return nil
}

func migrateLegacyPluginSettings(ctx context.Context, db *gorm.DB) error {
	var records []legacySystemConfigRecord
	if err := db.WithContext(ctx).
		Table("system_configs").
		Select("key", "value_json").
		Where("key LIKE ?", "plugin_settings:%").
		Find(&records).Error; err != nil {
		return fmt.Errorf("query legacy plugin settings: %w", err)
	}

	for _, record := range records {
		pluginKey := normalizePluginKey(strings.TrimPrefix(record.Key, "plugin_settings:"))
		if pluginKey == "" {
			continue
		}
		value := record.ValueJSON
		if len(value) == 0 || !json.Valid(value) {
			value = json.RawMessage("{}")
		}

		if err := db.WithContext(ctx).
			Clauses(clause.OnConflict{DoNothing: true}).
			Create(&PluginSettingsModel{
				PluginKey:    pluginKey,
				SettingsJSON: JSONBytes(value),
			}).Error; err != nil {
			return fmt.Errorf("migrate legacy plugin settings %s: %w", pluginKey, err)
		}
	}
	return nil
}

func defaultSystemSettingsSeed() SystemSettingsModel {
	return SystemSettingsModel{
		ID:                       systemSettingsSingletonID,
		AppName:                  "OctoManager",
		JobDefaultTimeoutMinutes: 30,
		JobMaxConcurrency:        10,
		AdminKeyHash:             "",
		PluginInternalAPIToken:   "",
	}
}

func defaultPluginServiceSeeds(cfg *platformconfig.Config) []PluginServiceConfigModel {
	services := platformconfig.DefaultPluginServices()
	if cfg != nil && len(cfg.Plugins.Services) > 0 {
		services = cfg.Plugins.Services
	}

	items := make([]PluginServiceConfigModel, 0, len(services))
	for key, entry := range services {
		normalizedKey := normalizePluginKey(key)
		address := strings.TrimSpace(entry.Address)
		if normalizedKey == "" || address == "" {
			continue
		}
		items = append(items, PluginServiceConfigModel{
			PluginKey:   normalizedKey,
			GRPCAddress: address,
		})
	}
	return items
}

func decodeLegacyString(raw []byte) string {
	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		return ""
	}
	return strings.TrimSpace(value)
}

func decodeLegacyInt(raw []byte) (int, bool) {
	var value int
	if err := json.Unmarshal(raw, &value); err == nil {
		return value, true
	}

	var floatValue float64
	if err := json.Unmarshal(raw, &floatValue); err == nil {
		return int(floatValue), true
	}
	return 0, false
}

func decodeLegacyPluginServices(raw []byte) map[string]string {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" || trimmed == "null" || trimmed == "{}" {
		return map[string]string{}
	}

	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		return map[string]string{}
	}

	items := make(map[string]string, len(payload))
	for key, value := range payload {
		normalizedKey := normalizePluginKey(key)
		if normalizedKey == "" {
			continue
		}

		switch item := value.(type) {
		case string:
			address := strings.TrimSpace(item)
			if address != "" {
				items[normalizedKey] = address
			}
		case map[string]any:
			address := strings.TrimSpace(asString(item["address"]))
			if address == "" {
				address = strings.TrimSpace(asString(item["Address"]))
			}
			if address != "" {
				items[normalizedKey] = address
			}
		}
	}
	return items
}

func asString(value any) string {
	if text, ok := value.(string); ok {
		return text
	}
	return ""
}

type legacySystemConfigRecord struct {
	Key       string `gorm:"column:key"`
	ValueJSON []byte `gorm:"column:value_json"`
}

type legacySystemConfigValueRecord struct {
	ValueJSON []byte `gorm:"column:value_json"`
}

func firstConfig(cfgs ...*platformconfig.Config) *platformconfig.Config {
	for _, cfg := range cfgs {
		if cfg != nil {
			return cfg
		}
	}
	return nil
}
