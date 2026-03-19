package jobapp

import (
	"context"
	"errors"
	"fmt"
	"go.uber.org/zap"
	"strings"
	"time"

	"github.com/robfig/cron/v3"

	jobdomain "octomanger/internal/domains/jobs/domain"
	jobpostgres "octomanger/internal/domains/jobs/infra/postgres"
	pluginapp "octomanger/internal/domains/plugins/app"
	plugindomain "octomanger/internal/domains/plugins/domain"
)

type Service struct {
	logger   *zap.Logger
	repo     jobpostgres.Repository
	plugins  pluginapp.Service
	workerID string
}

func New(
	logger *zap.Logger,
	repo jobpostgres.Repository,
	plugins pluginapp.Service,
	workerID string,
) Service {
	return Service{
		logger:   logger,
		repo:     repo,
		plugins:  plugins,
		workerID: strings.TrimSpace(workerID),
	}
}

func (s Service) ListDefinitions(ctx context.Context) ([]jobdomain.JobDefinition, error) {
	return s.repo.ListDefinitions(ctx)
}

func (s Service) GetDefinition(ctx context.Context, definitionID int64) (*jobdomain.JobDefinition, error) {
	return s.repo.GetDefinition(ctx, definitionID)
}

func (s Service) CreateDefinition(ctx context.Context, input jobdomain.CreateDefinitionInput) (*jobdomain.JobDefinition, error) {
	var nextRunAt *time.Time
	if input.Schedule != nil && input.Schedule.Enabled {
		next, err := nextScheduleTime(input.Schedule.CronExpression, input.Schedule.Timezone, time.Now().UTC())
		if err != nil {
			return nil, err
		}
		nextRunAt = &next
	}
	return s.repo.CreateDefinition(ctx, input, nextRunAt)
}

func (s Service) EnqueueExecution(
	ctx context.Context,
	definitionID int64,
	requestedBy string,
	source string,
	inputOverride map[string]any,
) (*jobdomain.JobExecution, error) {
	return s.repo.EnqueueExecution(ctx, definitionID, requestedBy, source, inputOverride)
}

func (s Service) ListExecutions(ctx context.Context) ([]jobdomain.JobExecution, error) {
	return s.repo.ListExecutions(ctx)
}

func (s Service) GetExecution(ctx context.Context, executionID int64) (*jobdomain.JobExecution, error) {
	return s.repo.GetExecution(ctx, executionID)
}

func (s Service) ListLogsAfter(ctx context.Context, executionID int64, afterID int64) ([]jobdomain.JobLog, error) {
	return s.repo.ListLogsAfter(ctx, executionID, afterID)
}

func (s Service) ExecuteDefinitionDirect(
	ctx context.Context,
	definitionID int64,
	inputOverride map[string]any,
) (map[string]any, []plugindomain.ExecutionEvent, error) {
	definition, err := s.repo.GetDefinition(ctx, definitionID)
	if err != nil {
		return nil, nil, err
	}

	events := make([]plugindomain.ExecutionEvent, 0, 8)
	var (
		resultPayload map[string]any
		errorMessage  string
	)

	err = s.plugins.Execute(ctx, definition.PluginKey, plugindomain.ExecutionRequest{
		Mode:   "job",
		Action: definition.Action,
		Input:  mergeMaps(definition.Input, inputOverride),
		Context: map[string]any{
			"worker_id": s.workerID,
			"source":    "trigger-sync",
		},
	}, func(event plugindomain.ExecutionEvent) {
		events = append(events, event)
		switch event.Type {
		case "result":
			resultPayload = event.Data
		case "error":
			errorMessage = event.Error
			if errorMessage == "" {
				errorMessage = event.Message
			}
		}
	})
	if err != nil {
		return resultPayload, events, err
	}
	if errorMessage != "" {
		return resultPayload, events, errors.New(errorMessage)
	}
	return resultPayload, events, nil
}

func (s Service) ProcessNextExecution(ctx context.Context) (bool, error) {
	execution, err := s.repo.ClaimNextQueuedExecution(ctx, s.workerID)
	if err != nil {
		return false, err
	}
	if execution == nil {
		return false, nil
	}

	s.logger.Sugar().Infow("processing job execution", "execution_id", execution.ID, "plugin_key", execution.PluginKey, "action", execution.Action)

	var (
		resultPayload map[string]any
		errorMessage  string
	)

	err = s.plugins.Execute(ctx, execution.PluginKey, plugindomain.ExecutionRequest{
		Mode:   "job",
		Action: execution.Action,
		Input:  execution.Input,
		Context: map[string]any{
			"execution_id": execution.ID,
			"worker_id":    s.workerID,
		},
	}, func(event plugindomain.ExecutionEvent) {
		payload := event.Data
		switch event.Type {
		case "result":
			resultPayload = payload
		case "error":
			errorMessage = event.Error
			if errorMessage == "" {
				errorMessage = event.Message
			}
		}

		if err := s.repo.AppendLog(ctx, execution.ID, "plugin", event.Type, event.Message, payload); err != nil {
			s.logger.Sugar().Errorw("append job log failed", "execution_id", execution.ID, "error", err)
		}
	})
	if err != nil {
		if finishErr := s.repo.FinishExecution(ctx, execution.ID, jobdomain.StatusFailed, "plugin execution failed", nil, err.Error()); finishErr != nil {
			return true, finishErr
		}
		return true, nil
	}

	if errorMessage != "" {
		if err := s.repo.FinishExecution(ctx, execution.ID, jobdomain.StatusFailed, "plugin reported an error event", resultPayload, errorMessage); err != nil {
			return true, err
		}
		return true, nil
	}

	if resultPayload == nil {
		resultPayload = map[string]any{
			"plugin_key": execution.PluginKey,
			"action":     execution.Action,
		}
	}

	if err := s.repo.FinishExecution(ctx, execution.ID, jobdomain.StatusSucceeded, "execution completed", resultPayload, ""); err != nil {
		return true, err
	}

	return true, nil
}

func mergeMaps(base map[string]any, override map[string]any) map[string]any {
	merged := map[string]any{}
	for key, value := range base {
		merged[key] = value
	}
	for key, value := range override {
		merged[key] = value
	}
	return merged
}

func (s Service) TickSchedules(ctx context.Context, limit int) (int, error) {
	processed := 0
	for index := 0; index < limit; index++ {
		schedule, err := s.repo.ClaimNextDueSchedule(ctx, s.workerID, 30*time.Second)
		if err != nil {
			return processed, err
		}
		if schedule == nil {
			return processed, nil
		}

		nextRunAt, err := nextScheduleTime(schedule.CronExpression, schedule.Timezone, schedule.NextRunAt)
		if err != nil {
			return processed, err
		}

		if err := s.repo.RescheduleAndEnqueue(ctx, *schedule, nextRunAt, s.workerID); err != nil {
			return processed, err
		}
		processed++
	}

	return processed, nil
}

func nextScheduleTime(expression string, timezone string, from time.Time) (time.Time, error) {
	if strings.TrimSpace(expression) == "" {
		return time.Time{}, fmt.Errorf("schedule cron_expression is required")
	}

	location := time.UTC
	if tz := strings.TrimSpace(timezone); tz != "" {
		nextLocation, err := time.LoadLocation(tz)
		if err != nil {
			return time.Time{}, fmt.Errorf("load schedule timezone: %w", err)
		}
		location = nextLocation
	}

	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	schedule, err := parser.Parse(expression)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse cron expression: %w", err)
	}

	return schedule.Next(from.In(location)).UTC(), nil
}
