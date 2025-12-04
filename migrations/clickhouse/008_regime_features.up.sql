-- migrations/clickhouse/008_regime_features.up.sql
-- Time-series storage for extracted regime features

CREATE TABLE IF NOT EXISTS regime_features (
    symbol LowCardinality(String),
    timestamp DateTime,
    
    -- Volatility features (4 features)
    atr_14 Float64,
    atr_pct Float64,
    bb_width Float64,
    historical_vol Float64,
    
    -- Trend features (8 features)
    adx Float64,
    ema_9 Float64,
    ema_21 Float64,
    ema_55 Float64,
    ema_200 Float64,
    ema_alignment String,  -- 'bullish', 'bearish', 'neutral'
    higher_highs_count UInt8,
    lower_lows_count UInt8,
    
    -- Volume features (3 features)
    volume_24h Float64,
    volume_change_pct Float64,
    volume_price_divergence Float64,
    
    -- Structure features (3 features)
    support_breaks UInt8,
    resistance_breaks UInt8,
    consolidation_periods UInt8,
    
    -- Cross-asset features (2 features)
    btc_dominance Float64,
    correlation_tightness Float64,
    
    -- Derivatives features (3 features)
    funding_rate Float64,
    funding_rate_ma Float64,
    liquidations_24h Float64,
    
    -- Calculated regime (for ML training labels)
    regime_label LowCardinality(String),  -- manual labeling for training
    
    INDEX idx_timestamp timestamp TYPE minmax GRANULARITY 3
) ENGINE = ReplacingMergeTree()
PARTITION BY toYYYYMM(timestamp)
ORDER BY (symbol, timestamp)
TTL timestamp + INTERVAL 1 YEAR;

-- Materialized view for hourly feature aggregation
CREATE MATERIALIZED VIEW IF NOT EXISTS regime_features_hourly_mv
ENGINE = AggregatingMergeTree()
PARTITION BY toYYYYMM(hour)
ORDER BY (symbol, hour)
AS SELECT
    symbol,
    toStartOfHour(timestamp) as hour,
    avgState(atr_14) as avg_atr,
    avgState(adx) as avg_adx,
    avgState(bb_width) as avg_bb_width,
    avgState(volume_change_pct) as avg_volume_change
FROM regime_features
GROUP BY symbol, hour;




