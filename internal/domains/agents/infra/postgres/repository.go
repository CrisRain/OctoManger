package agentpostgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	agentdomain "octomanger/internal/domains/agents/domain"
	"octomanger/internal/platform/database"
	"octomanger/internal/platform/dbutil"
)

var ErrNotFound = errors.New("agent not found")

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

func (r Repository) List(ctx context.Context) ([]agentdomain.Agent, error) {
	if items, ok := r.readAgentsCache(ctx); ok {
		return items, nil
	}

	var records []agentRecord
	if err := r.db.WithContext(ctx).
		Order("created_at DESC").
		Find(&records).Error; err != nil {
		return nil, fmt.Errorf("list agents: %w", err)
	}

	items := make([]agentdomain.Agent, len(records))
	for i, record := range records {
		items[i] = record.toDomain()
	}
	r.writeAgentsCache(ctx, items)
	return items, nil
}

func (r Repository) ListPage(ctx context.Context, limit int, offset int) ([]agentdomain.Agent, int64, error) {
	var total int64
	if err := r.db.WithContext(ctx).
		Model(&agentRecord{}).
		Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count agents: %w", err)
	}

	var records []agentRecord
	query := r.db.WithContext(ctx).Order("created_at DESC")
	if limit > 0 {
		query = query.Limit(limit).Offset(offset)
	}
	if err := query.Find(&records).Error; err != nil {
		return nil, 0, fmt.Errorf("list agents: %w", err)
	}

	items := make([]agentdomain.Agent, len(records))
	for i, record := range records {
		items[i] = record.toDomain()
	}
	return items, total, nil
}

func (r Repository) ListDesiredRunning(ctx context.Context) ([]agentdomain.Agent, error) {
	var records []agentRecord
	if err := r.db.WithContext(ctx).
		Where("desired_state = ?", agentdomain.DesiredStateRunning).
		Order("id ASC").
		Find(&records).Error; err != nil {
		return nil, fmt.Errorf("list runnable agents: %w", err)
	}

	items := make([]agentdomain.Agent, len(records))
	for i, record := range records {
		items[i] = record.toDomain()
	}
	return items, nil
}

func (r Repository) Create(ctx context.Context, input agentdomain.CreateAgentInput) (*agentdomain.Agent, error) {
	inputJSON, err := json.Marshal(dbutil.NormalizeMap(input.Input))
	if err != nil {
		return nil, fmt.Errorf("marshal agent input: %w", err)
	}

	record := agentRecord{
		Name:         input.Name,
		PluginKey:    input.PluginKey,
		Action:       input.Action,
		InputJSON:    database.JSONBytes(inputJSON),
		DesiredState: agentdomain.DesiredStateStopped,
		RuntimeState: agentdomain.RuntimeStateIdle,
	}
	if err := r.db.WithContext(ctx).Create(&record).Error; err != nil {
		return nil, fmt.Errorf("create agent: %w", err)
	}

	item := record.toDomain()
	r.invalidateAgentsCache(ctx)
	return &item, nil
}

func (r Repository) Get(ctx context.Context, agentID int64) (*agentdomain.Agent, error) {
	var record agentRecord
	if err := r.db.WithContext(ctx).
		First(&record, "id = ?", agentID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get agent: %w", err)
	}

	item := record.toDomain()
	return &item, nil
}

