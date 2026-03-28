package auth

import (
	"context"
	"testing"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
)

func TestFailedAttemptLimiterBlocksAfterThreshold(t *testing.T) {
	limiter := newFailedAttemptLimiter()
	now := time.Unix(1000, 0)

	for i := 0; i < maxFailedAttemptsPerWindow-1; i++ {
		if retryAfter, blocked := limiter.recordFailure("127.0.0.1", now); blocked {
			t.Fatalf("attempt %d unexpectedly blocked with retry_after=%s", i+1, retryAfter)
		}
	}

	retryAfter, blocked := limiter.recordFailure("127.0.0.1", now)
	if !blocked {
		t.Fatalf("expected limiter to block after threshold")
	}
	if retryAfter <= 0 {
		t.Fatalf("expected positive retry_after, got %s", retryAfter)
	}

	if _, stillBlocked := limiter.isBlocked("127.0.0.1", now.Add(failedAttemptsBlockWindow/2)); !stillBlocked {
		t.Fatalf("expected limiter to remain blocked within block window")
	}
	if _, stillBlocked := limiter.isBlocked("127.0.0.1", now.Add(failedAttemptsBlockWindow+time.Second)); stillBlocked {
		t.Fatalf("expected limiter to unblock after block window")
	}
}

func TestFailedAttemptLimiterResetClearsBlockState(t *testing.T) {
	limiter := newFailedAttemptLimiter()
	now := time.Unix(2000, 0)

	for i := 0; i < maxFailedAttemptsPerWindow; i++ {
		_, _ = limiter.recordFailure("10.0.0.2", now)
	}

	limiter.reset("10.0.0.2")

	if _, blocked := limiter.isBlocked("10.0.0.2", now); blocked {
		t.Fatalf("expected limiter reset to clear block state")
	}
}

func TestRequireAdminRejectsWhenNotConfigured(t *testing.T) {
	mw := RequireAdmin("")
	c := &app.RequestContext{}
	c.Request.SetRequestURI("/api/v2/accounts")

	mw(context.Background(), c)

	if c.Response.StatusCode() != StatusUnauthorized {
		t.Fatalf("unexpected status code: %d", c.Response.StatusCode())
	}
}

func TestRequireAdminAcceptsAnyConfiguredKey(t *testing.T) {
	mw := RequireAdmin("alpha-key, beta-key")
	c := &app.RequestContext{}
	c.Request.SetRequestURI("/api/v2/accounts")
	c.Request.Header.Set("X-Admin-Key", "beta-key")

	mw(context.Background(), c)

	if c.Response.StatusCode() != 200 {
		t.Fatalf("expected pass-through status, got status=%d", c.Response.StatusCode())
	}
}

func TestRequireAdminAcceptsAuthorizationBearer(t *testing.T) {
	mw := RequireAdmin("only-key")
	c := &app.RequestContext{}
	c.Request.SetRequestURI("/api/v2/accounts")
	c.Request.Header.Set("Authorization", "Bearer only-key")

	mw(context.Background(), c)

	if c.Response.StatusCode() != 200 {
		t.Fatalf("expected pass-through status, got status=%d", c.Response.StatusCode())
	}
}

func TestParseConfiguredKeys(t *testing.T) {
	keys := parseConfiguredKeys("a,\n b; c\t")
	if len(keys) != 3 {
		t.Fatalf("expected 3 keys, got %d (%v)", len(keys), keys)
	}
	if keys[0] != "a" || keys[1] != "b" || keys[2] != "c" {
		t.Fatalf("unexpected parsed keys: %#v", keys)
	}
}
