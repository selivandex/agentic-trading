package testsupport

import "testing"

func TestLoadDatabaseConfigsFromEnv(t *testing.T) {
	t.Setenv("POSTGRES_HOST", "localhost")
	t.Setenv("POSTGRES_USER", "user")
	t.Setenv("POSTGRES_PASSWORD", "pass")
	t.Setenv("POSTGRES_DB", "db")
	t.Setenv("POSTGRES_PORT", "5543")
	t.Setenv("POSTGRES_SSL_MODE", "disable")

	t.Setenv("CLICKHOUSE_HOST", "click")
	t.Setenv("CLICKHOUSE_DB", "analytics")
	t.Setenv("CLICKHOUSE_PORT", "8123")

	t.Setenv("REDIS_HOST", "redis")
	t.Setenv("REDIS_PORT", "6380")
	t.Setenv("REDIS_DB", "2")

	cfg := LoadDatabaseConfigsFromEnv(t)

	if cfg.Postgres.Host != "localhost" || cfg.Postgres.Port != 5543 {
		t.Fatalf("unexpected postgres config %+v", cfg.Postgres)
	}

	if cfg.ClickHouse.Host != "click" || cfg.ClickHouse.Port != 8123 {
		t.Fatalf("unexpected clickhouse config %+v", cfg.ClickHouse)
	}

	if cfg.Redis.Host != "redis" || cfg.Redis.Port != 6380 || cfg.Redis.DB != 2 {
		t.Fatalf("unexpected redis config %+v", cfg.Redis)
	}
}
