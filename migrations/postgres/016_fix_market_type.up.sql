-- Fix existing strategies with empty market_type
-- Set default value 'spot' for strategies where market_type is empty or null
-- Use text comparison to avoid enum validation error

UPDATE user_strategies
SET market_type = 'spot'::market_type
WHERE market_type::text = '' OR market_type IS NULL;
