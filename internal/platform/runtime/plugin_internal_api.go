package runtime

import (
	"net"
	"strconv"
	"strings"

	pluginapp "octomanger/internal/domains/plugins/app"
	"octomanger/internal/platform/config"
)

func buildPluginInternalAPIConfig(cfg *config.Config) pluginapp.InternalAPIConfig {
	if cfg == nil {
		return pluginapp.InternalAPIConfig{}
	}

	timeoutSeconds := int(cfg.Plugins.Timeout.Account.Seconds())
	if timeoutSeconds <= 0 {
		timeoutSeconds = 60
	}

	return pluginapp.InternalAPIConfig{
		URL:            normalizePluginInternalAPIURL(cfg.Server.APIAddr),
		Token:          strings.TrimSpace(cfg.Auth.PluginInternalAPIToken),
		TimeoutSeconds: timeoutSeconds,
	}
}

func normalizePluginInternalAPIURL(apiAddr string) string {
	addr := strings.TrimSpace(apiAddr)
	if addr == "" {
		return ""
	}

	if strings.HasPrefix(addr, "http://") || strings.HasPrefix(addr, "https://") {
		return strings.TrimRight(addr, "/")
	}

	if strings.HasPrefix(addr, ":") {
		port := strings.TrimPrefix(addr, ":")
		if _, err := strconv.Atoi(port); err == nil {
			return "http://127.0.0.1:" + port
		}
	}

	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return "http://" + addr
	}

	host = strings.TrimSpace(host)
	if host == "" || host == "0.0.0.0" || host == "::" || host == "[::]" {
		host = "127.0.0.1"
	}

	return "http://" + net.JoinHostPort(host, port)
}
