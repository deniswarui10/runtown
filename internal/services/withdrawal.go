package services

import (
	"fmt"

	"event-ticketing-platform/internal/models"
	"event-ticketing-platform/internal/repositories"
)

// WithdrawalService handles withdrawal business logic
type WithdrawalService struct {
	withdrawalRepo *repositories.WithdrawalRepository
}

// NewWithdrawalService creates a new withdrawal service
func NewWithdrawalService(withdrawalRepo *repositories.WithdrawalRepository) *WithdrawalService {
	return &WithdrawalService{
		withdrawalRepo: withdrawalRepo,
	}
}

// CreateWithdrawal creates a new withdrawal request
func (s *WithdrawalService) CreateWithdrawal(organizerID int, req *models.WithdrawalCreateRequest) (*models.Withdrawal, error) {
	// Validate request
	if err := req.Validate(); err != nil {
		return nil, err
	}

	// Check available balance
	availableBalance, err := s.withdrawalRepo.GetOrganizerBalance(organizerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get organizer balance: %w", err)
	}

	if req.Amount > availableBalance {
		return nil, fmt.Errorf("insufficient balance: requested %.2f, available %.2f", req.Amount, availableBalance)
	}

	// Create withdrawal
	return s.withdrawalRepo.Create(organizerID, req)
}

// GetWithdrawalByID retrieves a withdrawal by ID
func (s *WithdrawalService) GetWithdrawalByID(id int) (*models.Withdrawal, error) {
	return s.withdrawalRepo.GetByID(id)
}

// GetOrganizerWithdrawals retrieves withdrawals for an organizer
func (s *WithdrawalService) GetOrganizerWithdrawals(organizerID int, page, limit int) ([]*models.Withdrawal, int, error) {
	offset := (page - 1) * limit
	return s.withdrawalRepo.GetByOrganizer(organizerID, limit, offset)
}

// GetAllWithdrawals retrieves all withdrawals (for admin)
func (s *WithdrawalService) GetAllWithdrawals(page, limit int, status string) ([]*models.Withdrawal, int, error) {
	offset := (page - 1) * limit
	return s.withdrawalRepo.GetAll(limit, offset, status)
}

// UpdateWithdrawalStatus updates the status of a withdrawal (admin only)
func (s *WithdrawalService) UpdateWithdrawalStatus(id int, status models.WithdrawalStatus, adminNotes string) error {
	return s.withdrawalRepo.UpdateStatus(id, status, adminNotes)
}

// GetOrganizerBalance gets the available balance for an organizer
func (s *WithdrawalService) GetOrganizerBalance(organizerID int) (float64, error) {
	return s.withdrawalRepo.GetOrganizerBalance(organizerID)
}

// CanUserAccessWithdrawal checks if a user can access a specific withdrawal
func (s *WithdrawalService) CanUserAccessWithdrawal(withdrawalID int, userID int, userRole models.UserRole) (bool, error) {
	withdrawal, err := s.withdrawalRepo.GetByID(withdrawalID)
	if err != nil {
		return false, err
	}

	// Admin can access all withdrawals
	if userRole == models.UserRoleAdmin {
		return true, nil
	}

	// Organizer can only access their own withdrawals
	if userRole == models.UserRoleOrganizer && withdrawal.OrganizerID == userID {
		return true, nil
	}

	return false, nil
}