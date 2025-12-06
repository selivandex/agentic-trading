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

-- Indexes
CREATE INDEX idx_user_strategies_user_id ON user_strategies(user_id);
CREATE INDEX idx_user_strategies_status ON user_strategies(status);
CREATE INDEX idx_user_strategies_created_at ON user_strategies(created_at DESC);

-- Trigger for updated_at
CREATE OR REPLACE FUNCTION update_user_strategies_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_update_user_strategies_updated_at
    BEFORE UPDATE ON user_strategies
    FOR EACH ROW
    EXECUTE FUNCTION update_user_strategies_updated_at();

-- Comments
COMMENT ON TABLE user_strategies IS 'Portfolio strategies created during onboarding or rebalancing';
COMMENT ON COLUMN user_strategies.target_allocations IS 'Target allocation percentages as JSON: {"BTC/USDT": 0.5, "ETH/USDT": 0.3}';
COMMENT ON COLUMN user_strategies.reasoning_log IS 'AI reasoning chain-of-thought for strategy creation (explainability)';
