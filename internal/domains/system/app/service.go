package systemapp

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	plugindomain "octomanger/internal/domains/plugins/domain"
	"octomanger/internal/platform/database"
	platformlogging "octomanger/internal/platform/logging"
)

// pluginLister is the narrow interface systemapp needs from the plugin backend.
type pluginLister interface {
	List(ctx context.Context) ([]plugindomain.Plugin, error)
}

type Status struct {
	Now         time.Time `json:"now"`
	DatabaseOK  bool      `json:"database_ok"`
	PluginCount int       `json:"plugin_count"`
}

type RuntimeLog struct {
	ID        int64          `json:"id"`
	Source    string         `json:"source"`
	Level     string         `json:"level"`
	Logger    string         `json:"logger"`
	Caller    string         `json:"caller"`
	Message   string         `json:"message"`
	Fields    map[string]any `json:"fields"`
	CreatedAt time.Time      `json:"created_at"`
}

// DashboardSummary is returned by GET /api/v2/dashboard and replaces 8 parallel
// frontend requests with a single round-trip. Counts are produced by cheap SQL
// COUNT queries; recent executions fetch the latest 6 rows only.
type DashboardSummary struct {
	PluginCount        int               `json:"pluginCount"`
	AccountTypeCount   int               `json:"accountTypeCount"`
	AccountCount       int               `json:"accountCount"`
	EmailAccountCount  int               `json:"emailAccountCount"`
	JobDefinitionCount int               `json:"jobDefinitionCount"`
	JobExecutionCount  int               `json:"jobExecutionCount"`
	TriggerCount       int               `json:"triggerCount"`
	AgentCount         int               `json:"agentCount"`
	RecentExecutions   []RecentExecution `json:"recentExecutions"`
}

// RecentExecution is a lightweight projection of job_executions for the
// dashboard table. It mirrors the fields the frontend currently reads.
type RecentExecution struct {
	ID             int64  `json:"id"`
	DefinitionName string `json:"definition_name"`
	PluginKey      string `json:"plugin_key"`
	Action         string `json:"action"`
	Status         string `json:"status"`
	WorkerID       string `json:"worker_id"`
}

type Service struct {
	db      *gorm.DB
	plugins pluginLister
	rdb     *redis.Client
}

func New(db *gorm.DB, plugins pluginLister, rdb ...*redis.Client) Service {
	var client *redis.Client
	if len(rdb) > 0 {
		client = rdb[0]
	}
	return Service{db: db, plugins: plugins, rdb: client}
}

func (s Service) Status(ctx context.Context) (Status, error) {
	if item, ok := s.readStatusCache(ctx); ok {
		return item, nil
	}
	plugins, err := s.plugins.List(ctx)
	if err != nil {
		return Status{}, err
	}

	item := Status{
		Now:         time.Now().UTC(),
		DatabaseOK:  s.db != nil,
		PluginCount: len(plugins),
	}
	s.writeStatusCache(ctx, item)
	return item, nil
}

func (s Service) ListLogs(ctx context.Context, limit int) ([]RuntimeLog, error) {
	if limit <= 0 {
		limit = 200
	}
	if limit > 500 {
		limit = 500
	}
	if s.rdb == nil {
		return []RuntimeLog{}, nil
	}

	rawItems, err := s.rdb.LRange(ctx, platformlogging.SystemRuntimeLogsRedisKey, 0, int64(limit-1)).Result()
	if err != nil {
		return nil, err
	}

	items := make([]RuntimeLog, 0, len(rawItems))
	for _, raw := range rawItems {
		var record platformlogging.SystemRuntimeLogRecord
		if err := json.Unmarshal([]byte(raw), &record); err != nil {
			continue
		}
		items = append(items, RuntimeLog{
			ID:        record.ID,
			Source:    record.Source,
			Level:     record.Level,
			Logger:    record.Logger,
			Caller:    record.Caller,
			Message:   record.Message,
			Fields:    record.Fields,
			CreatedAt: record.CreatedAt,
		})
	}
	return items, nil
}

