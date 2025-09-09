package models

import (
	"time"
)

// WithdrawalStatus represents the status of a withdrawal request
type WithdrawalStatus string

const (
	WithdrawalStatusPending   WithdrawalStatus = "pending"
	WithdrawalStatusApproved  WithdrawalStatus = "approved"
	WithdrawalStatusRejected  WithdrawalStatus = "rejected"
	WithdrawalStatusCompleted WithdrawalStatus = "completed"
)

// Withdrawal represents a withdrawal request from an organizer
type Withdrawal struct {
	ID          int               `json:"id" db:"id"`
	OrganizerID int               `json:"organizer_id" db:"organizer_id"`
	Amount      float64           `json:"amount" db:"amount"`
	Status      WithdrawalStatus  `json:"status" db:"status"`
	Reason      string            `json:"reason" db:"reason"`
	BankDetails string            `json:"bank_details" db:"bank_details"`
	Notes       string            `json:"notes" db:"notes"`
	AdminNotes  string            `json:"admin_notes" db:"admin_notes"`
	RequestedAt time.Time         `json:"requested_at" db:"requested_at"`
	ProcessedAt *time.Time        `json:"processed_at" db:"processed_at"`
	CreatedAt   time.Time         `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at" db:"updated_at"`
	
	// Related data
	Organizer *User `json:"organizer,omitempty"`
}

// WithdrawalCreateRequest represents a request to create a withdrawal
type WithdrawalCreateRequest struct {
	Amount      float64 `json:"amount" validate:"required,min=10"`
	Reason      string  `json:"reason" validate:"required,max=500"`
	BankDetails string  `json:"bank_details" validate:"required,max=1000"`
}

// Validate validates the withdrawal create request
func (r *WithdrawalCreateRequest) Validate() error {
	if r.Amount < 10 {
		return ErrInvalidInput
	}
	if r.Reason == "" {
		return ErrInvalidInput
	}
	if r.BankDetails == "" {
		return ErrInvalidInput
	}
	return nil
}

// WithdrawalUpdateRequest represents a request to update a withdrawal
type WithdrawalUpdateRequest struct {
	Status     WithdrawalStatus `json:"status" validate:"required"`
	AdminNotes string           `json:"admin_notes" validate:"max=1000"`
}

