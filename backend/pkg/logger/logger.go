package logger

import (
	"context"
	"errors"
	"os"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"octomanger/backend/config"
)

type traceIDKey struct{}

func WithTraceID(ctx context.Context, traceID string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	trimmed := strings.TrimSpace(traceID)
	if trimmed == "" {
		return ctx
	}
	return context.WithValue(ctx, traceIDKey{}, trimmed)
}

func TraceIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if value := ctx.Value(traceIDKey{}); value != nil {
		if traceID, ok := value.(string); ok {
			return strings.TrimSpace(traceID)
		}
	}
	return ""
}

func WithContext(ctx context.Context, log *zap.Logger) *zap.Logger {
	if log == nil {
		return nil
	}
	traceID := TraceIDFromContext(ctx)
	if traceID == "" {
		return log
	}
	return log.With(zap.String("trace_id", traceID))
}

func Init(cfg config.LoggingConfig) (*zap.Logger, error) {
	level := zapcore.InfoLevel
	if err := level.Set(strings.ToLower(strings.TrimSpace(cfg.Level))); err != nil {
		return nil, err
	}

	ws, err := buildWriteSyncer(cfg.File)
	if err != nil {
		return nil, err
	}

	var enc zapcore.Encoder
	const consoleSep = "  "
	if strings.ToLower(strings.TrimSpace(cfg.Format)) == "json" {
		encCfg := zap.NewProductionEncoderConfig()
		encCfg.TimeKey = "ts"
		encCfg.EncodeTime = zapcore.ISO8601TimeEncoder
		enc = zapcore.NewJSONEncoder(encCfg)
	} else {
		// console format: human-readable with color level labels
		encCfg := zap.NewDevelopmentEncoderConfig()
		encCfg.EncodeTime = zapcore.TimeEncoderOfLayout("2006-01-02 15:04:05.000")
		encCfg.EncodeLevel = zapcore.CapitalColorLevelEncoder
		encCfg.EncodeCaller = zapcore.ShortCallerEncoder
		encCfg.ConsoleSeparator = consoleSep
		enc = zapcore.NewConsoleEncoder(encCfg)
		ws = newLogfmtSyncer(ws, consoleSep)
	}

	core := zapcore.NewCore(enc, ws, level)
	logger := zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))
	return logger, nil
}

func buildWriteSyncer(path string) (zapcore.WriteSyncer, error) {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return zapcore.AddSync(os.Stdout), nil
	}

	file, err := os.OpenFile(trimmed, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, err
	}

	if file == nil {
		return nil, errors.New("failed to open log file")
	}

	return zapcore.NewMultiWriteSyncer(
		zapcore.AddSync(os.Stdout),
		zapcore.AddSync(file),
	), nil
}
