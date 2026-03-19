package redisclient

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"

	"octomanger/internal/platform/config"
)

// New creates and pings a Redis client.
func New(cfg config.RedisConfig) (*redis.Client, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	if err := rdb.Ping(context.Background()).Err(); err != nil {
		return nil, fmt.Errorf("ping redis: %w", err)
	}

	return rdb, nil
}
