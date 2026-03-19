package outlook

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const tokenRefreshGrace = 2 * time.Minute

type AccountConfig struct {
	raw                 map[string]any
	AccessToken         string
	RefreshToken        string
	ClientID            string
	ClientSecret        string
	Tenant              string
	Scopes              []string
	TokenURL            string
	TokenExpiresAt      time.Time
	Mailbox             string
	Timeout             time.Duration
	GraphBaseURL        string
	Proxy               string
	RedirectURI         string
	LoginHint           string
	Prompt              string
	State               string
	CodeChallenge       string
	CodeChallengeMethod string
	CodeVerifier        string
	Username            string
}

func ParseAccountConfig(raw map[string]any) AccountConfig {
	config := AccountConfig{
		raw:                 cloneMap(raw),
		AccessToken:         normalizeAccessToken(stringField(raw, "access_token")),
		RefreshToken:        strings.TrimSpace(stringField(raw, "refresh_token")),
		ClientID:            strings.TrimSpace(stringField(raw, "client_id")),
		ClientSecret:        strings.TrimSpace(stringField(raw, "client_secret")),
		Tenant:              strings.TrimSpace(stringField(raw, "tenant")),
		Scopes:              parseScopes(raw["scope"]),
		TokenURL:            strings.TrimSpace(stringField(raw, "token_url")),
		TokenExpiresAt:      parseTimestamp(raw["token_expires_at"]),
		Mailbox:             strings.TrimSpace(stringField(raw, "mailbox")),
		Timeout:             parseTimeout(raw["timeout_seconds"]),
		GraphBaseURL:        strings.TrimSpace(stringField(raw, "graph_base_url")),
		Proxy:               strings.TrimSpace(stringField(raw, "proxy")),
		RedirectURI:         strings.TrimSpace(stringField(raw, "redirect_uri")),
		LoginHint:           strings.TrimSpace(stringField(raw, "login_hint")),
		Prompt:              strings.TrimSpace(stringField(raw, "prompt")),
		State:               strings.TrimSpace(stringField(raw, "state")),
		CodeChallenge:       strings.TrimSpace(stringField(raw, "code_challenge")),
		CodeChallengeMethod: strings.TrimSpace(stringField(raw, "code_challenge_method")),
		CodeVerifier:        strings.TrimSpace(stringField(raw, "code_verifier")),
		Username:            strings.TrimSpace(stringField(raw, "username")),
	}

	if config.TokenExpiresAt.IsZero() && config.AccessToken != "" {
		config.TokenExpiresAt = extractTokenExpiry(config.AccessToken)
	}
	if config.Username == "" {
		config.Username = extractUsername(config.AccessToken)
	}
	if config.Username == "" {
		config.Username = config.LoginHint
	}
	if config.GraphBaseURL == "" {
		config.GraphBaseURL = defaultGraphBaseURL
	}
	return config
}

func (c AccountConfig) RawMap() map[string]any {
	return cloneMap(c.raw)
}

func (c AccountConfig) GraphConfig() (GraphConfig, error) {
	accessToken := normalizeAccessToken(c.AccessToken)
	if accessToken == "" {
		return GraphConfig{}, errors.New("outlook access_token is required before reading messages")
	}

	return GraphConfig{
		AccessToken: accessToken,
		BaseURL:     normalizeGraphBaseURL(c.GraphBaseURL),
		Timeout:     c.Timeout,
		Proxy:       c.Proxy,
	}, nil
}

func (c AccountConfig) BuildAuthorizeURL(defaultLoginHint string) (string, error) {
	loginHint := strings.TrimSpace(c.LoginHint)
	if loginHint == "" {
		loginHint = strings.TrimSpace(defaultLoginHint)
	}

	return BuildAuthorizeURL(AuthorizeURLInput{
		Tenant:              resolveTenant(c.Tenant, loginHint),
		ClientID:            c.ClientID,
		RedirectURI:         c.RedirectURI,
		Scope:               c.Scopes,
		State:               c.State,
		Prompt:              c.Prompt,
		LoginHint:           loginHint,
		CodeChallenge:       c.CodeChallenge,
		CodeChallengeMethod: c.CodeChallengeMethod,
	})
}

func (c AccountConfig) ExchangeCode(ctx context.Context, code string, defaultLoginHint string) (AccountConfig, error) {
	loginHint := strings.TrimSpace(c.Username)
	if loginHint == "" {
		loginHint = strings.TrimSpace(c.LoginHint)
	}
	if loginHint == "" {
		loginHint = strings.TrimSpace(defaultLoginHint)
	}

	token, err := ExchangeCode(ctx, ExchangeCodeInput{
		Tenant:       resolveTenant(c.Tenant, loginHint),
		ClientID:     c.ClientID,
		ClientSecret: c.ClientSecret,
		RedirectURI:  c.RedirectURI,
		Code:         code,
		Scope:        c.Scopes,
		TokenURL:     c.TokenURL,
		CodeVerifier: c.CodeVerifier,
		Proxy:        c.Proxy,
	})
	if err != nil {
		return AccountConfig{}, err
	}

	next := c.applyToken(token)
	if next.Username == "" {
		next.Username = coalesceNonEmpty(extractUsername(token.AccessToken), loginHint)
	}
	next.raw["username"] = next.Username
	return next, nil
}

