-- migrations/clickhouse/006_macro.down.sql
-- Rollback macro tables

DROP TABLE IF EXISTS market_correlations;
DROP TABLE IF EXISTS macro_events;

