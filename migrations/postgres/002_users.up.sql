-- Users table with both Telegram and email/password authentication

CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

    -- Telegram authentication (optional)
    telegram_id BIGINT UNIQUE,
    telegram_username VARCHAR(255),
    first_name VARCHAR(255),
    last_name VARCHAR(255),
    language_code VARCHAR(10) DEFAULT 'en',

    -- Email/password authentication (optional)
    email VARCHAR(255) UNIQUE,
    password_hash VARCHAR(255),

    -- User settings
    is_active BOOLEAN DEFAULT true,
    is_premium BOOLEAN DEFAULT false,
    settings JSONB DEFAULT '{}',

    -- Timestamps
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),

    -- Constraint: user must have either telegram_id OR email
    CONSTRAINT users_auth_method_check CHECK (
        (telegram_id IS NOT NULL) OR
        (email IS NOT NULL AND password_hash IS NOT NULL)
    )
);

-- Indexes
CREATE INDEX idx_users_telegram_id ON users(telegram_id) WHERE telegram_id IS NOT NULL;
CREATE INDEX idx_users_email ON users(email) WHERE email IS NOT NULL;
CREATE INDEX idx_users_is_active ON users(is_active) WHERE is_active = true;

-- Comments
COMMENT ON TABLE users IS 'Users can authenticate via Telegram or email/password';
COMMENT ON COLUMN users.email IS 'User email for email/password authentication';
COMMENT ON COLUMN users.password_hash IS 'Bcrypt hashed password';

-- Function to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger for updated_at
CREATE TRIGGER users_updated_at BEFORE UPDATE ON users
FOR EACH ROW EXECUTE FUNCTION update_updated_at();
