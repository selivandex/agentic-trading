-- Portfolio Decisions - Agent reasoning history per asset
-- Stores BOTH:
-- 1. Final decisions by PortfolioManager (action=buy/sell/hold, parent_decision_id=NULL)
-- 2. Expert agent analyses that informed those decisions (action=analysis, parent_decision_id=decision.id)
CREATE TABLE
  portfolio_decisions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid (),
    -- Links
    strategy_id UUID NOT NULL REFERENCES user_strategies (id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    session_id VARCHAR(255), -- ADK session ID for full trace
    -- Agent context
    agent_id INTEGER NOT NULL REFERENCES agents (id), -- Link to agents table
    agent_session_id VARCHAR(255), -- Full ADK session for this decision
    -- Hierarchy (expert analyses link to parent decision)
    parent_decision_id UUID REFERENCES portfolio_decisions (id) ON DELETE CASCADE, -- NULL for main decisions, set for expert analyses
    -- Asset & Decision
    symbol VARCHAR(50) NOT NULL, -- BTC/USDT, ETH/USDT, SOL/USDT
    action VARCHAR(20) NOT NULL, -- buy, sell, hold, rebalance, close, analysis
    decision_reason TEXT NOT NULL, -- Human-readable reasoning
    -- Market context at decision time
    market_regime VARCHAR(50), -- bull_trending, bear, ranging, high_volatility
    market_price DECIMAL(20, 8), -- Price at decision time
    -- Position context (NULL for expert analyses, filled for PortfolioManager decisions)
    position_before DECIMAL(20, 8), -- Amount before decision
    position_after DECIMAL(20, 8), -- Amount after decision
    allocation_before DECIMAL(10, 4), -- % of portfolio before (0.35 = 35%)
    allocation_after DECIMAL(10, 4), -- % of portfolio after
    -- Decision metadata
    confidence_level DECIMAL(5, 4), -- 0.0 to 1.0 (agent's confidence)
    risk_score DECIMAL(5, 4), -- Risk assessment for this decision
    expected_return DECIMAL(10, 4), -- Expected return %
    -- Technical indicators (for expert agent analyses)
    timeframe VARCHAR(10), -- 1h, 4h, 1D (for TechnicalAnalyzer, SMCAnalyzer)
    indicators_used JSONB, -- {"rsi": 62, "macd": "bullish", "ema": "above"}
    -- Related entities
    order_id UUID REFERENCES orders (id), -- If action resulted in order (PortfolioManager only)
    -- Timestamps
    decision_timestamp TIMESTAMP NOT NULL DEFAULT NOW (),
    created_at TIMESTAMP NOT NULL DEFAULT NOW (),
    -- Indexes for fast queries
    INDEX idx_portfolio_decisions_strategy (strategy_id, decision_timestamp DESC),
    INDEX idx_portfolio_decisions_symbol (symbol, decision_timestamp DESC),
    INDEX idx_portfolio_decisions_agent (agent_id, decision_timestamp DESC),
    INDEX idx_portfolio_decisions_action (action, decision_timestamp DESC),
    INDEX idx_portfolio_decisions_parent (parent_decision_id) -- Find expert analyses for a decision
  );

-- Permissions
COMMENT ON TABLE portfolio_decisions IS 'Unified agent reasoning - stores both PortfolioManager decisions AND expert agent analyses';

COMMENT ON COLUMN portfolio_decisions.parent_decision_id IS 'NULL for PortfolioManager decisions, set to decision.id for expert analyses that informed it';

COMMENT ON COLUMN portfolio_decisions.agent_id IS 'References agents table - PortfolioManager for final decisions, expert agents for analyses';

COMMENT ON COLUMN portfolio_decisions.action IS 'buy/sell/hold/rebalance for PortfolioManager, "analysis" for expert agents';

COMMENT ON COLUMN portfolio_decisions.decision_reason IS 'Human-readable: "Buying BTC..." (PortfolioManager) or "RSI 62, uptrend..." (TechnicalAnalyzer)';

COMMENT ON COLUMN portfolio_decisions.indicators_used IS 'Technical indicators for expert analyses: {"rsi": 62, "macd": "bullish"}';

COMMENT ON COLUMN portfolio_decisions.timeframe IS 'Chart timeframe for technical analyses: 1h, 4h, 1D';
