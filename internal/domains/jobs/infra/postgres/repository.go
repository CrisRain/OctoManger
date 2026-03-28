package jobpostgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	jobdomain "octomanger/internal/domains/jobs/domain"
	"octomanger/internal/platform/database"
	"octomanger/internal/platform/dbutil"
)

var ErrNotFound = errors.New("job resource not found")

type Repository struct {
	db  *gorm.DB
	rdb *redis.Client
}

func New(db *gorm.DB, rdb ...*redis.Client) Repository {
	var client *redis.Client
	if len(rdb) > 0 {
		client = rdb[0]
	}
	return Repository{db: db, rdb: client}
}

func (r Repository) ListDefinitions(ctx context.Context) ([]jobdomain.JobDefinition, error) {
	items, _, err := r.ListDefinitionsPage(ctx, 0, 0)
	return items, err
}

func (r Repository) ListDefinitionsPage(ctx context.Context, limit int, offset int) ([]jobdomain.JobDefinition, int64, error) {
	var total int64
	if err := r.db.WithContext(ctx).
		Model(&database.JobDefinitionModel{}).
		Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count job definitions: %w", err)
	}

	var records []database.JobDefinitionModel
	query := r.db.WithContext(ctx).Order("created_at DESC")
	if limit > 0 {
		query = query.Limit(limit).Offset(offset)
	}
	if err := query.Find(&records).Error; err != nil {
		return nil, 0, fmt.Errorf("list job definitions: %w", err)
	}

	schedules, err := r.loadSchedulesByDefinitionID(ctx, definitionIDs(records))
	if err != nil {
		return nil, 0, err
	}

	items := make([]jobdomain.JobDefinition, len(records))
	for i, record := range records {
		items[i] = toDomainDefinition(record, schedules[record.ID])
	}
	return items, total, nil
}

func (r Repository) GetDefinition(ctx context.Context, definitionID int64) (*jobdomain.JobDefinition, error) {
	var record database.JobDefinitionModel
	if err := r.db.WithContext(ctx).
		First(&record, "id = ?", definitionID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get job definition: %w", err)
	}

	schedule, err := r.loadScheduleByDefinitionID(ctx, definitionID)
	if err != nil {
		return nil, err
	}

	item := toDomainDefinition(record, schedule)
	return &item, nil
}

func (r Repository) CreateDefinition(ctx context.Context, input jobdomain.CreateDefinitionInput, nextRunAt *time.Time) (*jobdomain.JobDefinition, error) {
	inputJSON, err := json.Marshal(dbutil.NormalizeMap(input.Input))
	if err != nil {
		return nil, fmt.Errorf("marshal definition input: %w", err)
	}

	record := database.JobDefinitionModel{
		Key:       input.Key,
		Name:      input.Name,
		PluginKey: input.PluginKey,
		Action:    input.Action,
		InputJSON: database.JSONBytes(inputJSON),
		Enabled:   true,
	}

	if err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&record).Error; err != nil {
			return fmt.Errorf("insert job definition: %w", err)
		}

		if input.Schedule != nil {
			zone := input.Schedule.Timezone
			if zone == "" {
				zone = "UTC"
			}
			schedule := database.ScheduleModel{
				JobDefinitionID: record.ID,
				CronExpression:  input.Schedule.CronExpression,
				Timezone:        zone,
				NextRunAt:       nextRunAt,
				Enabled:         input.Schedule.Enabled,
			}
			if err := tx.Create(&schedule).Error; err != nil {
				return fmt.Errorf("insert schedule: %w", err)
			}
		}
		return nil
	}); err != nil {
		return nil, err
	}

	return r.GetDefinition(ctx, record.ID)
}

