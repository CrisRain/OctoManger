package grpcclient

import (
	"fmt"
	"sort"
	"strings"
)

// PluginRegistry resolves a plugin key to a gRPC server address (host:port).
type PluginRegistry interface {
	Address(pluginKey string) (string, error)
	ServiceConfig(pluginKey string) (PluginServiceConfig, error)
	Keys() []string
}

// StaticRegistry holds a fixed map of plugin key → address, configured at
// startup from environment variables or config file entries.
//
// Address resolution rules:
//   - Exact match on the plugin key (case-insensitive).
//   - Keys with hyphens or underscores are normalised to underscores before lookup.
type StaticRegistry struct {
	entries map[string]PluginServiceConfig // normalised key → config
}

// NewStaticRegistry builds a registry from a map of raw plugin keys to addresses.
func NewStaticRegistry(services map[string]PluginServiceConfig) *StaticRegistry {
	entries := make(map[string]PluginServiceConfig, len(services))
	for key, cfg := range services {
		address := strings.TrimSpace(cfg.Address)
		if address == "" {
			continue
		}
		cfg.Address = address
		cfg.TLSServerName = strings.TrimSpace(cfg.TLSServerName)
		entries[normaliseKey(key)] = cfg
	}
	return &StaticRegistry{entries: entries}
}

// Address returns the gRPC server address for the given plugin key.
func (r *StaticRegistry) Address(pluginKey string) (string, error) {
	cfg, ok := r.entries[normaliseKey(pluginKey)]
	if !ok || cfg.Address == "" {
		return "", fmt.Errorf("grpcclient: no address registered for plugin %q", pluginKey)
	}
	return cfg.Address, nil
}

// ServiceConfig returns the full connection config for the given plugin key.
func (r *StaticRegistry) ServiceConfig(pluginKey string) (PluginServiceConfig, error) {
	cfg, ok := r.entries[normaliseKey(pluginKey)]
	if !ok || cfg.Address == "" {
		return PluginServiceConfig{}, fmt.Errorf("grpcclient: no address registered for plugin %q", pluginKey)
	}
	return cfg, nil
}

// Keys returns all registered plugin keys (normalised).
func (r *StaticRegistry) Keys() []string {
	keys := make([]string, 0, len(r.entries))
	for k := range r.entries {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// PluginServiceConfig holds the connection parameters for one plugin service.
type PluginServiceConfig struct {
	Address               string // host:port — required
	AllowInsecure         bool   // allow plaintext even for non-loopback addresses
	TLSServerName         string // optional SNI override
	TLSInsecureSkipVerify bool   // skip certificate verification (for private CA/dev only)
}

func normaliseKey(key string) string {
	return strings.ToLower(strings.ReplaceAll(strings.TrimSpace(key), "-", "_"))
}
