package database

import (
	"context"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestPluginSettingsStoreCRUD(t *testing.T) {
	db := newTestDB(t)
	store := NewPluginSettingsStore(db)
	ctx := context.Background()

	if _, err := store.GetSettings(ctx, ""); err == nil {
		t.Fatalf("expected error for empty key")
	}

	value, err := store.GetSettings(ctx, "missing")
	if err != nil {
		t.Fatalf("get settings: %v", err)
	}
	if string(value) != "{}" {
		t.Fatalf("expected empty settings, got %s", string(value))
	}

	if err := store.SetSettings(ctx, "", []byte(`{"a":1}`)); err == nil {
		t.Fatalf("expected error for empty key")
	}
	if err := store.SetSettings(ctx, "demo", []byte("not-json")); err == nil {
		t.Fatalf("expected error for invalid json")
	}

	if err := store.SetSettings(ctx, "demo", []byte(`{"a":1}`)); err != nil {
		t.Fatalf("set settings: %v", err)
	}

	value, err = store.GetSettings(ctx, "demo")
	if err != nil {
		t.Fatalf("get settings: %v", err)
	}
	if string(value) != `{"a":1}` {
		t.Fatalf("unexpected settings %s", string(value))
	}
}

func TestPluginServiceConfigStore(t *testing.T) {
	db := newTestDB(t)
	store := NewPluginServiceConfigStore(db)
	ctx := context.Background()

	if _, err := store.GetGRPCAddress(ctx, ""); err == nil {
		t.Fatalf("expected error for empty key")
	}
	if err := store.SetGRPCAddress(ctx, "", "addr"); err == nil {
		t.Fatalf("expected error for empty key")
	}
	if err := store.SetGRPCAddress(ctx, "demo", ""); err == nil {
		t.Fatalf("expected error for empty address")
	}

	if err := store.SetGRPCAddress(ctx, "octo-demo", "127.0.0.1:60051"); err != nil {
		t.Fatalf("set grpc address: %v", err)
	}

	addr, err := store.GetGRPCAddress(ctx, "octo_demo")
	if err != nil {
		t.Fatalf("get grpc address: %v", err)
	}
	if addr != "127.0.0.1:60051" {
		t.Fatalf("unexpected grpc address %q", addr)
	}

	items, err := store.ListGRPCAddresses(ctx)
	if err != nil {
		t.Fatalf("list grpc addresses: %v", err)
	}
	if items["octo_demo"] != "127.0.0.1:60051" {
		t.Fatalf("unexpected list output %#v", items)
	}
}

func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := AutoMigrate(context.Background(), db); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	return db
}
