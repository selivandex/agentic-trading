-- Rollback agents table migration

-- Drop trigger first (depends on function)
DROP TRIGGER IF EXISTS agents_updated_at ON agents;

-- Drop function
DROP FUNCTION IF EXISTS update_agents_updated_at();

-- Drop table (will cascade drop all indexes)
DROP TABLE IF EXISTS agents;