func (r Repository) PatchDefinition(ctx context.Context, id int64, input jobdomain.PatchDefinitionInput) (*jobdomain.JobDefinition, error) {
	current, err := r.GetDefinition(ctx, id)
	if err != nil {
		return nil, err
	}

	if input.Name != nil {
		current.Name = *input.Name
	}
	if input.PluginKey != nil {
		current.PluginKey = *input.PluginKey
	}
	if input.Action != nil {
		current.Action = *input.Action
	}
	if input.Input != nil {
		current.Input = input.Input
	}
	if input.Enabled != nil {
		current.Enabled = *input.Enabled
	}

	inputJSON, err := json.Marshal(dbutil.NormalizeMap(current.Input))
	if err != nil {
		return nil, fmt.Errorf("marshal definition input: %w", err)
	}

	if err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&database.JobDefinitionModel{}).
			Where("id = ?", id).
			Updates(map[string]any{
				"name":       current.Name,
				"plugin_key": current.PluginKey,
				"action":     current.Action,
				"input_json": inputJSON,
				"enabled":    current.Enabled,
			})
		if result.Error != nil {
			return fmt.Errorf("patch job definition: %w", result.Error)
		}
		if result.RowsAffected == 0 {
			return ErrNotFound
		}

		if input.Schedule != nil {
			zone := input.Schedule.Timezone
			if zone == "" {
				zone = "UTC"
			}

			var existing database.ScheduleModel
			err := tx.First(&existing, "job_definition_id = ?", id).Error
			switch {
			case errors.Is(err, gorm.ErrRecordNotFound):
				now := time.Now().UTC()
				schedule := database.ScheduleModel{
					JobDefinitionID: id,
					CronExpression:  input.Schedule.CronExpression,
					Timezone:        zone,
					NextRunAt:       &now,
					Enabled:         input.Schedule.Enabled,
				}
				if err := tx.Create(&schedule).Error; err != nil {
					return fmt.Errorf("create schedule: %w", err)
				}
			case err != nil:
				return fmt.Errorf("load schedule: %w", err)
			default:
				if err := tx.Model(&database.ScheduleModel{}).
					Where("job_definition_id = ?", id).
					Updates(map[string]any{
						"cron_expression": input.Schedule.CronExpression,
						"timezone":        zone,
						"enabled":         input.Schedule.Enabled,
					}).Error; err != nil {
					return fmt.Errorf("update schedule: %w", err)
				}
			}
		}
		return nil
	}); err != nil {
		return nil, err
	}

	return r.GetDefinition(ctx, id)
}

func (r Repository) DeleteDefinition(ctx context.Context, id int64) error {
	var executionIDs []int64

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&database.JobExecutionModel{}).
			Where("job_definition_id = ?", id).
			Pluck("id", &executionIDs).Error; err != nil {
			return fmt.Errorf("list executions for delete: %w", err)
		}

		if err := tx.Where("job_definition_id = ?", id).
			Delete(&database.ScheduleModel{}).Error; err != nil {
			return fmt.Errorf("delete schedules for job definition: %w", err)
		}

		if err := tx.Where("job_definition_id = ?", id).
			Delete(&database.TriggerModel{}).Error; err != nil {
			return fmt.Errorf("delete triggers for job definition: %w", err)
		}

		if len(executionIDs) > 0 {
			if err := tx.Where("job_execution_id IN ?", executionIDs).
				Delete(&database.JobLogModel{}).Error; err != nil {
				return fmt.Errorf("delete logs for job definition: %w", err)
			}
		}

		if err := tx.Where("job_definition_id = ?", id).
			Delete(&database.JobExecutionModel{}).Error; err != nil {
			return fmt.Errorf("delete executions for job definition: %w", err)
		}

		result := tx.Delete(&database.JobDefinitionModel{}, id)
		if result.Error != nil {
			return fmt.Errorf("delete job definition: %w", result.Error)
		}
		if result.RowsAffected == 0 {
			return ErrNotFound
		}
		return nil
	})
	if err != nil {
		return err
	}

	if r.rdb != nil {
		_ = r.rdb.Del(ctx, r.executionsCacheKey()).Err()
	}
	for _, executionID := range executionIDs {
		r.invalidateExecutionCache(ctx, executionID)
		if r.rdb != nil {
			_ = r.rdb.Del(ctx, r.executionLogsCacheKey(executionID)).Err()
		}
	}
	return nil
}

