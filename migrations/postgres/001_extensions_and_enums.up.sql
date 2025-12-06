-- Extensions and ENUM types
-- Enable extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE EXTENSION IF NOT EXISTS "vector";

-- Create ENUM types for type safety and low cardinality columns
CREATE TYPE exchange_type AS ENUM ('binance', 'bybit', 'okx', 'kucoin', 'gate');

CREATE TYPE market_type AS ENUM ('spot', 'futures');

CREATE TYPE order_side AS ENUM ('buy', 'sell');

CREATE TYPE order_type AS ENUM ('market', 'limit', 'stop_market', 'stop_limit');

CREATE TYPE order_status AS ENUM (
  'pending',
  'open',
  'filled',
  'partial',
  'canceled',
  'rejected',
  'expired'
);

CREATE TYPE position_side AS ENUM ('long', 'short');

CREATE TYPE position_status AS ENUM ('open', 'closed', 'liquidated');

CREATE TYPE memory_type AS ENUM (
  'observation',
  'decision',
  'trade',
  'lesson',
  'regime',
  'pattern'
);

CREATE TYPE risk_event_type AS ENUM (
  'drawdown_warning',
  'consecutive_loss',
  'circuit_breaker_triggered',
  'max_exposure_reached',
  'kill_switch_activated',
  'anomaly_detected'
);

CREATE TYPE strategy_mode AS ENUM ('auto', 'semi_auto', 'signals');

CREATE TYPE strategy_status AS ENUM ('active', 'paused', 'closed');

CREATE TYPE risk_tolerance AS ENUM ('conservative', 'moderate', 'aggressive');

CREATE TYPE rebalance_frequency AS ENUM ('daily', 'weekly', 'monthly', 'never');

CREATE TYPE memory_scope AS ENUM ('user', 'collective', 'working');

CREATE TYPE strategy_transaction_type AS ENUM (
  'deposit',
  'withdrawal',
  'position_open',
  'position_close',
  'fee',
  'funding_rate',
  'dividend',
  'interest',
  'adjustment'
);
