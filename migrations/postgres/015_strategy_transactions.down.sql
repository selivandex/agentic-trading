-- Drop indexes
DROP INDEX IF EXISTS idx_strategy_transactions_user_created;

DROP INDEX IF EXISTS idx_strategy_transactions_position_id;

DROP INDEX IF EXISTS idx_strategy_transactions_created_at;

DROP INDEX IF EXISTS idx_strategy_transactions_type;

DROP INDEX IF EXISTS idx_strategy_transactions_user_id;

DROP INDEX IF EXISTS idx_strategy_transactions_strategy_id;

-- Drop table
-- Note: strategy_transaction_type ENUM is dropped in 001_extensions_and_enums.down.sql
DROP TABLE IF EXISTS strategy_transactions;
