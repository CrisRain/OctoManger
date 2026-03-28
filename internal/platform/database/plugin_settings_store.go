package database

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type PluginSettingsStore struct {
	db *gorm.DB
}

func NewPluginSettingsStore(db *gorm.DB) PluginSettingsStore {
	return PluginSettingsStore{db: db}
}

func (s PluginSettingsStore) GetSettings(ctx context.Context, pluginKey string) (json.RawMessage, error) {
	normalizedKey := normalizePluginKey(pluginKey)
	if normalizedKey == "" {
		return nil, errors.New("plugin key is required")
	}

	var record PluginSettingsModel
	if err := s.db.WithContext(ctx).
		First(&record, "plugin_key = ?", normalizedKey).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return json.RawMessage("{}"), nil
		}
		return nil, err
	}
	value := record.SettingsJSON
	if len(value) == 0 {
		return json.RawMessage("{}"), nil
	}
	return json.RawMessage(value), nil
}

func (s PluginSettingsStore) SetSettings(ctx context.Context, pluginKey string, value json.RawMessage) error {
	normalizedKey := normalizePluginKey(pluginKey)
	if normalizedKey == "" {
		return errors.New("plugin key is required")
	}
	if len(value) == 0 || !json.Valid(value) {
		return errors.New("settings must be valid JSON")
	}

	return s.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "plugin_key"}},
			DoUpdates: clause.AssignmentColumns([]string{"settings_json", "updated_at"}),
		}).
		Create(&PluginSettingsModel{
			PluginKey:    normalizedKey,
			SettingsJSON: JSONBytes(value),
		}).Error
}

type PluginServiceConfigStore struct {
	db *gorm.DB
}

func NewPluginServiceConfigStore(db *gorm.DB) PluginServiceConfigStore {
	return PluginServiceConfigStore{db: db}
}

func (s PluginServiceConfigStore) ListGRPCAddresses(ctx context.Context) (map[string]string, error) {
	var records []PluginServiceConfigModel
	if err := s.db.WithContext(ctx).
		Order("plugin_key ASC").
		Find(&records).Error; err != nil {
		return nil, err
	}

	items := map[string]string{}
	for _, record := range records {
		normalizedKey := normalizePluginKey(record.PluginKey)
		normalizedAddress := strings.TrimSpace(record.GRPCAddress)
		if normalizedKey == "" || normalizedAddress == "" {
			continue
		}
		items[normalizedKey] = normalizedAddress
	}
	return items, nil
}

func (s PluginServiceConfigStore) GetGRPCAddress(ctx context.Context, pluginKey string) (string, error) {
	normalizedKey := normalizePluginKey(pluginKey)
	if normalizedKey == "" {
		return "", errors.New("plugin key is required")
	}

	var record PluginServiceConfigModel
	if err := s.db.WithContext(ctx).
		First(&record, "plugin_key = ?", normalizedKey).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", nil
		}
		return "", err
	}
	return strings.TrimSpace(record.GRPCAddress), nil
}

func (s PluginServiceConfigStore) SetGRPCAddress(ctx context.Context, pluginKey string, address string) error {
	normalizedKey := normalizePluginKey(pluginKey)
	normalizedAddress := strings.TrimSpace(address)
	if normalizedKey == "" {
		return errors.New("plugin key is required")
	}
	if normalizedAddress == "" {
		return errors.New("grpc address is required")
	}

	return s.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "plugin_key"}},
			DoUpdates: clause.AssignmentColumns([]string{"grpc_address", "updated_at"}),
		}).
		Create(&PluginServiceConfigModel{
			PluginKey:   normalizedKey,
			GRPCAddress: normalizedAddress,
		}).Error
}

func normalizePluginKey(value string) string {
	return strings.ToLower(strings.ReplaceAll(strings.TrimSpace(value), "-", "_"))
}
