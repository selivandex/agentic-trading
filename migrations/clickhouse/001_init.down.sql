-- migrations/clickhouse/001_init.down.sql
-- Rollback ClickHouse tables

-- Drop materialized views first (dependency order)
DROP VIEW IF EXISTS sentiment_hourly_mv;
DROP VIEW IF EXISTS funding_rate_stats_mv;
DROP VIEW IF EXISTS funding_rate_history_mv;
DROP VIEW IF EXISTS volume_profile_mv;
DROP VIEW IF EXISTS large_trades_mv;

-- Drop tables
DROP TABLE IF EXISTS liquidations;
DROP TABLE IF EXISTS social_sentiment;
DROP TABLE IF EXISTS news;
DROP TABLE IF EXISTS trades;
DROP TABLE IF EXISTS orderbook_snapshots;
DROP TABLE IF EXISTS tickers;
DROP TABLE IF EXISTS mark_price;
DROP TABLE IF EXISTS ohlcv;

