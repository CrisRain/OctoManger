package redisclient

import (
	"testing"

	"octomanger/internal/platform/config"
	"octomanger/internal/testutil"
)

func TestNewRedisSuccess(t *testing.T) {
	_, server := testutil.NewTestRedis(t)
	cfg := config.RedisConfig{Addr: server.Addr()}

	rdb, err := New(cfg)
	if err != nil {
		t.Fatalf("new redis: %v", err)
	}
	if rdb == nil {
		t.Fatalf("expected redis client")
	}
	_ = rdb.Close()
}

func TestNewRedisFailure(t *testing.T) {
	cfg := config.RedisConfig{Addr: "127.0.0.1:0"}
	if _, err := New(cfg); err == nil {
		t.Fatalf("expected error for invalid redis addr")
	}
}
