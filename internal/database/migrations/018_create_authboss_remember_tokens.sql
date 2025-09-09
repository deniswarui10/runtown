-- Create Authboss remember tokens table for "Remember Me" functionality
CREATE TABLE authboss_remember_tokens (
    selector VARCHAR(255) PRIMARY KEY,
    verifier VARCHAR(255) NOT NULL,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP NOT NULL
);

-- Create indexes for efficient lookups
CREATE INDEX idx_authboss_remember_tokens_user_id ON authboss_remember_tokens(user_id);
CREATE INDEX idx_authboss_remember_tokens_expires_at ON authboss_remember_tokens(expires_at);

-- Add comment for clarity
COMMENT ON TABLE authboss_remember_tokens IS 'Authboss: Stores remember me tokens for persistent authentication';
COMMENT ON COLUMN authboss_remember_tokens.selector IS 'Authboss: Public token selector';
COMMENT ON COLUMN authboss_remember_tokens.verifier IS 'Authboss: Hashed token verifier';
COMMENT ON COLUMN authboss_remember_tokens.user_id IS 'Reference to the user who owns this token';
COMMENT ON COLUMN authboss_remember_tokens.expires_at IS 'When this remember token expires';