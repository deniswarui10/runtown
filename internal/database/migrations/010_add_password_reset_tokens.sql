-- Add password reset token fields to users table
ALTER TABLE users 
ADD COLUMN password_reset_token VARCHAR(255),
ADD COLUMN password_reset_expires TIMESTAMP;

-- Create index for password reset token lookups
CREATE INDEX idx_users_password_reset_token ON users(password_reset_token) WHERE password_reset_token IS NOT NULL;

-- Create index for cleanup of expired tokens
CREATE INDEX idx_users_password_reset_expires ON users(password_reset_expires) WHERE password_reset_expires IS NOT NULL;