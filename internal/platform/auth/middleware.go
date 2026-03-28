package auth

import (
	"context"
	"crypto/subtle"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/cloudwego/hertz/pkg/app"

	"octomanger/internal/platform/httpx"
)

type AdminKeyVerifier interface {
	// VerifyAdminKey returns:
	// - configured: whether the backend has an admin API key configured.
	// - matched: whether providedKey is valid.
	VerifyAdminKey(ctx context.Context, providedKey string) (configured bool, matched bool, err error)
}

// RequireAdmin returns a Hertz HandlerFunc that validates the admin key.
// If adminKey is empty, all protected endpoints are rejected (fail-closed).
// Multiple keys can be configured via comma/semicolon/newline-separated values.
func RequireAdmin(adminKey string) app.HandlerFunc {
	configuredKeys := parseConfiguredKeys(adminKey)
	if len(configuredKeys) == 0 {
		return func(ctx context.Context, c *app.RequestContext) {
			httpx.Unauthorized(ctx, c, "unauthorized")
			c.Abort()
		}
	}

	limiter := newFailedAttemptLimiter()

	return func(ctx context.Context, c *app.RequestContext) {
		source := requestSource(c)
		if retryAfter, blocked := limiter.isBlocked(source, time.Now()); blocked {
			c.Header("Retry-After", strconv.FormatInt(int64(retryAfter.Seconds()), 10))
			httpx.TooManyRequests(ctx, c, "too many invalid admin key attempts")
			c.Abort()
			return
		}

		providedKey := extractProvidedKey(c)

		if !matchesAnyConfiguredKey(providedKey, configuredKeys) {
			if retryAfter, blocked := limiter.recordFailure(source, time.Now()); blocked {
				c.Header("Retry-After", strconv.FormatInt(int64(retryAfter.Seconds()), 10))
				httpx.TooManyRequests(ctx, c, "too many invalid admin key attempts")
				c.Abort()
				return
			}
			httpx.Unauthorized(ctx, c, "missing or invalid admin key")
			c.Abort()
			return
		}
		limiter.reset(source)

		c.Next(ctx)
	}
}

// RequireAdminWithVerifier validates admin API keys by delegating verification
// to a runtime key verifier (for example, database-backed key storage).
func RequireAdminWithVerifier(verifier AdminKeyVerifier) app.HandlerFunc {
	if verifier == nil {
		return func(ctx context.Context, c *app.RequestContext) {
			httpx.Unauthorized(ctx, c, "unauthorized")
			c.Abort()
		}
	}

	limiter := newFailedAttemptLimiter()

	return func(ctx context.Context, c *app.RequestContext) {
		source := requestSource(c)
		if retryAfter, blocked := limiter.isBlocked(source, time.Now()); blocked {
			c.Header("Retry-After", strconv.FormatInt(int64(retryAfter.Seconds()), 10))
			httpx.TooManyRequests(ctx, c, "too many invalid admin key attempts")
			c.Abort()
			return
		}

		providedKey := extractProvidedKey(c)
		configured, matched, err := verifier.VerifyAdminKey(ctx, providedKey)
		if err != nil {
			httpx.InternalServerError(ctx, c, err.Error())
			c.Abort()
			return
		}
		if !configured {
			httpx.Unauthorized(ctx, c, "admin key is not initialized")
			c.Abort()
			return
		}
		if !matched {
			if retryAfter, blocked := limiter.recordFailure(source, time.Now()); blocked {
				c.Header("Retry-After", strconv.FormatInt(int64(retryAfter.Seconds()), 10))
				httpx.TooManyRequests(ctx, c, "too many invalid admin key attempts")
				c.Abort()
				return
			}
			httpx.Unauthorized(ctx, c, "missing or invalid admin key")
			c.Abort()
			return
		}
		limiter.reset(source)

		c.Next(ctx)
	}
}

// RequireAdminForRouter returns Hertz middleware suitable for router.Use().
func RequireAdminForRouter(adminKey string) app.HandlerFunc {
	return RequireAdmin(adminKey)
}

// RequireAdminForRouterWithVerifier returns dynamic-admin middleware suitable
// for router.Use().
func RequireAdminForRouterWithVerifier(verifier AdminKeyVerifier) app.HandlerFunc {
	return RequireAdminWithVerifier(verifier)
}

// StatusUnauthorized is kept as a named constant for clarity.
const StatusUnauthorized = http.StatusUnauthorized

const (
	failedAttemptsWindow       = time.Minute
	maxFailedAttemptsPerWindow = 10
	failedAttemptsBlockWindow  = 2 * time.Minute
)

