-- Drop materialized views first (they depend on main table)
DROP VIEW IF EXISTS ai_usage_daily_by_agent;
DROP VIEW IF EXISTS ai_usage_daily_by_user;
DROP VIEW IF EXISTS ai_usage_daily_by_provider;

-- Drop indexes
DROP INDEX IF EXISTS idx_ai_usage_agent_type ON ai_usage;
DROP INDEX IF EXISTS idx_ai_usage_session_id ON ai_usage;
DROP INDEX IF EXISTS idx_ai_usage_user_id ON ai_usage;

-- Drop main table
DROP TABLE IF EXISTS ai_usage;



