-- Add email and password authentication fields to users table
-- This allows users to authenticate via email/password in addition to Telegram

ALTER TABLE users
    ADD COLUMN email VARCHAR(255) UNIQUE,
    ADD COLUMN password_hash VARCHAR(255);

-- Make telegram_id nullable (for email-only users)
ALTER TABLE users
    ALTER COLUMN telegram_id DROP NOT NULL;

-- Add index on email for fast lookups
CREATE INDEX idx_users_email ON users(email) WHERE email IS NOT NULL;

-- Add constraint: user must have either telegram_id OR email
ALTER TABLE users
    ADD CONSTRAINT users_auth_method_check
    CHECK (
        (telegram_id IS NOT NULL) OR
        (email IS NOT NULL AND password_hash IS NOT NULL)
    );

COMMENT ON COLUMN users.email IS 'User email for email/password authentication';
COMMENT ON COLUMN users.password_hash IS 'Bcrypt hashed password';
