package middleware

import (
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	applogger "octomanger/backend/pkg/logger"
)

const requestIDKey = "request_id"

// staticPrefixes contains path prefixes that are logged at Debug level rather
// than Info so they don't pollute normal log output.
var staticPrefixes = []string{"/assets/", "/favicon"}

func Logger(logger *zap.Logger) gin.HandlerFunc {
	if logger == nil {
		logger = zap.NewNop()
	}
	return func(c *gin.Context) {
		requestID := strings.TrimSpace(c.GetHeader("X-Request-ID"))
		if requestID == "" {
			requestID = uuid.NewString()
		}
		c.Set(requestIDKey, requestID)
		c.Writer.Header().Set("X-Request-ID", requestID)
		c.Request = c.Request.WithContext(applogger.WithTraceID(c.Request.Context(), requestID))

		start := time.Now()
		c.Next()

		status := c.Writer.Status()
		dur := time.Since(start)
		path := c.Request.URL.RequestURI()

		msg := fmt.Sprintf("%s %s %d %s",
			c.Request.Method,
			path,
			status,
			formatDuration(dur),
		)

		fields := []zap.Field{
			zap.String("request_id", requestID),
			zap.String("trace_id", requestID),
			zap.Int("bytes", c.Writer.Size()),
			zap.String("ip", c.ClientIP()),
		}

		switch {
		case status >= 500:
			logger.Error(msg, fields...)
		case status >= 400:
			logger.Warn(msg, fields...)
		case path == "/healthz" || isStaticPath(path):
			// health checks and static assets only visible at debug level
			logger.Debug(msg, fields...)
		default:
			logger.Info(msg, fields...)
		}
	}
}

func isStaticPath(path string) bool {
	for _, prefix := range staticPrefixes {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}
	return false
}

func RequestID(c *gin.Context) string {
	if c == nil {
		return "-"
	}
	if value, exists := c.Get(requestIDKey); exists {
		if requestID, ok := value.(string); ok && strings.TrimSpace(requestID) != "" {
			return requestID
		}
	}
	if requestID := strings.TrimSpace(c.GetHeader("X-Request-ID")); requestID != "" {
		return requestID
	}
	return "-"
}

func formatDuration(d time.Duration) string {
	switch {
	case d < time.Millisecond:
		return fmt.Sprintf("%.2fµs", float64(d.Microseconds()))
	case d < time.Second:
		return fmt.Sprintf("%.2fms", float64(d.Microseconds())/1000)
	default:
		return fmt.Sprintf("%.2fs", d.Seconds())
	}
}
