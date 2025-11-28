-- migrations/clickhouse/002_tool_stats.up.sql
-- Tool usage statistics (time-series data)
CREATE TABLE
  IF NOT EXISTS tool_usage_stats (
    user_id UUID,
    agent_id LowCardinality (String),
    tool_name LowCardinality (String),
    timestamp DateTime64 (3),
    -- Metrics
    duration_ms UInt32,
    success Bool,
    -- Context
    session_id String,
    symbol String
  ) ENGINE = MergeTree ()
PARTITION BY
  toYYYYMMDD (timestamp)
ORDER BY
  (user_id, agent_id, tool_name, timestamp) TTL toDateTime (timestamp) + INTERVAL 90 DAY;

-- Materialized view for hourly aggregation
CREATE MATERIALIZED VIEW IF NOT EXISTS tool_usage_hourly_mv ENGINE = SummingMergeTree ()
PARTITION BY
  toYYYYMM (hour)
ORDER BY
  (user_id, agent_id, tool_name, hour) AS
SELECT
  user_id,
  agent_id,
  tool_name,
  toStartOfHour (timestamp) as hour,
  count() as call_count,
  sum(duration_ms) as total_duration_ms,
  countIf (success) as success_count,
  countIf (NOT success) as error_count,
  avg(duration_ms) as avg_duration_ms
FROM
  tool_usage_stats
GROUP BY
  user_id,
  agent_id,
  tool_name,
  hour;
