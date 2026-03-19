package config

import (
	"fmt"
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
	APIAddr    string
	WebDistDir string
}

type DatabaseConfig struct {
	DSN              string
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
	AdminKey string
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
	ModulesDir string
	SDKDir     string
	PythonBin  string
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
	v.SetDefault("server.web_dist_dir", "apps/web/dist")
	v.SetDefault("database.max_connections", 8)
	v.SetDefault("database.connect_timeout", "5s")
	v.SetDefault("database.query_timeout", "10s")
	v.SetDefault("database.healthcheck_grace", "250ms")
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

	return Config{
		App: AppConfig{
			Env: v.GetString("app.env"),
		},
		Server: ServerConfig{
			APIAddr:    pickEnvOrKey(v, "API_ADDR", "server.api_addr"),
			WebDistDir: pickEnvOrKey(v, "WEB_DIST_DIR", "server.web_dist_dir"),
		},
		Database: DatabaseConfig{
			DSN:              dsn,
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
			AdminKey: strings.TrimSpace(pickEnvOrKey(v, "ADMIN_KEY", "auth.admin_key")),
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
		},
		Logging: LoggingConfig{
			Level:    pickEnvOrKey(v, "LOG_LEVEL", "logging.level"),
			Format:   pickEnvOrKey(v, "LOG_FORMAT", "logging.format"),
			Filename: pickEnvOrKey(v, "LOG_FILE", "logging.filename"),
		},
	}, nil
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