func (r Repository) EnqueueExecution(ctx context.Context, definitionID int64, requestedBy, source string, inputOverride map[string]any) (*jobdomain.JobExecution, error) {
	definition, err := r.GetDefinition(ctx, definitionID)
	if err != nil {
		return nil, err
	}

	executionInputJSON, err := json.Marshal(dbutil.MergeMaps(definition.Input, inputOverride))
	if err != nil {
		return nil, fmt.Errorf("marshal execution input: %w", err)
	}

	record := database.JobExecutionModel{
		JobDefinitionID: definitionID,
		InputJSON:       database.JSONBytes(executionInputJSON),
		Status:          jobdomain.StatusQueued,
		RequestedBy:     requestedBy,
		Source:          source,
	}
	if err := r.db.WithContext(ctx).Create(&record).Error; err != nil {
		return nil, fmt.Errorf("insert job execution: %w", err)
	}

	exec := toDomainExecution(record, executionDefinition{
		Key:       definition.Key,
		Name:      definition.Name,
		PluginKey: definition.PluginKey,
		Action:    definition.Action,
	})
	r.invalidateExecutionCache(ctx, exec.ID)
	r.writeExecutionCache(ctx, &exec)
	return &exec, nil
}

func (r Repository) ListExecutions(ctx context.Context) ([]jobdomain.JobExecution, error) {
	if items, ok := r.readExecutionsCache(ctx); ok {
		return items, nil
	}

	var records []database.JobExecutionModel
	if err := r.db.WithContext(ctx).
		Order("created_at DESC").
		Find(&records).Error; err != nil {
		return nil, fmt.Errorf("list job executions: %w", err)
	}

	definitions, err := r.loadExecutionDefinitionsByID(ctx, executionDefinitionIDs(records))
	if err != nil {
		return nil, err
	}

	items := make([]jobdomain.JobExecution, len(records))
	for i, record := range records {
		items[i] = toDomainExecution(record, definitions[record.JobDefinitionID])
	}
	r.writeExecutionsCache(ctx, items)
	return items, nil
}

func (r Repository) ListExecutionsPage(ctx context.Context, limit int, offset int) ([]jobdomain.JobExecution, int64, error) {
	var total int64
	if err := r.db.WithContext(ctx).
		Model(&database.JobExecutionModel{}).
		Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count job executions: %w", err)
	}

	var records []database.JobExecutionModel
	query := r.db.WithContext(ctx).Order("created_at DESC")
	if limit > 0 {
		query = query.Limit(limit).Offset(offset)
	}
	if err := query.Find(&records).Error; err != nil {
		return nil, 0, fmt.Errorf("list job executions: %w", err)
	}

	definitions, err := r.loadExecutionDefinitionsByID(ctx, executionDefinitionIDs(records))
	if err != nil {
		return nil, 0, err
	}

	items := make([]jobdomain.JobExecution, len(records))
	for i, record := range records {
		items[i] = toDomainExecution(record, definitions[record.JobDefinitionID])
	}
	return items, total, nil
}

func (r Repository) GetExecution(ctx context.Context, executionID int64) (*jobdomain.JobExecution, error) {
	if item, ok := r.readExecutionCache(ctx, executionID); ok {
		return item, nil
	}

	var record database.JobExecutionModel
	if err := r.db.WithContext(ctx).
		First(&record, "id = ?", executionID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get job execution: %w", err)
	}

	definitions, err := r.loadExecutionDefinitionsByID(ctx, []int64{record.JobDefinitionID})
	if err != nil {
		return nil, err
	}

	item := toDomainExecution(record, definitions[record.JobDefinitionID])
	r.writeExecutionCache(ctx, &item)
	return &item, nil
}

