package runtime

import (
	"context"
	"fmt"
	"strings"

	"octomanger/internal/platform/config"
)

type pluginConfigStore interface {
	ListGRPCAddresses(ctx context.Context) (map[string]string, error)
	SetGRPCAddress(ctx context.Context, pluginKey string, address string) error
}

func resolvePluginServices(
	ctx context.Context,
	store pluginConfigStore,
	defaults map[string]config.PluginServiceEntry,
) (map[string]config.PluginServiceEntry, error) {
	normalizedDefaults := normalizePluginServices(defaults)

	addresses, err := store.ListGRPCAddresses(ctx)
	if err != nil {
		return nil, fmt.Errorf("list plugin service configs: %w", err)
	}

	current := make(map[string]config.PluginServiceEntry, len(addresses))
	for key, address := range addresses {
		normalizedKey := normalizePluginKey(key)
		normalizedAddress := strings.TrimSpace(address)
		if normalizedKey == "" || normalizedAddress == "" {
			continue
		}
		current[normalizedKey] = config.PluginServiceEntry{Address: normalizedAddress}
	}

	for key, entry := range normalizedDefaults {
		if existing, ok := current[key]; ok && strings.TrimSpace(existing.Address) != "" {
			existing.AllowInsecure = existing.AllowInsecure || entry.AllowInsecure
			if existing.TLSServerName == "" {
				existing.TLSServerName = entry.TLSServerName
			}
			if entry.TLSInsecureSkipVerify {
				existing.TLSInsecureSkipVerify = true
			}
			current[key] = existing
			continue
		}
		current[key] = entry
		if err := store.SetGRPCAddress(ctx, key, entry.Address); err != nil {
			return nil, fmt.Errorf("persist plugin service config %s: %w", key, err)
		}
	}

	return current, nil
}

func normalizePluginServices(in map[string]config.PluginServiceEntry) map[string]config.PluginServiceEntry {
	out := make(map[string]config.PluginServiceEntry, len(in))
	for key, entry := range in {
		normalizedKey := normalizePluginKey(key)
		address := strings.TrimSpace(entry.Address)
		if normalizedKey == "" || address == "" {
			continue
		}
		out[normalizedKey] = config.PluginServiceEntry{
			Address:               address,
			AllowInsecure:         entry.AllowInsecure,
			TLSServerName:         strings.TrimSpace(entry.TLSServerName),
			TLSInsecureSkipVerify: entry.TLSInsecureSkipVerify,
		}
	}
	return out
}

func normalizePluginKey(key string) string {
	return strings.ToLower(strings.ReplaceAll(strings.TrimSpace(key), "-", "_"))
}
