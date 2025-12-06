-- Rollback unified memories table
-- Note: memory_scope ENUM is dropped in 001_extensions_and_enums.down.sql
DROP TABLE IF EXISTS memories CASCADE;
