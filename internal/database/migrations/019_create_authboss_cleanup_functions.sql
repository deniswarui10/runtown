-- Create functions and procedures for Authboss token cleanup

-- Function to clean up expired remember tokens
CREATE OR REPLACE FUNCTION cleanup_expired_remember_tokens()
RETURNS INTEGER AS $$
DECLARE
    deleted_count INTEGER;
BEGIN
    DELETE FROM authboss_remember_tokens 
    WHERE expires_at < NOW();
    
    GET DIAGNOSTICS deleted_count = ROW_COUNT;
    RETURN deleted_count;
END;
$$ LANGUAGE plpgsql;

-- Function to clean up expired recovery tokens
CREATE OR REPLACE FUNCTION cleanup_expired_recovery_tokens()
RETURNS INTEGER AS $$
DECLARE
    updated_count INTEGER;
BEGIN
    UPDATE users 
    SET recover_selector = NULL,
        recover_verifier = NULL,
        recover_token_expires = NULL
    WHERE recover_token_expires IS NOT NULL 
    AND recover_token_expires < NOW();
    
    GET DIAGNOSTICS updated_count = ROW_COUNT;
    RETURN updated_count;
END;
$$ LANGUAGE plpgsql;

-- Function to unlock accounts that have passed their lock time
CREATE OR REPLACE FUNCTION unlock_expired_accounts()
RETURNS INTEGER AS $$
DECLARE
    updated_count INTEGER;
BEGIN
    UPDATE users 
    SET locked_until = NULL,
        attempt_count = 0
    WHERE locked_until IS NOT NULL 
    AND locked_until < NOW();
    
    GET DIAGNOSTICS updated_count = ROW_COUNT;
    RETURN updated_count;
END;
$$ LANGUAGE plpgsql;

-- Function to clean up old confirmation tokens (for users who never confirmed)
CREATE OR REPLACE FUNCTION cleanup_old_confirmation_tokens()
RETURNS INTEGER AS $$
DECLARE
    updated_count INTEGER;
BEGIN
    -- Clean up confirmation tokens older than 7 days for unconfirmed users
    UPDATE users 
    SET confirm_selector = NULL,
        confirm_verifier = NULL
    WHERE confirmed_at IS NULL 
    AND confirm_selector IS NOT NULL
    AND created_at < NOW() - INTERVAL '7 days';
    
    GET DIAGNOSTICS updated_count = ROW_COUNT;
    RETURN updated_count;
END;
$$ LANGUAGE plpgsql;

-- Add comments
COMMENT ON FUNCTION cleanup_expired_remember_tokens() IS 'Authboss: Removes expired remember me tokens';
COMMENT ON FUNCTION cleanup_expired_recovery_tokens() IS 'Authboss: Clears expired password recovery tokens';
COMMENT ON FUNCTION unlock_expired_accounts() IS 'Authboss: Unlocks accounts that have passed their lock time';
COMMENT ON FUNCTION cleanup_old_confirmation_tokens() IS 'Authboss: Removes old confirmation tokens for unconfirmed users';