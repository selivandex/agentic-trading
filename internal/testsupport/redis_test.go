package testsupport

import (
	"context"
	"testing"
)

func TestRedisClientIsCleanedBetweenTests(t *testing.T) {
	client := NewRedisClient(t, GetConfig().Redis)
	if err := client.Set(context.Background(), "integration-key", "value", 0).Err(); err != nil {
		t.Fatalf("failed to set key: %v", err)
	}

	val, err := client.Get(context.Background(), "integration-key").Result()
	if err != nil {
		t.Fatalf("failed to get key: %v", err)
	}

	if val != "value" {
		t.Fatalf("unexpected redis value: %s", val)
	}
}
