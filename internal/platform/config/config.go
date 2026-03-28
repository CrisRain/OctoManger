package config

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	App      AppConfig
	Server   ServerConfig
	Database DatabaseConfig
	Redis    RedisConfig
	Auth     AuthConfig
	Worker   WorkerConfig
	Plugins  PluginsConfig
	Logging  LoggingConfig
}

type AppConfig struct {
	Env string
}

type ServerConfig struct {
	APIAddr     string
	ReadTimeout time.Duration
	IdleTimeout time.Duration
	CORS        CORSConfig
}

type CORSConfig struct {
	AllowedOrigins []string
	AllowedMethods []string
	AllowedHeaders []string
	MaxAge         time.Duration
}

type DatabaseConfig struct {
	DSN              string
	MigrationMode    string
	MaxConnections   int
	ConnectTimeout   time.Duration
	QueryTimeout     time.Duration
	HealthcheckGrace time.Duration
}

type RedisConfig struct {
	Addr     string
	Password string
	DB       int
}

type AuthConfig struct {
	AdminKey               string
	PluginInternalAPIToken string
}

type WorkerConfig struct {
	ID                 string
	PollInterval       time.Duration
	AgentScanInterval  time.Duration
	AgentLoopInterval  time.Duration
	AgentErrorBackoff  time.Duration
	SchedulePollLimit  int
	ExecutionPollLimit int
}

type PluginsConfig struct {
	// Local plugin process configuration used by the worker-managed gRPC launcher.
	ModulesDir string
	SDKDir     string
	PythonBin  string
	Timeout    PluginsTimeoutConfig
	GRPC       PluginsGRPCConfig

	// Plugin gRPC service defaults used to initialize database-backed config.
	// Set PLUGIN_GRPC_<KEY>_ADDR=host:port to override the initial address.
	Services map[string]PluginServiceEntry // plugin key → gRPC connection info
}

// PluginServiceEntry holds the gRPC connection config for one plugin microservice.
type PluginServiceEntry struct {
	Address               string `json:"address"` // host:port of the plugin's gRPC server
	AllowInsecure         bool   `json:"allow_insecure,omitempty"`
	TLSServerName         string `json:"tls_server_name,omitempty"`
	TLSInsecureSkipVerify bool   `json:"tls_insecure_skip_verify,omitempty"`
}

type PluginsGRPCConfig struct {
	AllowInsecureRemote bool
	InsecureSkipVerify  bool
}

type PluginsTimeoutConfig struct {
	Account time.Duration
	Job     time.Duration
	Agent   time.Duration
}

type LoggingConfig struct {
	Level    string
	Format   string
	Filename string
}

