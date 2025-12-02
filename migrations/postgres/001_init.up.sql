-- migrations/postgres/001_init.up.sql
-- Initial schema setup with ENUM types

-- Enable extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "vector";

-- Create ENUM types for type safety and low cardinality columns
CREATE TYPE exchange_type AS ENUM ('binance', 'bybit', 'okx', 'kucoin', 'gate');
CREATE TYPE market_type AS ENUM ('spot', 'futures');
CREATE TYPE order_side AS ENUM ('buy', 'sell');
CREATE TYPE order_type AS ENUM ('market', 'limit', 'stop_market', 'stop_limit');
CREATE TYPE order_status AS ENUM ('pending', 'open', 'filled', 'partial', 'canceled', 'rejected', 'expired');
CREATE TYPE position_side AS ENUM ('long', 'short');
CREATE TYPE position_status AS ENUM ('open', 'closed', 'liquidated');
CREATE TYPE memory_type AS ENUM ('observation', 'decision', 'trade', 'lesson', 'regime', 'pattern');
CREATE TYPE risk_event_type AS ENUM ('drawdown_warning', 'consecutive_loss', 'circuit_breaker_triggered', 'max_exposure_reached', 'kill_switch_activated', 'anomaly_detected');
CREATE TYPE strategy_mode AS ENUM ('auto', 'semi_auto', 'signals');
CREATE TYPE strategy_status AS ENUM ('active', 'paused', 'closed');
CREATE TYPE risk_tolerance AS ENUM ('conservative', 'moderate', 'aggressive');
CREATE TYPE rebalance_frequency AS ENUM ('daily', 'weekly', 'monthly', 'never');

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
    exchange exchange_type NOT NULL,
    label VARCHAR(255),
    api_key_encrypted BYTEA NOT NULL,
    secret_encrypted BYTEA NOT NULL,
    passphrase BYTEA,
    is_testnet BOOLEAN DEFAULT false,
    permissions TEXT[] DEFAULT '{}',
    is_active BOOLEAN DEFAULT true,
    last_sync_at TIMESTAMPTZ,
    
    -- User Data WebSocket fields (for real-time order/position updates)
    listen_key_encrypted BYTEA,
    listen_key_expires_at TIMESTAMPTZ,
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),

    CONSTRAINT unique_user_exchange_label UNIQUE(user_id, exchange, label)
);

CREATE INDEX idx_exchange_accounts_user ON exchange_accounts(user_id);
CREATE INDEX idx_exchange_accounts_active ON exchange_accounts(is_active) WHERE is_active = true;
CREATE INDEX idx_exchange_accounts_listen_key_expires ON exchange_accounts(listen_key_expires_at) 
    WHERE listen_key_expires_at IS NOT NULL AND is_active = true;

-- Fund Watchlist table (global symbols monitored by the fund)
CREATE TABLE fund_watchlist (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    symbol VARCHAR(50) NOT NULL UNIQUE,
    market_type market_type NOT NULL,
    
    -- Metadata
    category VARCHAR(50),  -- "major", "defi", "layer1", etc
    tier INTEGER DEFAULT 3,  -- 1=BTC/ETH, 2=top20, 3=top100
    
    -- State
    is_active BOOLEAN DEFAULT true,
    is_paused BOOLEAN DEFAULT false,
    paused_reason TEXT,
    
    -- Analytics
    last_analyzed_at TIMESTAMPTZ,
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    
    CONSTRAINT unique_symbol_market UNIQUE(symbol, market_type)
);

CREATE INDEX idx_fund_watchlist_active ON fund_watchlist(is_active, is_paused);
CREATE INDEX idx_fund_watchlist_symbol ON fund_watchlist(symbol);

-- Insert default watchlist
INSERT INTO fund_watchlist (symbol, market_type, category, tier) VALUES
    ('BTC/USDT', 'spot', 'major', 1),
    ('ETH/USDT', 'spot', 'major', 1),
    ('BNB/USDT', 'spot', 'exchange', 2),
    ('SOL/USDT', 'spot', 'layer1', 2),
    ('XRP/USDT', 'spot', 'major', 2),
    ('ADA/USDT', 'spot', 'layer1', 2),
    ('AVAX/USDT', 'spot', 'layer1', 2),
    ('DOT/USDT', 'spot', 'layer0', 2),
    ('MATIC/USDT', 'spot', 'layer2', 2),
    ('LINK/USDT', 'spot', 'oracle', 2);

