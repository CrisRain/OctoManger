package database

import (
	"database/sql/driver"
	"fmt"
	"time"
)

// JSONBytes is a []byte wrapper that tolerates drivers returning strings for JSON columns.
type JSONBytes []byte

func (j *JSONBytes) Scan(value any) error {
	if value == nil {
		*j = nil
		return nil
	}
	switch v := value.(type) {
	case []byte:
		*j = append((*j)[:0], v...)
		return nil
	case string:
		*j = append((*j)[:0], v...)
		return nil
	default:
		return fmt.Errorf("unsupported Scan, storing driver.Value type %T into JSONBytes", value)
	}
}

func (j JSONBytes) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	return []byte(j), nil
}

type AccountTypeModel struct {
	ID               int64     `gorm:"column:id;primaryKey;autoIncrement"`
	Key              string    `gorm:"column:key;type:text;not null;unique"`
	Name             string    `gorm:"column:name;type:text;not null"`
	Category         string    `gorm:"column:category;type:text;not null;default:generic"`
	SchemaJSON       JSONBytes `gorm:"column:schema_json;type:jsonb;not null;default:'{}'"`
	CapabilitiesJSON JSONBytes `gorm:"column:capabilities_json;type:jsonb;not null;default:'{}'"`
	CreatedAt        time.Time `gorm:"column:created_at;not null;autoCreateTime"`
	UpdatedAt        time.Time `gorm:"column:updated_at;not null;autoUpdateTime"`
}

func (AccountTypeModel) TableName() string {
	return "account_types"
}

type AccountModel struct {
	ID            int64     `gorm:"column:id;primaryKey;autoIncrement"`
	AccountTypeID *int64    `gorm:"column:account_type_id;uniqueIndex:accounts_account_type_id_identifier_key,priority:1"`
	Identifier    string    `gorm:"column:identifier;type:text;not null;uniqueIndex:accounts_account_type_id_identifier_key,priority:2"`
	SpecJSON      JSONBytes `gorm:"column:spec_json;type:jsonb;not null;default:'{}'"`
	Status        string    `gorm:"column:status;type:text;not null;default:active"`
	TagsJSON      JSONBytes `gorm:"column:tags_json;type:jsonb;not null;default:'[]'"`
	CreatedAt     time.Time `gorm:"column:created_at;not null;autoCreateTime"`
	UpdatedAt     time.Time `gorm:"column:updated_at;not null;autoUpdateTime"`
}

func (AccountModel) TableName() string {
	return "accounts"
}

type EmailAccountModel struct {
	ID         int64     `gorm:"column:id;primaryKey;autoIncrement"`
	Provider   string    `gorm:"column:provider;type:text;not null"`
	Address    string    `gorm:"column:address;type:text;not null;unique"`
	Status     string    `gorm:"column:status;type:text;not null;default:active"`
	ConfigJSON JSONBytes `gorm:"column:config_json;type:jsonb;not null;default:'{}'"`
	CreatedAt  time.Time `gorm:"column:created_at;not null;autoCreateTime"`
	UpdatedAt  time.Time `gorm:"column:updated_at;not null;autoUpdateTime"`
}

func (EmailAccountModel) TableName() string {
	return "email_accounts"
}

type JobDefinitionModel struct {
	ID        int64     `gorm:"column:id;primaryKey;autoIncrement"`
	Key       string    `gorm:"column:key;type:text;not null;unique"`
	Name      string    `gorm:"column:name;type:text;not null"`
	PluginKey string    `gorm:"column:plugin_key;type:text;not null"`
	Action    string    `gorm:"column:action;type:text;not null"`
	InputJSON JSONBytes `gorm:"column:input_json;type:jsonb;not null;default:'{}'"`
	Enabled   bool      `gorm:"column:enabled;not null;default:true"`
	CreatedAt time.Time `gorm:"column:created_at;not null;autoCreateTime"`
	UpdatedAt time.Time `gorm:"column:updated_at;not null;autoUpdateTime"`
}

func (JobDefinitionModel) TableName() string {
	return "job_definitions"
}