func Load() (Config, error) {
	v := viper.New()

	// Optional config file (config.yaml in working dir or ./configs/)
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(".")
	v.AddConfigPath("./configs")
	_ = v.ReadInConfig() // silently ignore if not found

	// Env vars take priority
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// Defaults
	v.SetDefault("app.env", "development")
	v.SetDefault("server.api_addr", ":8080")
	v.SetDefault("server.read_timeout", "15s")
	v.SetDefault("server.idle_timeout", "60s")
	v.SetDefault("server.cors.allowed_origins", []string{})
	v.SetDefault("server.cors.allowed_methods", []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"})
	v.SetDefault("server.cors.allowed_headers", []string{"Authorization", "Content-Type", "X-Admin-Key", "X-Api-Key", "X-Trigger-Token"})
	v.SetDefault("server.cors.max_age", "10m")
	v.SetDefault("database.max_connections", 8)
	v.SetDefault("database.connect_timeout", "5s")
	v.SetDefault("database.query_timeout", "10s")
	v.SetDefault("database.healthcheck_grace", "250ms")
	v.SetDefault("database.migration_mode", "versioned")
	v.SetDefault("redis.addr", "localhost:6379")
	v.SetDefault("redis.db", 0)
	v.SetDefault("worker.poll_interval", "2s")
	v.SetDefault("worker.agent_scan_interval", "3s")
	v.SetDefault("worker.agent_loop_interval", "10s")
	v.SetDefault("worker.agent_error_backoff", "15s")
	v.SetDefault("worker.schedule_poll_limit", 10)
	v.SetDefault("worker.execution_poll_limit", 10)
	v.SetDefault("plugins.python_bin", "python3")
	v.SetDefault("plugins.modules_dir", "plugins/modules")
	v.SetDefault("plugins.sdk_dir", "plugins/sdk/python")
	v.SetDefault("plugins.timeout.account", "60s")
	v.SetDefault("plugins.timeout.job", "10m")
	v.SetDefault("plugins.timeout.agent", "0s")
	v.SetDefault("plugins.grpc.allow_insecure_remote", false)
	v.SetDefault("plugins.grpc.insecure_skip_verify", false)
	v.SetDefault("auth.admin_keys", "")
	v.SetDefault("auth.plugin_internal_api_token", "")
	v.SetDefault("logging.level", "info")
	v.SetDefault("logging.format", "text")

	// DATABASE_DSN supports both legacy flat env and nested viper key
	dsn := strings.TrimSpace(v.GetString("DATABASE_DSN"))
	if dsn == "" {
		dsn = strings.TrimSpace(v.GetString("database.dsn"))
	}
	if dsn == "" {
		return Config{}, fmt.Errorf("DATABASE_DSN is required")
	}

	workerID := strings.TrimSpace(v.GetString("WORKER_ID"))
	if workerID == "" {
		workerID = strings.TrimSpace(v.GetString("worker.id"))
	}
	if workerID == "" {
		if h, err := os.Hostname(); err == nil {
			workerID = h
		} else {
			workerID = "worker-local"
		}
	}

	resolvedAdminKeys := resolveAdminKeys(v)

	return Config{
		App: AppConfig{
			Env: v.GetString("app.env"),
		},
		Server: ServerConfig{
			APIAddr:     pickEnvOrKey(v, "API_ADDR", "server.api_addr"),
			ReadTimeout: pickDurationEnvOrKey(v, "API_READ_TIMEOUT", "server.read_timeout"),
			IdleTimeout: pickDurationEnvOrKey(v, "API_IDLE_TIMEOUT", "server.idle_timeout"),
			CORS: CORSConfig{
				AllowedOrigins: pickCSVEnvOrKey(v, "CORS_ALLOWED_ORIGINS", "server.cors.allowed_origins"),
				AllowedMethods: pickCSVEnvOrKey(v, "CORS_ALLOWED_METHODS", "server.cors.allowed_methods"),
				AllowedHeaders: pickCSVEnvOrKey(v, "CORS_ALLOWED_HEADERS", "server.cors.allowed_headers"),
				MaxAge:         pickDurationEnvOrKey(v, "CORS_MAX_AGE", "server.cors.max_age"),
			},
		},
		Database: DatabaseConfig{
			DSN:              dsn,
			MigrationMode:    pickEnvOrKey(v, "MIGRATION_MODE", "database.migration_mode"),
			MaxConnections:   v.GetInt("database.max_connections"),
			ConnectTimeout:   v.GetDuration("database.connect_timeout"),
			QueryTimeout:     v.GetDuration("database.query_timeout"),
			HealthcheckGrace: v.GetDuration("database.healthcheck_grace"),
		},
		Redis: RedisConfig{
			Addr:     pickEnvOrKey(v, "REDIS_ADDR", "redis.addr"),
			Password: pickEnvOrKey(v, "REDIS_PASSWORD", "redis.password"),
			DB:       v.GetInt("redis.db"),
		},
		Auth: AuthConfig{
			AdminKey:               resolvedAdminKeys,
			PluginInternalAPIToken: resolvePluginInternalAPIToken(v, resolvedAdminKeys),
		},
		Worker: WorkerConfig{
			ID:                 workerID,
			PollInterval:       v.GetDuration("worker.poll_interval"),
			AgentScanInterval:  v.GetDuration("worker.agent_scan_interval"),
			AgentLoopInterval:  v.GetDuration("worker.agent_loop_interval"),
			AgentErrorBackoff:  v.GetDuration("worker.agent_error_backoff"),
			SchedulePollLimit:  v.GetInt("worker.schedule_poll_limit"),
			ExecutionPollLimit: v.GetInt("worker.execution_poll_limit"),
		},
		Plugins: PluginsConfig{
			ModulesDir: pickEnvOrKey(v, "PLUGINS_DIR", "plugins.modules_dir"),
			SDKDir:     pickEnvOrKey(v, "PLUGIN_SDK_DIR", "plugins.sdk_dir"),
			PythonBin:  pickEnvOrKey(v, "PYTHON_BIN", "plugins.python_bin"),
			Timeout: PluginsTimeoutConfig{
				Account: pickDurationEnvOrKey(v, "PLUGINS_TIMEOUT_ACCOUNT", "plugins.timeout.account"),
				Job:     pickDurationEnvOrKey(v, "PLUGINS_TIMEOUT_JOB", "plugins.timeout.job"),
				Agent:   pickDurationEnvOrKey(v, "PLUGINS_TIMEOUT_AGENT", "plugins.timeout.agent"),
			},
			GRPC: PluginsGRPCConfig{
				AllowInsecureRemote: pickBoolEnvOrKey(v, "PLUGIN_GRPC_ALLOW_INSECURE_REMOTE", "plugins.grpc.allow_insecure_remote"),
				InsecureSkipVerify:  pickBoolEnvOrKey(v, "PLUGIN_GRPC_INSECURE_SKIP_VERIFY", "plugins.grpc.insecure_skip_verify"),
			},
			Services: loadPluginServices(v),
		},
		Logging: LoggingConfig{
			Level:    pickEnvOrKey(v, "LOG_LEVEL", "logging.level"),
			Format:   pickEnvOrKey(v, "LOG_FORMAT", "logging.format"),
			Filename: pickEnvOrKey(v, "LOG_FILE", "logging.filename"),
		},
	}, nil
}

func resolveAdminKeys(v *viper.Viper) string {
	if v == nil {
		return ""
	}
	if configured := strings.TrimSpace(pickEnvOrKey(v, "ADMIN_KEYS", "auth.admin_keys")); configured != "" {
		return configured
	}
	return strings.TrimSpace(pickEnvOrKey(v, "ADMIN_KEY", "auth.admin_key"))
}

func resolvePluginInternalAPIToken(v *viper.Viper, adminKeys string) string {
	if v == nil {
		return ""
	}
	if configured := strings.TrimSpace(pickEnvOrKey(v, "PLUGIN_INTERNAL_API_TOKEN", "auth.plugin_internal_api_token")); configured != "" {
		return configured
	}
	keys := strings.Split(adminKeys, ",")
	for _, key := range keys {
		if trimmed := strings.TrimSpace(key); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func DefaultPluginServices() map[string]PluginServiceEntry {
	return map[string]PluginServiceEntry{
		"octo_demo": {Address: "127.0.0.1:50051"},
	}
}

// loadPluginServices reads gRPC service addresses from the viper config.
// Config-file path: plugins.services.<key>.address
// Env var pattern:  PLUGIN_GRPC_<KEY>_ADDR  (e.g. PLUGIN_GRPC_OCTO_DEMO_ADDR)
func loadPluginServices(v *viper.Viper) map[string]PluginServiceEntry {
	services := DefaultPluginServices()

	// Read from structured config (plugins.services.* YAML/TOML section).
	raw := v.GetStringMap("plugins.services")
	for key, val := range raw {
		if m, ok := val.(map[string]any); ok {
			entry := services[key]
			if addr, _ := m["address"].(string); strings.TrimSpace(addr) != "" {
				entry.Address = strings.TrimSpace(addr)
			}
			if allowInsecure, ok := asBool(m["allow_insecure"]); ok {
				entry.AllowInsecure = allowInsecure
			}
			if tlsServerName, _ := m["tls_server_name"].(string); strings.TrimSpace(tlsServerName) != "" {
				entry.TLSServerName = strings.TrimSpace(tlsServerName)
			}
			if skipVerify, ok := asBool(m["tls_insecure_skip_verify"]); ok {
				entry.TLSInsecureSkipVerify = skipVerify
			}
			if strings.TrimSpace(entry.Address) != "" {
				services[key] = entry
			}
		}
	}

	// Overlay with any PLUGIN_GRPC_<KEY>_ADDR environment variables so that
	// individual plugin addresses can be set without a config file.
	for _, env := range os.Environ() {
		const prefix = "PLUGIN_GRPC_"
		const suffix = "_ADDR"
		if !strings.HasPrefix(env, prefix) {
			continue
		}
		parts := strings.SplitN(env, "=", 2)
		if len(parts) != 2 {
			continue
		}
		name, addr := parts[0], strings.TrimSpace(parts[1])
		if !strings.HasSuffix(name, suffix) || addr == "" {
			continue
		}
		key := strings.ToLower(strings.TrimPrefix(strings.TrimSuffix(name, suffix), prefix))
		entry := services[key]
		entry.Address = addr
		services[key] = entry
	}

	return services
}

func MustLoad() Config {
	cfg, err := Load()
	if err != nil {
		panic(err)
	}
	return cfg
}

// pickEnvOrKey returns the flat uppercase env var when non-empty,
// otherwise falls back to the nested viper key.
func pickEnvOrKey(v *viper.Viper, envKey, viperKey string) string {
	if val := strings.TrimSpace(v.GetString(envKey)); val != "" {
		return val
	}
	return v.GetString(viperKey)
}

func pickDurationEnvOrKey(v *viper.Viper, envKey, viperKey string) time.Duration {
	if val := strings.TrimSpace(v.GetString(envKey)); val != "" {
		if parsed, err := time.ParseDuration(val); err == nil {
			return parsed
		} else {
			slog.Warn("invalid duration env var, using default", "key", envKey, "value", val, "error", err)
		}
	}
	return v.GetDuration(viperKey)
}

func pickCSVEnvOrKey(v *viper.Viper, envKey, viperKey string) []string {
	if val := strings.TrimSpace(v.GetString(envKey)); val != "" {
		return splitCSV(val)
	}
	return normalizeStringSlice(v.GetStringSlice(viperKey))
}

func pickBoolEnvOrKey(v *viper.Viper, envKey, viperKey string) bool {
	if val := strings.TrimSpace(v.GetString(envKey)); val != "" {
		parsed, err := parseBool(val)
		if err == nil {
			return parsed
		}
		slog.Warn("invalid bool env var, using default", "key", envKey, "value", val, "error", err)
	}
	return v.GetBool(viperKey)
}

func splitCSV(raw string) []string {
	parts := strings.Split(raw, ",")
	return normalizeStringSlice(parts)
}

func normalizeStringSlice(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, item := range values {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		result = append(result, trimmed)
	}
	return result
}

func asBool(raw any) (bool, bool) {
	switch value := raw.(type) {
	case bool:
		return value, true
	case string:
		parsed, err := parseBool(value)
		if err != nil {
			return false, false
		}
		return parsed, true
	default:
		return false, false
	}
}

func parseBool(raw string) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "1", "true", "yes", "on":
		return true, nil
	case "0", "false", "no", "off":
		return false, nil
	default:
		return false, fmt.Errorf("invalid bool value %q", raw)
	}
}
