-- Create withdrawals table
CREATE TABLE withdrawals (
    id SERIAL PRIMARY KEY,
    organizer_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    amount INTEGER NOT NULL, -- Amount in cents
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    
    -- Payment details
    payment_method VARCHAR(50) NOT NULL,
    account_name VARCHAR(255) NOT NULL,
    account_number VARCHAR(100) NOT NULL,
    bank_name VARCHAR(255),
    
    -- Request details
    requested_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    processed_at TIMESTAMP WITH TIME ZONE,
    processed_by INTEGER REFERENCES users(id),
    
    -- Notes
    organizer_notes TEXT,
    admin_notes TEXT,
    
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Create indexes
CREATE INDEX idx_withdrawals_organizer_id ON withdrawals(organizer_id);
CREATE INDEX idx_withdrawals_status ON withdrawals(status);
CREATE INDEX idx_withdrawals_requested_at ON withdrawals(requested_at);

-- Add check constraint for status
ALTER TABLE withdrawals ADD CONSTRAINT check_withdrawal_status 
    CHECK (status IN ('pending', 'approved', 'rejected', 'completed'));

-- Add check constraint for amount (must be positive)
ALTER TABLE withdrawals ADD CONSTRAINT check_withdrawal_amount 
    CHECK (amount > 0);