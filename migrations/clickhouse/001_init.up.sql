-- migrations/clickhouse/001_init.up.sql
-- ClickHouse schemas for time-series market data
-- OHLCV Data (candlesticks)
CREATE TABLE
  IF NOT EXISTS ohlcv (
    exchange LowCardinality (String),
    symbol LowCardinality (String),
    timeframe LowCardinality (String),
    open_time DateTime64 (3),
    open Float64,
    high Float64,
    low Float64,
    close Float64,
    volume Float64,
    quote_volume Float64,
    trades UInt64
  ) ENGINE = ReplacingMergeTree ()
PARTITION BY
  toYYYYMM (open_time)
ORDER BY
  (exchange, symbol, timeframe, open_time) TTL open_time + INTERVAL 2 YEAR;

-- Tickers (real-time price data)
CREATE TABLE
  IF NOT EXISTS tickers (
    exchange LowCardinality (String),
    symbol LowCardinality (String),
    timestamp DateTime64 (3),
    price Float64,
    bid Float64,
    ask Float64,
    volume_24h Float64,
    change_24h Float64,
    high_24h Float64,
    low_24h Float64,
    funding_rate Float64,
    open_interest Float64
  ) ENGINE = MergeTree ()
PARTITION BY
  toYYYYMMDD (timestamp)
ORDER BY
  (exchange, symbol, timestamp) TTL timestamp + INTERVAL 30 DAY;

-- Order Book Snapshots
CREATE TABLE
  IF NOT EXISTS orderbook_snapshots (
    exchange LowCardinality (String),
    symbol LowCardinality (String),
    timestamp DateTime64 (3),
    bids String, -- JSON array
    asks String, -- JSON array
    bid_depth Float64,
    ask_depth Float64
  ) ENGINE = MergeTree ()
PARTITION BY
  toYYYYMMDD (timestamp)
ORDER BY
  (exchange, symbol, timestamp) TTL timestamp + INTERVAL 7 DAY;

-- Trades (tape)
CREATE TABLE
  IF NOT EXISTS trades (
    exchange LowCardinality (String),
    symbol LowCardinality (String),
    timestamp DateTime64 (3),
    trade_id String,
    price Float64,
    quantity Float64,
    side LowCardinality (String),
    is_buyer Bool
  ) ENGINE = MergeTree ()
PARTITION BY
  toYYYYMMDD (timestamp)
ORDER BY
  (exchange, symbol, timestamp) TTL timestamp + INTERVAL 7 DAY;

-- News & Sentiment
CREATE TABLE
  IF NOT EXISTS news (
    id UUID DEFAULT generateUUIDv4 (),
    source LowCardinality (String),
    title String,
    content String,
    url String,
    sentiment Float32,
    symbols Array (LowCardinality (String)),
    published_at DateTime64 (3),
    collected_at DateTime64 (3) DEFAULT now64 (3)
  ) ENGINE = MergeTree ()
PARTITION BY
  toYYYYMM (published_at)
ORDER BY
  (published_at, source) TTL published_at + INTERVAL 1 YEAR;

-- Social Sentiment (Twitter, Reddit, etc.)
CREATE TABLE
  IF NOT EXISTS social_sentiment (
    platform LowCardinality (String),
    symbol LowCardinality (String),
    timestamp DateTime64 (3),
    mentions UInt32,
    sentiment_score Float32,
    positive_count UInt32,
    negative_count UInt32,
    neutral_count UInt32,
    influencer_sentiment Float32,
    trending_rank UInt16
  ) ENGINE = MergeTree ()
PARTITION BY
  toYYYYMMDD (timestamp)
ORDER BY
  (platform, symbol, timestamp) TTL timestamp + INTERVAL 90 DAY;

-- Liquidations
CREATE TABLE
  IF NOT EXISTS liquidations (
    exchange LowCardinality (String),
    symbol LowCardinality (String),
    timestamp DateTime64 (3),
    side LowCardinality (String), -- long, short
    price Float64,
    quantity Float64,
    value Float64 -- USD value
  ) ENGINE = MergeTree ()
PARTITION BY
  toYYYYMMDD (timestamp)
ORDER BY
  (exchange, symbol, timestamp) TTL timestamp + INTERVAL 30 DAY;

-- Materialized view for hourly sentiment aggregation
CREATE MATERIALIZED VIEW IF NOT EXISTS sentiment_hourly_mv ENGINE = SummingMergeTree ()
PARTITION BY
  toYYYYMM (hour)
ORDER BY
  (symbol, hour) AS
SELECT
  symbol,
  toStartOfHour (timestamp) as hour,
  avg(sentiment_score) as avg_sentiment,
  sum(mentions) as total_mentions,
  sum(positive_count) as total_positive,
  sum(negative_count) as total_negative
FROM
  social_sentiment
GROUP BY
  symbol,
  hour;
