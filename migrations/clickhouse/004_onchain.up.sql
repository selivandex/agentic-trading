-- migrations/clickhouse/004_onchain.up.sql
-- On-chain data tables for blockchain analytics
-- ============================================================================
-- Whale Movements Table
-- ============================================================================
-- Tracks large wallet transfers that may indicate market-moving activity
CREATE TABLE
  IF NOT EXISTS whale_movements (
    tx_hash String,
    blockchain LowCardinality (String),
    from_address String,
    to_address String,
    token LowCardinality (String),
    amount Float64,
    amount_usd Float64,
    timestamp DateTime64 (3),
    from_label String DEFAULT '',
    to_label String DEFAULT ''
  ) ENGINE = MergeTree ()
ORDER BY
  (blockchain, token, timestamp)
PARTITION BY
  toYYYYMM (timestamp) TTL timestamp + INTERVAL 90 DAY;

-- Index for querying large movements
CREATE INDEX IF NOT EXISTS idx_whale_amount_usd ON whale_movements (amount_usd) TYPE minmax GRANULARITY 4;

-- ============================================================================
-- Exchange Flows Table
-- ============================================================================
-- Aggregated inflow/outflow data for exchange reserves
CREATE TABLE
  IF NOT EXISTS exchange_flows (
    exchange LowCardinality (String),
    blockchain LowCardinality (String),
    token LowCardinality (String),
    inflow_usd Float64,
    outflow_usd Float64,
    net_flow_usd Float64,
    timestamp DateTime64 (3)
  ) ENGINE = MergeTree ()
ORDER BY
  (exchange, blockchain, token, timestamp)
PARTITION BY
  toYYYYMM (timestamp) TTL timestamp + INTERVAL 180 DAY;

-- ============================================================================
-- Network Metrics Table
-- ============================================================================
-- Blockchain health and activity metrics
CREATE TABLE
  IF NOT EXISTS network_metrics (
    blockchain LowCardinality (String),
    timestamp DateTime64 (3),
    -- Common metrics
    active_addresses UInt32,
    transaction_count UInt32,
    avg_fee_usd Float64,
    -- Bitcoin-specific
    hash_rate Float64 DEFAULT 0,
    mempool_size UInt32 DEFAULT 0,
    mempool_bytes UInt64 DEFAULT 0,
    -- Ethereum-specific
    gas_price Float64 DEFAULT 0,
    gas_used UInt64 DEFAULT 0,
    block_utilization Float64 DEFAULT 0
  ) ENGINE = MergeTree ()
ORDER BY
  (blockchain, timestamp)
PARTITION BY
  toYYYYMM (timestamp) TTL timestamp + INTERVAL 365 DAY;

-- ============================================================================
-- Miner Metrics Table
-- ============================================================================
-- Mining pool behavior and holdings
CREATE TABLE
  IF NOT EXISTS miner_metrics (
    pool_name LowCardinality (String),
    timestamp DateTime64 (3),
    -- Mining power
    hash_rate Float64,
    hash_rate_share Float64,
    -- Bitcoin holdings
    btc_balance Float64,
    btc_sent Float64,
    btc_received Float64,
    -- Miner Position Index
    mpi Float64
  ) ENGINE = MergeTree ()
ORDER BY
  (pool_name, timestamp)
PARTITION BY
  toYYYYMM (timestamp) TTL timestamp + INTERVAL 365 DAY;

-- ============================================================================
-- Indexes for performance
-- ============================================================================
-- Exchange flow indexes
CREATE INDEX IF NOT EXISTS idx_exchange_flows_net ON exchange_flows (net_flow_usd) TYPE minmax GRANULARITY 4;

-- Network metrics indexes
CREATE INDEX IF NOT EXISTS idx_network_hash_rate ON network_metrics (hash_rate) TYPE minmax GRANULARITY 4;

CREATE INDEX IF NOT EXISTS idx_network_gas_price ON network_metrics (gas_price) TYPE minmax GRANULARITY 4;

-- Miner metrics indexes
CREATE INDEX IF NOT EXISTS idx_miner_mpi ON miner_metrics (mpi) TYPE minmax GRANULARITY 4;
