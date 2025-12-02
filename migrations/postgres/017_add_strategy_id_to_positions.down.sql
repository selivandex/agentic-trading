-- Rollback: remove strategy_id from positions
DROP INDEX IF EXISTS idx_positions_strategy;

ALTER TABLE positions
DROP COLUMN IF EXISTS strategy_id;