type ScheduleModel struct {
	ID              int64      `gorm:"column:id;primaryKey;autoIncrement"`
	JobDefinitionID int64      `gorm:"column:job_definition_id;not null;unique"`
	CronExpression  string     `gorm:"column:cron_expression;type:text;not null"`
	Timezone        string     `gorm:"column:timezone;type:text;not null;default:UTC"`
	NextRunAt       *time.Time `gorm:"column:next_run_at;index:idx_schedules_next_run_at,where:enabled = true"`
	LeaseOwner      *string    `gorm:"column:lease_owner;type:text"`
	LeaseUntil      *time.Time `gorm:"column:lease_until"`
	Enabled         bool       `gorm:"column:enabled;not null;default:true"`
	CreatedAt       time.Time  `gorm:"column:created_at;not null;autoCreateTime"`
	UpdatedAt       time.Time  `gorm:"column:updated_at;not null;autoUpdateTime"`
}

func (ScheduleModel) TableName() string {
	return "schedules"
}

type JobExecutionModel struct {
	ID              int64      `gorm:"column:id;primaryKey;autoIncrement"`
	JobDefinitionID int64      `gorm:"column:job_definition_id;not null;index:idx_job_executions_def_id_created,priority:1"`
	InputJSON       JSONBytes  `gorm:"column:input_json;type:jsonb;not null;default:'{}'"`
	Status          string     `gorm:"column:status;type:text;not null;index:idx_job_executions_status_created_at,priority:1"`
	RequestedBy     string     `gorm:"column:requested_by;type:text;not null;default:''"`
	Source          string     `gorm:"column:source;type:text;not null;default:manual"`
	WorkerID        *string    `gorm:"column:worker_id;type:text"`
	Summary         *string    `gorm:"column:summary;type:text"`
	ResultJSON      JSONBytes  `gorm:"column:result_json;type:jsonb;not null;default:'{}'"`
	ErrorMessage    *string    `gorm:"column:error_message;type:text"`
	StartedAt       *time.Time `gorm:"column:started_at"`
	FinishedAt      *time.Time `gorm:"column:finished_at"`
	CreatedAt       time.Time  `gorm:"column:created_at;not null;autoCreateTime;index:idx_job_executions_status_created_at,priority:2;sort:desc;index:idx_job_executions_def_id_created,priority:2,sort:desc"`
	UpdatedAt       time.Time  `gorm:"column:updated_at;not null;autoUpdateTime"`
}

func (JobExecutionModel) TableName() string {
	return "job_executions"
}

type JobLogModel struct {
	ID             int64     `gorm:"column:id;primaryKey;autoIncrement"`
	JobExecutionID int64     `gorm:"column:job_execution_id;not null;index:idx_job_logs_execution_id_id,priority:1"`
	Stream         string    `gorm:"column:stream;type:text;not null"`
	EventType      string    `gorm:"column:event_type;type:text;not null"`
	Message        string    `gorm:"column:message;type:text;not null"`
	PayloadJSON    JSONBytes `gorm:"column:payload_json;type:jsonb;not null;default:'{}'"`
	CreatedAt      time.Time `gorm:"column:created_at;not null;autoCreateTime"`
}

func (JobLogModel) TableName() string {
	return "job_logs"
}

type TriggerModel struct {
	ID               int64     `gorm:"column:id;primaryKey;autoIncrement"`
	Key              string    `gorm:"column:key;type:text;not null;unique"`
	Name             string    `gorm:"column:name;type:text;not null"`
	JobDefinitionID  int64     `gorm:"column:job_definition_id;not null"`
	Mode             string    `gorm:"column:mode;type:text;not null;default:async"`
	DefaultInputJSON JSONBytes `gorm:"column:default_input_json;type:jsonb;not null;default:'{}'"`
	TokenHash        string    `gorm:"column:token_hash;type:text;not null"`
	TokenPrefix      string    `gorm:"column:token_prefix;type:text;not null"`
	Enabled          bool      `gorm:"column:enabled;not null;default:true"`
	CreatedAt        time.Time `gorm:"column:created_at;not null;autoCreateTime"`
	UpdatedAt        time.Time `gorm:"column:updated_at;not null;autoUpdateTime"`
}

func (TriggerModel) TableName() string {
	return "triggers"
}

