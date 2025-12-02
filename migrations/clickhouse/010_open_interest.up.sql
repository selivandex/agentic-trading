-- migrations/clickhouse/010_open_interest.up.sql
-- Open Interest tracking for futures/perpetuals
-- ============================================================================
-- Open Interest Table
-- ============================================================================
-- Stores periodic snapshots of open interest for futures contracts
CREATE TABLE IF NOT EXISTS open_interest (
    exchange LowCardinality(String),
    symbol LowCardinality(String),
    timestamp DateTime64(3),
    amount Float64  -- Open interest in contracts or USD
) ENGINE = ReplacingMergeTree()
ORDER BY (exchange, symbol, timestamp)
PARTITION BY toYYYYMM(timestamp)
TTL toDateTime(timestamp) + INTERVAL 90 DAY;

-- ============================================================================
-- Indexes
-- ============================================================================
CREATE INDEX IF NOT EXISTS idx_open_interest_amount ON open_interest (amount) TYPE minmax GRANULARITY 4;

