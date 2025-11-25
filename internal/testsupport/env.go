package testsupport

import (
	"testing"

	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"

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

	_ = godotenv.Load()

	// Restrict parsing only to DB-related sections so other required fields
	// (e.g. Kafka, Telegram) do not block integration tests.
	var cfg struct {
		Postgres   config.PostgresConfig
		ClickHouse config.ClickHouseConfig
		Redis      config.RedisConfig
	}

	if err := envconfig.Process("", &cfg); err != nil {
		t.Skipf("integration environment missing or invalid: %v", err)
	}

	return DatabaseConfigs{
		Postgres:   cfg.Postgres,
		ClickHouse: cfg.ClickHouse,
		Redis:      cfg.Redis,
	}
}
