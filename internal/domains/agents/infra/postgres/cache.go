package agentpostgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
	agentdomain "octomanger/internal/domains/agents/domain"
)

const (
	agentListCacheTTL = 3 * time.Second
	agentLogCacheTTL  = time.Hour
	agentLogLimit     = 200
)

func (r Repository) agentsCacheKey() string {
	return "agents:list"
}

func (r Repository) agentLogsCacheKey(agentID int64) string {
	return fmt.Sprintf("agents:%d:logs", agentID)
}

func (r Repository) readAgentsCache(ctx context.Context) ([]agentdomain.Agent, bool) {
	if r.rdb == nil {
		return nil, false
	}
	raw, err := r.rdb.Get(ctx, r.agentsCacheKey()).Bytes()
	if err != nil {
		return nil, false
	}
	var items []agentdomain.Agent
	if err := json.Unmarshal(raw, &items); err != nil {
		return nil, false
	}
	return items, true
}

func (r Repository) writeAgentsCache(ctx context.Context, items []agentdomain.Agent) {
	if r.rdb == nil {
		return
	}
	raw, err := json.Marshal(items)
	if err != nil {
		return
	}
	_ = r.rdb.Set(ctx, r.agentsCacheKey(), raw, agentListCacheTTL).Err()
}

func (r Repository) invalidateAgentsCache(ctx context.Context) {
	if r.rdb == nil {
		return
	}
	_ = r.rdb.Del(ctx, r.agentsCacheKey()).Err()
}

func (r Repository) readAgentLogsCache(ctx context.Context, agentID int64) ([]agentdomain.AgentLog, bool) {
	if r.rdb == nil {
		return nil, false
	}
	raw, err := r.rdb.Get(ctx, r.agentLogsCacheKey(agentID)).Bytes()
	if err != nil {
		return nil, false
	}
	var items []agentdomain.AgentLog
	if err := json.Unmarshal(raw, &items); err != nil {
		return nil, false
	}
	return items, true
}

func (r Repository) refreshAgentLogsCache(ctx context.Context, agentID int64) {
	if r.rdb == nil {
		return
	}
	items, err := r.loadRecentAgentLogs(ctx, agentID)
	if err != nil {
		return
	}
	raw, err := json.Marshal(items)
	if err != nil {
		return
	}
	_ = r.rdb.Set(ctx, r.agentLogsCacheKey(agentID), raw, agentLogCacheTTL).Err()
}

func (r Repository) loadRecentAgentLogs(ctx context.Context, agentID int64) ([]agentdomain.AgentLog, error) {
	var records []agentLogRecord
	if err := r.db.WithContext(ctx).
		Where("agent_id = ? AND (event_type <> ? OR message <> ?)", agentID, "", "").
		Order("id DESC").
		Limit(agentLogLimit).
		Find(&records).Error; err != nil {
		return nil, fmt.Errorf("load recent agent logs: %w", err)
	}

	items := make([]agentdomain.AgentLog, len(records))
	for i := range records {
		items[len(records)-1-i] = records[i].toDomain()
	}
	return items, nil
}

func (r Repository) trimAgentLogs(ctx context.Context, agentID int64) error {
	var threshold agentLogRecord
	err := r.db.WithContext(ctx).
		Where("agent_id = ?", agentID).
		Order("id DESC").
		Offset(agentLogLimit - 1).
		Limit(1).
		Take(&threshold).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return fmt.Errorf("trim agent logs threshold: %w", err)
	}

	if err := r.db.WithContext(ctx).
		Where("agent_id = ? AND id < ?", agentID, threshold.ID).
		Delete(&agentLogRecord{}).Error; err != nil {
		return fmt.Errorf("trim agent logs: %w", err)
	}
	return nil
}
