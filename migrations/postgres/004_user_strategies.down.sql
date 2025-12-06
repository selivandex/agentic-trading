-- Rollback user_strategies table
DROP TRIGGER IF EXISTS trigger_update_user_strategies_updated_at ON user_strategies;

DROP FUNCTION IF EXISTS update_user_strategies_updated_at ();

DROP TABLE IF EXISTS user_strategies;
