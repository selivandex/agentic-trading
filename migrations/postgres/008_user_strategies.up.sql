-- Migration 008: user_strategies table was moved to 001_init.up.sql
-- This migration now only handles additional indexes and trigger

-- Create trigger to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_user_strategies_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_update_user_strategies_updated_at
    BEFORE UPDATE ON user_strategies
    FOR EACH ROW
    EXECUTE FUNCTION update_user_strategies_updated_at();

-- Add comment for documentation
COMMENT ON TABLE user_strategies IS 'Portfolio strategies created during onboarding or rebalancing';
COMMENT ON COLUMN user_strategies.target_allocations IS 'Target allocation percentages as JSON: {"BTC/USDT": 0.5, "ETH/USDT": 0.3}';
COMMENT ON COLUMN user_strategies.reasoning_log IS 'AI reasoning chain-of-thought for strategy creation (explainability)';



