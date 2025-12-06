-- Rollback extensions and ENUM types
DROP TYPE IF EXISTS strategy_transaction_type;

DROP TYPE IF EXISTS memory_scope;

DROP TYPE IF EXISTS rebalance_frequency;

DROP TYPE IF EXISTS risk_tolerance;

DROP TYPE IF EXISTS strategy_status;

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

DROP EXTENSION IF EXISTS "vector";

DROP EXTENSION IF EXISTS "uuid-ossp";