func (r Repository) ClaimNextQueuedExecution(ctx context.Context, workerID string) (*jobdomain.JobExecution, error) {
	var executionID int64

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var records []database.JobExecutionModel
		query := tx
		if lock := lockingClause(tx); lock != nil {
			query = query.Clauses(lock)
		}
		if err := query.
			Where("status = ?", jobdomain.StatusQueued).
			Order("created_at ASC").
			Limit(1).
			Find(&records).Error; err != nil {
			return fmt.Errorf("claim queued execution: %w", err)
		}
		if len(records) == 0 {
			return nil
		}

		executionID = records[0].ID
		startedAt := time.Now().UTC()
		if err := tx.Model(&database.JobExecutionModel{}).
			Where("id = ?", executionID).
			Updates(map[string]any{
				"status":     jobdomain.StatusRunning,
				"worker_id":  workerID,
				"started_at": startedAt,
			}).Error; err != nil {
			return fmt.Errorf("update queued execution: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if executionID == 0 {
		return nil, nil
	}

	r.invalidateExecutionCache(ctx, executionID)
	return r.GetExecution(ctx, executionID)
}

func (r Repository) FinishExecution(ctx context.Context, executionID int64, status, summary string, result map[string]any, errorMessage string) error {
	resultJSON, err := json.Marshal(dbutil.NormalizeMap(result))
	if err != nil {
		return fmt.Errorf("marshal execution result: %w", err)
	}

	finishedAt := time.Now().UTC()
	res := r.db.WithContext(ctx).
		Model(&database.JobExecutionModel{}).
		Where("id = ?", executionID).
		Updates(map[string]any{
			"status":        status,
			"summary":       summary,
			"result_json":   resultJSON,
			"error_message": errorMessage,
			"finished_at":   finishedAt,
		})
	if res.Error != nil {
		return fmt.Errorf("finish execution: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	r.invalidateExecutionCache(ctx, executionID)
	return nil
}

func (r Repository) AppendLog(ctx context.Context, executionID int64, stream, eventType, message string, payload map[string]any) error {
	payloadJSON, err := json.Marshal(dbutil.NormalizeMap(payload))
	if err != nil {
		return fmt.Errorf("marshal job log payload: %w", err)
	}
	record := database.JobLogModel{
		JobExecutionID: executionID,
		Stream:         stream,
		EventType:      eventType,
		Message:        message,
		PayloadJSON:    database.JSONBytes(payloadJSON),
	}
	if err := r.db.WithContext(ctx).Create(&record).Error; err != nil {
		return fmt.Errorf("append job log: %w", err)
	}
	if err := r.trimExecutionLogs(ctx, executionID); err != nil {
		return err
	}
	r.refreshExecutionLogsCache(ctx, executionID)
	return nil
}

func (r Repository) AppendLogBatch(ctx context.Context, entries []jobdomain.JobLogEntry) error {
	if len(entries) == 0 {
		return nil
	}

	records := make([]database.JobLogModel, len(entries))
	for i, entry := range entries {
		payloadJSON, err := json.Marshal(dbutil.NormalizeMap(entry.Payload))
		if err != nil || len(payloadJSON) == 0 {
			payloadJSON = json.RawMessage("{}")
		}
		records[i] = database.JobLogModel{
			JobExecutionID: entry.ExecutionID,
			Stream:         entry.Stream,
			EventType:      entry.EventType,
			Message:        entry.Message,
			PayloadJSON:    database.JSONBytes(payloadJSON),
		}
	}

	if err := r.db.WithContext(ctx).Create(&records).Error; err != nil {
		return fmt.Errorf("append job log batch: %w", err)
	}

	seen := make(map[int64]struct{}, len(entries))
	for _, entry := range entries {
		if _, ok := seen[entry.ExecutionID]; ok {
			continue
		}
		seen[entry.ExecutionID] = struct{}{}
		if err := r.trimExecutionLogs(ctx, entry.ExecutionID); err != nil {
			return err
		}
		r.refreshExecutionLogsCache(ctx, entry.ExecutionID)
	}
	return nil
}

func (r Repository) ListLogsAfter(ctx context.Context, executionID, afterID int64) ([]jobdomain.JobLog, error) {
	if items, ok := r.readExecutionLogsCache(ctx, executionID); ok {
		filtered := make([]jobdomain.JobLog, 0, len(items))
		for _, item := range items {
			if item.ID > afterID {
				filtered = append(filtered, item)
			}
		}
		if len(filtered) > jobLogRetentionLimit {
			return filtered[:jobLogRetentionLimit], nil
		}
		return filtered, nil
	}

	var records []database.JobLogModel
	if err := r.db.WithContext(ctx).
		Where("job_execution_id = ? AND id > ?", executionID, afterID).
		Order("id ASC").
		Limit(jobLogRetentionLimit).
		Find(&records).Error; err != nil {
		return nil, fmt.Errorf("list execution logs: %w", err)
	}

	items := make([]jobdomain.JobLog, len(records))
	for i, record := range records {
		items[i] = toDomainJobLog(record)
	}
	r.refreshExecutionLogsCache(ctx, executionID)
	return items, nil
}

func (r Repository) ClaimNextDueSchedule(ctx context.Context, workerID string, leaseDuration time.Duration) (*jobdomain.Schedule, error) {
	var claimed *database.ScheduleModel

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		now := time.Now().UTC()

		var records []database.ScheduleModel
		query := tx
		if lock := lockingClause(tx); lock != nil {
			query = query.Clauses(lock)
		}
		if err := query.
			Where("enabled = ? AND next_run_at <= ? AND (lease_until IS NULL OR lease_until < ?)", true, now, now).
			Order("next_run_at ASC").
			Limit(1).
			Find(&records).Error; err != nil {
			return fmt.Errorf("claim due schedule: %w", err)
		}
		if len(records) == 0 {
			return nil
		}

		record := records[0]
		leaseUntil := now.Add(leaseDuration)
		if err := tx.Model(&database.ScheduleModel{}).
			Where("id = ?", record.ID).
			Updates(map[string]any{
				"lease_owner": workerID,
				"lease_until": leaseUntil,
			}).Error; err != nil {
			return fmt.Errorf("update schedule lease: %w", err)
		}

		record.LeaseOwner = stringPtr(workerID)
		record.LeaseUntil = &leaseUntil
		claimed = &record
		return nil
	})
	if err != nil {
		return nil, err
	}
	if claimed == nil {
		return nil, nil
	}

	item := toDomainSchedule(*claimed)
	return &item, nil
}

func (r Repository) RescheduleAndEnqueue(ctx context.Context, schedule jobdomain.Schedule, nextRunAt time.Time, workerID string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&database.ScheduleModel{}).
			Where("id = ?", schedule.ID).
			Updates(map[string]any{
				"next_run_at": nextRunAt,
				"lease_owner": nil,
				"lease_until": nil,
			}).Error; err != nil {
			return fmt.Errorf("update schedule next run: %w", err)
		}

		var definition database.JobDefinitionModel
		if err := tx.First(&definition, "id = ?", schedule.JobDefinitionID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrNotFound
			}
			return fmt.Errorf("load job definition for schedule enqueue: %w", err)
		}

		record := database.JobExecutionModel{
			JobDefinitionID: schedule.JobDefinitionID,
			InputJSON:       definition.InputJSON,
			Status:          jobdomain.StatusQueued,
			RequestedBy:     "system:scheduler",
			Source:          "schedule",
			WorkerID:        stringPtr(workerID),
		}
		if err := tx.Create(&record).Error; err != nil {
			return fmt.Errorf("enqueue schedule execution: %w", err)
		}
		return nil
	})
}

