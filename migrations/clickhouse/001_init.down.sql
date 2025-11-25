-- migrations/clickhouse/001_init.down.sql
-- Rollback ClickHouse tables

DROP VIEW IF EXISTS sentiment_hourly_mv;

DROP TABLE IF EXISTS liquidations;
DROP TABLE IF EXISTS social_sentiment;
DROP TABLE IF EXISTS news;
DROP TABLE IF EXISTS trades;
DROP TABLE IF EXISTS orderbook_snapshots;
DROP TABLE IF EXISTS tickers;
DROP TABLE IF EXISTS ohlcv;

