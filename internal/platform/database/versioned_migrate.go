package database

import (
	"context"
	"errors"
	"fmt"
	"math"
	"slices"
	"strconv"
	"strings"

	"gorm.io/gorm"

	platformconfig "octomanger/internal/platform/config"
)

const (
	MigrationModeVersioned = "versioned"
	MigrationModeAuto      = "auto"
)

type schemaMigrationRecord struct {
	Version int64  `gorm:"column:version"`
	Name    string `gorm:"column:name"`
}

type schemaMigrationRawRecord struct {
	Version any `gorm:"column:version"`
}

func (schemaMigrationRecord) TableName() string {
	return "schema_migrations"
}

type versionedMigration struct {
	Version int64
	Name    string
	Up      []string
	Down    []string
}

type migrationDeps struct {
	ensureSchema   func(context.Context, *gorm.DB) error
	loadApplied    func(context.Context, *gorm.DB) (map[int64]struct{}, error)
	applyMigration func(context.Context, *gorm.DB, versionedMigration) error
	migrateLegacy  func(context.Context, *gorm.DB) error
	seedSystem     func(context.Context, *gorm.DB) error
	seedServices   func(context.Context, *gorm.DB, *platformconfig.Config) error
}

func defaultMigrationDeps() migrationDeps {
	return migrationDeps{
		ensureSchema:   ensureSchemaMigrationTable,
		loadApplied:    loadAppliedVersions,
		applyMigration: applyMigration,
		migrateLegacy:  migrateLegacySystemConfigs,
		seedSystem:     seedSystemSettings,
		seedServices:   seedPluginServiceConfigs,
	}
}

func Migrate(ctx context.Context, db *gorm.DB, cfgs ...*platformconfig.Config) error {
	cfg := firstConfig(cfgs...)
	switch migrationMode(cfg) {
	case MigrationModeVersioned:
		return runVersionedMigrations(ctx, db, cfg)
	case MigrationModeAuto:
		return AutoMigrate(ctx, db, cfg)
	default:
		return fmt.Errorf("unknown migration mode %q", migrationMode(cfg))
	}
}

func RollbackLastMigration(ctx context.Context, db *gorm.DB, cfgs ...*platformconfig.Config) error {
	cfg := firstConfig(cfgs...)
	if migrationMode(cfg) != MigrationModeVersioned {
		return fmt.Errorf("rollback is only supported when migration mode is %q", MigrationModeVersioned)
	}
	return rollbackLastVersionedMigration(ctx, db)
}

func runVersionedMigrations(ctx context.Context, db *gorm.DB, cfg *platformconfig.Config) error {
	return runVersionedMigrationsWith(ctx, db, cfg, versionedMigrations())
}

func runVersionedMigrationsWith(ctx context.Context, db *gorm.DB, cfg *platformconfig.Config, migrations []versionedMigration) error {
	return runVersionedMigrationsWithDeps(ctx, db, cfg, migrations, defaultMigrationDeps())
}

func runVersionedMigrationsWithDeps(
	ctx context.Context,
	db *gorm.DB,
	cfg *platformconfig.Config,
	migrations []versionedMigration,
	deps migrationDeps,
) error {
	if err := deps.ensureSchema(ctx, db); err != nil {
		return err
	}

	applied, err := deps.loadApplied(ctx, db)
	if err != nil {
		return err
	}

	for _, migration := range migrations {
		if _, ok := applied[migration.Version]; ok {
			continue
		}
		if err := deps.applyMigration(ctx, db, migration); err != nil {
			return err
		}
	}

	if err := deps.migrateLegacy(ctx, db); err != nil {
		return err
	}
	if err := deps.seedSystem(ctx, db); err != nil {
		return err
	}
	if err := deps.seedServices(ctx, db, cfg); err != nil {
		return err
	}

	return nil
}

func rollbackLastVersionedMigration(ctx context.Context, db *gorm.DB) error {
	return rollbackLastVersionedMigrationWith(ctx, db, versionedMigrations())
}