// DashboardSummary returns entity counts and the 6 most recent executions in a
// single DB round-trip (one query per table — all cheap COUNT/LIMIT queries).
func (s Service) DashboardSummary(ctx context.Context) (DashboardSummary, error) {
	if item, ok := s.readDashboardCache(ctx); ok {
		return item, nil
	}
	countOf := func(model any) (int, error) {
		var count int64
		if err := s.db.WithContext(ctx).Model(model).Count(&count).Error; err != nil {
			return 0, err
		}
		return int(count), nil
	}

	pluginList, _ := s.plugins.List(ctx) // plugins live in-memory; no separate table

	accountTypeCount, err := countOf(&database.AccountTypeModel{})
	if err != nil {
		return DashboardSummary{}, err
	}

	accountCount, err := countOf(&database.AccountModel{})
	if err != nil {
		return DashboardSummary{}, err
	}

	emailAccountCount, err := countOf(&database.EmailAccountModel{})
	if err != nil {
		return DashboardSummary{}, err
	}

	jobDefinitionCount, err := countOf(&database.JobDefinitionModel{})
	if err != nil {
		return DashboardSummary{}, err
	}

	jobExecutionCount, err := countOf(&database.JobExecutionModel{})
	if err != nil {
		return DashboardSummary{}, err
	}

	triggerCount, err := countOf(&database.TriggerModel{})
	if err != nil {
		return DashboardSummary{}, err
	}

	agentCount, err := countOf(&database.AgentModel{})
	if err != nil {
		return DashboardSummary{}, err
	}

	var executions []database.JobExecutionModel
	if err := s.db.WithContext(ctx).
		Order("created_at DESC").
		Limit(6).
		Find(&executions).Error; err != nil {
		return DashboardSummary{}, err
	}

	definitionIDs := make([]int64, 0, len(executions))
	seenDefinitions := map[int64]struct{}{}
	for _, execution := range executions {
		if _, ok := seenDefinitions[execution.JobDefinitionID]; ok {
			continue
		}
		seenDefinitions[execution.JobDefinitionID] = struct{}{}
		definitionIDs = append(definitionIDs, execution.JobDefinitionID)
	}

	definitions := map[int64]database.JobDefinitionModel{}
	if len(definitionIDs) > 0 {
		var records []database.JobDefinitionModel
		if err := s.db.WithContext(ctx).
			Where("id IN ?", definitionIDs).
			Find(&records).Error; err != nil {
			return DashboardSummary{}, err
		}
		for _, record := range records {
			definitions[record.ID] = record
		}
	}

	recent := make([]RecentExecution, 0, len(executions))
	for _, execution := range executions {
		definition := definitions[execution.JobDefinitionID]
		workerID := ""
		if execution.WorkerID != nil {
			workerID = *execution.WorkerID
		}
		recent = append(recent, RecentExecution{
			ID:             execution.ID,
			DefinitionName: definition.Name,
			PluginKey:      definition.PluginKey,
			Action:         definition.Action,
			Status:         execution.Status,
			WorkerID:       workerID,
		})
	}

	item := DashboardSummary{
		PluginCount:        len(pluginList),
		AccountTypeCount:   accountTypeCount,
		AccountCount:       accountCount,
		EmailAccountCount:  emailAccountCount,
		JobDefinitionCount: jobDefinitionCount,
		JobExecutionCount:  jobExecutionCount,
		TriggerCount:       triggerCount,
		AgentCount:         agentCount,
		RecentExecutions:   recent,
	}
	s.writeDashboardCache(ctx, item)
	return item, nil
}

func (s Service) readStatusCache(ctx context.Context) (Status, bool) {
	if s.rdb == nil {
		return Status{}, false
	}
	raw, err := s.rdb.Get(ctx, "system:status").Bytes()
	if err != nil {
		return Status{}, false
	}
	var item Status
	if err := json.Unmarshal(raw, &item); err != nil {
		return Status{}, false
	}
	return item, true
}

func (s Service) writeStatusCache(ctx context.Context, item Status) {
	s.writeCache(ctx, "system:status", item)
}

func (s Service) readDashboardCache(ctx context.Context) (DashboardSummary, bool) {
	if s.rdb == nil {
		return DashboardSummary{}, false
	}
	raw, err := s.rdb.Get(ctx, "system:dashboard").Bytes()
	if err != nil {
		return DashboardSummary{}, false
	}
	var item DashboardSummary
	if err := json.Unmarshal(raw, &item); err != nil {
		return DashboardSummary{}, false
	}
	return item, true
}

func (s Service) writeDashboardCache(ctx context.Context, item DashboardSummary) {
	s.writeCache(ctx, "system:dashboard", item)
}

func (s Service) writeCache(ctx context.Context, key string, item any) {
	if s.rdb == nil {
		return
	}
	raw, err := json.Marshal(item)
	if err != nil {
		return
	}
	_ = s.rdb.Set(ctx, key, raw, 3*time.Second).Err()
}
