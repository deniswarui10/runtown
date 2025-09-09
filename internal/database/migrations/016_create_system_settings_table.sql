-- Create system_settings table
CREATE TABLE system_settings (
    id SERIAL PRIMARY KEY,
    platform_fee_percentage DECIMAL(5,2) NOT NULL DEFAULT 5.00,
    min_withdrawal_amount DECIMAL(10,2) NOT NULL DEFAULT 10.00,
    max_withdrawal_amount DECIMAL(10,2) NOT NULL DEFAULT 10000.00,
    withdrawal_processing_days INTEGER NOT NULL DEFAULT 3,
    event_moderation_enabled BOOLEAN NOT NULL DEFAULT true,
    auto_approve_organizers BOOLEAN NOT NULL DEFAULT false,
    maintenance_mode BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Create indexes
CREATE INDEX idx_system_settings_created_at ON system_settings(created_at DESC);

-- Insert default settings
INSERT INTO system_settings (
    platform_fee_percentage,
    min_withdrawal_amount,
    max_withdrawal_amount,
    withdrawal_processing_days,
    event_moderation_enabled,
    auto_approve_organizers,
    maintenance_mode
) VALUES (
    5.00,    -- 5% platform fee
    10.00,   -- $10 minimum withdrawal
    10000.00, -- $10,000 maximum withdrawal
    3,       -- 3 business days processing
    true,    -- Enable event moderation
    false,   -- Manual organizer approval
    false    -- Not in maintenance mode
);