-- Rollback: remove strategy_id from trading_pairs
DROP INDEX IF EXISTS idx_trading_pairs_strategy;

ALTER TABLE trading_pairs
DROP COLUMN IF EXISTS strategy_id;


