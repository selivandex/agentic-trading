-- migrations/clickhouse/003_funding_rates.up.sql
-- Funding rates for perpetual futures

CREATE TABLE IF NOT EXISTS funding_rates (
    exchange LowCardinality(String),
    symbol LowCardinality(String),
    timestamp DateTime64(3),
    funding_rate Float64,
    next_funding_time DateTime64(3),
    mark_price Float64,
    index_price Float64
) ENGINE = ReplacingMergeTree()
PARTITION BY toYYYYMMDD(timestamp)
ORDER BY (exchange, symbol, timestamp)
TTL timestamp + INTERVAL 90 DAY;

-- Create materialized view for funding rate statistics
CREATE MATERIALIZED VIEW IF NOT EXISTS funding_rates_hourly_mv
ENGINE = AggregatingMergeTree()
PARTITION BY toYYYYMM(hour)
ORDER BY (exchange, symbol, hour)
AS SELECT
    exchange,
    symbol,
    toStartOfHour(timestamp) as hour,
    avg(funding_rate) as avg_funding_rate,
    max(funding_rate) as max_funding_rate,
    min(funding_rate) as min_funding_rate,
    count() as samples_count
FROM funding_rates
GROUP BY exchange, symbol, hour;



