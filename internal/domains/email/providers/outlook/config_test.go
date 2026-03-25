package outlook

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestAccountConfigBuildAuthorizeURLUsesStoredSettings(t *testing.T) {
	t.Parallel()

	config := ParseAccountConfig(map[string]any{
		"client_id":             "client-123",
		"redirect_uri":          "http://localhost:5173/oauth/callback",
		"tenant":                "common",
		"scope":                 []any{"offline_access", "https://graph.microsoft.com/Mail.Read"},
		"login_hint":            "robot@example.com",
		"state":                 "state-123",
		"code_challenge":        "challenge-123",
		"code_challenge_method": "plain",
	})

	authorizeURL, err := config.BuildAuthorizeURL("fallback@example.com")
	if err != nil {
		t.Fatalf("build authorize url: %v", err)
	}

	parsed, err := url.Parse(authorizeURL)
	if err != nil {
		t.Fatalf("parse authorize url: %v", err)
	}
	query := parsed.Query()

	if got := query.Get("client_id"); got != "client-123" {
		t.Fatalf("unexpected client_id %q", got)
	}
	if got := query.Get("redirect_uri"); got != "http://localhost:5173/oauth/callback" {
		t.Fatalf("unexpected redirect_uri %q", got)
	}
	if got := query.Get("login_hint"); got != "robot@example.com" {
		t.Fatalf("unexpected login_hint %q", got)
	}
	if got := query.Get("state"); got != "state-123" {
		t.Fatalf("unexpected state %q", got)
	}
	if got := query.Get("code_challenge"); got != "challenge-123" {
		t.Fatalf("unexpected code_challenge %q", got)
	}
	if got := query.Get("code_challenge_method"); got != "plain" {
		t.Fatalf("unexpected code_challenge_method %q", got)
	}
}

func TestAccountConfigEnsureAccessTokenRetriesWithoutClientSecret(t *testing.T) {
	t.Parallel()

	var attempts int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if err := r.ParseForm(); err != nil {
			t.Fatalf("parse form: %v", err)
		}

		if got := r.Form.Get("refresh_token"); got != "refresh-token" {
			t.Fatalf("unexpected refresh_token %q", got)
		}

		if attempts == 1 && strings.TrimSpace(r.Form.Get("client_secret")) != "" {
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"error":             "invalid_client",
				"error_description": "client secret is invalid",
			})
			return
		}

		_ = json.NewEncoder(w).Encode(map[string]any{
			"token_type":    "Bearer",
			"scope":         "https://graph.microsoft.com/.default",
			"expires_in":    3600,
			"access_token":  makeJWT(time.Now().Add(1*time.Hour), "robot@example.com"),
			"refresh_token": "refresh-token-2",
		})
	}))
	defer server.Close()

	config := ParseAccountConfig(map[string]any{
		"client_id":        "client-123",
		"client_secret":    "secret-123",
		"refresh_token":    "refresh-token",
		"access_token":     makeJWT(time.Now().Add(-1*time.Hour), "robot@example.com"),
		"token_expires_at": time.Now().Add(-5 * time.Minute).UTC().Format(time.RFC3339),
		"token_url":        server.URL,
	})

	refreshed, changed, err := config.EnsureAccessToken(context.Background(), "robot@example.com")
	if err != nil {
		t.Fatalf("ensure access token: %v", err)
	}
	if !changed {
		t.Fatalf("expected config to be refreshed")
	}
	if attempts != 2 {
		t.Fatalf("expected 2 refresh attempts, got %d", attempts)
	}
	if refreshed.AccessToken == config.AccessToken {
		t.Fatalf("expected access token to change")
	}
	if refreshed.RefreshToken != "refresh-token-2" {
		t.Fatalf("unexpected refresh token %q", refreshed.RefreshToken)
	}
	if refreshed.TokenExpiresAt.IsZero() {
		t.Fatalf("expected token expiry to be populated")
	}
}

func TestAccountConfigExchangeCodeAppliesTokenResponse(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatalf("parse form: %v", err)
		}
		if got := r.Form.Get("code"); got != "code-123" {
			t.Fatalf("unexpected code %q", got)
		}
		if got := r.Form.Get("code_verifier"); got != "verifier-123" {
			t.Fatalf("unexpected code_verifier %q", got)
		}

		_ = json.NewEncoder(w).Encode(map[string]any{
			"token_type":    "Bearer",
			"scope":         "offline_access https://graph.microsoft.com/Mail.Read",
			"expires_in":    1800,
			"access_token":  makeJWT(time.Now().Add(30*time.Minute), "robot@example.com"),
			"refresh_token": "refresh-token-123",
		})
	}))
	defer server.Close()

	config := ParseAccountConfig(map[string]any{
		"client_id":     "client-123",
		"client_secret": "secret-123",
		"redirect_uri":  "http://localhost:5173/oauth/callback",
		"code_verifier": "verifier-123",
		"token_url":     server.URL,
	})

	updated, err := config.ExchangeCode(context.Background(), "code-123", "robot@example.com")
	if err != nil {
		t.Fatalf("exchange code: %v", err)
	}

	if updated.AccessToken == "" {
		t.Fatalf("expected access token to be stored")
	}
	if updated.RefreshToken != "refresh-token-123" {
		t.Fatalf("unexpected refresh token %q", updated.RefreshToken)
	}
	if updated.Username != "robot@example.com" {
		t.Fatalf("unexpected username %q", updated.Username)
	}
	raw := updated.RawMap()
	if raw["access_token"] == "" {
		t.Fatalf("expected raw config to contain access_token")
	}
	if raw["refresh_token"] != "refresh-token-123" {
		t.Fatalf("expected raw config to contain refresh_token")
	}
}

func makeJWT(expiry time.Time, username string) string {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"none","typ":"JWT"}`))
	payloadJSON, _ := json.Marshal(map[string]any{
		"exp":                expiry.Unix(),
		"preferred_username": username,
	})
	payload := base64.RawURLEncoding.EncodeToString(payloadJSON)
	return header + "." + payload + ".signature"
}