type AgentModel struct {
	ID              int64      `gorm:"column:id;primaryKey;autoIncrement"`
	Name            string     `gorm:"column:name;type:text;not null"`
	PluginKey       string     `gorm:"column:plugin_key;type:text;not null"`
	Action          string     `gorm:"column:action;type:text;not null"`
	InputJSON       JSONBytes  `gorm:"column:input_json;type:jsonb;not null;default:'{}'"`
	DesiredState    string     `gorm:"column:desired_state;type:text;not null;default:stopped"`
	RuntimeState    string     `gorm:"column:runtime_state;type:text;not null;default:idle"`
	LastError       *string    `gorm:"column:last_error;type:text"`
	LastHeartbeatAt *time.Time `gorm:"column:last_heartbeat_at"`
	CreatedAt       time.Time  `gorm:"column:created_at;not null;autoCreateTime"`
	UpdatedAt       time.Time  `gorm:"column:updated_at;not null;autoUpdateTime"`
}

func (AgentModel) TableName() string {
	return "agents"
}

type AgentLogModel struct {
	ID          int64     `gorm:"column:id;primaryKey;autoIncrement"`
	AgentID     int64     `gorm:"column:agent_id;not null;index:idx_agent_logs_agent_id_id,priority:1"`
	EventType   string    `gorm:"column:event_type;type:text;not null"`
	Message     string    `gorm:"column:message;type:text;not null"`
	PayloadJSON JSONBytes `gorm:"column:payload_json;type:jsonb;not null;default:'{}'"`
	CreatedAt   time.Time `gorm:"column:created_at;not null;autoCreateTime"`
}

func (AgentLogModel) TableName() string {
	return "agent_logs"
}

type SystemSettingsModel struct {
	ID                       int64     `gorm:"column:id;primaryKey;autoIncrement:false"`
	AppName                  string    `gorm:"column:app_name;type:text;not null;default:'OctoManager'"`
	JobDefaultTimeoutMinutes int       `gorm:"column:job_default_timeout_minutes;not null;default:30"`
	JobMaxConcurrency        int       `gorm:"column:job_max_concurrency;not null;default:10"`
	AdminKeyHash             string    `gorm:"column:admin_key_hash;type:text;not null;default:''"`
	PluginInternalAPIToken   string    `gorm:"column:plugin_internal_api_token;type:text;not null;default:''"`
	CreatedAt                time.Time `gorm:"column:created_at;not null;autoCreateTime"`
	UpdatedAt                time.Time `gorm:"column:updated_at;not null;autoUpdateTime"`
}

func (SystemSettingsModel) TableName() string {
	return "system_settings"
}

type PluginSettingsModel struct {
	PluginKey    string    `gorm:"column:plugin_key;primaryKey;type:text"`
	SettingsJSON JSONBytes `gorm:"column:settings_json;type:jsonb;not null;default:'{}'"`
	CreatedAt    time.Time `gorm:"column:created_at;not null;autoCreateTime"`
	UpdatedAt    time.Time `gorm:"column:updated_at;not null;autoUpdateTime"`
}

func (PluginSettingsModel) TableName() string {
	return "plugin_settings"
}

type PluginServiceConfigModel struct {
	PluginKey   string    `gorm:"column:plugin_key;primaryKey;type:text"`
	GRPCAddress string    `gorm:"column:grpc_address;type:text;not null;default:''"`
	CreatedAt   time.Time `gorm:"column:created_at;not null;autoCreateTime"`
	UpdatedAt   time.Time `gorm:"column:updated_at;not null;autoUpdateTime"`
}

func (PluginServiceConfigModel) TableName() string {
	return "plugin_service_configs"
}

type SystemLogModel struct {
	ID         int64     `gorm:"column:id;primaryKey;autoIncrement"`
	Source     string    `gorm:"column:source;type:text;not null;default:''"`
	Level      string    `gorm:"column:level;type:text;not null;default:info;index:idx_system_logs_level_created,priority:1"`
	Logger     string    `gorm:"column:logger;type:text;not null;default:''"`
	Caller     string    `gorm:"column:caller;type:text;not null;default:''"`
	Message    string    `gorm:"column:message;type:text;not null"`
	FieldsJSON JSONBytes `gorm:"column:fields_json;type:jsonb;not null;default:'{}'"`
	CreatedAt  time.Time `gorm:"column:created_at;not null;autoCreateTime;index:idx_system_logs_level_created,priority:2,sort:desc"`
}

func (SystemLogModel) TableName() string {
	return "system_logs"
}
