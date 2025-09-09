-- Add Authboss required fields to users table
-- This migration adds fields needed for Authboss while preserving existing functionality

-- Add Authboss confirmation fields (maps to existing email verification)
ALTER TABLE users 
ADD COLUMN confirmed_at TIMESTAMP NULL,
ADD COLUMN confirm_selector VARCHAR(255) NULL,
ADD COLUMN confirm_verifier VARCHAR(255) NULL;

-- Add Authboss account locking fields
ALTER TABLE users 
ADD COLUMN locked_until TIMESTAMP NULL,
ADD COLUMN attempt_count INTEGER DEFAULT 0 NOT NULL,
ADD COLUMN last_attempt TIMESTAMP NULL;

-- Add Authboss password management fields
ALTER TABLE users 
ADD COLUMN password_changed_at TIMESTAMP NULL;

-- Add Authboss recovery fields (separate from existing password reset for compatibility)
ALTER TABLE users 
ADD COLUMN recover_selector VARCHAR(255) NULL,
ADD COLUMN recover_verifier VARCHAR(255) NULL,
ADD COLUMN recover_token_expires TIMESTAMP NULL;

-- Create indexes for Authboss fields
CREATE INDEX idx_users_confirmed_at ON users(confirmed_at);
CREATE INDEX idx_users_confirm_selector ON users(confirm_selector) WHERE confirm_selector IS NOT NULL;
CREATE INDEX idx_users_locked_until ON users(locked_until) WHERE locked_until IS NOT NULL;
CREATE INDEX idx_users_recover_selector ON users(recover_selector) WHERE recover_selector IS NOT NULL;

-- Migrate existing email verification data to Authboss format
UPDATE users 
SET confirmed_at = email_verified_at 
WHERE email_verified = TRUE AND email_verified_at IS NOT NULL;

-- Set password_changed_at for existing users (use created_at as default)
UPDATE users 
SET password_changed_at = created_at 
WHERE password_changed_at IS NULL;

-- Add comments for clarity
COMMENT ON COLUMN users.confirmed_at IS 'Authboss: When the user confirmed their email address';
COMMENT ON COLUMN users.confirm_selector IS 'Authboss: Email confirmation selector token';
COMMENT ON COLUMN users.confirm_verifier IS 'Authboss: Email confirmation verifier token';
COMMENT ON COLUMN users.locked_until IS 'Authboss: Account locked until this timestamp';
COMMENT ON COLUMN users.attempt_count IS 'Authboss: Number of failed login attempts';
COMMENT ON COLUMN users.last_attempt IS 'Authboss: Timestamp of last login attempt';
COMMENT ON COLUMN users.password_changed_at IS 'Authboss: When the password was last changed';
COMMENT ON COLUMN users.recover_selector IS 'Authboss: Password recovery selector token';
COMMENT ON COLUMN users.recover_verifier IS 'Authboss: Password recovery verifier token';
COMMENT ON COLUMN users.recover_token_expires IS 'Authboss: When the recovery token expires';