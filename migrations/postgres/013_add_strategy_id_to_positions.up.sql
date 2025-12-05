-- Migration 017: strategy_id column was moved to 001_init.up.sql
-- This migration is now a no-op for backward compatibility
-- Ensure index exists (in case it's missing)
CREATE INDEX IF NOT EXISTS idx_positions_strategy ON positions (strategy_id);

-- Add comment
COMMENT ON COLUMN positions.strategy_id IS 'Link to user_strategies table for capital allocation and PnL tracking';

