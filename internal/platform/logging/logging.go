package logging

import (
	"os"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/lumberjack.v2"

	"octomanger/internal/platform/config"
)

// New returns a *zap.Logger configured from LoggingConfig.
// When Filename is empty, logs go to stdout.
// Lumberjack handles rotation when writing to file.
func New(cfg config.LoggingConfig) *zap.Logger {
	level := parseLevel(cfg.Level)

	encoderCfg := zap.NewProductionEncoderConfig()
	encoderCfg.TimeKey = "time"
	encoderCfg.EncodeTime = zapcore.ISO8601TimeEncoder

	var encoder zapcore.Encoder
	if strings.ToLower(strings.TrimSpace(cfg.Format)) == "json" {
		encoder = zapcore.NewJSONEncoder(encoderCfg)
	} else {
		encoderCfg.EncodeLevel = zapcore.CapitalColorLevelEncoder
		encoder = zapcore.NewConsoleEncoder(encoderCfg)
	}

	var sink zapcore.WriteSyncer
	if cfg.Filename != "" {
		sink = zapcore.AddSync(&lumberjack.Logger{
			Filename:   cfg.Filename,
			MaxSize:    100, // MB
			MaxBackups: 5,
			MaxAge:     30, // days
			Compress:   true,
		})
	} else {
		sink = zapcore.AddSync(os.Stdout)
	}

	core := zapcore.NewCore(encoder, sink, level)
	return zap.New(core, zap.AddCaller(), zap.AddCallerSkip(0))
}

func parseLevel(raw string) zapcore.Level {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "debug":
		return zapcore.DebugLevel
	case "warn", "warning":
		return zapcore.WarnLevel
	case "error":
		return zapcore.ErrorLevel
	default:
		return zapcore.InfoLevel
	}
}