func rollbackLastVersionedMigrationWith(ctx context.Context, db *gorm.DB, migrations []versionedMigration) error {
	if err := ensureSchemaMigrationTable(ctx, db); err != nil {
		return err
	}
	versionType, err := schemaMigrationVersionColumnType(ctx, db)
	if err != nil {
		return err
	}

	var current schemaMigrationRecord
	if err := db.WithContext(ctx).
		Table(current.TableName()).
		Order("version DESC").
		Limit(1).
		Take(&current).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return fmt.Errorf("load last schema migration: %w", err)
	}

	var migration *versionedMigration
	for _, item := range migrations {
		if item.Version == current.Version {
			itemCopy := item
			migration = &itemCopy
			break
		}
	}
	if migration == nil {
		return fmt.Errorf("migration version %d is applied but not found in code", current.Version)
	}

	tx := db.WithContext(ctx).Begin()
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	for _, statement := range migration.Down {
		stmt := strings.TrimSpace(statement)
		if stmt == "" {
			continue
		}
		if err := tx.Exec(stmt).Error; err != nil {
			return fmt.Errorf("rollback migration %d (%s): %w", migration.Version, migration.Name, err)
		}
	}

	if err := tx.Table(current.TableName()).
		Where("version = ?", schemaMigrationVersionValue(migration.Version, versionType)).
		Delete(&schemaMigrationRecord{}).Error; err != nil {
		return fmt.Errorf("delete schema migration record %d: %w", migration.Version, err)
	}

	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("commit rollback migration %d: %w", migration.Version, err)
	}
	committed = true
	return nil
}

func ensureSchemaMigrationTable(ctx context.Context, db *gorm.DB) error {
	if err := db.WithContext(ctx).Exec(`
CREATE TABLE IF NOT EXISTS schema_migrations (
	version BIGINT PRIMARY KEY,
	name TEXT NOT NULL,
	applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
)`).Error; err != nil {
		return err
	}

	compatStatements := []string{
		`ALTER TABLE schema_migrations ADD COLUMN IF NOT EXISTS name TEXT`,
		`ALTER TABLE schema_migrations ADD COLUMN IF NOT EXISTS applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()`,
		`UPDATE schema_migrations SET name = COALESCE(name, '') WHERE name IS NULL`,
	}
	for _, statement := range compatStatements {
		if err := db.WithContext(ctx).Exec(statement).Error; err != nil {
			return fmt.Errorf("ensure schema_migrations compatibility: %w", err)
		}
	}
	return nil
}

func loadAppliedVersions(ctx context.Context, db *gorm.DB) (map[int64]struct{}, error) {
	var rows []map[string]any
	if err := db.WithContext(ctx).
		Table(schemaMigrationRecord{}.TableName()).
		Select("version").
		Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("load applied schema migrations: %w", err)
	}
	records := make([]schemaMigrationRawRecord, 0, len(rows))
	for _, row := range rows {
		records = append(records, schemaMigrationRawRecord{Version: row["version"]})
	}
	return parseAppliedVersions(records), nil
}

func applyMigration(ctx context.Context, db *gorm.DB, migration versionedMigration) error {
	tx := db.WithContext(ctx).Begin()
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	versionType, err := schemaMigrationVersionColumnType(ctx, tx)
	if err != nil {
		return err
	}

	for _, statement := range migration.Up {
		stmt := strings.TrimSpace(statement)
		if stmt == "" {
			continue
		}
		if err := tx.Exec(stmt).Error; err != nil {
			return fmt.Errorf("apply migration %d (%s): %w", migration.Version, migration.Name, err)
		}
	}

	if err := tx.Table(schemaMigrationRecord{}.TableName()).
		Create(map[string]any{
			"version": schemaMigrationVersionValue(migration.Version, versionType),
			"name":    migration.Name,
		}).Error; err != nil {
		return fmt.Errorf("record migration %d (%s): %w", migration.Version, migration.Name, err)
	}

	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("commit migration %d (%s): %w", migration.Version, migration.Name, err)
	}
	committed = true
	return nil
}

