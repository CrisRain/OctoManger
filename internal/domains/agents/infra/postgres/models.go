package agentpostgres

import (
	"time"

	agentdomain "octomanger/internal/domains/agents/domain"
	"octomanger/internal/platform/database"
	"octomanger/internal/platform/dbutil"
)

type agentRecord struct {
	ID              int64              `gorm:"column:id;primaryKey;autoIncrement"`
	Name            string             `gorm:"column:name;type:text;not null"`
	PluginKey       string             `gorm:"column:plugin_key;type:text;not null"`
	Action          string             `gorm:"column:action;type:text;not null"`
	InputJSON       database.JSONBytes `gorm:"column:input_json;type:jsonb;not null;default:'{}'"`
	DesiredState    string             `gorm:"column:desired_state;type:text;not null;default:stopped"`
	RuntimeState    string             `gorm:"column:runtime_state;type:text;not null;default:idle"`
	LastError       *string            `gorm:"column:last_error;type:text"`
	LastHeartbeatAt *time.Time         `gorm:"column:last_heartbeat_at"`
	CreatedAt       time.Time          `gorm:"column:created_at;not null;autoCreateTime"`
	UpdatedAt       time.Time          `gorm:"column:updated_at;not null;autoUpdateTime"`
}

func (agentRecord) TableName() string {
	return "agents"
}

func (record agentRecord) toDomain() agentdomain.Agent {
	lastError := ""
	if record.LastError != nil {
		lastError = *record.LastError
	}
	return agentdomain.Agent{
		ID:              record.ID,
		Name:            record.Name,
		PluginKey:       record.PluginKey,
		Action:          record.Action,
		Input:           dbutil.DecodeJSONMap([]byte(record.InputJSON)),
		DesiredState:    record.DesiredState,
		RuntimeState:    record.RuntimeState,
		LastError:       lastError,
		LastHeartbeatAt: record.LastHeartbeatAt,
		CreatedAt:       record.CreatedAt,
		UpdatedAt:       record.UpdatedAt,
	}
}

type agentLogRecord struct {
	ID          int64              `gorm:"column:id;primaryKey;autoIncrement"`
	AgentID     int64              `gorm:"column:agent_id;not null;index:idx_agent_logs_agent_id_id,priority:1"`
	EventType   string             `gorm:"column:event_type;type:text;not null"`
	Message     string             `gorm:"column:message;type:text;not null"`
	PayloadJSON database.JSONBytes `gorm:"column:payload_json;type:jsonb;not null;default:'{}'"`
	CreatedAt   time.Time          `gorm:"column:created_at;not null;autoCreateTime"`
}

func (agentLogRecord) TableName() string {
	return "agent_logs"
}

func (record agentLogRecord) toDomain() agentdomain.AgentLog {
	return agentdomain.AgentLog{
		ID:        record.ID,
		AgentID:   record.AgentID,
		EventType: record.EventType,
		Message:   record.Message,
		Payload:   dbutil.DecodeJSONMap([]byte(record.PayloadJSON)),
		CreatedAt: record.CreatedAt,
	}
}
