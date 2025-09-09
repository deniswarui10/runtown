package models

import (
	"time"
)

// SystemSettings represents system-wide configuration settings
type SystemSettings struct {
	ID                    int       `json:"id" db:"id"`
	PlatformFeePercentage float64   `json:"platform_fee_percentage" db:"platform_fee_percentage"`
	MinWithdrawalAmount   float64   `json:"min_withdrawal_amount" db:"min_withdrawal_amount"`
	MaxWithdrawalAmount   float64   `json:"max_withdrawal_amount" db:"max_withdrawal_amount"`
	WithdrawalProcessingDays int    `json:"withdrawal_processing_days" db:"withdrawal_processing_days"`
	EventModerationEnabled bool     `json:"event_moderation_enabled" db:"event_moderation_enabled"`
	AutoApproveOrganizers bool      `json:"auto_approve_organizers" db:"auto_approve_organizers"`
	MaintenanceMode       bool      `json:"maintenance_mode" db:"maintenance_mode"`
	CreatedAt             time.Time `json:"created_at" db:"created_at"`
	UpdatedAt             time.Time `json:"updated_at" db:"updated_at"`
}

// SettingsUpdateRequest represents a request to update system settings
type SettingsUpdateRequest struct {
	PlatformFeePercentage    *float64 `json:"platform_fee_percentage" validate:"omitempty,min=0,max=50"`
	MinWithdrawalAmount      *float64 `json:"min_withdrawal_amount" validate:"omitempty,min=1"`
	MaxWithdrawalAmount      *float64 `json:"max_withdrawal_amount" validate:"omitempty,min=1"`
	WithdrawalProcessingDays *int     `json:"withdrawal_processing_days" validate:"omitempty,min=1,max=30"`
	EventModerationEnabled   *bool    `json:"event_moderation_enabled"`
	AutoApproveOrganizers    *bool    `json:"auto_approve_organizers"`
	MaintenanceMode          *bool    `json:"maintenance_mode"`
}

// DefaultSettings returns the default system settings
func DefaultSettings() *SystemSettings {
	return &SystemSettings{
		PlatformFeePercentage:    5.0,  // 5% platform fee
		MinWithdrawalAmount:      10.0, // $10 minimum
		MaxWithdrawalAmount:      10000.0, // $10,000 maximum
		WithdrawalProcessingDays: 3,    // 3 business days
		EventModerationEnabled:   true, // Enable moderation by default
		AutoApproveOrganizers:    false, // Manual organizer approval
		MaintenanceMode:          false, // Not in maintenance mode
		CreatedAt:                time.Now(),
		UpdatedAt:                time.Now(),
	}
}