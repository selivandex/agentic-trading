-- Add strategy_id to positions table to link positions with strategies
ALTER TABLE positions 
ADD COLUMN strategy_id UUID REFERENCES user_strategies(id);

-- Create index for faster queries by strategy
CREATE INDEX idx_positions_strategy ON positions(strategy_id);

-- Add comment
COMMENT ON COLUMN positions.strategy_id IS 'Link to user_strategies table for capital allocation and PnL tracking';