func schemaMigrationVersionColumnType(ctx context.Context, db *gorm.DB) (string, error) {
	var row struct {
		DataType string `gorm:"column:data_type"`
	}
	err := db.WithContext(ctx).Raw(`
SELECT data_type
FROM information_schema.columns
WHERE table_schema = current_schema()
  AND table_name = ?
  AND column_name = 'version'
LIMIT 1
`, schemaMigrationRecord{}.TableName()).Scan(&row).Error
	if err != nil {
		return "", fmt.Errorf("query schema_migrations version column type: %w", err)
	}

	dataType := strings.ToLower(strings.TrimSpace(row.DataType))
	if dataType == "" {
		return "bigint", nil
	}
	return dataType, nil
}

func schemaMigrationVersionValue(version int64, columnType string) any {
	switch strings.ToLower(strings.TrimSpace(columnType)) {
	case "text", "character varying", "character", "varchar":
		return strconv.FormatInt(version, 10)
	default:
		return version
	}
}

func migrationMode(cfg *platformconfig.Config) string {
	if cfg == nil {
		return MigrationModeVersioned
	}
	mode := strings.ToLower(strings.TrimSpace(cfg.Database.MigrationMode))
	if mode == "" {
		return MigrationModeVersioned
	}
	if mode == "automigrate" {
		return MigrationModeAuto
	}
	return mode
}

func versionedMigrations() []versionedMigration {
	items := []versionedMigration{
		{
			Version: 1,
			Name:    "initial_schema",
			Up:      initialSchemaUpStatements(),
			Down:    initialSchemaDownStatements(),
		},
		{
			Version: 2,
			Name:    "system_settings_admin_key_hash",
			Up:      systemSettingsAdminKeyHashUpStatements(),
			Down:    systemSettingsAdminKeyHashDownStatements(),
		},
		{
			Version: 3,
			Name:    "system_settings_plugin_internal_api_token",
			Up:      systemSettingsPluginInternalAPITokenUpStatements(),
			Down:    systemSettingsPluginInternalAPITokenDownStatements(),
		},
	}
	slices.SortFunc(items, func(a, b versionedMigration) int {
		switch {
		case a.Version < b.Version:
			return -1
		case a.Version > b.Version:
			return 1
		default:
			return 0
		}
	})
	return items
}

func systemSettingsAdminKeyHashUpStatements() []string {
	return []string{
		`ALTER TABLE system_settings
			ADD COLUMN IF NOT EXISTS admin_key_hash TEXT NOT NULL DEFAULT ''`,
	}
}

func systemSettingsAdminKeyHashDownStatements() []string {
	return []string{
		`ALTER TABLE system_settings
			DROP COLUMN IF EXISTS admin_key_hash`,
	}
}

func systemSettingsPluginInternalAPITokenUpStatements() []string {
	return []string{
		`ALTER TABLE system_settings
			ADD COLUMN IF NOT EXISTS plugin_internal_api_token TEXT NOT NULL DEFAULT ''`,
	}
}

func systemSettingsPluginInternalAPITokenDownStatements() []string {
	return []string{
		`ALTER TABLE system_settings
			DROP COLUMN IF EXISTS plugin_internal_api_token`,
	}
}

