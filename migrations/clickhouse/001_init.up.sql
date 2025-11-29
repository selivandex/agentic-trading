-- migrations/clickhouse/001_init.up.sql
-- ClickHouse schemas for time-series market data
-- OHLCV Data (candlesticks)
CREATE TABLE
  IF NOT EXISTS ohlcv (
    exchange LowCardinality (String),
    symbol LowCardinality (String),
    timeframe LowCardinality (String),
    market_type LowCardinality (String) DEFAULT 'spot',
    open_time DateTime,
    close_time DateTime,
    open Float64,
    high Float64,
    low Float64,
    close Float64,
    volume Float64,
    quote_volume Float64,
    trades UInt64,
    taker_buy_base_volume Float64,
    taker_buy_quote_volume Float64,
    is_closed Bool DEFAULT true,
    event_time DateTime64 (3),
    collected_at DateTime DEFAULT now ()
  ) ENGINE = ReplacingMergeTree (event_time)
PARTITION BY
  toYYYYMM (open_time)
ORDER BY
  (exchange, symbol, timeframe, open_time) TTL open_time + INTERVAL 2 YEAR;

-- Mark Price & Funding Rate (real-time derivatives data)
-- Updates every 1-3 seconds for perpetual contracts
-- ONLY for futures/perpetuals (does not exist for spot)
CREATE TABLE
  IF NOT EXISTS mark_price (
    exchange LowCardinality (String),
    symbol LowCardinality (String),
    market_type LowCardinality (String) DEFAULT 'futures',
    timestamp DateTime,
    mark_price Float64,
    index_price Float64,
    estimated_settle_price Float64,
    funding_rate Float64,
    next_funding_time DateTime,
    event_time DateTime64 (3),
    collected_at DateTime DEFAULT now ()
  ) ENGINE = ReplacingMergeTree (event_time)
PARTITION BY
  toYYYYMMDD (timestamp)
ORDER BY
  (exchange, market_type, symbol, timestamp) TTL timestamp + INTERVAL 90 DAY;

-- 24hr Ticker Statistics (rolling window)
-- For both spot and futures markets
CREATE TABLE
  IF NOT EXISTS tickers (
    exchange LowCardinality (String),
    symbol LowCardinality (String),
    market_type LowCardinality (String) DEFAULT 'futures',
    timestamp DateTime,
    last_price Float64,
    open_price Float64,
    high_price Float64,
    low_price Float64,
    volume Float64,
    quote_volume Float64,
    price_change Float64,
    price_change_percent Float64,
    weighted_avg_price Float64,
    trade_count UInt64,
    event_time DateTime64 (3),
    collected_at DateTime DEFAULT now ()
  ) ENGINE = ReplacingMergeTree (event_time)
PARTITION BY
  toYYYYMMDD (timestamp)
ORDER BY
  (exchange, market_type, symbol, timestamp) TTL timestamp + INTERVAL 30 DAY;

-- Order Book Snapshots
-- For both spot and futures markets
CREATE TABLE
  IF NOT EXISTS orderbook_snapshots (
    exchange LowCardinality (String),
    symbol LowCardinality (String),
    market_type LowCardinality (String) DEFAULT 'futures',
    timestamp DateTime,
    bids String,
    asks String,
    bid_depth Float64,
    ask_depth Float64,
    event_time DateTime64 (3),
    collected_at DateTime DEFAULT now ()
  ) ENGINE = ReplacingMergeTree (event_time)
PARTITION BY
  toYYYYMMDD (timestamp)
ORDER BY
  (exchange, market_type, symbol, timestamp) TTL timestamp + INTERVAL 7 DAY;

-- Aggregated Trades (tape/time & sales)
-- Aggregates trades that fill from the same taker order
-- For both spot and futures markets
CREATE TABLE
  IF NOT EXISTS trades (
    exchange LowCardinality (String),
    symbol LowCardinality (String),
    market_type LowCardinality (String) DEFAULT 'futures',
    timestamp DateTime,
    trade_id Int64,
    agg_trade_id Int64,
    price Float64,
    quantity Float64,
    first_trade_id Int64,
    last_trade_id Int64,
    is_buyer_maker Bool,
    event_time DateTime64 (3),
    collected_at DateTime DEFAULT now ()
  ) ENGINE = MergeTree ()