type failedAttemptLimiter struct {
	mu      sync.Mutex
	entries map[string]failedAttemptEntry
}

type failedAttemptEntry struct {
	windowStartedAt time.Time
	failures        int
	blockedUntil    time.Time
}

func newFailedAttemptLimiter() *failedAttemptLimiter {
	return &failedAttemptLimiter{
		entries: make(map[string]failedAttemptEntry),
	}
}

func (l *failedAttemptLimiter) isBlocked(source string, now time.Time) (time.Duration, bool) {
	key := normaliseSource(source)

	l.mu.Lock()
	defer l.mu.Unlock()

	entry, ok := l.entries[key]
	if !ok {
		return 0, false
	}
	if !entry.blockedUntil.IsZero() && now.After(entry.blockedUntil) {
		if now.Sub(entry.windowStartedAt) > failedAttemptsWindow {
			delete(l.entries, key)
			return 0, false
		}
		entry.blockedUntil = time.Time{}
		entry.failures = 0
		entry.windowStartedAt = now
		l.entries[key] = entry
		return 0, false
	}
	if !entry.blockedUntil.IsZero() {
		return ceilSecondDuration(entry.blockedUntil.Sub(now)), true
	}
	return 0, false
}

func (l *failedAttemptLimiter) recordFailure(source string, now time.Time) (time.Duration, bool) {
	key := normaliseSource(source)

	l.mu.Lock()
	defer l.mu.Unlock()

	entry := l.entries[key]
	if entry.windowStartedAt.IsZero() || now.Sub(entry.windowStartedAt) > failedAttemptsWindow {
		entry = failedAttemptEntry{windowStartedAt: now}
	}
	if !entry.blockedUntil.IsZero() && now.Before(entry.blockedUntil) {
		return ceilSecondDuration(entry.blockedUntil.Sub(now)), true
	}

	entry.failures++
	if entry.failures >= maxFailedAttemptsPerWindow {
		entry.blockedUntil = now.Add(failedAttemptsBlockWindow)
	}
	l.entries[key] = entry

	if entry.blockedUntil.IsZero() {
		return 0, false
	}
	return ceilSecondDuration(entry.blockedUntil.Sub(now)), true
}

func (l *failedAttemptLimiter) reset(source string) {
	key := normaliseSource(source)
	l.mu.Lock()
	delete(l.entries, key)
	l.mu.Unlock()
}

func normaliseSource(source string) string {
	source = strings.TrimSpace(source)
	if source == "" {
		return "unknown"
	}
	return source
}

func requestSource(c *app.RequestContext) string {
	if xff := strings.TrimSpace(string(c.GetHeader("X-Forwarded-For"))); xff != "" {
		forwarded := strings.Split(xff, ",")
		if len(forwarded) > 0 {
			return strings.TrimSpace(forwarded[0])
		}
	}
	if realIP := strings.TrimSpace(string(c.GetHeader("X-Real-IP"))); realIP != "" {
		return realIP
	}
	if clientIP := strings.TrimSpace(c.ClientIP()); clientIP != "" {
		return clientIP
	}
	return "unknown"
}

func extractProvidedKey(c *app.RequestContext) string {
	providedKey := strings.TrimSpace(string(c.GetHeader("X-Admin-Key")))
	if providedKey == "" {
		authHeader := strings.TrimSpace(string(c.GetHeader("Authorization")))
		providedKey = strings.TrimPrefix(authHeader, "Bearer ")
		providedKey = strings.TrimSpace(providedKey)
	}
	return providedKey
}

func ceilSecondDuration(d time.Duration) time.Duration {
	if d <= 0 {
		return time.Second
	}
	secs := int64(d / time.Second)
	if d%time.Second != 0 {
		secs++
	}
	if secs <= 0 {
		secs = 1
	}
	return time.Duration(secs) * time.Second
}

func parseConfiguredKeys(raw string) []string {
	fields := strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == ';' || r == '\n' || r == '\r' || r == '\t'
	})
	keys := make([]string, 0, len(fields))
	for _, field := range fields {
		key := strings.TrimSpace(field)
		if key == "" {
			continue
		}
		keys = append(keys, key)
	}
	return keys
}

func matchesAnyConfiguredKey(provided string, configured []string) bool {
	if strings.TrimSpace(provided) == "" || len(configured) == 0 {
		return false
	}

	matched := 0
	for _, key := range configured {
		matched |= subtle.ConstantTimeCompare([]byte(provided), []byte(key))
	}
	return matched == 1
}
