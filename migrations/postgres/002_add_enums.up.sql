-- migrations/postgres/002_add_enums.up.sql
-- Add PostgreSQL ENUM types for type safety

-- Exchange types
CREATE TYPE exchange_type AS ENUM ('binance', 'bybit', 'okx', 'kucoin', 'gate');

-- Market types
CREATE TYPE market_type AS ENUM ('spot', 'futures');

-- Order sides
CREATE TYPE order_side AS ENUM ('buy', 'sell');

-- Order types
CREATE TYPE order_type AS ENUM ('market', 'limit', 'stop_market', 'stop_limit');

-- Order statuses
CREATE TYPE order_status AS ENUM (
    'pending',
    'open',
    'filled',
    'partial',
    'canceled',
    'rejected',
    'expired'
);

-- Position sides
CREATE TYPE position_side AS ENUM ('long', 'short');

-- Position statuses
CREATE TYPE position_status AS ENUM ('open', 'closed', 'liquidated');

-- Memory types
CREATE TYPE memory_type AS ENUM (
    'observation',
    'decision',
    'trade',
    'lesson',
    'regime',
    'pattern'
);

-- Risk event types
CREATE TYPE risk_event_type AS ENUM (
    'drawdown_warning',
    'consecutive_loss',
    'circuit_breaker_triggered',
    'max_exposure_reached',
    'kill_switch_activated',
    'anomaly_detected'
);

-- Strategy modes
CREATE TYPE strategy_mode AS ENUM ('auto', 'semi_auto', 'signals');

-- Update existing tables to use ENUMs
ALTER TABLE exchange_accounts
    ALTER COLUMN exchange TYPE exchange_type USING exchange::exchange_type;

ALTER TABLE trading_pairs
    ALTER COLUMN market_type TYPE market_type USING market_type::market_type,
    ALTER COLUMN strategy_mode TYPE strategy_mode USING strategy_mode::strategy_mode;

ALTER TABLE orders
    ALTER COLUMN side TYPE order_side USING side::order_side,
    ALTER COLUMN type TYPE order_type USING type::order_type,
    ALTER COLUMN status TYPE order_status USING status::order_status,
    ALTER COLUMN market_type TYPE market_type USING market_type::market_type;

ALTER TABLE positions
    ALTER COLUMN side TYPE position_side USING side::position_side,
    ALTER COLUMN status TYPE position_status USING status::position_status,
    ALTER COLUMN market_type TYPE market_type USING market_type::market_type;

ALTER TABLE memories
    ALTER COLUMN type TYPE memory_type USING type::memory_type;

ALTER TABLE collective_memories
    ALTER COLUMN type TYPE memory_type USING type::memory_type;

ALTER TABLE risk_events
    ALTER COLUMN event_type TYPE risk_event_type USING event_type::risk_event_type;

