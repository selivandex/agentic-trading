-- Drop indexes
DROP INDEX IF EXISTS idx_strategy_transactions_user_created;
DROP INDEX IF EXISTS idx_strategy_transactions_position_id;
DROP INDEX IF EXISTS idx_strategy_transactions_created_at;
DROP INDEX IF EXISTS idx_strategy_transactions_type;
DROP INDEX IF EXISTS idx_strategy_transactions_user_id;
DROP INDEX IF EXISTS idx_strategy_transactions_strategy_id;

-- Drop table
DROP TABLE IF EXISTS strategy_transactions;

-- Drop enum type
DROP TYPE IF EXISTS strategy_transaction_type;


