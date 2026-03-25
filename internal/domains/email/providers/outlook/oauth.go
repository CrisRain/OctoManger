package outlook

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	defaultTenant       = "common"
	defaultAuthorizeURL = "https://login.microsoftonline.com/%s/oauth2/v2.0/authorize"
	defaultTokenURL     = "https://login.microsoftonline.com/%s/oauth2/v2.0/token"
)

var (
	defaultAuthorizeScopes = []string{
		"offline_access",
		"openid",
		"profile",
		"email",
		"https://graph.microsoft.com/Mail.Read",
	}
	defaultRefreshScopes = []string{
		"https://graph.microsoft.com/.default",
	}
)

type AuthorizeURLInput struct {
	Tenant              string
	ClientID            string
	RedirectURI         string
	Scope               []string
	State               string
	Prompt              string
	LoginHint           string
	CodeChallenge       string
	CodeChallengeMethod string
}

type ExchangeCodeInput struct {
	Tenant       string
	ClientID     string
	ClientSecret string
	RedirectURI  string
	Code         string
	Scope        []string
	TokenURL     string
	CodeVerifier string
	Proxy        string
}

type RefreshTokenInput struct {
	Tenant       string
	ClientID     string
	ClientSecret string
	RefreshToken string
	Scope        []string
	TokenURL     string
	Proxy        string
}

type TokenResponse struct {
	TokenType    string
	Scope        string
	ExpiresIn    int64
	AccessToken  string
	RefreshToken string
	ExpiresAt    time.Time
	TokenURL     string
}

type tokenEndpointError struct {
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
}

