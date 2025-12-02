-- Add strategy_id to trading_pairs table to link trading pairs with strategies
-- This creates proper hierarchy: User → Strategy → TradingPair → Position
ALTER TABLE trading_pairs
ADD COLUMN strategy_id UUID REFERENCES user_strategies (id);

-- Create index for faster queries by strategy
CREATE INDEX idx_trading_pairs_strategy ON trading_pairs (strategy_id);

-- Add comment
COMMENT ON COLUMN trading_pairs.strategy_id IS 'Link to user_strategies table - each trading pair belongs to a strategy';

-- Note: This is a nullable column for backward compatibility.
-- New trading pairs SHOULD have strategy_id set.
-- Consider adding NOT NULL constraint after data migration.
