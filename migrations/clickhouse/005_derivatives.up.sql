-- migrations/clickhouse/005_derivatives.up.sql
-- Derivatives (options) data tables

-- ============================================================================
-- Options Snapshots Table
-- ============================================================================
-- Stores periodic snapshots of options market state
CREATE TABLE IF NOT EXISTS options_snapshots (
    symbol LowCardinality(String),
    timestamp DateTime64(3),
    
    -- Open Interest
    call_oi Float64,
    put_oi Float64,
    total_oi Float64,
    put_call_ratio Float64,
    
    -- Max Pain
    max_pain_price Float64,
    max_pain_delta Float64,
    
    -- Gamma Exposure
    gamma_exposure Float64,
    gamma_flip Float64,
    
    -- Implied Volatility
    iv_index Float64,
    iv_25d_call Float64,
    iv_25d_put Float64,
    iv_skew Float64,
    
    -- Large Trades
    large_call_vol Float64,
    large_put_vol Float64
) ENGINE = MergeTree()
ORDER BY (symbol, timestamp)
PARTITION BY toYYYYMM(timestamp)
TTL timestamp + INTERVAL 365 DAY;

-- ============================================================================
-- Options Flows Table
-- ============================================================================
-- Stores large options trades
CREATE TABLE IF NOT EXISTS options_flows (
    id String,
    timestamp DateTime64(3),
    symbol LowCardinality(String),
    side LowCardinality(String), -- call, put
    strike Float64,
    expiry DateTime,
    premium Float64,
    size Int32,
    spot Float64,
    trade_type LowCardinality(String), -- buy, sell
    sentiment LowCardinality(String) -- bullish, bearish, neutral
) ENGINE = MergeTree()
ORDER BY (symbol, timestamp)
PARTITION BY toYYYYMM(timestamp)
TTL timestamp + INTERVAL 90 DAY;

-- ============================================================================
-- Indexes
-- ============================================================================

CREATE INDEX IF NOT EXISTS idx_options_gamma 
    ON options_snapshots (gamma_exposure) 
    TYPE minmax 
    GRANULARITY 4;

CREATE INDEX IF NOT EXISTS idx_options_flow_premium 
    ON options_flows (premium) 
    TYPE minmax 
    GRANULARITY 4;