type executionDefinition struct {
	Key       string
	Name      string
	PluginKey string
	Action    string
}

func (r Repository) loadSchedulesByDefinitionID(ctx context.Context, ids []int64) (map[int64]*database.ScheduleModel, error) {
	if len(ids) == 0 {
		return map[int64]*database.ScheduleModel{}, nil
	}

	var records []database.ScheduleModel
	if err := r.db.WithContext(ctx).
		Where("job_definition_id IN ?", ids).
		Find(&records).Error; err != nil {
		return nil, fmt.Errorf("load schedules: %w", err)
	}

	items := make(map[int64]*database.ScheduleModel, len(records))
	for _, record := range records {
		recordCopy := record
		items[record.JobDefinitionID] = &recordCopy
	}
	return items, nil
}

func (r Repository) loadScheduleByDefinitionID(ctx context.Context, definitionID int64) (*database.ScheduleModel, error) {
	var record database.ScheduleModel
	if err := r.db.WithContext(ctx).
		First(&record, "job_definition_id = ?", definitionID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("load schedule: %w", err)
	}
	return &record, nil
}

func (r Repository) loadExecutionDefinitionsByID(ctx context.Context, ids []int64) (map[int64]executionDefinition, error) {
	if len(ids) == 0 {
		return map[int64]executionDefinition{}, nil
	}

	var records []database.JobDefinitionModel
	if err := r.db.WithContext(ctx).
		Where("id IN ?", ids).
		Find(&records).Error; err != nil {
		return nil, fmt.Errorf("load job execution definitions: %w", err)
	}

	items := make(map[int64]executionDefinition, len(records))
	for _, record := range records {
		items[record.ID] = executionDefinition{
			Key:       record.Key,
			Name:      record.Name,
			PluginKey: record.PluginKey,
			Action:    record.Action,
		}
	}
	return items, nil
}

