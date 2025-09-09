-- Add missing columns to withdrawals table to match the model
ALTER TABLE withdrawals ADD COLUMN reason TEXT;
ALTER TABLE withdrawals ADD COLUMN bank_details TEXT;
ALTER TABLE withdrawals ADD COLUMN notes TEXT;

-- Update existing data to use the new columns
UPDATE withdrawals SET 
    reason = COALESCE(organizer_notes, 'Withdrawal request'),
    bank_details = CONCAT('Payment Method: ', payment_method, ', Account: ', account_name, ', Number: ', account_number, ', Bank: ', COALESCE(bank_name, 'N/A')),
    notes = organizer_notes;

-- The old columns can be kept for backward compatibility or dropped later
-- For now, we'll keep them to avoid breaking existing code