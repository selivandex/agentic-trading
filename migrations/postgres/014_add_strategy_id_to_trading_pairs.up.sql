-- Migration 014: strategy_id column was moved to 001_init.up.sql
-- This migration is now a no-op for backward compatibility

-- Ensure column exists (in case it's missing)
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 
        FROM information_schema.columns 
        WHERE table_name='trading_pairs' 
        AND column_name='strategy_id'
    ) THEN
        ALTER TABLE trading_pairs
        ADD COLUMN strategy_id UUID REFERENCES user_strategies(id) ON DELETE CASCADE;
    END IF;
END $$;

-- Ensure index exists (in case it's missing)
CREATE INDEX IF NOT EXISTS idx_trading_pairs_strategy ON trading_pairs (strategy_id);

-- Add comment
COMMENT ON COLUMN trading_pairs.strategy_id IS 'Link to user_strategies table for strategy-level portfolio management';

