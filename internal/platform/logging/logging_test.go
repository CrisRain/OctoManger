package logging

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"octomanger/internal/platform/config"
	"octomanger/internal/testutil"
)

func TestParseLevel(t *testing.T) {
	if parseLevel("debug") != zapcore.DebugLevel {
		t.Fatalf("expected debug level")
	}
	if parseLevel("warn") != zapcore.WarnLevel {
		t.Fatalf("expected warn level")
	}
	if parseLevel("warning") != zapcore.WarnLevel {
		t.Fatalf("expected warning level")
	}
	if parseLevel("error") != zapcore.ErrorLevel {
		t.Fatalf("expected error level")
	}
	if parseLevel("") != zapcore.InfoLevel {
		t.Fatalf("expected default info level")
	}
}

func TestAttachSystemLogSinkNil(t *testing.T) {
	base := New(config.LoggingConfig{Level: "info"})
	logger, flush := AttachSystemLogSink(base, nil, config.LoggingConfig{}, "source")
	if logger != base {
		t.Fatalf("expected logger to be unchanged when redis nil")
	}
	flush()
}

func TestSystemLogCoreWriteStoresLog(t *testing.T) {
	rdb, _ := testutil.NewTestRedis(t)
	base := New(config.LoggingConfig{Level: "info"})
	logger, _ := AttachSystemLogSink(base, rdb, config.LoggingConfig{Level: "info"}, "worker")

	logger.Info("hello", zap.String("k", "v"))

	ctx := context.Background()
	raw, err := rdb.LPop(ctx, SystemRuntimeLogsRedisKey).Result()
	if err != nil {
		t.Fatalf("read redis log: %v", err)
	}

	var record SystemRuntimeLogRecord
	if err := json.Unmarshal([]byte(raw), &record); err != nil {
		t.Fatalf("decode log record: %v", err)
	}
	if record.Source != "worker" {
		t.Fatalf("unexpected source %q", record.Source)
	}
	if record.Message != "hello" {
		t.Fatalf("unexpected message %q", record.Message)
	}
	if record.Fields["k"] != "v" {
		t.Fatalf("unexpected fields %#v", record.Fields)
	}
	if record.CreatedAt.IsZero() {
		t.Fatalf("expected created_at to be set")
	}
}

func TestSystemLogCoreWith(t *testing.T) {
	core := &systemLogCore{fields: []zapcore.Field{zap.String("a", "b")}}
	clone := core.With([]zapcore.Field{zap.String("c", "d")}).(*systemLogCore)
	if len(clone.fields) != 2 {
		t.Fatalf("expected merged fields, got %d", len(clone.fields))
	}
	if len(core.fields) != 1 {
		t.Fatalf("expected original core unchanged")
	}
}

func TestEncodeSystemLogFields(t *testing.T) {
	payload := encodeSystemLogFields([]zapcore.Field{zap.String("foo", "bar")})
	if payload["foo"] != "bar" {
		t.Fatalf("unexpected payload %#v", payload)
	}
	if got := encodeSystemLogFields(nil); len(got) != 0 {
		t.Fatalf("expected empty payload")
	}
}

func TestNormalizeCallerAndAPIAccessLog(t *testing.T) {
	caller := zapcore.EntryCaller{Defined: false}
	if normalizeCaller(caller) != "" {
		t.Fatalf("expected empty caller for undefined")
	}

	caller = zapcore.EntryCaller{Defined: true, File: "C:/app/apiserver/logging.go", Line: 10}
	normalized := normalizeCaller(caller)
	if normalized == "" {
		t.Fatalf("expected normalized caller")
	}

	entry := zapcore.Entry{
		LoggerName: "api",
		Caller:     caller,
		Time:       time.Now(),
	}
	if !isAPIAccessLog(entry) {
		t.Fatalf("expected api access log to be detected")
	}
}
