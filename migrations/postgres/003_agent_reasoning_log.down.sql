-- migrations/postgres/003_agent_reasoning_log.down.sql
-- Rollback reasoning log table
DROP TABLE IF EXISTS agent_reasoning_logs;
