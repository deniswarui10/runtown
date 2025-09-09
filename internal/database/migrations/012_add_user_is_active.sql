-- Add is_active column to users table
ALTER TABLE users ADD COLUMN is_active BOOLEAN NOT NULL DEFAULT true;

-- Create index for performance
CREATE INDEX idx_users_is_active ON users(is_active);

-- Update existing users to be active by default
UPDATE users SET is_active = true WHERE is_active IS NULL;