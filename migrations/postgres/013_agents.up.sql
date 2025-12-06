-- Agents - AI agent definitions with prompts stored in DB
-- Allows changing agent behavior (prompts) without code deployment

CREATE TABLE agents (
    id SERIAL PRIMARY KEY,

    -- Agent identifier (unique key for code references)
    identifier VARCHAR(100) NOT NULL UNIQUE, -- portfolio_manager, technical_analyzer, smc_analyzer

    -- Metadata
    name VARCHAR(255) NOT NULL, -- "Portfolio Manager", "Technical Analyzer"
    description TEXT, -- What this agent does
    category VARCHAR(50), -- expert, coordinator, specialist

    -- Prompts (editable in production!)
    system_prompt TEXT NOT NULL, -- Base system prompt for this agent
    instructions TEXT, -- Additional instructions/guidelines

    -- Configuration
    model_provider VARCHAR(50) DEFAULT 'anthropic', -- anthropic, openai, google
    model_name VARCHAR(100) DEFAULT 'claude-3-5-sonnet-20241022',
    temperature DECIMAL(3,2) DEFAULT 0.7,
    max_tokens INTEGER DEFAULT 4000,

    -- Tools assignment (JSONB array of tool identifiers)
    available_tools JSONB, -- ["get_account_status", "place_order", "call_technical_analyzer"]

    -- Limits
    max_cost_per_run DECIMAL(10,4) DEFAULT 0.50, -- Max $ per single run
    timeout_seconds INTEGER DEFAULT 120,

    -- Status
    is_active BOOLEAN DEFAULT true,
    version INTEGER DEFAULT 1, -- Increment on prompt updates for tracking

    -- Timestamps
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Indexes
CREATE INDEX idx_agents_identifier ON agents(identifier);
CREATE INDEX idx_agents_category ON agents(category);
CREATE INDEX idx_agents_active ON agents(is_active) WHERE is_active = true;

-- Trigger to auto-update updated_at
CREATE OR REPLACE FUNCTION update_agents_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    NEW.version = OLD.version + 1; -- Increment version on update
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER agents_updated_at
    BEFORE UPDATE ON agents
    FOR EACH ROW
    EXECUTE FUNCTION update_agents_updated_at();

-- Permissions
COMMENT ON TABLE agents IS 'AI agent definitions with prompts - editable in production for tuning';
COMMENT ON COLUMN agents.identifier IS 'Unique code identifier: portfolio_manager, technical_analyzer, smc_analyzer';
COMMENT ON COLUMN agents.system_prompt IS 'Base system prompt - can be updated without deployment';
COMMENT ON COLUMN agents.available_tools IS 'JSON array of tool identifiers this agent can use';
COMMENT ON COLUMN agents.version IS 'Auto-incremented on prompt updates for tracking changes';
