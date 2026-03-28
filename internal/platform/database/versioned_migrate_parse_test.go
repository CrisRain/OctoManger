package database

import "testing"

func TestParseAppliedVersionValueEdges(t *testing.T) {
	if _, ok := parseAppliedVersionValue(float64(1.5)); ok {
		t.Fatalf("expected fractional float to be rejected")
	}
	if _, ok := parseAppliedVersionValue(uint64(^uint64(0))); ok {
		t.Fatalf("expected oversized uint64 to be rejected")
	}
	if v, ok := parseAppliedVersionValue(" 42 "); !ok || v != 42 {
		t.Fatalf("expected parsed string version, got %d ok=%v", v, ok)
	}
}

func TestParseStrictIntString(t *testing.T) {
	if _, ok := parseStrictIntString(""); ok {
		t.Fatalf("expected empty string to be rejected")
	}
	if _, ok := parseStrictIntString("abc"); ok {
		t.Fatalf("expected invalid string to be rejected")
	}
	if v, ok := parseStrictIntString("7"); !ok || v != 7 {
		t.Fatalf("expected parsed value, got %d ok=%v", v, ok)
	}
}

func TestIsLegacyVersionMarker(t *testing.T) {
	if !isLegacyVersionMarker("0001_initial.sql") {
		t.Fatalf("expected legacy marker to be detected")
	}
	if isLegacyVersionMarker("12345") {
		t.Fatalf("expected numeric-only value to be rejected")
	}
	if isLegacyVersionMarker(123) {
		t.Fatalf("expected non-string value to be rejected")
	}
}

func TestSchemaMigrationVersionValue(t *testing.T) {
	if v := schemaMigrationVersionValue(5, "text"); v != "5" {
		t.Fatalf("expected string version for text column, got %#v", v)
	}
	if v := schemaMigrationVersionValue(5, "bigint"); v != int64(5) {
		t.Fatalf("expected int64 version for bigint column, got %#v", v)
	}
}
