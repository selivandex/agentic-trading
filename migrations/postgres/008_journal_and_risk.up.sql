-- Journal Entries, Circuit Breaker, and Risk Events tables

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

-- Trigger for circuit breaker updated_at
CREATE TRIGGER circuit_breaker_states_updated_at BEFORE UPDATE ON circuit_breaker_states
FOR EACH ROW EXECUTE FUNCTION update_updated_at();

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
