-- migrations/clickhouse/004_onchain.down.sql
-- Rollback on-chain data tables

DROP TABLE IF EXISTS miner_metrics;
DROP TABLE IF EXISTS network_metrics;
DROP TABLE IF EXISTS exchange_flows;
DROP TABLE IF EXISTS whale_movements;



