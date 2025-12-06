-- Unified memory table for user, collective, and working memories
-- Note: memory_scope ENUM is created in 001_extensions_and_enums.up.sql

-- Create unified memories table
CREATE TABLE IF NOT EXISTS memories (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

    -- Scope determines memory level (user/collective/working)
    scope memory_scope NOT NULL,

    -- User-specific (NULL for collective/working)
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,

    -- Agent that created memory (required for user/working, optional for collective)
    agent_id VARCHAR(100),

    -- Session context (for user/working memories)
    session_id VARCHAR(100),

    -- Memory classification
    type memory_type NOT NULL,
    content TEXT NOT NULL,

    -- Vector embedding for semantic search
    embedding vector(1536),
    embedding_model VARCHAR(50),
    embedding_dimensions INTEGER,

    -- Validation metrics (for collective memories)
    validation_score DECIMAL(5,2),
    validation_trades INTEGER DEFAULT 0,
    validated_at TIMESTAMPTZ,

    -- Context metadata
    symbol VARCHAR(50),
    timeframe VARCHAR(10),
    importance DECIMAL(3,2) DEFAULT 0.5,
    tags TEXT[],

    -- Additional context (flexible JSON)
    metadata JSONB DEFAULT '{}'::jsonb,

    -- Personality filter (for collective memories)
    personality VARCHAR(50),

    -- Source tracking (for collective memories promoted from user memories)
    source_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    source_trade_id UUID,

    -- Lifecycle
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    expires_at TIMESTAMPTZ,

    -- Constraints
    CONSTRAINT user_memory_requires_user CHECK (
        scope != 'user' OR user_id IS NOT NULL
    ),
    CONSTRAINT collective_memory_no_user CHECK (
        scope != 'collective' OR user_id IS NULL
    ),
    CONSTRAINT collective_requires_validation CHECK (
        scope != 'collective' OR validated_at IS NOT NULL
    )
);

-- Indexes for efficient querying

-- User memories: by user + agent + type
CREATE INDEX idx_memories_user_scope ON memories(user_id, agent_id, type)
    WHERE scope = 'user';

-- Collective memories: by agent + personality + validated
CREATE INDEX idx_memories_collective_agent ON memories(agent_id, personality)
    WHERE scope = 'collective';
CREATE INDEX idx_memories_collective_validated ON memories(validation_score DESC, validation_trades DESC)
    WHERE scope = 'collective' AND validated_at IS NOT NULL;

-- By type
CREATE INDEX idx_memories_type ON memories(type);

-- By symbol
CREATE INDEX idx_memories_symbol ON memories(symbol)
    WHERE symbol IS NOT NULL;

-- Vector similarity search
CREATE INDEX idx_memories_embedding ON memories
    USING ivfflat (embedding vector_cosine_ops);

-- Session lookup
CREATE INDEX idx_memories_session ON memories(session_id)
    WHERE session_id IS NOT NULL;

-- Created date
CREATE INDEX idx_memories_created ON memories(created_at DESC);

-- Expiry
CREATE INDEX idx_memories_expires ON memories(expires_at)
    WHERE expires_at IS NOT NULL;

-- Scope + importance for general queries
CREATE INDEX idx_memories_scope_importance ON memories(scope, importance DESC);

-- Metadata JSON queries
CREATE INDEX idx_memories_metadata ON memories USING gin(metadata);

-- Embedding model filter (for compatibility)
CREATE INDEX idx_memories_embedding_model ON memories(embedding_model)
    WHERE embedding_model IS NOT NULL;

-- Trigger for updated_at
CREATE TRIGGER memories_updated_at BEFORE UPDATE ON memories
FOR EACH ROW EXECUTE FUNCTION update_updated_at();

-- Comments
COMMENT ON TABLE memories IS 'Unified memory storage for agent-level memories: user (agent memory for specific user), collective (agent shared knowledge), working (temporary)';
COMMENT ON COLUMN memories.scope IS 'Memory scope: user (agent-level for user), collective (agent-level shared), working (temporary)';
COMMENT ON COLUMN memories.user_id IS 'User for agent-level user memories, NULL for collective/working';
COMMENT ON COLUMN memories.agent_id IS 'Agent that owns this memory';
COMMENT ON COLUMN memories.validation_score IS 'Performance score 0-100 for collective memories';
COMMENT ON COLUMN memories.validation_trades IS 'Number of trades that validated this pattern (collective only)';
COMMENT ON COLUMN memories.personality IS 'Risk personality filter for collective memories (NULL = all)';
COMMENT ON COLUMN memories.source_user_id IS 'Original user who contributed this collective memory';
