package config

import (
	"log/slog"
	"testing"
	"time"

	"github.com/spf13/viper"
)

func TestPickEnvOrKey(t *testing.T) {
	v := viper.New()
	v.Set("ENV_KEY", "value")
	v.Set("nested.key", "fallback")
	if got := pickEnvOrKey(v, "ENV_KEY", "nested.key"); got != "value" {
		t.Fatalf("expected env value, got %q", got)
	}

	v.Set("ENV_KEY", "")
	if got := pickEnvOrKey(v, "ENV_KEY", "nested.key"); got != "fallback" {
		t.Fatalf("expected fallback value, got %q", got)
	}
}

func TestPickDurationEnvOrKey(t *testing.T) {
	v := viper.New()
	v.Set("duration.key", "10s")
	v.Set("ENV_DURATION", "bad")
	if got := pickDurationEnvOrKey(v, "ENV_DURATION", "duration.key"); got != 10*time.Second {
		t.Fatalf("expected fallback duration, got %s", got)
	}

	v.Set("ENV_DURATION", "2m")
	if got := pickDurationEnvOrKey(v, "ENV_DURATION", "duration.key"); got != 2*time.Minute {
		t.Fatalf("expected env duration, got %s", got)
	}
}

func TestPickCSVEnvOrKey(t *testing.T) {
	v := viper.New()
	v.Set("csv.key", []string{"x"})
	v.Set("ENV_CSV", "a, b,,c")
	values := pickCSVEnvOrKey(v, "ENV_CSV", "csv.key")
	if len(values) != 3 || values[0] != "a" || values[1] != "b" || values[2] != "c" {
		t.Fatalf("unexpected csv values %#v", values)
	}
}

func TestPickBoolEnvOrKey(t *testing.T) {
	v := viper.New()
	v.Set("flag.key", false)
	v.Set("ENV_FLAG", "true")
	if got := pickBoolEnvOrKey(v, "ENV_FLAG", "flag.key"); !got {
		t.Fatalf("expected true from env")
	}

	v.Set("ENV_FLAG", "")
	v.Set("flag.key", true)
	if got := pickBoolEnvOrKey(v, "ENV_FLAG", "flag.key"); !got {
		t.Fatalf("expected true from viper")
	}
}

func TestSplitCSVAndNormalize(t *testing.T) {
	values := splitCSV(" a, ,b, c ")
	if len(values) != 3 || values[0] != "a" || values[1] != "b" || values[2] != "c" {
		t.Fatalf("unexpected values %#v", values)
	}

	if got := normalizeStringSlice([]string{" a ", "", "b"}); len(got) != 2 || got[0] != "a" || got[1] != "b" {
		t.Fatalf("unexpected normalized values %#v", got)
	}
}

func TestAsBoolAndParseBool(t *testing.T) {
	if v, ok := asBool(true); !ok || v != true {
		t.Fatalf("expected bool true")
	}
	if v, ok := asBool("1"); !ok || v != true {
		t.Fatalf("expected string bool true")
	}
	if _, ok := asBool(slog.LevelInfo); ok {
		t.Fatalf("expected non-bool to be rejected")
	}

	if _, err := parseBool("maybe"); err == nil {
		t.Fatalf("expected parse error")
	}
}

func TestLoadPluginServicesOverrides(t *testing.T) {
	v := viper.New()
	v.Set("plugins.services", map[string]any{
		"demo": map[string]any{
			"address":                  "127.0.0.1:60051",
			"allow_insecure":           true,
			"tls_server_name":          "server",
			"tls_insecure_skip_verify": true,
		},
	})
	t.Setenv("PLUGIN_GRPC_EXTRA_ADDR", "127.0.0.1:70051")

	services := loadPluginServices(v)
	if services["demo"].Address != "127.0.0.1:60051" {
		t.Fatalf("unexpected demo address %q", services["demo"].Address)
	}
	if !services["demo"].AllowInsecure || services["demo"].TLSServerName != "server" || !services["demo"].TLSInsecureSkipVerify {
		t.Fatalf("unexpected demo flags %#v", services["demo"])
	}
	if services["extra"].Address != "127.0.0.1:70051" {
		t.Fatalf("unexpected extra address %q", services["extra"].Address)
	}
}