func (c AccountConfig) EnsureAccessToken(ctx context.Context, defaultUsername string) (AccountConfig, bool, error) {
	if !shouldRefreshToken(c) {
		return c, false, nil
	}

	username := coalesceNonEmpty(c.Username, c.LoginHint, defaultUsername)
	scopeCandidates := buildRefreshScopeCandidates(c.Scopes)
	tenantCandidates := uniqueStrings([]string{
		resolveTenant(c.Tenant, username),
		"common",
	})
	if isConsumerAddress(username) {
		tenantCandidates = uniqueStrings(append(tenantCandidates, "consumers"))
	}

	var lastErr error
	for _, tenant := range tenantCandidates {
		tokenURLs := uniqueStrings([]string{
			strings.TrimSpace(c.TokenURL),
			fmt.Sprintf(defaultTokenURL, url.PathEscape(tenant)),
		})
		for _, tokenURL := range tokenURLs {
			for _, scopes := range scopeCandidates {
				token, err := refreshTokenWithRetry(ctx, RefreshTokenInput{
					Tenant:       tenant,
					ClientID:     c.ClientID,
					ClientSecret: c.ClientSecret,
					RefreshToken: c.RefreshToken,
					Scope:        scopes,
					TokenURL:     tokenURL,
					Proxy:        c.Proxy,
				})
				if err != nil {
					lastErr = err
					continue
				}

				next := c.applyToken(token)
				next.Tenant = tenant
				next.raw["tenant"] = tenant
				next.Username = coalesceNonEmpty(extractUsername(token.AccessToken), username)
				if next.Username != "" {
					next.raw["username"] = next.Username
				}
				return next, true, nil
			}
		}
	}

	if lastErr == nil {
		lastErr = errors.New("unable to refresh outlook access token")
	}
	return AccountConfig{}, false, lastErr
}

func (c AccountConfig) applyToken(token TokenResponse) AccountConfig {
	next := c
	next.raw = cloneMap(c.raw)
	next.AccessToken = normalizeAccessToken(token.AccessToken)
	next.raw["access_token"] = next.AccessToken

	if refreshToken := strings.TrimSpace(token.RefreshToken); refreshToken != "" {
		next.RefreshToken = refreshToken
		next.raw["refresh_token"] = refreshToken
	}
	if scope := strings.TrimSpace(token.Scope); scope != "" {
		next.Scopes = normalizeScopes(strings.Fields(scope))
		next.raw["scope"] = append([]string(nil), next.Scopes...)
	}
	if token.TokenURL != "" {
		next.TokenURL = token.TokenURL
		next.raw["token_url"] = token.TokenURL
	}
	if !token.ExpiresAt.IsZero() {
		next.TokenExpiresAt = token.ExpiresAt.UTC()
		next.raw["token_expires_at"] = next.TokenExpiresAt.Format(time.RFC3339)
	}
	return next
}

func refreshTokenWithRetry(ctx context.Context, input RefreshTokenInput) (TokenResponse, error) {
	token, err := RefreshToken(ctx, input)
	if err == nil {
		return token, nil
	}
	if strings.TrimSpace(input.ClientSecret) == "" || !shouldRetryWithoutClientSecret(err) {
		return TokenResponse{}, err
	}

	input.ClientSecret = ""
	return RefreshToken(ctx, input)
}

func shouldRetryWithoutClientSecret(err error) bool {
	if err == nil {
		return false
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "aadsts7000215") ||
		(strings.Contains(message, "invalid_client") && strings.Contains(message, "client secret"))
}

func shouldRefreshToken(config AccountConfig) bool {
	if strings.TrimSpace(config.RefreshToken) == "" {
		return false
	}
	if normalizeAccessToken(config.AccessToken) == "" {
		return true
	}
	if config.TokenExpiresAt.IsZero() {
		return false
	}
	return time.Now().UTC().Add(tokenRefreshGrace).After(config.TokenExpiresAt.UTC())
}

func buildRefreshScopeCandidates(scopes []string) [][]string {
	normalized := normalizeScopes(scopes)
	candidates := make([][]string, 0, 2)
	if len(normalized) > 0 {
		candidates = append(candidates, normalized)
	}
	candidates = append(candidates, append([]string(nil), defaultRefreshScopes...))
	return uniqueScopeCandidates(candidates)
}

func uniqueScopeCandidates(candidates [][]string) [][]string {
	result := make([][]string, 0, len(candidates))
	seen := map[string]struct{}{}
	for _, candidate := range candidates {
		normalized := normalizeScopes(candidate)
		key := strings.Join(normalized, " ")
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, normalized)
	}
	if len(result) == 0 {
		return [][]string{nil}
	}
	return result
}

func resolveTenant(rawTenant string, username string) string {
	if tenant := strings.TrimSpace(rawTenant); tenant != "" {
		return tenant
	}
	if isConsumerAddress(username) {
		return "consumers"
	}
	return "common"
}

