-- Migration 14: Add event moderation statuses
-- Add new event statuses for moderation workflow

-- First, add the new status values to the enum
-- Note: PostgreSQL doesn't allow direct enum modification, so we need to use a different approach

-- Add new columns for moderation
ALTER TABLE events ADD COLUMN IF NOT EXISTS reviewed_at TIMESTAMP;
ALTER TABLE events ADD COLUMN IF NOT EXISTS reviewed_by INTEGER REFERENCES users(id);
ALTER TABLE events ADD COLUMN IF NOT EXISTS rejection_reason TEXT;

-- Update the status column to allow new values
-- Since we're using VARCHAR for status, we just need to update any existing constraints
-- The application will handle validation of the new status values

-- Create index for moderation queries
CREATE INDEX IF NOT EXISTS idx_events_status_reviewed ON events(status, reviewed_at);
CREATE INDEX IF NOT EXISTS idx_events_reviewed_by ON events(reviewed_by);

-- Add audit log table for administrative actions
CREATE TABLE IF NOT EXISTS admin_audit_log (
    id SERIAL PRIMARY KEY,
    admin_user_id INTEGER NOT NULL REFERENCES users(id),
    action VARCHAR(100) NOT NULL,
    target_type VARCHAR(50) NOT NULL, -- 'event', 'user', 'category', etc.
    target_id INTEGER NOT NULL,
    details JSONB,
    ip_address INET,
    user_agent TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for audit log
CREATE INDEX IF NOT EXISTS idx_audit_log_admin_user ON admin_audit_log(admin_user_id);
CREATE INDEX IF NOT EXISTS idx_audit_log_action ON admin_audit_log(action);
CREATE INDEX IF NOT EXISTS idx_audit_log_target ON admin_audit_log(target_type, target_id);
CREATE INDEX IF NOT EXISTS idx_audit_log_created_at ON admin_audit_log(created_at);