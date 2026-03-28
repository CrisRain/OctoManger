package runtime

import (
	"context"
	"errors"
	"fmt"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"octomanger/internal/platform/database"
)

const (
	startupJobLogRetentionLimit   = 200
	startupAgentLogRetentionLimit = 200
)

func enforceLogRetentionOnStartup(ctx context.Context, app *App) {
	steps := []struct {
		name string
		run  func(context.Context, *gorm.DB) (int64, error)
	}{
		{name: "job_logs", run: trimJobLogsOnStartup},
		{name: "agent_logs", run: trimAgentLogsOnStartup},
	}

	for _, step := range steps {
		deletedRows, err := step.run(ctx, app.DB)
		if err != nil {
			app.Logger.Warn("enforce log retention failed", zap.String("table", step.name), zap.Error(err))
			continue
		}
		if deletedRows > 0 {
			app.Logger.Info("enforced log retention", zap.String("table", step.name), zap.Int64("deleted_rows", deletedRows))
		}
	}
}

func trimJobLogsOnStartup(ctx context.Context, db *gorm.DB) (int64, error) {
	var executionIDs []int64
	if err := db.WithContext(ctx).
		Model(&database.JobLogModel{}).
		Distinct("job_execution_id").
		Pluck("job_execution_id", &executionIDs).Error; err != nil {
		return 0, fmt.Errorf("list job log execution ids: %w", err)
	}

	var deleted int64
	for _, executionID := range executionIDs {
		rows, err := trimGroupedJobLogs(ctx, db, executionID)
		if err != nil {
			return deleted, err
		}
		deleted += rows
	}
	return deleted, nil
}

func trimGroupedJobLogs(ctx context.Context, db *gorm.DB, executionID int64) (int64, error) {
	var threshold database.JobLogModel
	err := db.WithContext(ctx).
		Where("job_execution_id = ?", executionID).
		Order("id DESC").
		Offset(startupJobLogRetentionLimit - 1).
		Limit(1).
		Take(&threshold).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, nil
		}
		return 0, fmt.Errorf("find job log threshold for execution %d: %w", executionID, err)
	}

	result := db.WithContext(ctx).
		Where("job_execution_id = ? AND id < ?", executionID, threshold.ID).
		Delete(&database.JobLogModel{})
	if result.Error != nil {
		return 0, fmt.Errorf("trim job logs for execution %d: %w", executionID, result.Error)
	}
	return result.RowsAffected, nil
}

func trimAgentLogsOnStartup(ctx context.Context, db *gorm.DB) (int64, error) {
	var agentIDs []int64
	if err := db.WithContext(ctx).
		Model(&database.AgentLogModel{}).
		Distinct("agent_id").
		Pluck("agent_id", &agentIDs).Error; err != nil {
		return 0, fmt.Errorf("list agent log ids: %w", err)
	}

	var deleted int64
	for _, agentID := range agentIDs {
		rows, err := trimGroupedAgentLogs(ctx, db, agentID)
		if err != nil {
			return deleted, err
		}
		deleted += rows
	}
	return deleted, nil
}

func trimGroupedAgentLogs(ctx context.Context, db *gorm.DB, agentID int64) (int64, error) {
	var threshold database.AgentLogModel
	err := db.WithContext(ctx).
		Where("agent_id = ?", agentID).
		Order("id DESC").
		Offset(startupAgentLogRetentionLimit - 1).
		Limit(1).
		Take(&threshold).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, nil
		}
		return 0, fmt.Errorf("find agent log threshold for agent %d: %w", agentID, err)
	}

	result := db.WithContext(ctx).
		Where("agent_id = ? AND id < ?", agentID, threshold.ID).
		Delete(&database.AgentLogModel{})
	if result.Error != nil {
		return 0, fmt.Errorf("trim agent logs for agent %d: %w", agentID, result.Error)
	}
	return result.RowsAffected, nil
}