func isConsumerAddress(username string) bool {
	lowerUser := strings.ToLower(strings.TrimSpace(username))
	at := strings.LastIndex(lowerUser, "@")
	if at < 0 || at == len(lowerUser)-1 {
		return false
	}
	switch lowerUser[at+1:] {
	case "outlook.com", "hotmail.com", "live.com", "msn.com":
		return true
	default:
		return false
	}
}

func normalizeGraphBaseURL(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return defaultGraphBaseURL
	}
	return strings.TrimSuffix(trimmed, "/")
}

func buildHTTPClient(proxy string, timeout time.Duration) (*http.Client, error) {
	transport, ok := http.DefaultTransport.(*http.Transport)
	if !ok || transport == nil {
		transport = &http.Transport{}
	}
	cloned := transport.Clone()

	if strings.TrimSpace(proxy) != "" {
		proxyURL, err := url.Parse(proxy)
		if err != nil || proxyURL.Scheme == "" {
			return nil, errors.New("invalid proxy url")
		}
		cloned.Proxy = http.ProxyURL(proxyURL)
	}

	client := &http.Client{Transport: cloned}
	if timeout > 0 {
		client.Timeout = timeout
	}
	return client, nil
}

func normalizeAccessToken(value string) string {
	token := strings.TrimSpace(value)
	if len(token) >= 7 && strings.EqualFold(token[:7], "bearer ") {
		token = strings.TrimSpace(token[7:])
	}
	return token
}

func normalizeScopes(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	result := make([]string, 0, len(values))
	seen := map[string]struct{}{}
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, exists := seen[trimmed]; exists {
			continue
		}
		seen[trimmed] = struct{}{}
		result = append(result, trimmed)
	}
	return result
}

func parseScopes(value any) []string {
	switch typed := value.(type) {
	case nil:
		return nil
	case string:
		return normalizeScopes(strings.Fields(strings.ReplaceAll(typed, ",", " ")))
	case []string:
		return normalizeScopes(typed)
	case []any:
		items := make([]string, 0, len(typed))
		for _, item := range typed {
			asString, ok := item.(string)
			if !ok {
				continue
			}
			items = append(items, asString)
		}
		return normalizeScopes(items)
	default:
		return nil
	}
}

func parseTimeout(value any) time.Duration {
	seconds, ok := parseInt(value)
	if !ok || seconds <= 0 {
		return 0
	}
	return time.Duration(seconds) * time.Second
}

func parseTimestamp(value any) time.Time {
	text, ok := value.(string)
	if !ok {
		return time.Time{}
	}
	parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(text))
	if err != nil {
		return time.Time{}
	}
	return parsed.UTC()
}

func parseInt(value any) (int64, bool) {
	switch typed := value.(type) {
	case int:
		return int64(typed), true
	case int64:
		return typed, true
	case float64:
		return int64(typed), true
	case json.Number:
		parsed, err := typed.Int64()
		if err != nil {
			return 0, false
		}
		return parsed, true
	case string:
		parsed, err := strconv.ParseInt(strings.TrimSpace(typed), 10, 64)
		if err != nil {
			return 0, false
		}
		return parsed, true
	default:
		return 0, false
	}
}

func extractUsername(accessToken string) string {
	claims, ok := parseTokenClaims(accessToken)
	if !ok {
		return ""
	}
	for _, key := range []string{"preferred_username", "upn", "email", "unique_name"} {
		value, ok := claims[key].(string)
		if ok && strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func extractTokenExpiry(accessToken string) time.Time {
	claims, ok := parseTokenClaims(accessToken)
	if !ok {
		return time.Time{}
	}
	value, ok := parseInt(claims["exp"])
	if !ok || value <= 0 {
		return time.Time{}
	}
	if value >= 1_000_000_000_000 {
		value /= 1000
	}
	return time.Unix(value, 0).UTC()
}

func parseTokenClaims(accessToken string) (map[string]any, bool) {
	parts := strings.Split(normalizeAccessToken(accessToken), ".")
	if len(parts) < 2 {
		return nil, false
	}

	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, false
	}

	var claims map[string]any
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil, false
	}
	return claims, true
}

func stringField(raw map[string]any, key string) string {
	if raw == nil {
		return ""
	}
	value, ok := raw[key]
	if !ok {
		return ""
	}
	switch typed := value.(type) {
	case string:
		return typed
	default:
		return fmt.Sprintf("%v", typed)
	}
}

func cloneMap(value map[string]any) map[string]any {
	if value == nil {
		return map[string]any{}
	}
	cloned := make(map[string]any, len(value))
	for key, item := range value {
		cloned[key] = item
	}
	return cloned
}

func uniqueStrings(items []string) []string {
	result := make([]string, 0, len(items))
	seen := map[string]struct{}{}
	for _, item := range items {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" {
			continue
		}
		if _, exists := seen[trimmed]; exists {
			continue
		}
		seen[trimmed] = struct{}{}
		result = append(result, trimmed)
	}
	return result
}

func coalesceNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
