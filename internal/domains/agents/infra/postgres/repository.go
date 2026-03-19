package agentpostgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	agentdomain "octomanger/internal/domains/agents/domain"
)

var ErrNotFound = errors.New("agent not found")

type Repository struct {
	db *gorm.DB
}

func New(db *gorm.DB) Repository {
	return Repository{db: db}
}

func (r Repository) List(ctx context.Context) ([]agentdomain.Agent, error) {
	rows, err := r.db.WithContext(ctx).Raw(`
		SELECT id, name, plugin_key, action, input_json, desired_state, runtime_state,
		       COALESCE(last_error, ''), last_heartbeat_at, created_at, updated_at
		FROM agents ORDER BY created_at DESC`).Rows()
	if err != nil {
		return nil, fmt.Errorf("list agents: %w", err)
	}
	defer rows.Close()
	return scanAgents(rows)
}

func (r Repository) ListDesiredRunning(ctx context.Context) ([]agentdomain.Agent, error) {
	rows, err := r.db.WithContext(ctx).Raw(`
		SELECT id, name, plugin_key, action, input_json, desired_state, runtime_state,
		       COALESCE(last_error, ''), last_heartbeat_at, created_at, updated_at
		FROM agents WHERE desired_state = 'running' ORDER BY id ASC`).Rows()
	if err != nil {
		return nil, fmt.Errorf("list runnable agents: %w", err)
	}
	defer rows.Close()
	return scanAgents(rows)
}

func (r Repository) Create(ctx context.Context, input agentdomain.CreateAgentInput) (*agentdomain.Agent, error) {
	inputJSON, err := json.Marshal(normalizeMap(input.Input))
	if err != nil {
		return nil, fmt.Errorf("marshal agent input: %w", err)
	}

	var agentID int64
	row := r.db.WithContext(ctx).Raw(`
		INSERT INTO agents (name, plugin_key, action, input_json, desired_state, runtime_state)
		VALUES ($1, $2, $3, $4, 'stopped', 'idle')
		RETURNING id`,
		input.Name, input.PluginKey, input.Action, inputJSON,
	).Row()
	if err := row.Scan(&agentID); err != nil {
		return nil, fmt.Errorf("create agent: %w", err)
	}

	return r.Get(ctx, agentID)
}

func (r Repository) Get(ctx context.Context, agentID int64) (*agentdomain.Agent, error) {
	row := r.db.WithContext(ctx).Raw(`
		SELECT id, name, plugin_key, action, input_json, desired_state, runtime_state,
		       COALESCE(last_error, ''), last_heartbeat_at, created_at, updated_at
		FROM agents WHERE id = $1`, agentID).Row()
	item, err := scanAgent(row)
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (r Repository) SetDesiredState(ctx context.Context, agentID int64, desiredState string) error {
	result := r.db.WithContext(ctx).Exec(`
		UPDATE agents SET desired_state = $2, updated_at = NOW() WHERE id = $1`,
		agentID, desiredState,
	)
	if result.Error != nil {
		return fmt.Errorf("update agent desired state: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (r Repository) UpdateRuntimeState(ctx context.Context, agentID int64, runtimeState, lastError string, heartbeat *time.Time) error {
	result := r.db.WithContext(ctx).Exec(`
		UPDATE agents
		SET runtime_state = $2, last_error = $3, last_heartbeat_at = $4, updated_at = NOW()
		WHERE id = $1`,
		agentID, runtimeState, lastError, heartbeat,
	)
	if result.Error != nil {
		return fmt.Errorf("update agent runtime state: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (r Repository) AppendLog(ctx context.Context, agentID int64, eventType, message string, payload map[string]any) error {
	payloadJSON, err := json.Marshal(normalizeMap(payload))
	if err != nil {
		return fmt.Errorf("marshal agent log payload: %w", err)
	}
	result := r.db.WithContext(ctx).Exec(`
		INSERT INTO agent_logs (agent_id, event_type, message, payload_json)
		VALUES ($1, $2, $3, $4)`,
		agentID, eventType, message, payloadJSON,
	)
	if result.Error != nil {
		return fmt.Errorf("append agent log: %w", result.Error)
	}
	return nil
}

func (r Repository) ListLogsAfter(ctx context.Context, agentID, afterID int64) ([]agentdomain.AgentLog, error) {
	rows, err := r.db.WithContext(ctx).Raw(`
		SELECT id, agent_id, event_type, message, payload_json, created_at
		FROM agent_logs
		WHERE agent_id = $1 AND id > $2 AND (event_type != '' OR message != '')
		ORDER BY id ASC
		LIMIT 200`, agentID, afterID).Rows()
	if err != nil {
		return nil, fmt.Errorf("list agent logs: %w", err)
	}
	defer rows.Close()

	items := make([]agentdomain.AgentLog, 0)
	for rows.Next() {
		var item agentdomain.AgentLog
		var payloadJSON []byte
		if err := rows.Scan(&item.ID, &item.AgentID, &item.EventType, &item.Message, &payloadJSON, &item.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan agent log: %w", err)
		}
		item.Payload = decodeJSONMap(payloadJSON)
		items = append(items, item)
	}
	return items, rows.Err()
}

type scanner interface {
	Scan(dest ...any) error
}

func scanAgents(rows *sql.Rows) ([]agentdomain.Agent, error) {
	items := make([]agentdomain.Agent, 0)
	for rows.Next() {
		item, err := scanAgent(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func scanAgent(row scanner) (agentdomain.Agent, error) {
	var item agentdomain.Agent
	var inputJSON []byte
	var lastHeartbeat sql.NullTime
	if err := row.Scan(
		&item.ID, &item.Name, &item.PluginKey, &item.Action, &inputJSON,
		&item.DesiredState, &item.RuntimeState, &item.LastError,
		&lastHeartbeat, &item.CreatedAt, &item.UpdatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return agentdomain.Agent{}, ErrNotFound
		}
		return agentdomain.Agent{}, fmt.Errorf("scan agent: %w", err)
	}
	item.Input = decodeJSONMap(inputJSON)
	if lastHeartbeat.Valid {
		t := lastHeartbeat.Time
		item.LastHeartbeatAt = &t
	}
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