type tokenEndpointResponse struct {
	TokenType    string `json:"token_type"`
	Scope        string `json:"scope"`
	ExpiresIn    int64  `json:"expires_in"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

func BuildAuthorizeURL(input AuthorizeURLInput) (string, error) {
	clientID := strings.TrimSpace(input.ClientID)
	redirectURI := strings.TrimSpace(input.RedirectURI)
	if clientID == "" {
		return "", errors.New("outlook client_id is required")
	}
	if redirectURI == "" {
		return "", errors.New("outlook redirect_uri is required")
	}

	query := url.Values{}
	query.Set("client_id", clientID)
	query.Set("response_type", "code")
	query.Set("redirect_uri", redirectURI)
	query.Set("response_mode", "query")
	query.Set("scope", normalizeScopeStringWithDefault(input.Scope, defaultAuthorizeScopes))

	state := strings.TrimSpace(input.State)
	if state == "" {
		state = "octomanger-email"
	}
	query.Set("state", state)

	if prompt := strings.TrimSpace(input.Prompt); prompt != "" {
		query.Set("prompt", prompt)
	}
	if loginHint := strings.TrimSpace(input.LoginHint); loginHint != "" {
		query.Set("login_hint", loginHint)
	}
	if codeChallenge := strings.TrimSpace(input.CodeChallenge); codeChallenge != "" {
		query.Set("code_challenge", codeChallenge)
		method := strings.TrimSpace(input.CodeChallengeMethod)
		if method == "" {
			method = "S256"
		}
		query.Set("code_challenge_method", method)
	}

	baseURL := fmt.Sprintf(defaultAuthorizeURL, url.PathEscape(normalizeTenant(input.Tenant)))
	return baseURL + "?" + query.Encode(), nil
}

func ExchangeCode(ctx context.Context, input ExchangeCodeInput) (TokenResponse, error) {
	clientID := strings.TrimSpace(input.ClientID)
	redirectURI := strings.TrimSpace(input.RedirectURI)
	code := strings.TrimSpace(input.Code)
	if clientID == "" {
		return TokenResponse{}, errors.New("outlook client_id is required")
	}
	if redirectURI == "" {
		return TokenResponse{}, errors.New("outlook redirect_uri is required")
	}
	if code == "" {
		return TokenResponse{}, errors.New("outlook code is required")
	}

	form := url.Values{}
	form.Set("client_id", clientID)
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	form.Set("redirect_uri", redirectURI)
	form.Set("scope", normalizeScopeStringWithDefault(input.Scope, defaultAuthorizeScopes))
	if clientSecret := strings.TrimSpace(input.ClientSecret); clientSecret != "" {
		form.Set("client_secret", clientSecret)
	}
	if codeVerifier := strings.TrimSpace(input.CodeVerifier); codeVerifier != "" {
		form.Set("code_verifier", codeVerifier)
	}

	return callTokenEndpoint(ctx, resolveTokenURL(input.TokenURL, input.Tenant), form, input.Proxy)
}

func RefreshToken(ctx context.Context, input RefreshTokenInput) (TokenResponse, error) {
	clientID := strings.TrimSpace(input.ClientID)
	refreshToken := strings.TrimSpace(input.RefreshToken)
	if clientID == "" {
		return TokenResponse{}, errors.New("outlook client_id is required")
	}
	if refreshToken == "" {
		return TokenResponse{}, errors.New("outlook refresh_token is required")
	}

	form := url.Values{}
	form.Set("client_id", clientID)
	form.Set("grant_type", "refresh_token")
	form.Set("refresh_token", refreshToken)
	if clientSecret := strings.TrimSpace(input.ClientSecret); clientSecret != "" {
		form.Set("client_secret", clientSecret)
	}
	if scope := normalizeScopeStringWithDefault(input.Scope, nil); scope != "" {
		form.Set("scope", scope)
	}

	return callTokenEndpoint(ctx, resolveTokenURL(input.TokenURL, input.Tenant), form, input.Proxy)
}

func resolveTokenURL(rawTokenURL string, tenant string) string {
	trimmed := strings.TrimSpace(rawTokenURL)
	if trimmed != "" {
		return trimmed
	}
	return fmt.Sprintf(defaultTokenURL, url.PathEscape(normalizeTenant(tenant)))
}

func normalizeTenant(tenant string) string {
	trimmed := strings.TrimSpace(tenant)
	if trimmed == "" {
		return defaultTenant
	}
	return trimmed
}

func normalizeScopeStringWithDefault(scopes []string, fallback []string) string {
	normalized := normalizeScopes(scopes)
	if len(normalized) == 0 && len(fallback) > 0 {
		normalized = append(normalized, fallback...)
	}
	return strings.Join(normalized, " ")
}

func callTokenEndpoint(ctx context.Context, endpoint string, form url.Values, proxy string) (TokenResponse, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return TokenResponse{}, err
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client, err := buildHTTPClient(proxy, 0)
	if err != nil {
		return TokenResponse{}, err
	}

	response, err := client.Do(request)
	if err != nil {
		return TokenResponse{}, err
	}
	defer response.Body.Close()

	if response.StatusCode >= 200 && response.StatusCode < 300 {
		var payload tokenEndpointResponse
		if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
			return TokenResponse{}, fmt.Errorf("parse outlook token response: %w", err)
		}
		accessToken := normalizeAccessToken(payload.AccessToken)
		if accessToken == "" {
			return TokenResponse{}, errors.New("outlook token response does not include access_token")
		}

		expiresIn := payload.ExpiresIn
		if expiresIn <= 0 {
			expiresIn = 3600
		}

		return TokenResponse{
			TokenType:    strings.TrimSpace(payload.TokenType),
			Scope:        strings.TrimSpace(payload.Scope),
			ExpiresIn:    expiresIn,
			AccessToken:  accessToken,
			RefreshToken: strings.TrimSpace(payload.RefreshToken),
			ExpiresAt:    time.Now().UTC().Add(time.Duration(expiresIn) * time.Second),
			TokenURL:     strings.TrimSpace(endpoint),
		}, nil
	}

	var failure tokenEndpointError
	if err := json.NewDecoder(response.Body).Decode(&failure); err != nil {
		return TokenResponse{}, fmt.Errorf("outlook token endpoint failed with status %d", response.StatusCode)
	}

	errorCode := strings.TrimSpace(failure.Error)
	if errorCode == "" {
		errorCode = fmt.Sprintf("http_%d", response.StatusCode)
	}
	errorDescription := strings.TrimSpace(failure.ErrorDescription)
	if errorDescription == "" {
		errorDescription = fmt.Sprintf("outlook token endpoint failed with status %d", response.StatusCode)
	}
	return TokenResponse{}, fmt.Errorf("%s: %s", errorCode, errorDescription)
}
