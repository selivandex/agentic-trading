-- Create limit_profiles table
CREATE TABLE
    IF NOT EXISTS limit_profiles (
        id UUID PRIMARY KEY DEFAULT gen_random_uuid (),
        name VARCHAR(100) NOT NULL UNIQUE,
        description TEXT NOT NULL DEFAULT '',
        limits JSONB NOT NULL DEFAULT '{}',
        is_active BOOLEAN NOT NULL DEFAULT true,
        created_at TIMESTAMP
        WITH
            TIME ZONE NOT NULL DEFAULT NOW (),
            updated_at TIMESTAMP
        WITH
            TIME ZONE NOT NULL DEFAULT NOW ()
    );

-- Create index on name for faster lookups
CREATE INDEX idx_limit_profiles_name ON limit_profiles (name)
WHERE
    is_active = true;

-- Create index on is_active
CREATE INDEX idx_limit_profiles_active ON limit_profiles (is_active);

-- Add limit_profile_id to users table
ALTER TABLE users
ADD COLUMN IF NOT EXISTS limit_profile_id UUID;

-- Add foreign key constraint
ALTER TABLE users ADD CONSTRAINT fk_users_limit_profile FOREIGN KEY (limit_profile_id) REFERENCES limit_profiles (id) ON DELETE SET NULL;

-- Create index on limit_profile_id
CREATE INDEX idx_users_limit_profile_id ON users (limit_profile_id);

-- Add comment
COMMENT ON TABLE limit_profiles IS 'Defines usage limits and features for different user tiers';

COMMENT ON COLUMN limit_profiles.limits IS 'JSONB field containing dynamic limit configurations';

COMMENT ON COLUMN users.limit_profile_id IS 'References the user''s current limit profile (tier)';
