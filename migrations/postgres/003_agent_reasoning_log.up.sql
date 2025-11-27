-- migrations/postgres/003_agent_reasoning_log.up.sql
-- Chain-of-Thought reasoning logging
CREATE TABLE
  agent_reasoning_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4 (),
    user_id UUID REFERENCES users (id) ON DELETE CASCADE,
    agent_id VARCHAR(100) NOT NULL,
    agent_type VARCHAR(100) NOT NULL,
    session_id VARCHAR(255),
    -- Context
    symbol VARCHAR(50),
    trading_pair_id UUID REFERENCES trading_pairs (id),
    -- Reasoning steps (array of step objects)
    reasoning_steps JSONB NOT NULL,
    /*
    Example structure:
    [
    {
    "step": 1,
    "action": "tool_call",
    "tool": "get_price",
    "input": {"symbol": "BTC/USDT"},
    "output": {"price": 45000},
    "timestamp": "2025-11-25T10:00:00Z"
    },
    {
    "step": 2,
    "action": "thinking",
    "content": "Price is above 200 EMA, bullish signal",
    "timestamp": "2025-11-25T10:00:01Z"
    },
    {
    "step": 3,
    "action": "decision",
    "content": "{\"signal\": \"buy\", \"confidence\": 80}",
    "timestamp": "2025-11-25T10:00:05Z"
    }
    ]
     */
    -- Final decision
    decision JSONB,
    confidence DECIMAL(5, 2),
    -- Performance metrics
    tokens_used INTEGER,
    cost_usd DECIMAL(10, 6),
    duration_ms INTEGER,
    tool_calls_count INTEGER,
    created_at TIMESTAMPTZ DEFAULT NOW ()
  );

CREATE INDEX idx_reasoning_logs_user ON agent_reasoning_logs (user_id);

CREATE INDEX idx_reasoning_logs_agent ON agent_reasoning_logs (agent_id);

CREATE INDEX idx_reasoning_logs_session ON agent_reasoning_logs (session_id);

CREATE INDEX idx_reasoning_logs_symbol ON agent_reasoning_logs (symbol);

CREATE INDEX idx_reasoning_logs_created ON agent_reasoning_logs (created_at DESC);

CREATE INDEX idx_reasoning_logs_confidence ON agent_reasoning_logs (confidence DESC);
