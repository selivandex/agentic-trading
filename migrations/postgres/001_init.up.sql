-- migrations/postgres/001_init.up.sql
-- Initial schema setup

-- Enable extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "vector";

-- Users table
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    telegram_id BIGINT UNIQUE NOT NULL,
    telegram_username VARCHAR(255),
    first_name VARCHAR(255),
    last_name VARCHAR(255),
    language_code VARCHAR(10) DEFAULT 'en',
    is_active BOOLEAN DEFAULT true,
    is_premium BOOLEAN DEFAULT false,
    settings JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_users_telegram_id ON users(telegram_id);
CREATE INDEX idx_users_is_active ON users(is_active) WHERE is_active = true;

-- Exchange Accounts table
CREATE TABLE exchange_accounts (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    exchange VARCHAR(50) NOT NULL,
    label VARCHAR(255),
    api_key_encrypted BYTEA NOT NULL,
    secret_encrypted BYTEA NOT NULL,
    passphrase BYTEA,
    is_testnet BOOLEAN DEFAULT false,
    permissions TEXT[] DEFAULT '{}',
    is_active BOOLEAN DEFAULT true,
    last_sync_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),

    CONSTRAINT unique_user_exchange_label UNIQUE(user_id, exchange, label)
);

CREATE INDEX idx_exchange_accounts_user ON exchange_accounts(user_id);
CREATE INDEX idx_exchange_accounts_active ON exchange_accounts(is_active) WHERE is_active = true;

-- Trading Pairs table
CREATE TABLE trading_pairs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    exchange_account_id UUID NOT NULL REFERENCES exchange_accounts(id) ON DELETE CASCADE,
    symbol VARCHAR(50) NOT NULL,
    market_type VARCHAR(20) NOT NULL,

    -- Budget & Risk
    budget DECIMAL(20,8) NOT NULL,
    max_position_size DECIMAL(20,8),
    max_leverage INTEGER DEFAULT 1,
    stop_loss_percent DECIMAL(5,2) DEFAULT 2.0,
    take_profit_percent DECIMAL(5,2) DEFAULT 4.0,

    -- Strategy
    ai_provider VARCHAR(50) DEFAULT 'claude',
    strategy_mode VARCHAR(20) DEFAULT 'signals',
    timeframes TEXT[] DEFAULT '{"1h","4h","1d"}',

    -- State
    is_active BOOLEAN DEFAULT true,
    is_paused BOOLEAN DEFAULT false,
    paused_reason TEXT,

    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),

    CONSTRAINT unique_pair_per_account UNIQUE(exchange_account_id, symbol, market_type)
);

CREATE INDEX idx_trading_pairs_user ON trading_pairs(user_id);
CREATE INDEX idx_trading_pairs_active ON trading_pairs(is_active, is_paused);
CREATE INDEX idx_trading_pairs_exchange_account ON trading_pairs(exchange_account_id);

-- Orders table
CREATE TABLE orders (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id),
    trading_pair_id UUID REFERENCES trading_pairs(id),
    exchange_account_id UUID NOT NULL REFERENCES exchange_accounts(id),

    exchange_order_id VARCHAR(255),
    symbol VARCHAR(50) NOT NULL,
    market_type VARCHAR(20) NOT NULL,

    side VARCHAR(10) NOT NULL,
    type VARCHAR(20) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',

    price DECIMAL(20,8),
    amount DECIMAL(20,8) NOT NULL,
    filled_amount DECIMAL(20,8) DEFAULT 0,
    avg_fill_price DECIMAL(20,8),

    stop_price DECIMAL(20,8),
    reduce_only BOOLEAN DEFAULT false,

    agent_id VARCHAR(100),
    reasoning TEXT,
    parent_order_id UUID REFERENCES orders(id),

    fee DECIMAL(20,8) DEFAULT 0,
    fee_currency VARCHAR(20),

    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    filled_at TIMESTAMPTZ
);

CREATE INDEX idx_orders_user ON orders(user_id);
CREATE INDEX idx_orders_status ON orders(status);
CREATE INDEX idx_orders_exchange ON orders(exchange_order_id);
CREATE INDEX idx_orders_trading_pair ON orders(trading_pair_id);
CREATE INDEX idx_orders_created ON orders(created_at DESC);

