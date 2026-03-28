package httpx

import "testing"

func TestSanitizeClientDetailBlocksSQLLikeMessage(t *testing.T) {
	got := sanitizeClientDetail(`pq: duplicate key value violates unique constraint "accounts_key"`)
	if got != "invalid request" {
		t.Fatalf("unexpected sanitized detail: %q", got)
	}
}

func TestSanitizeClientDetailKeepsSafeValidationMessage(t *testing.T) {
	got := sanitizeClientDetail("action is required")
	if got != "action is required" {
		t.Fatalf("unexpected sanitized detail: %q", got)
	}
}
