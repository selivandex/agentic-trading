-- Rollback orders table
DROP TRIGGER IF EXISTS orders_updated_at ON orders;

DROP TABLE IF EXISTS orders;
