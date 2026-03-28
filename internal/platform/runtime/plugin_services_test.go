package runtime

import (
	"context"
	"fmt"
	"testing"

	"octomanger/internal/platform/config"
)

func TestResolvePluginServicesInitializesDefaultsIntoStore(t *testing.T) {
	store := &stubPluginConfigStore{
		values: map[string]string{},
	}

	services, err := resolvePluginServices(context.Background(), store, map[string]config.PluginServiceEntry{
		"octo_demo": {Address: "127.0.0.1:50051"},
	})
	if err != nil {
		t.Fatalf("resolve plugin services: %v", err)
	}

	if services["octo_demo"].Address != "127.0.0.1:50051" {
		t.Fatalf("unexpected resolved address %q", services["octo_demo"].Address)
	}

	if store.values["octo_demo"] != "127.0.0.1:50051" {
		t.Fatalf("expected plugin services config to be initialized in store")
	}
}

func TestResolvePluginServicesUsesDatabaseValueAndBackfillsMissingDefaults(t *testing.T) {
	store := &stubPluginConfigStore{
		values: map[string]string{
			"octo_demo": "127.0.0.1:60051",
		},
	}

	services, err := resolvePluginServices(context.Background(), store, map[string]config.PluginServiceEntry{
		"octo_demo":  {Address: "127.0.0.1:50051"},
		"other_demo": {Address: "127.0.0.1:50052"},
	})
	if err != nil {
		t.Fatalf("resolve plugin services: %v", err)
	}

	if services["octo_demo"].Address != "127.0.0.1:60051" {
		t.Fatalf("expected database value to win, got %q", services["octo_demo"].Address)
	}
	if services["other_demo"].Address != "127.0.0.1:50052" {
		t.Fatalf("expected missing default to be backfilled, got %q", services["other_demo"].Address)
	}
}

func TestResolvePluginServicesErrorFromStore(t *testing.T) {
	store := &errorPluginConfigStore{}
	if _, err := resolvePluginServices(context.Background(), store, map[string]config.PluginServiceEntry{}); err == nil {
		t.Fatalf("expected error when store fails")
	}
}

type stubPluginConfigStore struct {
	values map[string]string
}

type errorPluginConfigStore struct{}

func (e *errorPluginConfigStore) ListGRPCAddresses(ctx context.Context) (map[string]string, error) {
	return nil, fmt.Errorf("boom")
}

func (e *errorPluginConfigStore) SetGRPCAddress(ctx context.Context, key string, address string) error {
	return nil
}

func (s *stubPluginConfigStore) ListGRPCAddresses(ctx context.Context) (map[string]string, error) {
	_ = ctx
	out := make(map[string]string, len(s.values))
	for key, value := range s.values {
		out[key] = value
	}
	return out, nil
}

func (s *stubPluginConfigStore) SetGRPCAddress(ctx context.Context, key string, address string) error {
	_ = ctx
	if s.values == nil {
		s.values = map[string]string{}
	}
	s.values[key] = address
	return nil
}