-- Positions table
CREATE TABLE positions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id),
    trading_pair_id UUID REFERENCES trading_pairs(id),
    exchange_account_id UUID NOT NULL REFERENCES exchange_accounts(id),

    symbol VARCHAR(50) NOT NULL,
    market_type VARCHAR(20) NOT NULL,
    side VARCHAR(10) NOT NULL,

    size DECIMAL(20,8) NOT NULL,
    entry_price DECIMAL(20,8) NOT NULL,
    current_price DECIMAL(20,8),
    liquidation_price DECIMAL(20,8),

    leverage INTEGER DEFAULT 1,
    margin_mode VARCHAR(20) DEFAULT 'cross',

    unrealized_pnl DECIMAL(20,8) DEFAULT 0,
    unrealized_pnl_pct DECIMAL(10,4) DEFAULT 0,
    realized_pnl DECIMAL(20,8) DEFAULT 0,

    stop_loss_price DECIMAL(20,8),
    take_profit_price DECIMAL(20,8),
    trailing_stop_pct DECIMAL(5,2),

    stop_loss_order_id UUID REFERENCES orders(id),
    take_profit_order_id UUID REFERENCES orders(id),

    open_reasoning TEXT,

    status VARCHAR(20) NOT NULL DEFAULT 'open',
    opened_at TIMESTAMPTZ DEFAULT NOW(),
    closed_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_positions_user ON positions(user_id);
CREATE INDEX idx_positions_status ON positions(status);
CREATE INDEX idx_positions_symbol ON positions(symbol);
CREATE INDEX idx_positions_trading_pair ON positions(trading_pair_id);

-- Agent Memory table (with vector embeddings)
CREATE TABLE memories (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    agent_id VARCHAR(100) NOT NULL,
    session_id VARCHAR(255),

    type VARCHAR(50) NOT NULL,
    content TEXT NOT NULL,
    embedding vector(1536), -- OpenAI ada-002 / text-embedding-3-small dimension

    symbol VARCHAR(50),
    timeframe VARCHAR(10),
    importance DECIMAL(3,2) DEFAULT 0.5,

    related_ids UUID[] DEFAULT '{}',
    trade_id UUID REFERENCES orders(id),

    created_at TIMESTAMPTZ DEFAULT NOW(),
    expires_at TIMESTAMPTZ
);

CREATE INDEX idx_memories_user ON memories(user_id);
CREATE INDEX idx_memories_agent ON memories(agent_id);
CREATE INDEX idx_memories_type ON memories(type);
CREATE INDEX idx_memories_embedding ON memories USING ivfflat (embedding vector_cosine_ops) WITH (lists = 100);
CREATE INDEX idx_memories_created ON memories(created_at DESC);

-- Collective Memory table (shared knowledge)
CREATE TABLE collective_memories (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

    -- Scope
    agent_type VARCHAR(100) NOT NULL,
    personality VARCHAR(50),

    -- Content
    type VARCHAR(50) NOT NULL,
    content TEXT NOT NULL,
    embedding vector(1536),

    -- Validation
    validation_score DECIMAL(3,2),
    validation_trades INTEGER,
    validated_at TIMESTAMPTZ,

    -- Metadata
    symbol VARCHAR(50),
    timeframe VARCHAR(10),
    importance DECIMAL(3,2) DEFAULT 0.5,

    -- Source
    source_user_id UUID REFERENCES users(id),
    source_trade_id UUID,

    created_at TIMESTAMPTZ DEFAULT NOW(),
    expires_at TIMESTAMPTZ
);

CREATE INDEX idx_collective_memories_agent ON collective_memories(agent_type);
CREATE INDEX idx_collective_memories_personality ON collective_memories(personality);
CREATE INDEX idx_collective_memories_embedding ON collective_memories USING ivfflat (embedding vector_cosine_ops) WITH (lists = 100);
CREATE INDEX idx_collective_memories_validated ON collective_memories(validation_score DESC) WHERE validated_at IS NOT NULL;