PARTITION BY
  toYYYYMMDD (timestamp)
ORDER BY
  (
    exchange,
    market_type,
    symbol,
    timestamp,
    trade_id
  ) TTL timestamp + INTERVAL 7 DAY;

-- Large Trades (whale alerts)
-- Materialized view for trades above threshold
CREATE MATERIALIZED VIEW IF NOT EXISTS large_trades_mv ENGINE = MergeTree ()
PARTITION BY
  toYYYYMMDD (timestamp)
ORDER BY
  (exchange, market_type, symbol, timestamp) AS
SELECT
  exchange,
  symbol,
  market_type,
  timestamp,
  trade_id,
  price,
  quantity,
  price * quantity as value_usd,
  is_buyer_maker,
  event_time
FROM
  trades
WHERE
  price * quantity > 50000;

-- Volume Profile (aggregated by price levels)
CREATE MATERIALIZED VIEW IF NOT EXISTS volume_profile_mv ENGINE = SummingMergeTree ()
PARTITION BY
  toYYYYMM (hour)
ORDER BY
  (exchange, market_type, symbol, hour, price_level) AS
SELECT
  exchange,
  symbol,
  market_type,
  toStartOfHour (timestamp) as hour,
  round(price, 0) as price_level,
  sum(quantity) as total_volume,
  countIf (is_buyer_maker = 0) as buy_volume_count,
  countIf (is_buyer_maker = 1) as sell_volume_count
FROM
  trades
GROUP BY
  exchange,
  symbol,
  market_type,
  hour,
  price_level;

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
    published_at DateTime,
    collected_at DateTime DEFAULT now ()
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
    timestamp DateTime,
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

-- Liquidations (force liquidation orders)
-- ONLY for futures/perpetuals (does not exist for spot)
CREATE TABLE
  IF NOT EXISTS liquidations (
    exchange LowCardinality (String),
    symbol LowCardinality (String),
    market_type LowCardinality (String) DEFAULT 'futures',
    timestamp DateTime,
    side LowCardinality (String),
    order_type LowCardinality (String),
    price Float64,
    quantity Float64,
    value Float64,
    event_time DateTime64 (3),
    collected_at DateTime DEFAULT now ()
  ) ENGINE = MergeTree ()
PARTITION BY
  toYYYYMMDD (timestamp)
ORDER BY
  (exchange, market_type, symbol, timestamp) TTL timestamp + INTERVAL 30 DAY;

-- Funding Rate History (snapshots every 8 hours)
-- Aggregated view for historical analysis
CREATE MATERIALIZED VIEW IF NOT EXISTS funding_rate_history_mv ENGINE = ReplacingMergeTree (event_time)
PARTITION BY
  toYYYYMM (funding_time)
ORDER BY
  (exchange, market_type, symbol, funding_time) AS
SELECT
  exchange,
  symbol,
  market_type,
  next_funding_time as funding_time,
  funding_rate,
  mark_price,
  index_price,
  event_time
FROM
  mark_price
WHERE
  funding_rate != 0;

-- Funding Rate Statistics (hourly aggregates)
CREATE MATERIALIZED VIEW IF NOT EXISTS funding_rate_stats_mv ENGINE = AggregatingMergeTree ()
PARTITION BY
  toYYYYMM (hour)
ORDER BY
  (exchange, market_type, symbol, hour) AS
SELECT
  exchange,
  symbol,
  market_type,
  toStartOfHour (timestamp) as hour,
  avgState (funding_rate) as avg_funding_rate,
  maxState (funding_rate) as max_funding_rate,
  minState (funding_rate) as min_funding_rate,
  countState () as sample_count
FROM
  mark_price
WHERE
  funding_rate != 0
GROUP BY
  exchange,
  symbol,
  market_type,
  hour;

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
