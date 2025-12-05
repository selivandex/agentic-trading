-- Rollback portfolio_decisions table

-- Drop indexes
DROP INDEX IF EXISTS idx_portfolio_decisions_strategy;
DROP INDEX IF EXISTS idx_portfolio_decisions_user;
DROP INDEX IF EXISTS idx_portfolio_decisions_symbol;
DROP INDEX IF EXISTS idx_portfolio_decisions_parent;
DROP INDEX IF EXISTS idx_portfolio_decisions_agent;
DROP INDEX IF EXISTS idx_portfolio_decisions_session;

-- Drop table
DROP TABLE IF EXISTS portfolio_decisions;

