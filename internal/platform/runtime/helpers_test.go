package runtime

import (
	"context"
	"os"
	"testing"
	"time"

	pluginapp "octomanger/internal/domains/plugins/app"
	"octomanger/internal/domains/plugins/grpcclient"
	"octomanger/internal/platform/config"
)

func TestNormalizePluginInternalAPIURL(t *testing.T) {
	cases := map[string]string{
		"":                        "",
		":8080":                   "http://127.0.0.1:8080",
		"0.0.0.0:8080":            "http://127.0.0.1:8080",
		"[::]:8080":               "http://127.0.0.1:8080",
		"example.com:9000":        "http://example.com:9000",
		"http://example.com/":     "http://example.com",
		"https://example.com/api": "https://example.com/api",
	}

	for input, want := range cases {
		if got := normalizePluginInternalAPIURL(input); got != want {
			t.Fatalf("normalize %q -> %q, want %q", input, got, want)
		}
	}
}

func TestBuildPluginInternalAPIConfig(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{APIAddr: ":8080"},
		Auth:   config.AuthConfig{PluginInternalAPIToken: "token"},
		Plugins: config.PluginsConfig{
			Timeout: config.PluginsTimeoutConfig{Account: 0},
		},
	}

	got := buildPluginInternalAPIConfig(cfg)
	if got.URL != "http://127.0.0.1:8080" {
		t.Fatalf("unexpected url %q", got.URL)
	}
	if got.Token != "token" {
		t.Fatalf("unexpected token %q", got.Token)
	}
	if got.TimeoutSeconds != 60 {
		t.Fatalf("expected default timeout 60, got %d", got.TimeoutSeconds)
	}

	cfg.Plugins.Timeout.Account = 30 * time.Second
	got = buildPluginInternalAPIConfig(cfg)
	if got.TimeoutSeconds != 30 {
		t.Fatalf("expected custom timeout, got %d", got.TimeoutSeconds)
	}
}

func TestToGRPCServiceMap(t *testing.T) {
	input := map[string]config.PluginServiceEntry{
		"demo": {Address: "127.0.0.1:60051", AllowInsecure: true},
	}
	out := toGRPCServiceMap(input)
	if out["demo"].Address != "127.0.0.1:60051" || !out["demo"].AllowInsecure {
		t.Fatalf("unexpected grpc service map %#v", out)
	}
}

func TestCurrentProcessName(t *testing.T) {
	origArgs := os.Args
	os.Args = []string{""}
	if got := currentProcessName(); got != "octomanger" {
		t.Fatalf("expected default process name, got %q", got)
	}
	os.Args = []string{"/usr/bin/octo-worker"}
	if got := currentProcessName(); got != "octo-worker" {
		t.Fatalf("expected basename, got %q", got)
	}
	os.Args = origArgs
}

func TestNormalizePluginKey(t *testing.T) {
	if got := normalizePluginKey("Octo-Demo"); got != "octo_demo" {
		t.Fatalf("unexpected normalized key %q", got)
	}
}

func TestNormalizePluginServices(t *testing.T) {
	services := map[string]config.PluginServiceEntry{
		"demo": {Address: " 127.0.0.1:60051 ", TLSServerName: " server "},
		"bad":  {Address: ""},
	}
	out := normalizePluginServices(services)
	if out["demo"].Address != "127.0.0.1:60051" || out["demo"].TLSServerName != "server" {
		t.Fatalf("unexpected normalized services %#v", out)
	}
	if _, ok := out["bad"]; ok {
		t.Fatalf("expected empty address entry to be dropped")
	}
}

func TestNormalizePluginServicesMergeDefaults(t *testing.T) {
	current := map[string]config.PluginServiceEntry{
		"demo": {Address: "127.0.0.1:60051"},
	}
	defaults := map[string]config.PluginServiceEntry{
		"demo":  {Address: "127.0.0.1:50051", AllowInsecure: true, TLSServerName: "server", TLSInsecureSkipVerify: true},
		"extra": {Address: "127.0.0.1:70051"},
	}
	store := &stubPluginConfigStore{values: map[string]string{"demo": current["demo"].Address}}

	resolved, err := resolvePluginServices(context.Background(), store, defaults)
	if err != nil {
		t.Fatalf("resolve services: %v", err)
	}
	if resolved["demo"].Address != "127.0.0.1:60051" {
		t.Fatalf("expected stored address, got %q", resolved["demo"].Address)
	}
	if resolved["demo"].AllowInsecure != true || resolved["demo"].TLSServerName != "server" || !resolved["demo"].TLSInsecureSkipVerify {
		t.Fatalf("expected merged flags %#v", resolved["demo"])
	}
	if resolved["extra"].Address != "127.0.0.1:70051" {
		t.Fatalf("expected extra service to be seeded")
	}
}

func TestToGRPCServiceMapUsesFields(t *testing.T) {
	cfg := map[string]config.PluginServiceEntry{
		"demo": {
			Address:               "127.0.0.1:50051",
			AllowInsecure:         true,
			TLSServerName:         "server",
			TLSInsecureSkipVerify: true,
		},
	}
	out := toGRPCServiceMap(cfg)
	if out["demo"].TLSServerName != "server" || !out["demo"].TLSInsecureSkipVerify {
		t.Fatalf("unexpected grpc config %#v", out["demo"])
	}
}

func TestBuildPluginInternalAPIConfigNil(t *testing.T) {
	if got := buildPluginInternalAPIConfig(nil); got != (pluginapp.InternalAPIConfig{}) {
		t.Fatalf("expected empty config, got %#v", got)
	}
}

func TestPluginInternalAPIURLWithHostOnly(t *testing.T) {
	if got := normalizePluginInternalAPIURL("localhost"); got != "http://localhost" {
		t.Fatalf("unexpected url %q", got)
	}
}

func TestGRPCServiceMapType(t *testing.T) {
	out := toGRPCServiceMap(map[string]config.PluginServiceEntry{"demo": {Address: "127.0.0.1:50051"}})
	if _, ok := out["demo"]; !ok {
		t.Fatalf("expected demo service in map")
	}
	_ = grpcclient.PluginServiceConfig(out["demo"])
}
