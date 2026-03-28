package runtime

import (
	"context"
	"errors"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"octomanger/internal/platform/apikey"
	"octomanger/internal/platform/config"
	"octomanger/internal/platform/database"
)

const runtimeSystemSettingsID int64 = 1

// ensurePluginInternalAPIToken guarantees the plugin internal API token is
// available to both API and worker processes, using DB state as shared source.
func ensurePluginInternalAPIToken(ctx context.Context, db *gorm.DB, cfg *config.Config) error {
	if db == nil || cfg == nil {
		return nil
	}

	// Old schemas (before migration) should not fail process boot.
	if !db.Migrator().HasTable(database.SystemSettingsModel{}.TableName()) {
		return nil
	}
	if !db.Migrator().HasColumn(&database.SystemSettingsModel{}, "PluginInternalAPIToken") {
		return nil
	}

	configured := strings.TrimSpace(cfg.Auth.PluginInternalAPIToken)
	if configured != "" {
		if err := upsertPluginInternalAPIToken(ctx, db, configured, true); err != nil {
			return err
		}
		cfg.Auth.PluginInternalAPIToken = configured
		return nil
	}

	stored, err := loadPluginInternalAPIToken(ctx, db)
	if err != nil {
		return err
	}
	if stored != "" {
		cfg.Auth.PluginInternalAPIToken = stored
		return nil
	}

	generated, err := apikey.Generate()
	if err != nil {
		return err
	}
	if err := upsertPluginInternalAPIToken(ctx, db, generated, false); err != nil {
		return err
	}

	stored, err = loadPluginInternalAPIToken(ctx, db)
	if err != nil {
		return err
	}
	if stored == "" {
		stored = generated
	}
	cfg.Auth.PluginInternalAPIToken = stored
	return nil
}

func loadPluginInternalAPIToken(ctx context.Context, db *gorm.DB) (string, error) {
	var row struct {
		Token string `gorm:"column:plugin_internal_api_token"`
	}
	err := db.WithContext(ctx).
		Table(database.SystemSettingsModel{}.TableName()).
		Select("plugin_internal_api_token").
		Where("id = ?", runtimeSystemSettingsID).
		Take(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", nil
		}
		return "", err
	}
	return strings.TrimSpace(row.Token), nil
}

func upsertPluginInternalAPIToken(
	ctx context.Context,
	db *gorm.DB,
	token string,
	overwrite bool,
) error {
	token = strings.TrimSpace(token)
	if token == "" {
		return nil
	}

	record := database.SystemSettingsModel{
		ID:                       runtimeSystemSettingsID,
		AppName:                  "OctoManager",
		JobDefaultTimeoutMinutes: 30,
		JobMaxConcurrency:        10,
		AdminKeyHash:             "",
		PluginInternalAPIToken:   token,
	}

	updateTokenExpr := gorm.Expr(
		"CASE WHEN COALESCE(system_settings.plugin_internal_api_token, '') = '' THEN EXCLUDED.plugin_internal_api_token ELSE system_settings.plugin_internal_api_token END",
	)
	if overwrite {
		updateTokenExpr = gorm.Expr("EXCLUDED.plugin_internal_api_token")
	}

	return db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "id"}},
			DoUpdates: clause.Assignments(map[string]any{
				"plugin_internal_api_token": updateTokenExpr,
				"updated_at":                nowExpr(db),
			}),
		}).
		Create(&record).Error
}

func nowExpr(db *gorm.DB) clause.Expr {
	if db == nil {
		return gorm.Expr("NOW()")
	}
	switch strings.ToLower(db.Dialector.Name()) {
	case "sqlite":
		return gorm.Expr("CURRENT_TIMESTAMP")
	default:
		return gorm.Expr("NOW()")
	}
}
