package logging

import (
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"octomanger/internal/platform/config"
)

const (
	SystemRuntimeLogsRedisKey    = "system:runtime_logs"
	SystemRuntimeLogsSeqRedisKey = "system:runtime_logs:next_id"
	SystemRuntimeLogsRetention   = 5000
)

type SystemRuntimeLogRecord struct {
	ID        int64          `json:"id"`
	Source    string         `json:"source"`
	Level     string         `json:"level"`
	Logger    string         `json:"logger"`
	Caller    string         `json:"caller"`
	Message   string         `json:"message"`
	Fields    map[string]any `json:"fields"`
	CreatedAt time.Time      `json:"created_at"`
}

type systemLogCore struct {
	level  zapcore.LevelEnabler
	source string
	fields []zapcore.Field
	rdb    *redis.Client
}

func AttachSystemLogSink(base *zap.Logger, rdb *redis.Client, cfg config.LoggingConfig, source string) (*zap.Logger, func()) {
	if base == nil || rdb == nil {
		return base, func() {}
	}

	core := &systemLogCore{
		level:  parseLevel(cfg.Level),
		source: strings.TrimSpace(source),
		fields: nil,
		rdb:    rdb,
	}
	logger := base.WithOptions(zap.WrapCore(func(existing zapcore.Core) zapcore.Core {
		return zapcore.NewTee(existing, core)
	}))

	return logger, func() {}
}

func (c *systemLogCore) Enabled(level zapcore.Level) bool {
	return c.level.Enabled(level)
}

func (c *systemLogCore) With(fields []zapcore.Field) zapcore.Core {
	clone := *c
	clone.fields = append(append([]zapcore.Field{}, c.fields...), fields...)
	return &clone
}

func (c *systemLogCore) Check(entry zapcore.Entry, checked *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if c.Enabled(entry.Level) {
		return checked.AddCore(entry, c)
	}
	return checked
}

func (c *systemLogCore) Write(entry zapcore.Entry, fields []zapcore.Field) error {
	if c.rdb == nil {
		return nil
	}
	if isAPIAccessLog(entry) {
		return nil
	}

	payload := encodeSystemLogFields(append(append([]zapcore.Field{}, c.fields...), fields...))
	if entry.Stack != "" {
		payload["stacktrace"] = entry.Stack
	}

	ctx := context.Background()
	record := SystemRuntimeLogRecord{
		Source:    c.source,
		Level:     strings.ToLower(entry.Level.String()),
		Logger:    strings.TrimSpace(entry.LoggerName),
		Caller:    normalizeCaller(entry.Caller),
		Message:   entry.Message,
		Fields:    payload,
		CreatedAt: entry.Time.UTC(),
	}

	nextID, err := c.rdb.Incr(ctx, SystemRuntimeLogsSeqRedisKey).Result()
	if err != nil {
		return nil
	}
	record.ID = nextID

	raw, err := json.Marshal(record)
	if err != nil {
		return nil
	}

	pipe := c.rdb.Pipeline()
	pipe.LPush(ctx, SystemRuntimeLogsRedisKey, raw)
	pipe.LTrim(ctx, SystemRuntimeLogsRedisKey, 0, SystemRuntimeLogsRetention-1)
	_, _ = pipe.Exec(ctx)
	return nil
}

func isAPIAccessLog(entry zapcore.Entry) bool {
	if strings.TrimSpace(entry.LoggerName) != "api" {
		return false
	}
	caller := normalizeCaller(entry.Caller)
	return strings.Contains(caller, "apiserver/logging.go")
}

func (c *systemLogCore) Sync() error {
	return nil
}

func encodeSystemLogFields(fields []zapcore.Field) map[string]any {
	if len(fields) == 0 {
		return map[string]any{}
	}

	encoder := zapcore.NewMapObjectEncoder()
	for _, field := range fields {
		field.AddTo(encoder)
	}

	payload := make(map[string]any, len(encoder.Fields))
	for key, value := range encoder.Fields {
		payload[key] = value
	}
	return payload
}

func normalizeCaller(caller zapcore.EntryCaller) string {
	if !caller.Defined {
		return ""
	}
	trimmed := strings.TrimSpace(caller.TrimmedPath())
	if trimmed != "" {
		return trimmed
	}
	return filepath.ToSlash(caller.File)
}
