package logger

import (
	"bytes"
	"log"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// NewStdLogger returns a *log.Logger that writes to the given zap.Logger at
// the specified level. Messages containing any of the suppressSubstrings
// fragments are silently dropped, which is useful for suppressing expected
// TLS noise (e.g. "tls: unknown certificate" from self-signed certs).
func NewStdLogger(z *zap.Logger, level zapcore.Level, suppressSubstrings ...string) *log.Logger {
	return log.New(&zapWriter{z: z.WithOptions(zap.AddCallerSkip(3)), level: level, suppress: suppressSubstrings}, "", 0)
}

type zapWriter struct {
	z        *zap.Logger
	level    zapcore.Level
	suppress []string
}

func (w *zapWriter) Write(p []byte) (int, error) {
	msg := strings.TrimRight(string(bytes.TrimSpace(p)), "\n")
	if msg == "" {
		return len(p), nil
	}
	for _, s := range w.suppress {
		if strings.Contains(msg, s) {
			return len(p), nil
		}
	}
	if ce := w.z.Check(w.level, msg); ce != nil {
		ce.Write()
	}
	return len(p), nil
}
