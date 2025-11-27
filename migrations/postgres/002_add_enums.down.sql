-- migrations/postgres/002_add_enums.down.sql
-- Rollback ENUMs to VARCHAR
-- Revert columns back to VARCHAR
ALTER TABLE risk_events
ALTER COLUMN event_type TYPE VARCHAR(50);

ALTER TABLE collective_memories
ALTER COLUMN type TYPE VARCHAR(50);

ALTER TABLE memories
ALTER COLUMN type TYPE VARCHAR(50);

ALTER TABLE positions
ALTER COLUMN side TYPE VARCHAR(10),
ALTER COLUMN status TYPE VARCHAR(20),
ALTER COLUMN market_type TYPE VARCHAR(20);

ALTER TABLE orders
ALTER COLUMN side TYPE VARCHAR(10),
ALTER COLUMN type TYPE VARCHAR(20),
ALTER COLUMN status TYPE VARCHAR(20),
ALTER COLUMN market_type TYPE VARCHAR(20);

ALTER TABLE trading_pairs
ALTER COLUMN market_type TYPE VARCHAR(20);

-- Handle strategy_mode separately to restore default
ALTER TABLE trading_pairs
ALTER COLUMN strategy_mode TYPE VARCHAR(20);

ALTER TABLE trading_pairs
ALTER COLUMN strategy_mode SET DEFAULT 'signals';

ALTER TABLE exchange_accounts
ALTER COLUMN exchange TYPE VARCHAR(50);

-- Drop ENUM types
DROP TYPE IF EXISTS strategy_mode;

DROP TYPE IF EXISTS risk_event_type;

DROP TYPE IF EXISTS memory_type;

DROP TYPE IF EXISTS position_status;

DROP TYPE IF EXISTS position_side;

DROP TYPE IF EXISTS order_status;

DROP TYPE IF EXISTS order_type;

DROP TYPE IF EXISTS order_side;

DROP TYPE IF EXISTS market_type;

DROP TYPE IF EXISTS exchange_type;
