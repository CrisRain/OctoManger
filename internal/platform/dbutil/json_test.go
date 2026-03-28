package dbutil

import "testing"

func TestDecodeJSONMap(t *testing.T) {
	if got := DecodeJSONMap(nil); len(got) != 0 {
		t.Fatalf("expected empty map for nil input")
	}
	if got := DecodeJSONMap([]byte("not-json")); len(got) != 0 {
		t.Fatalf("expected empty map for invalid JSON")
	}
	value := DecodeJSONMap([]byte(`{"hello":"world"}`))
	if value["hello"] != "world" {
		t.Fatalf("expected decoded map value, got %#v", value)
	}
}

func TestNormalizeMap(t *testing.T) {
	if got := NormalizeMap(nil); got == nil || len(got) != 0 {
		t.Fatalf("expected empty map for nil input")
	}
	original := map[string]any{"a": 1}
	if got := NormalizeMap(original); got["a"] != 1 {
		t.Fatalf("expected original map content, got %#v", got)
	}
}

func TestMergeMaps(t *testing.T) {
	base := map[string]any{"a": 1, "b": 2}
	override := map[string]any{"b": 9, "c": 3}
	merged := MergeMaps(base, override)
	if merged["a"] != 1 || merged["b"] != 9 || merged["c"] != 3 {
		t.Fatalf("unexpected merge result: %#v", merged)
	}
}