-- User Strategies table (portfolio management)
CREATE TABLE user_strategies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    
    -- Strategy metadata
    name VARCHAR(255) NOT NULL,
    description TEXT,
    status strategy_status NOT NULL DEFAULT 'active',
    
    -- Capital allocation
    allocated_capital DECIMAL(20, 8) NOT NULL,
    current_equity DECIMAL(20, 8) NOT NULL,
    cash_reserve DECIMAL(20, 8) NOT NULL DEFAULT 0,
    
    -- Strategy configuration
    market_type market_type NOT NULL DEFAULT 'spot',
    risk_tolerance risk_tolerance NOT NULL,
    rebalance_frequency rebalance_frequency,
    target_allocations JSONB NOT NULL,
    
    -- Performance metrics
    total_pnl DECIMAL(20, 8) DEFAULT 0,
    total_pnl_percent DECIMAL(10, 4) DEFAULT 0,
    sharpe_ratio DECIMAL(10, 4),
    max_drawdown DECIMAL(10, 4),
    win_rate DECIMAL(10, 4),
    
    -- Timestamps
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    closed_at TIMESTAMP,
    last_rebalanced_at TIMESTAMP,
    
    -- Reasoning (for explainability)
    reasoning_log JSONB,
    
    -- Constraints
    CONSTRAINT positive_capital CHECK (allocated_capital > 0),
    CONSTRAINT valid_equity CHECK (current_equity >= 0),
    CONSTRAINT valid_cash CHECK (cash_reserve >= 0)
);

CREATE INDEX idx_user_strategies_user_id ON user_strategies(user_id);
CREATE INDEX idx_user_strategies_status ON user_strategies(status);
CREATE INDEX idx_user_strategies_created_at ON user_strategies(created_at DESC);

-- Orders table
CREATE TABLE orders (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id),
    strategy_id UUID REFERENCES user_strategies(id) ON DELETE SET NULL,
    exchange_account_id UUID NOT NULL REFERENCES exchange_accounts(id),

    exchange_order_id VARCHAR(255),
    symbol VARCHAR(50) NOT NULL,
    market_type market_type NOT NULL,

    side order_side NOT NULL,
    type order_type NOT NULL,
    status order_status NOT NULL DEFAULT 'pending',

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
CREATE INDEX idx_orders_symbol ON orders(symbol);
CREATE INDEX idx_orders_created ON orders(created_at DESC);

-- Positions table
CREATE TABLE positions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id),
    strategy_id UUID REFERENCES user_strategies(id) ON DELETE SET NULL,
    exchange_account_id UUID NOT NULL REFERENCES exchange_accounts(id),

    symbol VARCHAR(50) NOT NULL,
    market_type market_type NOT NULL,
    side position_side NOT NULL,

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

    status position_status NOT NULL DEFAULT 'open',
    opened_at TIMESTAMPTZ DEFAULT NOW(),
    closed_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_positions_user ON positions(user_id);
CREATE INDEX idx_positions_status ON positions(status);
CREATE INDEX idx_positions_symbol ON positions(symbol);
CREATE INDEX idx_positions_strategy ON positions(strategy_id);

-- Agent Memory tables moved to 005_agent_memories.up.sql (unified structure)

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

    event_type risk_event_type NOT NULL,
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

CREATE TRIGGER orders_updated_at BEFORE UPDATE ON orders
FOR EACH ROW EXECUTE FUNCTION update_updated_at();

CREATE TRIGGER positions_updated_at BEFORE UPDATE ON positions
FOR EACH ROW EXECUTE FUNCTION update_updated_at();

CREATE TRIGGER circuit_breaker_states_updated_at BEFORE UPDATE ON circuit_breaker_states
FOR EACH ROW EXECUTE FUNCTION update_updated_at();

