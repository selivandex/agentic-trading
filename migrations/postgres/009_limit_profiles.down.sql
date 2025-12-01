-- Remove limit_profile_id from users table
ALTER TABLE users DROP CONSTRAINT IF EXISTS fk_users_limit_profile;
DROP INDEX IF EXISTS idx_users_limit_profile_id;
ALTER TABLE users DROP COLUMN IF EXISTS limit_profile_id;

-- Drop limit_profiles table
DROP INDEX IF EXISTS idx_limit_profiles_active;
DROP INDEX IF EXISTS idx_limit_profiles_name;
DROP TABLE IF EXISTS limit_profiles;


