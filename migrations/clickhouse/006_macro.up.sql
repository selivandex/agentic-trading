-- migrations/clickhouse/006_macro.up.sql
-- Macro economic data tables

-- ============================================================================
-- Macro Events Table
-- ============================================================================
-- Stores economic calendar events (CPI, NFP, FOMC, GDP, etc.)
CREATE TABLE IF NOT EXISTS macro_events (
    id String,
    event_type LowCardinality(String), -- cpi, nfp, fomc, gdp, etc.
    title String,
    country LowCardinality(String),
    currency LowCardinality(String),
    event_time DateTime,
    
    -- Event values
    previous String,
    forecast String,
    actual String,
    
    -- Impact
    impact LowCardinality(String), -- low, medium, high
    
    -- BTC reaction (populated after event)
    btc_reaction_1h Float64 DEFAULT 0,
    btc_reaction_24h Float64 DEFAULT 0,
    
    collected_at DateTime64(3)
) ENGINE = MergeTree()
ORDER BY (event_type, event_time)
PARTITION BY toYYYYMM(event_time)
TTL event_time + INTERVAL 730 DAY;

-- ============================================================================
-- Market Correlations Table
-- ============================================================================
-- Stores BTC correlation with traditional markets (SPX, Gold, DXY, Bonds)
CREATE TABLE IF NOT EXISTS market_correlations (
    timestamp DateTime64(3),
    asset LowCardinality(String), -- SPY, GLD, DXY, TLT
    correlation Float64, -- -1 to 1
    period_days Int32, -- Rolling window (e.g., 30 days)
    btc_return Float64,
    asset_return Float64
) ENGINE = MergeTree()
ORDER BY (asset, timestamp)
PARTITION BY toYYYYMM(timestamp)
TTL toDateTime(timestamp) + INTERVAL 365 DAY;

-- ============================================================================
-- Indexes
-- ============================================================================

CREATE INDEX IF NOT EXISTS idx_macro_impact 
    ON macro_events (impact) 
    TYPE set(0) 
    GRANULARITY 4;

CREATE INDEX IF NOT EXISTS idx_correlation_value 
    ON market_correlations (correlation) 
    TYPE minmax 
    GRANULARITY 4;



