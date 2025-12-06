-- migrations/postgres/006_performance_indexes.up.sql
-- Performance indexes for frequently queried tables
-- ==================================================
-- Orders table indexes
-- ==================================================
-- Index for querying open/partial orders by user
CREATE INDEX IF NOT EXISTS idx_orders_user_status ON orders (user_id, status)
WHERE
    status IN ('open', 'partial');

-- Index for looking up orders by exchange order ID
CREATE INDEX IF NOT EXISTS idx_orders_exchange_id ON orders (exchange_order_id)
WHERE
    exchange_order_id IS NOT NULL;

-- Index for querying orders by symbol
CREATE INDEX IF NOT EXISTS idx_orders_symbol_time ON orders (symbol, created_at DESC);

-- ==================================================
-- Positions table indexes
-- ==================================================
-- Index for querying open positions by user
CREATE INDEX IF NOT EXISTS idx_positions_user_open ON positions (user_id, status)
WHERE
    status = 'open';

-- Index for querying open positions by symbol
CREATE INDEX IF NOT EXISTS idx_positions_symbol_open ON positions (symbol, status)
WHERE
    status = 'open';

-- Index for querying positions by symbol and time
CREATE INDEX IF NOT EXISTS idx_positions_symbol_time ON positions (symbol, opened_at DESC);

-- ==================================================
-- Memories table indexes (pgvector semantic search)
-- ==================================================
-- IVFFlat index for cosine similarity search on embeddings (pgvector)
CREATE INDEX IF NOT EXISTS idx_memories_embedding_ivfflat ON memories USING ivfflat (embedding vector_cosine_ops)
WITH
    (lists = 100);

-- Index for querying memories by user and type
CREATE INDEX IF NOT EXISTS idx_memories_user_type ON memories (user_id, type, created_at DESC);

-- Index for querying by session
CREATE INDEX IF NOT EXISTS idx_memories_session ON memories (session_id, created_at DESC)
WHERE
    session_id IS NOT NULL;

-- ==================================================
-- Collective memory indexes moved to 005_agent_memories.up.sql (unified with user memories)
-- ==================================================
-- Journal entries indexes
-- ==================================================
-- Index for querying user's journal entries chronologically
CREATE INDEX IF NOT EXISTS idx_journal_user_time ON journal_entries (user_id, created_at DESC);

-- Index for analyzing strategy performance
CREATE INDEX IF NOT EXISTS idx_journal_strategy ON journal_entries (strategy_used, created_at DESC)
WHERE
    strategy_used IS NOT NULL;

-- Index for querying profitable/unprofitable trades
CREATE INDEX IF NOT EXISTS idx_journal_pnl ON journal_entries (user_id, pnl DESC)
WHERE
    pnl IS NOT NULL;

-- ==================================================
-- Circuit breaker states indexes
-- ==================================================
-- Index for fast user lookups (heavily queried in risk checks)
CREATE INDEX IF NOT EXISTS idx_circuit_breaker_states_user ON circuit_breaker_states (user_id);

-- Index for querying triggered circuit breakers
CREATE INDEX IF NOT EXISTS idx_circuit_breaker_states_triggered ON circuit_breaker_states (is_triggered, triggered_at DESC)
WHERE
    is_triggered = true;

-- ==================================================
-- Risk events indexes
-- ==================================================
-- Index for querying risk events by user
CREATE INDEX IF NOT EXISTS idx_risk_events_user ON risk_events (user_id, timestamp DESC);

-- Index for unacknowledged events
CREATE INDEX IF NOT EXISTS idx_risk_events_unacknowledged ON risk_events (user_id, event_type)
WHERE
    acknowledged = false;

-- ==================================================
-- Trading pairs indexes (table removed - trading_pairs no longer used)
-- ==================================================
-- Indexes removed as trading_pairs table was deprecated

-- ==================================================
-- Exchange accounts indexes
-- ==================================================
-- Index for active exchange accounts
CREATE INDEX IF NOT EXISTS idx_exchange_accounts_active ON exchange_accounts (user_id, is_active)
WHERE
    is_active = true;

-- ==================================================
-- Agent reasoning logs indexes (table removed - agent_reasoning_logs no longer used)
-- ==================================================
-- Indexes removed as agent_reasoning_logs table was deprecated
