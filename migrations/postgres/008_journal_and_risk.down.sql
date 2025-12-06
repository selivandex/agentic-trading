-- Rollback journal and risk tables

DROP TABLE IF EXISTS risk_events;
DROP TRIGGER IF EXISTS circuit_breaker_states_updated_at ON circuit_breaker_states;
DROP TABLE IF EXISTS circuit_breaker_states;
DROP TABLE IF EXISTS journal_entries;
