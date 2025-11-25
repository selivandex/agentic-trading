-- migrations/postgres/001_init.down.sql
-- Rollback initial schema
DROP TRIGGER IF EXISTS circuit_breaker_states_updated_at ON circuit_breaker_states;

DROP TRIGGER IF EXISTS positions_updated_at ON positions;

DROP TRIGGER IF EXISTS orders_updated_at ON orders;

DROP TRIGGER IF EXISTS trading_pairs_updated_at ON trading_pairs;

DROP TRIGGER IF EXISTS exchange_accounts_updated_at ON exchange_accounts;

DROP TRIGGER IF EXISTS users_updated_at ON users;

DROP FUNCTION IF EXISTS update_updated_at ();

DROP TABLE IF EXISTS risk_events;

DROP TABLE IF EXISTS circuit_breaker_states;

DROP TABLE IF EXISTS journal_entries;

DROP TABLE IF EXISTS collective_memories;

DROP TABLE IF EXISTS memories;

DROP TABLE IF EXISTS positions;

DROP TABLE IF EXISTS orders;

DROP TABLE IF EXISTS trading_pairs;

DROP TABLE IF EXISTS exchange_accounts;

DROP TABLE IF EXISTS users;

DROP EXTENSION IF EXISTS "vector";

DROP EXTENSION IF EXISTS "uuid-ossp";
