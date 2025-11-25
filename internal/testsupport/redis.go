package testsupport

import (
	"context"
	"testing"

	"github.com/redis/go-redis/v9"

	"prometheus/internal/adapters/config"
)

// NewRedisClient creates a redis client for integration tests and ensures database cleanup.
func NewRedisClient(t *testing.T, cfg config.RedisConfig) *redis.Client {
	t.Helper()

	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr(),
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	if err := client.Ping(context.Background()).Err(); err != nil {
		t.Fatalf("failed to connect to redis: %v", err)
	}

	if err := client.FlushDB(context.Background()).Err(); err != nil {
		t.Fatalf("failed to flush redis before test: %v", err)
	}

	t.Cleanup(func() {
		_ = client.FlushDB(context.Background()).Err()
		_ = client.Close()
	})

	return client
}
