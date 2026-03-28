package testutil

import (
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

// NewTestRedis returns a live in-memory redis and a connected client.
func NewTestRedis(t *testing.T) (*redis.Client, *miniredis.Miniredis) {
	t.Helper()

	server, err := miniredis.Run()
	if err != nil {
		t.Fatalf("start miniredis: %v", err)
	}
	t.Cleanup(server.Close)

	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	t.Cleanup(func() {
		_ = client.Close()
	})

	return client, server
}
