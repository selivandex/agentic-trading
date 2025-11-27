-- Drop trigger and function
DROP TRIGGER IF EXISTS trigger_update_user_strategies_updated_at ON user_strategies;
DROP FUNCTION IF EXISTS update_user_strategies_updated_at();

-- Drop indexes
DROP INDEX IF EXISTS idx_positions_strategy_id;
DROP INDEX IF EXISTS idx_user_strategies_created_at;
DROP INDEX IF EXISTS idx_user_strategies_status;
DROP INDEX IF EXISTS idx_user_strategies_user_id;

-- Remove strategy_id column from positions
ALTER TABLE positions DROP COLUMN IF EXISTS strategy_id;

-- Drop user_strategies table
DROP TABLE IF EXISTS user_strategies;



