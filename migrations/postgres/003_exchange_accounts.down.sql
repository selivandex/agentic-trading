-- Rollback exchange_accounts table
DROP TRIGGER IF EXISTS exchange_accounts_updated_at ON exchange_accounts;

DROP TABLE IF EXISTS exchange_accounts;