func initialSchemaUpStatements() []string {
	return []string{
		`CREATE TABLE IF NOT EXISTS account_types (
			id BIGSERIAL PRIMARY KEY,
			key TEXT NOT NULL UNIQUE,
			name TEXT NOT NULL,
			category TEXT NOT NULL DEFAULT 'generic',
			schema_json JSONB NOT NULL DEFAULT '{}'::jsonb,
			capabilities_json JSONB NOT NULL DEFAULT '{}'::jsonb,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS accounts (
			id BIGSERIAL PRIMARY KEY,
			account_type_id BIGINT NULL,
			identifier TEXT NOT NULL,
			spec_json JSONB NOT NULL DEFAULT '{}'::jsonb,
			status TEXT NOT NULL DEFAULT 'active',
			tags_json JSONB NOT NULL DEFAULT '[]'::jsonb,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS accounts_account_type_id_identifier_key
			ON accounts(account_type_id, identifier)`,
		`CREATE TABLE IF NOT EXISTS email_accounts (
			id BIGSERIAL PRIMARY KEY,
			provider TEXT NOT NULL,
			address TEXT NOT NULL UNIQUE,
			status TEXT NOT NULL DEFAULT 'active',
			config_json JSONB NOT NULL DEFAULT '{}'::jsonb,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS job_definitions (
			id BIGSERIAL PRIMARY KEY,
			key TEXT NOT NULL UNIQUE,
			name TEXT NOT NULL,
			plugin_key TEXT NOT NULL,
			action TEXT NOT NULL,
			input_json JSONB NOT NULL DEFAULT '{}'::jsonb,
			enabled BOOLEAN NOT NULL DEFAULT TRUE,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS schedules (
			id BIGSERIAL PRIMARY KEY,
			job_definition_id BIGINT NOT NULL UNIQUE,
			cron_expression TEXT NOT NULL,
			timezone TEXT NOT NULL DEFAULT 'UTC',
			next_run_at TIMESTAMPTZ NULL,
			lease_owner TEXT NULL,
			lease_until TIMESTAMPTZ NULL,
			enabled BOOLEAN NOT NULL DEFAULT TRUE,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_schedules_next_run_at
			ON schedules(next_run_at) WHERE enabled = TRUE`,
		`CREATE TABLE IF NOT EXISTS job_executions (
			id BIGSERIAL PRIMARY KEY,
			job_definition_id BIGINT NOT NULL,
			input_json JSONB NOT NULL DEFAULT '{}'::jsonb,
			status TEXT NOT NULL,
			requested_by TEXT NOT NULL DEFAULT '',
			source TEXT NOT NULL DEFAULT 'manual',
			worker_id TEXT NULL,
			summary TEXT NULL,
			result_json JSONB NOT NULL DEFAULT '{}'::jsonb,
			error_message TEXT NULL,
			started_at TIMESTAMPTZ NULL,
			finished_at TIMESTAMPTZ NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_job_executions_status_created_at
			ON job_executions(status, created_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_job_executions_def_id_created
			ON job_executions(job_definition_id, created_at DESC)`,
		`CREATE TABLE IF NOT EXISTS job_logs (
			id BIGSERIAL PRIMARY KEY,
			job_execution_id BIGINT NOT NULL,
			stream TEXT NOT NULL,
			event_type TEXT NOT NULL,
			message TEXT NOT NULL,
			payload_json JSONB NOT NULL DEFAULT '{}'::jsonb,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_job_logs_execution_id_id
			ON job_logs(job_execution_id, id)`,
		`CREATE TABLE IF NOT EXISTS triggers (
			id BIGSERIAL PRIMARY KEY,
			key TEXT NOT NULL UNIQUE,
			name TEXT NOT NULL,
			job_definition_id BIGINT NOT NULL,
			mode TEXT NOT NULL DEFAULT 'async',
			default_input_json JSONB NOT NULL DEFAULT '{}'::jsonb,
			token_hash TEXT NOT NULL,
			token_prefix TEXT NOT NULL,
			enabled BOOLEAN NOT NULL DEFAULT TRUE,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS agents (
			id BIGSERIAL PRIMARY KEY,
			name TEXT NOT NULL,
			plugin_key TEXT NOT NULL,
			action TEXT NOT NULL,
			input_json JSONB NOT NULL DEFAULT '{}'::jsonb,
			desired_state TEXT NOT NULL DEFAULT 'stopped',
			runtime_state TEXT NOT NULL DEFAULT 'idle',
			last_error TEXT NULL,
			last_heartbeat_at TIMESTAMPTZ NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS agent_logs (
			id BIGSERIAL PRIMARY KEY,
			agent_id BIGINT NOT NULL,
			event_type TEXT NOT NULL,
			message TEXT NOT NULL,
			payload_json JSONB NOT NULL DEFAULT '{}'::jsonb,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_agent_logs_agent_id_id
			ON agent_logs(agent_id, id)`,
		`CREATE TABLE IF NOT EXISTS system_settings (
			id BIGINT PRIMARY KEY,
			app_name TEXT NOT NULL DEFAULT 'OctoManager',
			job_default_timeout_minutes INTEGER NOT NULL DEFAULT 30,
			job_max_concurrency INTEGER NOT NULL DEFAULT 10,
			admin_key_hash TEXT NOT NULL DEFAULT '',
			plugin_internal_api_token TEXT NOT NULL DEFAULT '',
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS plugin_settings (
			plugin_key TEXT PRIMARY KEY,
			settings_json JSONB NOT NULL DEFAULT '{}'::jsonb,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS plugin_service_configs (
			plugin_key TEXT PRIMARY KEY,
			grpc_address TEXT NOT NULL DEFAULT '',
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`UPDATE accounts
			SET account_type_id = NULL
			WHERE account_type_id IS NOT NULL
				AND NOT EXISTS (SELECT 1 FROM account_types WHERE account_types.id = accounts.account_type_id)`,
		`DELETE FROM schedules
			WHERE NOT EXISTS (SELECT 1 FROM job_definitions WHERE job_definitions.id = schedules.job_definition_id)`,
		`DELETE FROM job_executions
			WHERE NOT EXISTS (SELECT 1 FROM job_definitions WHERE job_definitions.id = job_executions.job_definition_id)`,
		`DELETE FROM job_logs
			WHERE NOT EXISTS (SELECT 1 FROM job_executions WHERE job_executions.id = job_logs.job_execution_id)`,
		`DELETE FROM triggers
			WHERE NOT EXISTS (SELECT 1 FROM job_definitions WHERE job_definitions.id = triggers.job_definition_id)`,
		`DELETE FROM agent_logs
			WHERE NOT EXISTS (SELECT 1 FROM agents WHERE agents.id = agent_logs.agent_id)`,
		`DO $$
		BEGIN
			IF NOT EXISTS (
				SELECT 1 FROM pg_constraint WHERE conname = 'fk_accounts_account_type_id'
			) THEN
				ALTER TABLE accounts
					ADD CONSTRAINT fk_accounts_account_type_id
					FOREIGN KEY (account_type_id)
					REFERENCES account_types(id)
					ON DELETE SET NULL;
			END IF;
		END$$`,
		`DO $$
		BEGIN
			IF NOT EXISTS (
				SELECT 1 FROM pg_constraint WHERE conname = 'fk_schedules_job_definition_id'
			) THEN
				ALTER TABLE schedules
					ADD CONSTRAINT fk_schedules_job_definition_id
					FOREIGN KEY (job_definition_id)
					REFERENCES job_definitions(id)
					ON DELETE CASCADE;
			END IF;
		END$$`,
		`DO $$
		BEGIN
			IF NOT EXISTS (
				SELECT 1 FROM pg_constraint WHERE conname = 'fk_job_executions_job_definition_id'
			) THEN
				ALTER TABLE job_executions
					ADD CONSTRAINT fk_job_executions_job_definition_id
					FOREIGN KEY (job_definition_id)
					REFERENCES job_definitions(id)
					ON DELETE CASCADE;
			END IF;
		END$$`,
		`DO $$
		BEGIN
			IF NOT EXISTS (
				SELECT 1 FROM pg_constraint WHERE conname = 'fk_job_logs_job_execution_id'
			) THEN
				ALTER TABLE job_logs
					ADD CONSTRAINT fk_job_logs_job_execution_id
					FOREIGN KEY (job_execution_id)
					REFERENCES job_executions(id)
					ON DELETE CASCADE;
			END IF;
		END$$`,
		`DO $$
		BEGIN
			IF NOT EXISTS (
				SELECT 1 FROM pg_constraint WHERE conname = 'fk_triggers_job_definition_id'
			) THEN
				ALTER TABLE triggers
					ADD CONSTRAINT fk_triggers_job_definition_id
					FOREIGN KEY (job_definition_id)
					REFERENCES job_definitions(id)
					ON DELETE CASCADE;
			END IF;
		END$$`,
		`DO $$
		BEGIN
			IF NOT EXISTS (
				SELECT 1 FROM pg_constraint WHERE conname = 'fk_agent_logs_agent_id'
			) THEN
				ALTER TABLE agent_logs
					ADD CONSTRAINT fk_agent_logs_agent_id
					FOREIGN KEY (agent_id)
					REFERENCES agents(id)
					ON DELETE CASCADE;
			END IF;
		END$$`,
	}
}

func initialSchemaDownStatements() []string {
	return []string{
		`DROP TABLE IF EXISTS system_logs`,
		`DROP TABLE IF EXISTS plugin_service_configs`,
		`DROP TABLE IF EXISTS plugin_settings`,
		`DROP TABLE IF EXISTS system_settings`,
		`DROP TABLE IF EXISTS agent_logs`,
		`DROP TABLE IF EXISTS agents`,
		`DROP TABLE IF EXISTS triggers`,
		`DROP TABLE IF EXISTS job_logs`,
		`DROP TABLE IF EXISTS job_executions`,
		`DROP TABLE IF EXISTS schedules`,
		`DROP TABLE IF EXISTS job_definitions`,
		`DROP TABLE IF EXISTS email_accounts`,
		`DROP TABLE IF EXISTS accounts`,
		`DROP TABLE IF EXISTS account_types`,
	}
}

func parseAppliedVersions(records []schemaMigrationRawRecord) map[int64]struct{} {
	items := make(map[int64]struct{}, len(records))
	legacyMarkerFound := false

	for _, item := range records {
		if version, ok := parseAppliedVersionValue(item.Version); ok {
			items[version] = struct{}{}
			continue
		}
		if isLegacyVersionMarker(item.Version) {
			legacyMarkerFound = true
		}
	}

	// Compatibility: legacy deployments may store filename-style migration
	// markers (e.g. 0001_v2_core.sql) in schema_migrations.version.
	// These environments already have the base schema, so treat the current
	// initial migration (version 1) as applied.
	if legacyMarkerFound {
		items[1] = struct{}{}
	}

	return items
}

func parseAppliedVersionValue(value any) (int64, bool) {
	switch v := value.(type) {
	case int:
		return int64(v), true
	case int8:
		return int64(v), true
	case int16:
		return int64(v), true
	case int32:
		return int64(v), true
	case int64:
		return v, true
	case uint:
		if uint64(v) > math.MaxInt64 {
			return 0, false
		}
		return int64(v), true
	case uint8:
		return int64(v), true
	case uint16:
		return int64(v), true
	case uint32:
		return int64(v), true
	case uint64:
		if v > math.MaxInt64 {
			return 0, false
		}
		return int64(v), true
	case float32:
		f := float64(v)
		if math.Trunc(f) != f || f < math.MinInt64 || f > math.MaxInt64 {
			return 0, false
		}
		return int64(f), true
	case float64:
		if math.Trunc(v) != v || v < math.MinInt64 || v > math.MaxInt64 {
			return 0, false
		}
		return int64(v), true
	case []byte:
		return parseStrictIntString(string(v))
	case string:
		return parseStrictIntString(v)
	default:
		return 0, false
	}
}

func parseStrictIntString(raw string) (int64, bool) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return 0, false
	}
	version, err := strconv.ParseInt(trimmed, 10, 64)
	if err != nil {
		return 0, false
	}
	return version, true
}

func isLegacyVersionMarker(value any) bool {
	raw, ok := value.(string)
	if !ok {
		if bytes, isBytes := value.([]byte); isBytes {
			raw = string(bytes)
			ok = true
		}
	}
	if !ok {
		return false
	}

	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return false
	}

	hasDigit := false
	hasNonDigit := false
	for _, r := range trimmed {
		if r >= '0' && r <= '9' {
			hasDigit = true
			continue
		}
		hasNonDigit = true
	}
	return hasDigit && hasNonDigit
}
