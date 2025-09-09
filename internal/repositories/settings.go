package repositories

import (
	"database/sql"
	"fmt"
	"time"

	"event-ticketing-platform/internal/models"
)

// SettingsRepository handles system settings data operations
type SettingsRepository struct {
	db *sql.DB
}

// NewSettingsRepository creates a new settings repository
func NewSettingsRepository(db *sql.DB) *SettingsRepository {
	return &SettingsRepository{db: db}
}

// GetSettings retrieves the current system settings
func (r *SettingsRepository) GetSettings() (*models.SystemSettings, error) {
	query := `
		SELECT id, platform_fee_percentage, min_withdrawal_amount, max_withdrawal_amount,
		       withdrawal_processing_days, event_moderation_enabled, auto_approve_organizers,
		       maintenance_mode, created_at, updated_at
		FROM system_settings
		ORDER BY id DESC
		LIMIT 1`

	settings := &models.SystemSettings{}
	err := r.db.QueryRow(query).Scan(
		&settings.ID,
		&settings.PlatformFeePercentage,
		&settings.MinWithdrawalAmount,
		&settings.MaxWithdrawalAmount,
		&settings.WithdrawalProcessingDays,
		&settings.EventModerationEnabled,
		&settings.AutoApproveOrganizers,
		&settings.MaintenanceMode,
		&settings.CreatedAt,
		&settings.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		// Return default settings if none exist
		return models.DefaultSettings(), nil
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get settings: %w", err)
	}

	return settings, nil
}

// UpdateSettings updates the system settings
func (r *SettingsRepository) UpdateSettings(req *models.SettingsUpdateRequest) (*models.SystemSettings, error) {
	// Get current settings first
	current, err := r.GetSettings()
	if err != nil {
		return nil, fmt.Errorf("failed to get current settings: %w", err)
	}

	// Apply updates
	if req.PlatformFeePercentage != nil {
		current.PlatformFeePercentage = *req.PlatformFeePercentage
	}
	if req.MinWithdrawalAmount != nil {
		current.MinWithdrawalAmount = *req.MinWithdrawalAmount
	}
	if req.MaxWithdrawalAmount != nil {
		current.MaxWithdrawalAmount = *req.MaxWithdrawalAmount
	}
	if req.WithdrawalProcessingDays != nil {
		current.WithdrawalProcessingDays = *req.WithdrawalProcessingDays
	}
	if req.EventModerationEnabled != nil {
		current.EventModerationEnabled = *req.EventModerationEnabled
	}
	if req.AutoApproveOrganizers != nil {
		current.AutoApproveOrganizers = *req.AutoApproveOrganizers
	}
	if req.MaintenanceMode != nil {
		current.MaintenanceMode = *req.MaintenanceMode
	}

	current.UpdatedAt = time.Now()

	// Insert new settings record (we keep history)
	query := `
		INSERT INTO system_settings (
			platform_fee_percentage, min_withdrawal_amount, max_withdrawal_amount,
			withdrawal_processing_days, event_moderation_enabled, auto_approve_organizers,
			maintenance_mode, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id`

	err = r.db.QueryRow(query,
		current.PlatformFeePercentage,
		current.MinWithdrawalAmount,
		current.MaxWithdrawalAmount,
		current.WithdrawalProcessingDays,
		current.EventModerationEnabled,
		current.AutoApproveOrganizers,
		current.MaintenanceMode,
		current.CreatedAt,
		current.UpdatedAt,
	).Scan(&current.ID)

	if err != nil {
		return nil, fmt.Errorf("failed to update settings: %w", err)
	}

	return current, nil
}

// InitializeDefaultSettings creates default settings if none exist
func (r *SettingsRepository) InitializeDefaultSettings() error {
	// Check if settings already exist
	_, err := r.GetSettings()
	if err == nil {
		return nil // Settings already exist
	}

	// Create default settings
	defaults := models.DefaultSettings()
	query := `
		INSERT INTO system_settings (
			platform_fee_percentage, min_withdrawal_amount, max_withdrawal_amount,
			withdrawal_processing_days, event_moderation_enabled, auto_approve_organizers,
			maintenance_mode, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`

	_, err = r.db.Exec(query,
		defaults.PlatformFeePercentage,
		defaults.MinWithdrawalAmount,
		defaults.MaxWithdrawalAmount,
		defaults.WithdrawalProcessingDays,
		defaults.EventModerationEnabled,
		defaults.AutoApproveOrganizers,
		defaults.MaintenanceMode,
		defaults.CreatedAt,
		defaults.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to initialize default settings: %w", err)
	}

	return nil
}