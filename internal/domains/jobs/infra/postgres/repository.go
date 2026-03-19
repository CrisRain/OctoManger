package jobpostgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	jobdomain "octomanger/internal/domains/jobs/domain"
)

var ErrNotFound = errors.New("job resource not found")

type Repository struct {
	db *gorm.DB
}

func New(db *gorm.DB) Repository {
	return Repository{db: db}
}

func (r Repository) ListDefinitions(ctx context.Context) ([]jobdomain.JobDefinition, error) {
	rows, err := r.db.WithContext(ctx).Raw(`
		SELECT
			d.id, d.key, d.name, d.plugin_key, d.action, d.input_json, d.enabled, d.created_at, d.updated_at,
			s.id, s.cron_expression, s.timezone, s.next_run_at,
			s.lease_owner, s.lease_until, s.enabled, s.created_at, s.updated_at
		FROM job_definitions d
		LEFT JOIN schedules s ON s.job_definition_id = d.id
		ORDER BY d.created_at DESC`).Rows()
	if err != nil {
		return nil, fmt.Errorf("list job definitions: %w", err)
	}
	defer rows.Close()

	items := make([]jobdomain.JobDefinition, 0)
	for rows.Next() {
		item, err := scanDefinition(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r Repository) GetDefinition(ctx context.Context, definitionID int64) (*jobdomain.JobDefinition, error) {
	row := r.db.WithContext(ctx).Raw(`
		SELECT
			d.id, d.key, d.name, d.plugin_key, d.action, d.input_json, d.enabled, d.created_at, d.updated_at,
			s.id, s.cron_expression, s.timezone, s.next_run_at,
			s.lease_owner, s.lease_until, s.enabled, s.created_at, s.updated_at
		FROM job_definitions d
		LEFT JOIN schedules s ON s.job_definition_id = d.id
		WHERE d.id = $1`, definitionID).Row()
	item, err := scanDefinition(row)
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (r Repository) CreateDefinition(ctx context.Context, input jobdomain.CreateDefinitionInput, nextRunAt *time.Time) (*jobdomain.JobDefinition, error) {
	inputJSON, err := json.Marshal(normalizeMap(input.Input))
	if err != nil {
		return nil, fmt.Errorf("marshal definition input: %w", err)
	}

	var definitionID int64

	txErr := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		row := tx.Raw(`
			INSERT INTO job_definitions (key, name, plugin_key, action, input_json, enabled)
			VALUES ($1, $2, $3, $4, $5, TRUE)
			RETURNING id`,
			input.Key, input.Name, input.PluginKey, input.Action, inputJSON,
		).Row()
		if err := row.Scan(&definitionID); err != nil {
			return fmt.Errorf("insert job definition: %w", err)
		}

		if input.Schedule != nil {
			zone := input.Schedule.Timezone
			if zone == "" {
				zone = "UTC"
			}
			result := tx.Exec(`
				INSERT INTO schedules (job_definition_id, cron_expression, timezone, next_run_at, enabled)
				VALUES ($1, $2, $3, $4, $5)`,
				definitionID, input.Schedule.CronExpression, zone, nextRunAt, input.Schedule.Enabled,
			)
			if result.Error != nil {
				return fmt.Errorf("insert schedule: %w", result.Error)
			}
		}
		return nil
	})
	if txErr != nil {
		return nil, txErr
	}

	return r.GetDefinition(ctx, definitionID)
}

func (r Repository) EnqueueExecution(ctx context.Context, definitionID int64, requestedBy, source string, inputOverride map[string]any) (*jobdomain.JobExecution, error) {
	definition, err := r.GetDefinition(ctx, definitionID)
	if err != nil {
		return nil, err
	}

	executionInputJSON, err := json.Marshal(mergeMaps(definition.Input, inputOverride))
	if err != nil {
		return nil, fmt.Errorf("marshal execution input: %w", err)
	}

	var executionID int64
	row := r.db.WithContext(ctx).Raw(`
		INSERT INTO job_executions (job_definition_id, input_json, status, requested_by, source)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id`,
		definitionID, executionInputJSON, jobdomain.StatusQueued, requestedBy, source,
	).Row()
	if err := row.Scan(&executionID); err != nil {
		return nil, fmt.Errorf("insert job execution: %w", err)
	}

	return r.GetExecution(ctx, executionID)
}

func (r Repository) ListExecutions(ctx context.Context) ([]jobdomain.JobExecution, error) {
	rows, err := r.db.WithContext(ctx).Raw(baseExecutionQuery + ` ORDER BY e.created_at DESC`).Rows()
	if err != nil {
		return nil, fmt.Errorf("list job executions: %w", err)
	}
	defer rows.Close()

	items := make([]jobdomain.JobExecution, 0)
	for rows.Next() {
		item, err := scanExecution(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r Repository) GetExecution(ctx context.Context, executionID int64) (*jobdomain.JobExecution, error) {
	row := r.db.WithContext(ctx).Raw(baseExecutionQuery+` WHERE e.id = $1`, executionID).Row()
	item, err := scanExecution(row)
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (r Repository) ClaimNextQueuedExecution(ctx context.Context, workerID string) (*jobdomain.JobExecution, error) {
	row := r.db.WithContext(ctx).Raw(`
		WITH candidate AS (
			SELECT id FROM job_executions
			WHERE status = $1
			ORDER BY created_at
			FOR UPDATE SKIP LOCKED
			LIMIT 1
		)
		UPDATE job_executions e
		SET status = $2, worker_id = $3, started_at = NOW(), updated_at = NOW()
		FROM candidate
		WHERE e.id = candidate.id
		RETURNING e.id`,
		jobdomain.StatusQueued, jobdomain.StatusRunning, workerID,
	).Row()

	var executionID int64
	if err := row.Scan(&executionID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("claim queued execution: %w", err)
	}

	return r.GetExecution(ctx, executionID)
}

func (r Repository) FinishExecution(ctx context.Context, executionID int64, status, summary string, result map[string]any, errorMessage string) error {
	resultJSON, err := json.Marshal(normalizeMap(result))
	if err != nil {
		return fmt.Errorf("marshal execution result: %w", err)
	}

	res := r.db.WithContext(ctx).Exec(`
		UPDATE job_executions
		SET status = $2, summary = $3, result_json = $4, error_message = $5,
		    finished_at = NOW(), updated_at = NOW()
		WHERE id = $1`,
		executionID, status, summary, resultJSON, errorMessage,
	)
	if res.Error != nil {
		return fmt.Errorf("finish execution: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (r Repository) AppendLog(ctx context.Context, executionID int64, stream, eventType, message string, payload map[string]any) error {
	payloadJSON, err := json.Marshal(normalizeMap(payload))
	if err != nil {
		return fmt.Errorf("marshal job log payload: %w", err)
	}
	result := r.db.WithContext(ctx).Exec(`
		INSERT INTO job_logs (job_execution_id, stream, event_type, message, payload_json)
		VALUES ($1, $2, $3, $4, $5)`,
		executionID, stream, eventType, message, payloadJSON,
	)
	if result.Error != nil {
		return fmt.Errorf("append job log: %w", result.Error)
	}
	return nil
}

func (r Repository) ListLogsAfter(ctx context.Context, executionID, afterID int64) ([]jobdomain.JobLog, error) {
	rows, err := r.db.WithContext(ctx).Raw(`
		SELECT id, job_execution_id, stream, event_type, message, payload_json, created_at
		FROM job_logs
		WHERE job_execution_id = $1 AND id > $2
		ORDER BY id ASC
		LIMIT 200`, executionID, afterID).Rows()
	if err != nil {
		return nil, fmt.Errorf("list execution logs: %w", err)
	}
	defer rows.Close()

	items := make([]jobdomain.JobLog, 0)
	for rows.Next() {
		var item jobdomain.JobLog
		var payloadJSON []byte
		if err := rows.Scan(&item.ID, &item.JobExecutionID, &item.Stream, &item.EventType, &item.Message, &payloadJSON, &item.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan execution log: %w", err)
		}
		item.Payload = decodeJSONMap(payloadJSON)
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r Repository) ClaimNextDueSchedule(ctx context.Context, workerID string, leaseDuration time.Duration) (*jobdomain.Schedule, error) {
	row := r.db.WithContext(ctx).Raw(`
		WITH candidate AS (
			SELECT id FROM schedules
			WHERE enabled = TRUE AND next_run_at <= NOW()
				AND (lease_until IS NULL OR lease_until < NOW())
			ORDER BY next_run_at ASC
			FOR UPDATE SKIP LOCKED
			LIMIT 1
		)
		UPDATE schedules s
		SET lease_owner = $1, lease_until = NOW() + ($2::text)::interval, updated_at = NOW()
		FROM candidate
		WHERE s.id = candidate.id
		RETURNING s.id, s.job_definition_id, s.cron_expression, s.timezone, s.next_run_at,
			s.lease_owner, s.lease_until, s.enabled, s.created_at, s.updated_at`,
		workerID, formatInterval(leaseDuration),
	).Row()

	var item jobdomain.Schedule
	var leaseUntil sql.NullTime
	if err := row.Scan(
		&item.ID, &item.JobDefinitionID, &item.CronExpression, &item.Timezone, &item.NextRunAt,
		&item.LeaseOwner, &leaseUntil, &item.Enabled, &item.CreatedAt, &item.UpdatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("claim due schedule: %w", err)
	}

	item.LeaseUntil = nullableTime(leaseUntil)
	return &item, nil
}

func (r Repository) RescheduleAndEnqueue(ctx context.Context, schedule jobdomain.Schedule, nextRunAt time.Time, workerID string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if result := tx.Exec(`
			UPDATE schedules
			SET next_run_at = $2, lease_owner = NULL, lease_until = NULL, updated_at = NOW()
			WHERE id = $1`, schedule.ID, nextRunAt); result.Error != nil {
			return fmt.Errorf("update schedule next run: %w", result.Error)
		}

		if result := tx.Exec(`
			INSERT INTO job_executions (job_definition_id, input_json, status, requested_by, source, worker_id)
			SELECT id, input_json, $2, $3, $4, $5
			FROM job_definitions WHERE id = $1`,
			schedule.JobDefinitionID, jobdomain.StatusQueued, "system:scheduler", "schedule", workerID,
		); result.Error != nil {
			return fmt.Errorf("enqueue schedule execution: %w", result.Error)
		}

		return nil
	})
}

const baseExecutionQuery = `
	SELECT
		e.id, e.job_definition_id,
		d.key, d.name, d.plugin_key, d.action,
		e.input_json, e.status, e.requested_by, e.source,
		COALESCE(e.worker_id, ''), COALESCE(e.summary, ''),
		e.result_json, COALESCE(e.error_message, ''),
		e.started_at, e.finished_at, e.created_at, e.updated_at
	FROM job_executions e
	JOIN job_definitions d ON d.id = e.job_definition_id`

type scanner interface {
	Scan(dest ...any) error
}

func scanDefinition(row scanner) (jobdomain.JobDefinition, error) {
	var item jobdomain.JobDefinition
	var inputJSON []byte
	var scheduleID sql.NullInt64
	var cronExpr, timezone, leaseOwner sql.NullString
	var nextRunAt, leaseUntil, scheduleCreatedAt, scheduleUpdatedAt sql.NullTime
	var scheduleEnabled sql.NullBool

	if err := row.Scan(
		&item.ID, &item.Key, &item.Name, &item.PluginKey, &item.Action,
		&inputJSON, &item.Enabled, &item.CreatedAt, &item.UpdatedAt,
		&scheduleID, &cronExpr, &timezone, &nextRunAt,
		&leaseOwner, &leaseUntil, &scheduleEnabled,
		&scheduleCreatedAt, &scheduleUpdatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return jobdomain.JobDefinition{}, ErrNotFound
		}
		return jobdomain.JobDefinition{}, fmt.Errorf("scan job definition: %w", err)
	}

	item.Input = decodeJSONMap(inputJSON)
	if scheduleID.Valid {
		item.Schedule = &jobdomain.Schedule{
			ID:              scheduleID.Int64,
			JobDefinitionID: item.ID,
			CronExpression:  cronExpr.String,
			Timezone:        timezone.String,
			NextRunAt:       nextRunAt.Time,
			LeaseOwner:      leaseOwner.String,
			LeaseUntil:      nullableTime(leaseUntil),
			Enabled:         scheduleEnabled.Bool,
			CreatedAt:       scheduleCreatedAt.Time,
			UpdatedAt:       scheduleUpdatedAt.Time,
		}
	}
	return item, nil
}

func scanExecution(row scanner) (jobdomain.JobExecution, error) {
	var item jobdomain.JobExecution
	var inputJSON, resultJSON []byte
	var startedAt, finishedAt sql.NullTime

	if err := row.Scan(
		&item.ID, &item.JobDefinitionID,
		&item.DefinitionKey, &item.DefinitionName, &item.PluginKey, &item.Action,
		&inputJSON, &item.Status, &item.RequestedBy, &item.Source,
		&item.WorkerID, &item.Summary,
		&resultJSON, &item.ErrorMessage,
		&startedAt, &finishedAt, &item.CreatedAt, &item.UpdatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return jobdomain.JobExecution{}, ErrNotFound
		}
		return jobdomain.JobExecution{}, fmt.Errorf("scan job execution: %w", err)
	}

	item.Input = decodeJSONMap(inputJSON)
	item.Result = decodeJSONMap(resultJSON)
	item.StartedAt = nullableTime(startedAt)
	item.FinishedAt = nullableTime(finishedAt)
	return item, nil
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

func mergeMaps(base, override map[string]any) map[string]any {
	merged := map[string]any{}
	for k, v := range normalizeMap(base) {
		merged[k] = v
	}
	for k, v := range normalizeMap(override) {
		merged[k] = v
	}
	return merged
}

func nullableTime(v sql.NullTime) *time.Time {
	if !v.Valid {
		return nil
	}
	t := v.Time
	return &t
}

func formatInterval(d time.Duration) string {
	return fmt.Sprintf("%f seconds", d.Seconds())
}
