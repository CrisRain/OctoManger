package grpcclient

import (
	"context"
	"testing"

	pluginapp "octomanger/internal/domains/plugins/app"
	plugindomain "octomanger/internal/domains/plugins/domain"
)

func TestDecodeManifestPreservesPluginUIAndActions(t *testing.T) {
	raw := []byte(`{
		"key":"octo_demo",
		"name":"Octo Demo",
		"schema":{"type":"object"},
		"settings":[{"key":"token","label":"Token"}],
		"ui":{"tabs":[{"key":"overview"}]},
		"capabilities":{
			"actions":[
				{"key":"VERIFY","description":"Verify credentials"},
				{"key":"SYNC","name":"Sync Profile","modes":["sync","job"]}
			],
			"webhook": true
		}
	}`)

	manifest, err := decodeManifest(raw, "fallback_key")
	if err != nil {
		t.Fatalf("decode manifest: %v", err)
	}

	if manifest.Key != "octo_demo" {
		t.Fatalf("unexpected key %q", manifest.Key)
	}
	if manifest.Name != "Octo Demo" {
		t.Fatalf("unexpected name %q", manifest.Name)
	}
	if len(manifest.Actions) != 2 {
		t.Fatalf("expected 2 actions, got %d", len(manifest.Actions))
	}
	if manifest.Actions[0].Name != "VERIFY" {
		t.Fatalf("expected missing action name to fall back to key, got %q", manifest.Actions[0].Name)
	}
	if len(manifest.Settings) != 1 {
		t.Fatalf("expected settings to be preserved")
	}
	if len(manifest.UI) == 0 {
		t.Fatalf("expected ui payload to be preserved")
	}
	if len(manifest.Capabilities) != 1 || manifest.Capabilities[0] != "webhook" {
		t.Fatalf("unexpected capabilities %#v", manifest.Capabilities)
	}
}

func TestDecodeManifestFallsBackToRegistryKey(t *testing.T) {
	manifest, err := decodeManifest([]byte(`{"name":"Demo"}`), "octo_demo")
	if err != nil {
		t.Fatalf("decode manifest: %v", err)
	}

	if manifest.Key != "octo_demo" {
		t.Fatalf("expected fallback key, got %q", manifest.Key)
	}
	if manifest.Name != "Demo" {
		t.Fatalf("unexpected name %q", manifest.Name)
	}
}

func TestInjectSettingsIncludesInternalAPIContext(t *testing.T) {
	client := New(stubRegistry{}).WithInternalAPI(pluginapp.InternalAPIConfig{
		URL:            "http://127.0.0.1:8080",
		Token:          "secret-token",
		TimeoutSeconds: 45,
	})

	request, err := client.injectSettings(context.Background(), "octo_demo", plugindomain.ExecutionRequest{
		Mode:   "account",
		Action: "LIST_TASKS",
	})
	if err != nil {
		t.Fatalf("inject settings: %v", err)
	}

	if got := request.Context["plugin_key"]; got != "octo_demo" {
		t.Fatalf("expected plugin_key octo_demo, got %#v", got)
	}
	if got := request.Context["api_url"]; got != "http://127.0.0.1:8080" {
		t.Fatalf("expected api_url to be injected, got %#v", got)
	}
	if got := request.Context["api_token"]; got != "secret-token" {
		t.Fatalf("expected api_token to be injected, got %#v", got)
	}
	if got := request.Context["api_timeout_seconds"]; got != 45 {
		t.Fatalf("expected api timeout to be injected, got %#v", got)
	}
}

func TestShouldUseInsecureTransport(t *testing.T) {
	tests := []struct {
		name     string
		service  PluginServiceConfig
		security TransportSecurityConfig
		want     bool
	}{
		{
			name:    "loopback defaults to insecure",
			service: PluginServiceConfig{Address: "127.0.0.1:50051"},
			want:    true,
		},
		{
			name:    "localhost defaults to insecure",
			service: PluginServiceConfig{Address: "localhost:50051"},
			want:    true,
		},
		{
			name:    "remote defaults to tls",
			service: PluginServiceConfig{Address: "example.com:50051"},
			want:    false,
		},
		{
			name:     "remote can be explicitly insecure",
			service:  PluginServiceConfig{Address: "example.com:50051"},
			security: TransportSecurityConfig{AllowInsecureRemote: true},
			want:     true,
		},
		{
			name:    "service override allows insecure",
			service: PluginServiceConfig{Address: "example.com:50051", AllowInsecure: true},
			want:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := shouldUseInsecureTransport(tt.service, tt.security)
			if got != tt.want {
				t.Fatalf("shouldUseInsecureTransport() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHostFromAddress(t *testing.T) {
	tests := []struct {
		address string
		want    string
	}{
		{address: "127.0.0.1:50051", want: "127.0.0.1"},
		{address: "localhost:50051", want: "localhost"},
		{address: "dns:///plugins.internal:50051", want: "plugins.internal"},
		{address: "unix:///tmp/plugin.sock", want: "localhost"},
	}

	for _, tt := range tests {
		t.Run(tt.address, func(t *testing.T) {
			if got := hostFromAddress(tt.address); got != tt.want {
				t.Fatalf("hostFromAddress(%q) = %q, want %q", tt.address, got, tt.want)
			}
		})
	}
}

type stubRegistry struct{}

func (stubRegistry) Address(string) (string, error) { return "", nil }
func (stubRegistry) ServiceConfig(string) (PluginServiceConfig, error) {
	return PluginServiceConfig{}, nil
}
func (stubRegistry) Keys() []string { return nil }
