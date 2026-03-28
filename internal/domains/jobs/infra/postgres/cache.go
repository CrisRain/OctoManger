package jobpostgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	jobdomain "octomanger/internal/domains/jobs/domain"
	"octomanger/internal/platform/database"
)

const (
	jobExecutionCacheTTL = 3 * time.Second
	jobLogCacheTTL       = time.Hour
	jobLogRetentionLimit = 200
)

func (r Repository) executionCacheKey(executionID int64) string {
	return fmt.Sprintf("jobs:execution:%d", executionID)
}

func (r Repository) executionsCacheKey() string {
	return "jobs:executions:list"
}

func (r Repository) executionLogsCacheKey(executionID int64) string {
	return fmt.Sprintf("jobs:execution:%d:logs", executionID)
}

func (r Repository) readExecutionCache(ctx context.Context, executionID int64) (*jobdomain.JobExecution, bool) {
	if r.rdb == nil {
		return nil, false
	}
	raw, err := r.rdb.Get(ctx, r.executionCacheKey(executionID)).Bytes()
	if err != nil {
		return nil, false
	}
	var item jobdomain.JobExecution
	if err := json.Unmarshal(raw, &item); err != nil {
		return nil, false
	}
	return &item, true
}

func (r Repository) writeExecutionCache(ctx context.Context, item *jobdomain.JobExecution) {
	if r.rdb == nil || item == nil {
		return
	}
	raw, err := json.Marshal(item)
	if err != nil {
		return
	}
	_ = r.rdb.Set(ctx, r.executionCacheKey(item.ID), raw, jobExecutionCacheTTL).Err()
}

func (r Repository) invalidateExecutionCache(ctx context.Context, executionID int64) {
	if r.rdb == nil {
		return
	}
	_ = r.rdb.Del(ctx, r.executionCacheKey(executionID), r.executionsCacheKey()).Err()
}

func (r Repository) readExecutionsCache(ctx context.Context) ([]jobdomain.JobExecution, bool) {
	if r.rdb == nil {
		return nil, false
	}
	raw, err := r.rdb.Get(ctx, r.executionsCacheKey()).Bytes()
	if err != nil {
		return nil, false
	}
	var items []jobdomain.JobExecution
	if err := json.Unmarshal(raw, &items); err != nil {
		return nil, false
	}
	return items, true
}

func (r Repository) writeExecutionsCache(ctx context.Context, items []jobdomain.JobExecution) {
	if r.rdb == nil {
		return
	}
	raw, err := json.Marshal(items)
	if err != nil {
		return
	}
	_ = r.rdb.Set(ctx, r.executionsCacheKey(), raw, jobExecutionCacheTTL).Err()
}

func (r Repository) readExecutionLogsCache(ctx context.Context, executionID int64) ([]jobdomain.JobLog, bool) {
	if r.rdb == nil {
		return nil, false
	}
	raw, err := r.rdb.Get(ctx, r.executionLogsCacheKey(executionID)).Bytes()
	if err != nil {
		return nil, false
	}
	var items []jobdomain.JobLog
	if err := json.Unmarshal(raw, &items); err != nil {
		return nil, false
	}
	return items, true
}

func (r Repository) refreshExecutionLogsCache(ctx context.Context, executionID int64) {
	if r.rdb == nil {
		return
	}
	items, err := r.loadRecentExecutionLogs(ctx, executionID)
	if err != nil {
		return
	}
	raw, err := json.Marshal(items)
	if err != nil {
		return
	}
	_ = r.rdb.Set(ctx, r.executionLogsCacheKey(executionID), raw, jobLogCacheTTL).Err()
}

func (r Repository) loadRecentExecutionLogs(ctx context.Context, executionID int64) ([]jobdomain.JobLog, error) {
	var records []database.JobLogModel
	if err := r.db.WithContext(ctx).
		Where("job_execution_id = ?", executionID).
		Order("id DESC").
		Limit(jobLogRetentionLimit).
		Find(&records).Error; err != nil {
		return nil, fmt.Errorf("load recent execution logs: %w", err)
	}

	items := make([]jobdomain.JobLog, len(records))
	for i := range records {
		items[len(records)-1-i] = toDomainJobLog(records[i])
	}
	return items, nil
}

func (r Repository) trimExecutionLogs(ctx context.Context, executionID int64) error {
	var threshold database.JobLogModel
	err := r.db.WithContext(ctx).
		Where("job_execution_id = ?", executionID).
		Order("id DESC").
		Offset(jobLogRetentionLimit - 1).
		Limit(1).
		Take(&threshold).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return fmt.Errorf("trim execution logs threshold: %w", err)
	}

	if err := r.db.WithContext(ctx).
		Where("job_execution_id = ? AND id < ?", executionID, threshold.ID).
		Delete(&database.JobLogModel{}).Error; err != nil {
		return fmt.Errorf("trim execution logs: %w", err)
	}
	return nil
}
