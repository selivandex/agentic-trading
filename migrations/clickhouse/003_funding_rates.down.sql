-- migrations/clickhouse/003_funding_rates.down.sql
-- Rollback funding rates tables

DROP VIEW IF EXISTS funding_rates_hourly_mv;
DROP TABLE IF EXISTS funding_rates;



