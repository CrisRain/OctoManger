package apikey

import "testing"

func TestGenerateAndMatch(t *testing.T) {
	key, err := Generate()
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	if len(key) == 0 || key[:4] != "okm_" {
		t.Fatalf("expected okm_ prefix, got %q", key)
	}

	hash := Hash(key)
	if hash == "" {
		t.Fatalf("expected hash to be non-empty")
	}
	if !Match(key, hash) {
		t.Fatalf("expected hash to match key")
	}
	if Match("other", hash) {
		t.Fatalf("expected hash mismatch for different key")
	}
}

func TestHashEmpty(t *testing.T) {
	if Hash("") != "" {
		t.Fatalf("expected empty hash for blank input")
	}
	if Match("", "") {
		t.Fatalf("expected empty hash to never match")
	}
}
