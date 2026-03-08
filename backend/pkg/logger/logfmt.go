package logger

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"go.uber.org/zap/zapcore"
)

// logfmtSyncer wraps a WriteSyncer and converts trailing JSON field objects
// produced by zap's ConsoleEncoder into key=value pairs.
//
// Console encoder output format:
//
//	<time>  <LEVEL>  <caller>  <message>  {"k":"v",...}
//
// logfmtSyncer rewrites the trailing JSON blob to:
//
//	<time>  <LEVEL>  <caller>  <message>  k=v  k2=v2
type logfmtSyncer struct {
	ws  zapcore.WriteSyncer
	sep string // field separator used by the console encoder (e.g. "  ")
}

func newLogfmtSyncer(ws zapcore.WriteSyncer, sep string) zapcore.WriteSyncer {
	return &logfmtSyncer{ws: ws, sep: sep}
}

func (s *logfmtSyncer) Write(p []byte) (int, error) {
	line := strings.TrimRight(string(p), "\n")

	needle := s.sep + "{"
	idx := strings.LastIndex(line, needle)
	if idx >= 0 {
		jsonStr := line[idx+len(s.sep):]
		if strings.HasPrefix(jsonStr, "{") && strings.HasSuffix(jsonStr, "}") {
			var m map[string]interface{}
			if json.Unmarshal([]byte(jsonStr), &m) == nil && len(m) > 0 {
				line = line[:idx] + s.sep + mapToLogfmt(m)
			}
		}
	}

	return s.ws.Write([]byte(line + "\n"))
}

func (s *logfmtSyncer) Sync() error { return s.ws.Sync() }

func mapToLogfmt(m map[string]interface{}) string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, k+"="+formatValue(m[k]))
	}
	return strings.Join(parts, "  ")
}

func formatValue(v interface{}) string {
	switch t := v.(type) {
	case string:
		if strings.ContainsAny(t, " \t") {
			// quote strings that contain whitespace
			if len(t) > 80 {
				t = t[:77] + "..."
			}
			return fmt.Sprintf("%q", t)
		}
		return t
	case float64:
		if t == float64(int64(t)) {
			return fmt.Sprintf("%d", int64(t))
		}
		return fmt.Sprintf("%g", t)
	case bool:
		if t {
			return "true"
		}
		return "false"
	case nil:
		return "null"
	default:
		b, err := json.Marshal(v)
		if err != nil {
			return fmt.Sprintf("%v", v)
		}
		return string(b)
	}
}