-- Journal Entries table
CREATE TABLE journal_entries (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    trade_id UUID NOT NULL,

    -- Trade details
    symbol VARCHAR(50) NOT NULL,
    side VARCHAR(10) NOT NULL,
    entry_price DECIMAL(20,8) NOT NULL,
    exit_price DECIMAL(20,8) NOT NULL,
    size DECIMAL(20,8) NOT NULL,
    pnl DECIMAL(20,8) NOT NULL,
    pnl_percent DECIMAL(10,4) NOT NULL,

    -- Strategy info
    strategy_used VARCHAR(100) NOT NULL,
    timeframe VARCHAR(10) NOT NULL,
    setup_type VARCHAR(50),

    -- Decision context
    market_regime VARCHAR(50),
    entry_reasoning TEXT,
    exit_reasoning TEXT,
    confidence_score DECIMAL(5,2),

    -- Indicators at entry
    rsi_at_entry DECIMAL(10,4),
    atr_at_entry DECIMAL(20,8),
    volume_at_entry DECIMAL(20,8),

    -- Outcome analysis
    was_correct_entry BOOLEAN,
    was_correct_exit BOOLEAN,
    max_drawdown DECIMAL(20,8),
    max_profit DECIMAL(20,8),
    hold_duration BIGINT, -- Duration in seconds

    -- Lessons learned (AI generated)
    lessons_learned TEXT,
    improvement_tips TEXT,

    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_journal_entries_user ON journal_entries(user_id);
CREATE INDEX idx_journal_entries_strategy ON journal_entries(strategy_used);
CREATE INDEX idx_journal_entries_created ON journal_entries(created_at DESC);
CREATE INDEX idx_journal_entries_symbol ON journal_entries(symbol);

-- Circuit Breaker State table
CREATE TABLE circuit_breaker_states (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,

    -- Current state
    is_triggered BOOLEAN DEFAULT false,
    triggered_at TIMESTAMPTZ,
    trigger_reason TEXT,

    -- Daily stats
    daily_pnl DECIMAL(20,8) DEFAULT 0,
    daily_pnl_percent DECIMAL(10,4) DEFAULT 0,
    daily_trade_count INTEGER DEFAULT 0,
    daily_wins INTEGER DEFAULT 0,
    daily_losses INTEGER DEFAULT 0,
    consecutive_losses INTEGER DEFAULT 0,

    -- Thresholds
    max_daily_drawdown DECIMAL(10,4) DEFAULT 5.0,
    max_consecutive_loss INTEGER DEFAULT 3,

    -- Auto-reset
    reset_at TIMESTAMPTZ DEFAULT (DATE_TRUNC('day', NOW()) + INTERVAL '1 day'),

    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_circuit_breaker_is_triggered ON circuit_breaker_states(is_triggered) WHERE is_triggered = true;

-- Risk Events table
CREATE TABLE risk_events (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    timestamp TIMESTAMPTZ DEFAULT NOW(),

    event_type VARCHAR(50) NOT NULL,
    severity VARCHAR(20) NOT NULL,
    message TEXT NOT NULL,
    data JSONB,

    acknowledged BOOLEAN DEFAULT false
);

CREATE INDEX idx_risk_events_user ON risk_events(user_id);
CREATE INDEX idx_risk_events_timestamp ON risk_events(timestamp DESC);
CREATE INDEX idx_risk_events_type ON risk_events(event_type);
CREATE INDEX idx_risk_events_acknowledged ON risk_events(acknowledged) WHERE acknowledged = false;

-- Function to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Triggers for updated_at
CREATE TRIGGER users_updated_at BEFORE UPDATE ON users
FOR EACH ROW EXECUTE FUNCTION update_updated_at();

CREATE TRIGGER exchange_accounts_updated_at BEFORE UPDATE ON exchange_accounts
FOR EACH ROW EXECUTE FUNCTION update_updated_at();

CREATE TRIGGER trading_pairs_updated_at BEFORE UPDATE ON trading_pairs
FOR EACH ROW EXECUTE FUNCTION update_updated_at();

CREATE TRIGGER orders_updated_at BEFORE UPDATE ON orders
FOR EACH ROW EXECUTE FUNCTION update_updated_at();

CREATE TRIGGER positions_updated_at BEFORE UPDATE ON positions
FOR EACH ROW EXECUTE FUNCTION update_updated_at();

CREATE TRIGGER circuit_breaker_states_updated_at BEFORE UPDATE ON circuit_breaker_states
FOR EACH ROW EXECUTE FUNCTION update_updated_at();