func definitionIDs(records []database.JobDefinitionModel) []int64 {
	ids := make([]int64, len(records))
	for i, record := range records {
		ids[i] = record.ID
	}
	return ids
}

func executionDefinitionIDs(records []database.JobExecutionModel) []int64 {
	ids := make([]int64, 0, len(records))
	seen := map[int64]struct{}{}
	for _, record := range records {
		if _, ok := seen[record.JobDefinitionID]; ok {
			continue
		}
		seen[record.JobDefinitionID] = struct{}{}
		ids = append(ids, record.JobDefinitionID)
	}
	return ids
}

func toDomainDefinition(record database.JobDefinitionModel, schedule *database.ScheduleModel) jobdomain.JobDefinition {
	item := jobdomain.JobDefinition{
		ID:        record.ID,
		Key:       record.Key,
		Name:      record.Name,
		PluginKey: record.PluginKey,
		Action:    record.Action,
		Input:     dbutil.DecodeJSONMap([]byte(record.InputJSON)),
		Enabled:   record.Enabled,
		CreatedAt: record.CreatedAt,
		UpdatedAt: record.UpdatedAt,
	}
	if schedule != nil {
		value := toDomainSchedule(*schedule)
		item.Schedule = &value
	}
	return item
}

func toDomainSchedule(record database.ScheduleModel) jobdomain.Schedule {
	nextRunAt := time.Time{}
	if record.NextRunAt != nil {
		nextRunAt = *record.NextRunAt
	}

	leaseOwner := ""
	if record.LeaseOwner != nil {
		leaseOwner = *record.LeaseOwner
	}

	return jobdomain.Schedule{
		ID:              record.ID,
		JobDefinitionID: record.JobDefinitionID,
		CronExpression:  record.CronExpression,
		Timezone:        record.Timezone,
		NextRunAt:       nextRunAt,
		LeaseOwner:      leaseOwner,
		LeaseUntil:      cloneTime(record.LeaseUntil),
		Enabled:         record.Enabled,
		CreatedAt:       record.CreatedAt,
		UpdatedAt:       record.UpdatedAt,
	}
}

func toDomainExecution(record database.JobExecutionModel, definition executionDefinition) jobdomain.JobExecution {
	workerID := ""
	if record.WorkerID != nil {
		workerID = *record.WorkerID
	}

	summary := ""
	if record.Summary != nil {
		summary = *record.Summary
	}

	errorMessage := ""
	if record.ErrorMessage != nil {
		errorMessage = *record.ErrorMessage
	}

	return jobdomain.JobExecution{
		ID:              record.ID,
		JobDefinitionID: record.JobDefinitionID,
		DefinitionKey:   definition.Key,
		DefinitionName:  definition.Name,
		PluginKey:       definition.PluginKey,
		Action:          definition.Action,
		Status:          record.Status,
		Input:           dbutil.DecodeJSONMap([]byte(record.InputJSON)),
		RequestedBy:     record.RequestedBy,
		Source:          record.Source,
		WorkerID:        workerID,
		Summary:         summary,
		Result:          dbutil.DecodeJSONMap([]byte(record.ResultJSON)),
		ErrorMessage:    errorMessage,
		StartedAt:       cloneTime(record.StartedAt),
		FinishedAt:      cloneTime(record.FinishedAt),
		CreatedAt:       record.CreatedAt,
		UpdatedAt:       record.UpdatedAt,
	}
}

func toDomainJobLog(record database.JobLogModel) jobdomain.JobLog {
	return jobdomain.JobLog{
		ID:             record.ID,
		JobExecutionID: record.JobExecutionID,
		Stream:         record.Stream,
		EventType:      record.EventType,
		Message:        record.Message,
		Payload:        dbutil.DecodeJSONMap([]byte(record.PayloadJSON)),
		CreatedAt:      record.CreatedAt,
	}
}

func cloneTime(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

func stringPtr(value string) *string {
	v := value
	return &v
}

func lockingClause(db *gorm.DB) clause.Expression {
	if db == nil {
		return nil
	}
	if strings.ToLower(db.Dialector.Name()) != "postgres" {
		return nil
	}
	return clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}
}
