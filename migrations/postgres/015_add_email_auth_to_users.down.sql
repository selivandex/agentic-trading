-- Rollback email and password authentication fields

DROP INDEX IF EXISTS idx_users_email;

ALTER TABLE users
    DROP CONSTRAINT IF EXISTS users_auth_method_check;

ALTER TABLE users
    ALTER COLUMN telegram_id SET NOT NULL;

ALTER TABLE users
    DROP COLUMN IF EXISTS password_hash,
    DROP COLUMN IF EXISTS email;