func (r Repository) Patch(ctx context.Context, agentID int64, input agentdomain.PatchAgentInput) (*agentdomain.Agent, error) {
	current, err := r.Get(ctx, agentID)
	if err != nil {
		return nil, err
	}

	if input.Name != nil {
		current.Name = *input.Name
	}
	if input.Input != nil {
		current.Input = input.Input
	}

	inputJSON, err := json.Marshal(dbutil.NormalizeMap(current.Input))
	if err != nil {
		return nil, fmt.Errorf("marshal agent input: %w", err)
	}

	result := r.db.WithContext(ctx).
		Model(&agentRecord{}).
		Where("id = ?", agentID).
		Updates(map[string]any{
			"name":       current.Name,
			"input_json": inputJSON,
		})
	if result.Error != nil {
		return nil, fmt.Errorf("patch agent: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return nil, ErrNotFound
	}

	item, err := r.Get(ctx, agentID)
	if err != nil {
		return nil, err
	}
	r.invalidateAgentsCache(ctx)
	return item, nil
}

func (r Repository) SetDesiredState(ctx context.Context, agentID int64, desiredState string) error {
	result := r.db.WithContext(ctx).
		Model(&agentRecord{}).
		Where("id = ?", agentID).
		Update("desired_state", desiredState)
	if result.Error != nil {
		return fmt.Errorf("update agent desired state: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	r.invalidateAgentsCache(ctx)
	return nil
}

func (r Repository) UpdateRuntimeState(ctx context.Context, agentID int64, runtimeState, lastError string, heartbeat *time.Time) error {
	result := r.db.WithContext(ctx).
		Model(&agentRecord{}).
		Where("id = ?", agentID).
		Updates(map[string]any{
			"runtime_state":     runtimeState,
			"last_error":        lastError,
			"last_heartbeat_at": heartbeat,
		})
	if result.Error != nil {
		return fmt.Errorf("update agent runtime state: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	r.invalidateAgentsCache(ctx)
	return nil
}

func (r Repository) Delete(ctx context.Context, agentID int64) error {
	result := r.db.WithContext(ctx).Delete(&agentRecord{}, agentID)
	if result.Error != nil {
		return fmt.Errorf("delete agent: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	r.invalidateAgentsCache(ctx)
	return nil
}

func (r Repository) AppendLog(ctx context.Context, agentID int64, eventType, message string, payload map[string]any) error {
	payloadJSON, err := json.Marshal(dbutil.NormalizeMap(payload))
	if err != nil {
		return fmt.Errorf("marshal agent log payload: %w", err)
	}

	record := agentLogRecord{
		AgentID:     agentID,
		EventType:   eventType,
		Message:     message,
		PayloadJSON: database.JSONBytes(payloadJSON),
	}
	if err := r.db.WithContext(ctx).Create(&record).Error; err != nil {
		return fmt.Errorf("append agent log: %w", err)
	}

	if err := r.trimAgentLogs(ctx, agentID); err != nil {
		return err
	}
	r.refreshAgentLogsCache(ctx, agentID)
	return nil
}

func (r Repository) AppendLogBatch(ctx context.Context, entries []agentdomain.AgentLogEntry) error {
	if len(entries) == 0 {
		return nil
	}

	records := make([]agentLogRecord, len(entries))
	for i, entry := range entries {
		payloadJSON, err := json.Marshal(dbutil.NormalizeMap(entry.Payload))
		if err != nil {
			return fmt.Errorf("marshal agent log payload: %w", err)
		}
		records[i] = agentLogRecord{
			AgentID:     entry.AgentID,
			EventType:   entry.EventType,
			Message:     entry.Message,
			PayloadJSON: database.JSONBytes(payloadJSON),
		}
	}

	if err := r.db.WithContext(ctx).Create(&records).Error; err != nil {
		return fmt.Errorf("append agent log batch: %w", err)
	}

	seen := make(map[int64]struct{}, len(entries))
	for _, entry := range entries {
		if _, ok := seen[entry.AgentID]; ok {
			continue
		}
		seen[entry.AgentID] = struct{}{}
		if err := r.trimAgentLogs(ctx, entry.AgentID); err != nil {
			return err
		}
		r.refreshAgentLogsCache(ctx, entry.AgentID)
	}
	return nil
}

func (r Repository) ListLogsAfter(ctx context.Context, agentID, afterID int64) ([]agentdomain.AgentLog, error) {
	if items, ok := r.readAgentLogsCache(ctx, agentID); ok {
		filtered := make([]agentdomain.AgentLog, 0, len(items))
		for _, item := range items {
			if item.ID > afterID {
				filtered = append(filtered, item)
			}
		}
		if len(filtered) > agentLogLimit {
			return filtered[:agentLogLimit], nil
		}
		return filtered, nil
	}

	var records []agentLogRecord
	if err := r.db.WithContext(ctx).
		Where("agent_id = ? AND id > ? AND (event_type <> ? OR message <> ?)", agentID, afterID, "", "").
		Order("id ASC").
		Limit(agentLogLimit).
		Find(&records).Error; err != nil {
		return nil, fmt.Errorf("list agent logs: %w", err)
	}

	items := make([]agentdomain.AgentLog, len(records))
	for i, record := range records {
		items[i] = record.toDomain()
	}
	r.refreshAgentLogsCache(ctx, agentID)
	return items, nil
}
