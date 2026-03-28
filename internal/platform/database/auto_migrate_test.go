package database

import (
	"testing"

	platformconfig "octomanger/internal/platform/config"
)

func TestDefaultSystemSettingsSeed(t *testing.T) {
	item := defaultSystemSettingsSeed()

	if item.ID != systemSettingsSingletonID {
		t.Fatalf("unexpected system settings id %d", item.ID)
	}
	if item.AppName != "OctoManager" {
		t.Fatalf("unexpected app name seed %q", item.AppName)
	}
	if item.JobDefaultTimeoutMinutes != 30 {
		t.Fatalf("unexpected timeout seed %d", item.JobDefaultTimeoutMinutes)
	}
	if item.JobMaxConcurrency != 10 {
		t.Fatalf("unexpected concurrency seed %d", item.JobMaxConcurrency)
	}
	if item.PluginInternalAPIToken != "" {
		t.Fatalf("expected empty plugin internal api token seed")
	}
}

func TestDefaultPluginServiceSeeds(t *testing.T) {
	cfg := &platformconfig.Config{
		Plugins: platformconfig.PluginsConfig{
			Services: map[string]platformconfig.PluginServiceEntry{
				"octo-demo": {Address: "127.0.0.1:60051"},
			},
		},
	}

	items := defaultPluginServiceSeeds(cfg)
	if len(items) != 1 {
		t.Fatalf("expected 1 plugin service seed, got %d", len(items))
	}
	if items[0].PluginKey != "octo_demo" {
		t.Fatalf("unexpected plugin key %q", items[0].PluginKey)
	}
	if items[0].GRPCAddress != "127.0.0.1:60051" {
		t.Fatalf("unexpected grpc address %q", items[0].GRPCAddress)
	}
}

func TestDecodeLegacyPluginServicesAcceptsUppercaseAddress(t *testing.T) {
	items := decodeLegacyPluginServices([]byte(`{
		"octo_demo": {"Address": "127.0.0.1:70051"}
	}`))

	if items["octo_demo"] != "127.0.0.1:70051" {
		t.Fatalf("unexpected decoded legacy address %q", items["octo_demo"])
	}
}
