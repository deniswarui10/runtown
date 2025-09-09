package services

import (
	"fmt"

	"event-ticketing-platform/internal/models"
	"event-ticketing-platform/internal/repositories"
)

// SettingsService handles system settings business logic
type SettingsService struct {
	settingsRepo *repositories.SettingsRepository
}

// NewSettingsService creates a new settings service
func NewSettingsService(settingsRepo *repositories.SettingsRepository) *SettingsService {
	return &SettingsService{
		settingsRepo: settingsRepo,
	}
}

// GetSettings retrieves the current system settings
func (s *SettingsService) GetSettings() (*models.SystemSettings, error) {
	return s.settingsRepo.GetSettings()
}

// UpdateSettings updates the system settings with validation
func (s *SettingsService) UpdateSettings(req *models.SettingsUpdateRequest) (*models.SystemSettings, error) {
	// Validate the request
	if err := s.validateSettingsUpdate(req); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	return s.settingsRepo.UpdateSettings(req)
}

// GetPlatformFeePercentage returns the current platform fee percentage
func (s *SettingsService) GetPlatformFeePercentage() (float64, error) {
	settings, err := s.GetSettings()
	if err != nil {
		return 5.0, err // Default to 5% on error
	}
	return settings.PlatformFeePercentage, nil
}

// GetMinWithdrawalAmount returns the minimum withdrawal amount
func (s *SettingsService) GetMinWithdrawalAmount() (float64, error) {
	settings, err := s.GetSettings()
	if err != nil {
		return 10.0, err // Default to $10 on error
	}
	return settings.MinWithdrawalAmount, nil
}

// IsEventModerationEnabled returns whether event moderation is enabled
func (s *SettingsService) IsEventModerationEnabled() (bool, error) {
	settings, err := s.GetSettings()
	if err != nil {
		return true, err // Default to enabled on error
	}
	return settings.EventModerationEnabled, nil
}

// InitializeDefaultSettings initializes default settings if none exist
func (s *SettingsService) InitializeDefaultSettings() error {
	return s.settingsRepo.InitializeDefaultSettings()
}

// validateSettingsUpdate validates the settings update request
func (s *SettingsService) validateSettingsUpdate(req *models.SettingsUpdateRequest) error {
	if req.PlatformFeePercentage != nil {
		if *req.PlatformFeePercentage < 0 || *req.PlatformFeePercentage > 50 {
			return fmt.Errorf("platform fee percentage must be between 0 and 50")
		}
	}

	if req.MinWithdrawalAmount != nil {
		if *req.MinWithdrawalAmount < 1 {
			return fmt.Errorf("minimum withdrawal amount must be at least $1")
		}
	}

	if req.MaxWithdrawalAmount != nil {
		if *req.MaxWithdrawalAmount < 1 {
			return fmt.Errorf("maximum withdrawal amount must be at least $1")
		}
	}

	if req.MinWithdrawalAmount != nil && req.MaxWithdrawalAmount != nil {
		if *req.MinWithdrawalAmount >= *req.MaxWithdrawalAmount {
			return fmt.Errorf("minimum withdrawal amount must be less than maximum")
		}
	}

	if req.WithdrawalProcessingDays != nil {
		if *req.WithdrawalProcessingDays < 1 || *req.WithdrawalProcessingDays > 30 {
			return fmt.Errorf("withdrawal processing days must be between 1 and 30")
		}
	}

	return nil
}