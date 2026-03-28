package apiserver

import "testing"

func TestMatchAllowedOrigin(t *testing.T) {
	allowed := []string{"https://admin.example.com", "http://localhost:5173"}
	if got := matchAllowedOrigin("https://admin.example.com", allowed); got != "https://admin.example.com" {
		t.Fatalf("expected exact origin to be allowed, got %q", got)
	}
	if got := matchAllowedOrigin("https://other.example.com", allowed); got != "" {
		t.Fatalf("expected unknown origin to be rejected, got %q", got)
	}
}

func TestMatchAllowedOriginWildcard(t *testing.T) {
	if got := matchAllowedOrigin("https://any.example.com", []string{"*"}); got != "*" {
		t.Fatalf("expected wildcard to allow origin, got %q", got)
	}
}

func TestJoinHeaderValuesDeduplicates(t *testing.T) {
	got := joinHeaderValues([]string{"Authorization", "authorization", " X-Admin-Key "}, nil)
	want := "Authorization, X-Admin-Key"
	if got != want {
		t.Fatalf("joinHeaderValues() = %q, want %q", got, want)
	}
}
