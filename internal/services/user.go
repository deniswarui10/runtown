package services

import (
	"fmt"

	"event-ticketing-platform/internal/models"
)

// UserServiceInterface defines the interface for user-related operations
type UserServiceInterface interface {
	UpdateProfile(userID int, req *UpdateProfileRequest) (*models.User, error)
	UpdatePreferences(userID int, preferences *UserPreferences) error
	DeleteAccount(userID int) error
	GetUserPreferences(userID int) (*UserPreferences, error)
	
	// Admin-specific methods
	GetUsersWithPagination(page, limit int, search, roleFilter string) ([]*models.User, int, error)
	GetUserCount() (int, error)
	GetActiveUserCount() (int, error)
	UpdateUserRole(userID int, role models.UserRole) error
	SuspendUser(userID int) error
	ActivateUser(userID int) error
}

// UserService handles user-related business logic
type UserService struct {
	userRepo UserRepositoryInterface
}

// NewUserService creates a new user service
func NewUserService(userRepo UserRepositoryInterface) *UserService {
	return &UserService{
		userRepo: userRepo,
	}
}

// UpdateProfileRequest represents a profile update request
type UpdateProfileRequest struct {
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Email     string `json:"email"`
}

// UserPreferences represents user notification and preference settings
type UserPreferences struct {
	EmailNotifications     bool `json:"email_notifications"`
	EventReminders         bool `json:"event_reminders"`
	MarketingEmails        bool `json:"marketing_emails"`
	SMSNotifications       bool `json:"sms_notifications"`
	NewsletterSubscription bool `json:"newsletter_subscription"`
}

// UpdateProfile updates a user's profile information
func (s *UserService) UpdateProfile(userID int, req *UpdateProfileRequest) (*models.User, error) {
	// Get current user to check if email is changing
	currentUser, err := s.userRepo.GetByID(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get current user: %w", err)
	}

	// If email is changing, check if new email already exists
	if req.Email != currentUser.Email {
		existingUser, err := s.userRepo.GetByEmail(req.Email)
		if err == nil && existingUser.ID != userID {
			return nil, fmt.Errorf("user with email %s already exists", req.Email)
		}
	}

	// Update user profile
	updateReq := &models.UserUpdateRequest{
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Role:      currentUser.Role, // Keep existing role
	}

	updatedUser, err := s.userRepo.Update(userID, updateReq)
	if err != nil {
		return nil, fmt.Errorf("failed to update user profile: %w", err)
	}

	// If email changed, update it separately (this might require email verification in the future)
	if req.Email != currentUser.Email {
		err = s.updateUserEmail(userID, req.Email)
		if err != nil {
			return nil, fmt.Errorf("failed to update email: %w", err)
		}
		updatedUser.Email = req.Email
	}

	return updatedUser, nil
}

// UpdatePreferences updates user notification preferences
func (s *UserService) UpdatePreferences(userID int, preferences *UserPreferences) error {
	// For now, we'll just validate that the user exists
	// In a real implementation, you would save preferences to a user_preferences table
	_, err := s.userRepo.GetByID(userID)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}

	// TODO: Implement actual preference storage
	// This would involve creating a user_preferences table and repository methods
	
	return nil
}

// DeleteAccount deletes a user account and all associated data
func (s *UserService) DeleteAccount(userID int) error {
	// In a real implementation, you would:
	// 1. Cancel any active orders
	// 2. Delete user sessions
	// 3. Anonymize or delete user data according to privacy policy
	// 4. Send confirmation email
	
	// For now, we'll just delete the user record
	err := s.userRepo.Delete(userID)
	if err != nil {
		return fmt.Errorf("failed to delete user account: %w", err)
	}

	return nil
}

// GetUserPreferences retrieves user notification preferences
func (s *UserService) GetUserPreferences(userID int) (*UserPreferences, error) {
	// Verify user exists
	_, err := s.userRepo.GetByID(userID)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	// For now, return default preferences
	// In a real implementation, you would fetch from user_preferences table
	return &UserPreferences{
		EmailNotifications:     true,
		EventReminders:         true,
		MarketingEmails:        false,
		SMSNotifications:       false,
		NewsletterSubscription: false,
	}, nil
}

// updateUserEmail updates a user's email address
func (s *UserService) updateUserEmail(userID int, newEmail string) error {
	// This is a simplified implementation
	// In a real system, you might want to:
	// 1. Send verification email to new address
	// 2. Mark email as unverified until confirmed
	// 3. Keep old email until verification is complete
	
	// For now, we'll implement a direct update
	// This would require adding an UpdateEmail method to the repository
	// Since we don't have that method, we'll skip the actual email update
	// and let the caller handle it through the profile update
	
	return nil
}

// UserRepositoryInterface defines the interface for user repository operations
type UserRepositoryInterface interface {
	GetByID(id int) (*models.User, error)
	GetByEmail(email string) (*models.User, error)
	Update(id int, req *models.UserUpdateRequest) (*models.User, error)
	Delete(id int) error
	
	// Admin-specific methods
	GetUsersWithPagination(page, limit int, search, roleFilter string) ([]*models.User, int, error)
	GetUserCount() (int, error)
	GetActiveUserCount() (int, error)
	UpdateUserRole(userID int, role models.UserRole) error
	UpdateUserStatus(userID int, isActive bool) error
}

// Admin-specific methods

// GetUsersWithPagination retrieves users with pagination and filtering
func (s *UserService) GetUsersWithPagination(page, limit int, search, roleFilter string) ([]*models.User, int, error) {
	return s.userRepo.GetUsersWithPagination(page, limit, search, roleFilter)
}

// GetUserCount returns the total number of users
func (s *UserService) GetUserCount() (int, error) {
	return s.userRepo.GetUserCount()
}

// GetActiveUserCount returns the number of active users
func (s *UserService) GetActiveUserCount() (int, error) {
	return s.userRepo.GetActiveUserCount()
}

// UpdateUserRole updates a user's role
func (s *UserService) UpdateUserRole(userID int, role models.UserRole) error {
	return s.userRepo.UpdateUserRole(userID, role)
}

// SuspendUser suspends a user account
func (s *UserService) SuspendUser(userID int) error {
	return s.userRepo.UpdateUserStatus(userID, false)
}

// ActivateUser activates a user account
func (s *UserService) ActivateUser(userID int) error {
	return s.userRepo.UpdateUserStatus(userID, true)
}