-- Add email verification columns to users table
ALTER TABLE users 
ADD COLUMN email_verified BOOLEAN DEFAULT FALSE,
ADD COLUMN email_verified_at TIMESTAMP NULL,
ADD COLUMN verification_token VARCHAR(255) NULL;

-- Create index on verification token for faster lookups
CREATE INDEX idx_users_verification_token ON users(verification_token);

-- Update existing users to be verified (for backward compatibility)
UPDATE users SET email_verified = TRUE, email_verified_at = created_at WHERE email_verified = FALSE;