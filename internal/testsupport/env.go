package testsupport

import (
	"fmt"
	"os"
	"testing"

	"prometheus/internal/adapters/config"
)

// DatabaseConfigs bundles config sections required for integration tests.
type DatabaseConfigs struct {
	Postgres   config.PostgresConfig
	ClickHouse config.ClickHouseConfig
	Redis      config.RedisConfig
}

// LoadDatabaseConfigsFromEnv reads minimal configuration for integration tests.
// Tests are skipped when required environment variables are missing.
func LoadDatabaseConfigsFromEnv(t *testing.T) DatabaseConfigs {
	t.Helper()

	required := []string{
		"POSTGRES_HOST", "POSTGRES_USER", "POSTGRES_PASSWORD", "POSTGRES_DB",
		"CLICKHOUSE_HOST", "CLICKHOUSE_DB",
		"REDIS_HOST",
	}

	missing := make([]string, 0)
	for _, key := range required {
		if os.Getenv(key) == "" {
			missing = append(missing, key)
		}
	}

	if len(missing) > 0 {
		t.Skipf("integration environment missing, set %v to run", missing)
	}

	return DatabaseConfigs{
		Postgres: config.PostgresConfig{
			Host:     os.Getenv("POSTGRES_HOST"),
			Port:     intValue("POSTGRES_PORT", 5432),
			User:     os.Getenv("POSTGRES_USER"),
			Password: os.Getenv("POSTGRES_PASSWORD"),
			Database: os.Getenv("POSTGRES_DB"),
			SSLMode:  valueWithDefault("POSTGRES_SSL_MODE", "disable"),
			MaxConns: 10,
		},
		ClickHouse: config.ClickHouseConfig{
			Host:     os.Getenv("CLICKHOUSE_HOST"),
			Port:     intValue("CLICKHOUSE_PORT", 9000),
			User:     valueWithDefault("CLICKHOUSE_USER", "default"),
			Password: os.Getenv("CLICKHOUSE_PASSWORD"),
			Database: os.Getenv("CLICKHOUSE_DB"),
		},
		Redis: config.RedisConfig{
			Host:     os.Getenv("REDIS_HOST"),
			Port:     intValue("REDIS_PORT", 6379),
			Password: os.Getenv("REDIS_PASSWORD"),
			DB:       intValue("REDIS_DB", 0),
		},
	}
}

func valueWithDefault(key string, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}

	return fallback
}

func intValue(key string, fallback int) int {
	if val := os.Getenv(key); val != "" {
		var parsed int
		_, err := fmt.Sscanf(val, "%d", &parsed)
		if err == nil {
			return parsed
		}
	}

	return fallback
}
