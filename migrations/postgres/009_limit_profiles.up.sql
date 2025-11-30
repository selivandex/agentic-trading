-- Create limit_profiles table
CREATE TABLE IF NOT EXISTS limit_profiles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) NOT NULL UNIQUE,
    description TEXT NOT NULL DEFAULT '',
    limits JSONB NOT NULL DEFAULT '{}',
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Create index on name for faster lookups
CREATE INDEX idx_limit_profiles_name ON limit_profiles(name) WHERE is_active = true;

-- Create index on is_active
CREATE INDEX idx_limit_profiles_active ON limit_profiles(is_active);

-- Insert default limit profiles
INSERT INTO limit_profiles (id, name, description, limits, is_active) VALUES
(
    gen_random_uuid(),
    'free',
    'Free tier with basic features',
    '{
        "exchanges_count": 1,
        "active_positions": 2,
        "daily_trades_count": 5,
        "monthly_trades_count": 50,
        "trading_pairs_count": 5,
        "monthly_ai_requests": 100,
        "max_agents_count": 2,
        "advanced_agents_access": false,
        "custom_agents_allowed": false,
        "max_agent_memory_mb": 50,
        "max_prompt_tokens": 4000,
        "advanced_ai_models": false,
        "agent_history_days": 7,
        "reasoning_history_days": 3,
        "max_memories_per_agent": 20,
        "data_retention_days": 30,
        "analysis_history_days": 7,
        "session_history_days": 7,
        "backtesting_allowed": false,
        "max_backtest_months": 0,
        "paper_trading_allowed": true,
        "live_trading_allowed": false,
        "max_leverage": 1.0,
        "max_position_size_usd": 100,
        "max_total_exposure_usd": 200,
        "advanced_order_types": false,
        "priority_execution": false,
        "webhooks_allowed": false,
        "max_webhooks": 0,
        "api_access_allowed": false,
        "api_rate_limit": 0,
        "realtime_data_access": false,
        "historical_data_years": 0,
        "advanced_charts": false,
        "custom_indicators": false,
        "portfolio_analytics": false,
        "risk_management_tools": false,
        "alerts_count": 5,
        "custom_reports_allowed": false,
        "priority_support": false,
        "dedicated_support": false
    }'::jsonb,
    true
),
(
    gen_random_uuid(),
    'basic',
    'Basic tier with enhanced features',
    '{
        "exchanges_count": 2,
        "active_positions": 5,
        "daily_trades_count": 20,
        "monthly_trades_count": 300,
        "trading_pairs_count": 20,
        "monthly_ai_requests": 1000,
        "max_agents_count": 5,
        "advanced_agents_access": true,
        "custom_agents_allowed": false,
        "max_agent_memory_mb": 200,
        "max_prompt_tokens": 8000,
        "advanced_ai_models": true,
        "agent_history_days": 30,
        "reasoning_history_days": 14,
        "max_memories_per_agent": 100,
        "data_retention_days": 90,
        "analysis_history_days": 30,
        "session_history_days": 30,
        "backtesting_allowed": true,
        "max_backtest_months": 6,
        "paper_trading_allowed": true,
        "live_trading_allowed": true,
        "max_leverage": 3.0,
        "max_position_size_usd": 1000,
        "max_total_exposure_usd": 5000,
        "advanced_order_types": true,
        "priority_execution": false,
        "webhooks_allowed": true,
        "max_webhooks": 5,
        "api_access_allowed": true,
        "api_rate_limit": 60,
        "realtime_data_access": true,
        "historical_data_years": 2,
        "advanced_charts": true,
        "custom_indicators": true,
        "portfolio_analytics": true,
        "risk_management_tools": true,
        "alerts_count": 50,
        "custom_reports_allowed": true,
        "priority_support": true,
        "dedicated_support": false
    }'::jsonb,
    true
),
(
    gen_random_uuid(),
    'premium',
    'Premium tier with all features unlocked',
    '{
        "exchanges_count": 10,
        "active_positions": 20,
        "daily_trades_count": -1,
        "monthly_trades_count": -1,
        "trading_pairs_count": -1,
        "monthly_ai_requests": 10000,
        "max_agents_count": 20,
        "advanced_agents_access": true,
        "custom_agents_allowed": true,
        "max_agent_memory_mb": 1000,
        "max_prompt_tokens": 32000,
        "advanced_ai_models": true,
        "agent_history_days": 365,
        "reasoning_history_days": 90,
        "max_memories_per_agent": 1000,
        "data_retention_days": 365,
        "analysis_history_days": 180,
        "session_history_days": 180,
        "backtesting_allowed": true,
        "max_backtest_months": 60,
        "paper_trading_allowed": true,
        "live_trading_allowed": true,
        "max_leverage": 10.0,
        "max_position_size_usd": 50000,
        "max_total_exposure_usd": 250000,
        "advanced_order_types": true,
        "priority_execution": true,
        "webhooks_allowed": true,
        "max_webhooks": 50,
        "api_access_allowed": true,
        "api_rate_limit": 600,
        "realtime_data_access": true,
        "historical_data_years": 10,
        "advanced_charts": true,
        "custom_indicators": true,
        "portfolio_analytics": true,
        "risk_management_tools": true,
        "alerts_count": 500,
        "custom_reports_allowed": true,
        "priority_support": true,
        "dedicated_support": true
    }'::jsonb,
    true
);

-- Add limit_profile_id to users table
ALTER TABLE users ADD COLUMN IF NOT EXISTS limit_profile_id UUID;

-- Add foreign key constraint
ALTER TABLE users ADD CONSTRAINT fk_users_limit_profile 
    FOREIGN KEY (limit_profile_id) 
    REFERENCES limit_profiles(id) 
    ON DELETE SET NULL;

-- Create index on limit_profile_id
CREATE INDEX idx_users_limit_profile_id ON users(limit_profile_id);

-- Set all existing users to 'free' tier
UPDATE users 
SET limit_profile_id = (SELECT id FROM limit_profiles WHERE name = 'free' LIMIT 1)
WHERE limit_profile_id IS NULL;

-- Add comment
COMMENT ON TABLE limit_profiles IS 'Defines usage limits and features for different user tiers';
COMMENT ON COLUMN limit_profiles.limits IS 'JSONB field containing dynamic limit configurations';
COMMENT ON COLUMN users.limit_profile_id IS 'References the user''s current limit profile (tier)';

