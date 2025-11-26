-- Drop collective memories table
DROP TRIGGER IF EXISTS collective_memories_updated_at ON collective_memories;
DROP INDEX IF EXISTS idx_collective_memories_created;
DROP INDEX IF EXISTS idx_collective_memories_importance;
DROP INDEX IF EXISTS idx_collective_memories_embedding;
DROP INDEX IF EXISTS idx_collective_memories_symbol;
DROP INDEX IF EXISTS idx_collective_memories_validated;
DROP INDEX IF EXISTS idx_collective_memories_type;
DROP INDEX IF EXISTS idx_collective_memories_personality;
DROP INDEX IF EXISTS idx_collective_memories_agent;
DROP TABLE IF EXISTS collective_memories;

