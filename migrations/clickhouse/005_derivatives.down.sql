-- migrations/clickhouse/005_derivatives.down.sql
-- Rollback derivatives tables

DROP TABLE IF EXISTS options_flows;
DROP TABLE IF EXISTS options_snapshots;



