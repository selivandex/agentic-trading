-- migrations/postgres/006_performance_indexes.down.sql
-- Rollback performance indexes

-- Orders table
DROP INDEX IF EXISTS idx_orders_user_status;
DROP INDEX IF EXISTS idx_orders_exchange_id;
DROP INDEX IF EXISTS idx_orders_trading_pair;

-- Positions table
DROP INDEX IF EXISTS idx_positions_user_open;
DROP INDEX IF EXISTS idx_positions_symbol_open;
DROP INDEX IF EXISTS idx_positions_trading_pair;

-- Memories table
DROP INDEX IF EXISTS idx_memories_embedding_gist;
DROP INDEX IF EXISTS idx_memories_user_type;
DROP INDEX IF EXISTS idx_memories_session;

-- Collective memory
DROP INDEX IF EXISTS idx_collective_memory_pattern;
DROP INDEX IF EXISTS idx_collective_memory_embedding;

-- Journal entries
DROP INDEX IF EXISTS idx_journal_user_time;
DROP INDEX IF EXISTS idx_journal_strategy;
DROP INDEX IF EXISTS idx_journal_pnl;

-- Circuit breaker state
DROP INDEX IF EXISTS idx_circuit_breaker_user;
DROP INDEX IF EXISTS idx_circuit_breaker_triggered;

-- Risk events
DROP INDEX IF EXISTS idx_risk_events_user;
DROP INDEX IF EXISTS idx_risk_events_unacknowledged;

-- Trading pairs
DROP INDEX IF EXISTS idx_trading_pairs_user_active;
DROP INDEX IF EXISTS idx_trading_pairs_not_paused;

-- Exchange accounts
DROP INDEX IF EXISTS idx_exchange_accounts_active;

-- Agent reasoning logs
DROP INDEX IF EXISTS idx_reasoning_user_time;
DROP INDEX IF EXISTS idx_reasoning_agent;
DROP INDEX IF EXISTS idx_reasoning_cost;

