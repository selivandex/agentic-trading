-- migrations/clickhouse/002_tool_stats.down.sql
-- Rollback tool stats tables

DROP VIEW IF EXISTS tool_usage_hourly_mv;
DROP TABLE IF EXISTS tool_usage_stats;

-- Rollback tool stats tables

DROP VIEW IF EXISTS tool_usage_hourly_mv;
DROP TABLE IF EXISTS tool_usage_stats;

