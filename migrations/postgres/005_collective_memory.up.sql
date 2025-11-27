-- Create collective memories table for shared knowledge across users
CREATE TABLE IF NOT EXISTS collective_memories (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

    -- Scope
    agent_type VARCHAR(100) NOT NULL,  -- 'market_analyst', 'risk_manager', etc.
    personality VARCHAR(50),            -- 'conservative', 'aggressive', 'balanced', NULL for all

    -- Content  
    type memory_type NOT NULL,
    content TEXT NOT NULL,
    embedding vector(1536),            -- OpenAI ada-002 or similar

    -- Validation (promoted only when proven)
    validation_score DECIMAL(5,2),     -- 0-100, how well this lesson performed
    validation_trades INTEGER DEFAULT 0, -- Number of trades that validated this
    validated_at TIMESTAMPTZ,

    -- Metadata
    symbol VARCHAR(50),                 -- NULL for general lessons
    timeframe VARCHAR(10),             -- NULL for general lessons
    importance DECIMAL(3,2) DEFAULT 0.5, -- 0-1, for retrieval ranking
    tags TEXT[],                       -- Additional categorization

    -- Source (anonymized)
    source_user_id UUID REFERENCES users(id) ON DELETE SET NULL, -- Who contributed this
    source_trade_id UUID,              -- Which trade led to this lesson

    -- Lifecycle
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    expires_at TIMESTAMPTZ             -- TTL for time-sensitive lessons
);

-- Indexes for efficient querying
CREATE INDEX idx_collective_memories_agent ON collective_memories(agent_type);
CREATE INDEX idx_collective_memories_personality ON collective_memories(personality) WHERE personality IS NOT NULL;
CREATE INDEX idx_collective_memories_type ON collective_memories(type);
CREATE INDEX idx_collective_memories_validated ON collective_memories(validation_score DESC, validation_trades DESC) WHERE validated_at IS NOT NULL;
CREATE INDEX idx_collective_memories_symbol ON collective_memories(symbol) WHERE symbol IS NOT NULL;
CREATE INDEX idx_collective_memories_embedding ON collective_memories USING ivfflat (embedding vector_cosine_ops);
CREATE INDEX idx_collective_memories_importance ON collective_memories(importance DESC);
CREATE INDEX idx_collective_memories_created ON collective_memories(created_at DESC);

-- Trigger for updated_at
CREATE TRIGGER collective_memories_updated_at BEFORE UPDATE ON collective_memories
FOR EACH ROW EXECUTE FUNCTION update_updated_at();

-- Comments
COMMENT ON TABLE collective_memories IS 'Shared knowledge base validated across multiple users';
COMMENT ON COLUMN collective_memories.agent_type IS 'Which agent type this knowledge is for';
COMMENT ON COLUMN collective_memories.personality IS 'Risk personality this applies to (NULL = all)';
COMMENT ON COLUMN collective_memories.validation_score IS 'Performance score 0-100 from actual trades';
COMMENT ON COLUMN collective_memories.validation_trades IS 'Number of trades that confirmed this pattern';
COMMENT ON COLUMN collective_memories.source_user_id IS 'Original contributor (anonymized after validation)';



