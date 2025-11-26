-- AI model usage tracking (timeseries for cost analysis)
CREATE TABLE
  IF NOT EXISTS ai_usage (
    timestamp DateTime DEFAULT now (),
    event_id String,
    -- User context
    user_id String,
    session_id String,
    -- Agent context
    agent_name LowCardinality (String), -- Limited set of agent names
    agent_type LowCardinality (String), -- Limited set of agent types
    -- Model details
    provider LowCardinality (String), -- anthropic, openai, google, deepseek (4 values)
    model_id LowCardinality (String), -- claude-sonnet-4, gpt-4o, gemini-pro, etc. (dozens)
    model_family LowCardinality (String), -- claude-3.5, gpt-4o, gemini-1.5, etc. (limited)
    -- Token usage
    prompt_tokens UInt32,
    completion_tokens UInt32,
    total_tokens UInt32,
    -- Cost
    input_cost_usd Float64,
    output_cost_usd Float64,
    total_cost_usd Float64,
    -- Request metadata
    tool_calls_count UInt16,
    is_cached Boolean DEFAULT false,
    cache_hit Boolean DEFAULT false,
    -- Performance
    latency_ms UInt32,
    -- Additional context
    reasoning_step UInt16, -- Which step in reasoning chain (1, 2, 3...)
    workflow_name LowCardinality (String), -- MarketResearchWorkflow, PersonalTradingWorkflow, etc.
    created_at DateTime DEFAULT now ()
  ) ENGINE = MergeTree ()
PARTITION BY
  toYYYYMM (timestamp)
ORDER BY
  (provider, model_id, timestamp) TTL timestamp + INTERVAL 90 DAY;

-- Keep 90 days of usage data
-- Index for fast user queries
CREATE INDEX IF NOT EXISTS idx_ai_usage_user_id ON ai_usage (user_id) TYPE bloom_filter GRANULARITY 1;

-- Index for session queries
CREATE INDEX IF NOT EXISTS idx_ai_usage_session_id ON ai_usage (session_id) TYPE bloom_filter GRANULARITY 1;

-- Index for agent type queries
CREATE INDEX IF NOT EXISTS idx_ai_usage_agent_type ON ai_usage (agent_type) TYPE bloom_filter GRANULARITY 1;

-- Materialized view for daily costs by provider
CREATE MATERIALIZED VIEW IF NOT EXISTS ai_usage_daily_by_provider ENGINE = SummingMergeTree ()
PARTITION BY
  toYYYYMM (date)
ORDER BY
  (provider, date) AS
SELECT
  toDate (timestamp) AS date,
  provider,
  model_id,
  sum(total_tokens) AS total_tokens,
  sum(total_cost_usd) AS total_cost_usd,
  count() AS request_count
FROM
  ai_usage
GROUP BY
  date,
  provider,
  model_id;

-- Materialized view for user daily costs
CREATE MATERIALIZED VIEW IF NOT EXISTS ai_usage_daily_by_user ENGINE = SummingMergeTree ()
PARTITION BY
  toYYYYMM (date)
ORDER BY
  (user_id, date) AS
SELECT
  toDate (timestamp) AS date,
  user_id,
  provider,
  sum(total_tokens) AS total_tokens,
  sum(total_cost_usd) AS total_cost_usd,
  count() AS request_count
FROM
  ai_usage
GROUP BY
  date,
  user_id,
  provider;

-- Materialized view for agent type costs
CREATE MATERIALIZED VIEW IF NOT EXISTS ai_usage_daily_by_agent ENGINE = SummingMergeTree ()
PARTITION BY
  toYYYYMM (date)
ORDER BY
  (agent_type, date) AS
SELECT
  toDate (timestamp) AS date,
  agent_type,
  provider,
  model_id,
  sum(total_tokens) AS total_tokens,
  sum(total_cost_usd) AS total_cost_usd,
  count() AS request_count
FROM
  ai_usage
GROUP BY
  date,
  agent_type,
  provider,
  model_id;
