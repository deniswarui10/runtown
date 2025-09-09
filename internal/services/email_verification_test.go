package services

import (
	"fmt"
	"testing"
	"time"

	"event-ticketing-platform/internal/models"
	"event-ticketing-platform/internal/repositories"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockUserRepositoryForEmailVerification for testing email verification
type MockUserRepositoryForEmailVerification struct {
	mock.Mock
}

func (m *MockUserRepositoryForEmailVerification) Create(req *models.UserCreateRequest) (*models.User, error) {
	args := m.Called(req)
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserRepositoryForEmailVerification) GetByID(id int) (*models.User, error) {
	args := m.Called(id)
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserRepositoryForEmailVerification) GetByEmail(email string) (*models.User, error) {
	args := m.Called(email)
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserRepositoryForEmailVerification) Update(id int, req *models.UserUpdateRequest) (*models.User, error) {
	args := m.Called(id, req)
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserRepositoryForEmailVerification) UpdatePassword(id int, passwordHash string) error {
	args := m.Called(id, passwordHash)
	return args.Error(0)
}

func (m *MockUserRepositoryForEmailVerification) Delete(id int) error {
	args := m.Called(id)
	return args.Error(0)
}

func (m *MockUserRepositoryForEmailVerification) Search(filters repositories.UserSearchFilters) ([]*models.User, int, error) {
	args := m.Called(filters)
	return args.Get(0).([]*models.User), args.Int(1), args.Error(2)
}

func (m *MockUserRepositoryForEmailVerification) GetByRole(role models.UserRole) ([]*models.User, error) {
	args := m.Called(role)
	return args.Get(0).([]*models.User), args.Error(1)
}

func (m *MockUserRepositoryForEmailVerification) CreateSession(userID int, sessionID string, expiresAt time.Time) error {
	args := m.Called(userID, sessionID, expiresAt)
	return args.Error(0)
}

func (m *MockUserRepositoryForEmailVerification) GetUserBySession(sessionID string) (*models.User, error) {
	args := m.Called(sessionID)
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserRepositoryForEmailVerification) DeleteSession(sessionID string) error {
	args := m.Called(sessionID)
	return args.Error(0)
}

func (m *MockUserRepositoryForEmailVerification) DeleteExpiredSessions() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockUserRepositoryForEmailVerification) DeleteUserSessions(userID int) error {
	args := m.Called(userID)
	return args.Error(0)
}

func (m *MockUserRepositoryForEmailVerification) ExtendSession(sessionID string, expiresAt time.Time) error {
	args := m.Called(sessionID, expiresAt)
	return args.Error(0)
}

func (m *MockUserRepositoryForEmailVerification) SetVerificationToken(userID int, token string) error {
	args := m.Called(userID, token)
	return args.Error(0)
}

func (m *MockUserRepositoryForEmailVerification) GetByVerificationToken(token string) (*models.User, error) {
	args := m.Called(token)
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserRepositoryForEmailVerification) VerifyEmail(userID int) error {
	args := m.Called(userID)
	return args.Error(0)
}

func (m *MockUserRepositoryForEmailVerification) SetPasswordResetToken(userID int, token string, expiresAt time.Time) error {
	args := m.Called(userID, token, expiresAt)
	return args.Error(0)
}

func (m *MockUserRepositoryForEmailVerification) GetByPasswordResetToken(token string) (*models.User, error) {
	args := m.Called(token)
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserRepositoryForEmailVerification) ClearPasswordResetToken(userID int) error {
	args := m.Called(userID)
	return args.Error(0)
}

func (m *MockUserRepositoryForEmailVerification) CleanupExpiredTokens() error {
	args := m.Called()
	return args.Error(0)
}

// MockEmailServiceForVerification for testing email verification
type MockEmailServiceForVerification struct {
	mock.Mock
}

func (m *MockEmailServiceForVerification) SendPasswordResetEmail(email, token string) error {
	args := m.Called(email, token)
	return args.Error(0)
}

func (m *MockEmailServiceForVerification) SendWelcomeEmail(email, userName string) error {
	args := m.Called(email, userName)
	return args.Error(0)
}

func (m *MockEmailServiceForVerification) SendVerificationEmail(email, userName, token string) error {
	args := m.Called(email, userName, token)
	return args.Error(0)
}

func (m *MockEmailServiceForVerification) SendOrderConfirmationWithTickets(email, userName, subject, htmlContent, textContent string, order *models.Order, tickets []*models.Ticket) error {
	args := m.Called(email, userName, subject, htmlContent, textContent, order, tickets)
	return args.Error(0)
}

func createTestUserForEmailVerification() *models.User {
	return &models.User{
		ID:            1,
		Email:         "test@example.com",
		FirstName:     "Test",
		LastName:      "User",
		Role:          models.RoleAttendee,
		EmailVerified: false,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
}

func TestAuthService_VerifyEmail(t *testing.T) {
	tests := []struct {
		name        string
		token       string
		setupMocks  func(*MockUserRepositoryForEmailVerification, *MockEmailServiceForVerification)
		expectError bool
		errorMsg    string
	}{
		{
			name:  "successful email verification",
			token: "valid-token-123",
			setupMocks: func(userRepo *MockUserRepositoryForEmailVerification, emailService *MockEmailServiceForVerification) {
				user := createTestUserForEmailVerification()
				userRepo.On("GetByVerificationToken", "valid-token-123").Return(user, nil)
				userRepo.On("VerifyEmail", 1).Return(nil)
				userRepo.On("GetByID", 1).Return(user, nil)
				emailService.On("SendWelcomeEmail", "test@example.com", "Test User").Return(nil)
			},
			expectError: false,
		},
		{
			name:  "empty token",
			token: "",
			setupMocks: func(userRepo *MockUserRepositoryForEmailVerification, emailService *MockEmailServiceForVerification) {
				// No mocks needed for validation error
			},
			expectError: true,
			errorMsg:    "verification token is required",
		},
		{
			name:  "invalid token",
			token: "invalid-token",
			setupMocks: func(userRepo *MockUserRepositoryForEmailVerification, emailService *MockEmailServiceForVerification) {
				userRepo.On("GetByVerificationToken", "invalid-token").Return((*models.User)(nil), fmt.Errorf("user not found"))
			},
			expectError: true,
			errorMsg:    "invalid or expired verification token",
		},
		{
			name:  "already verified email",
			token: "valid-token-123",
			setupMocks: func(userRepo *MockUserRepositoryForEmailVerification, emailService *MockEmailServiceForVerification) {
				user := createTestUserForEmailVerification()
				user.EmailVerified = true
				userRepo.On("GetByVerificationToken", "valid-token-123").Return(user, nil)
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mocks
			mockUserRepo := &MockUserRepositoryForEmailVerification{}
			mockEmailService := &MockEmailServiceForVerification{}
			tt.setupMocks(mockUserRepo, mockEmailService)

			// Create service
			authService := NewAuthService(mockUserRepo, mockEmailService)

			// Call method
			user, err := authService.VerifyEmail(tt.token)

			// Check results
			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
				assert.Nil(t, user)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, user)
			}

			// Verify mocks
			mockUserRepo.AssertExpectations(t)
			mockEmailService.AssertExpectations(t)
		})
	}
}

func TestAuthService_ResendVerificationEmail(t *testing.T) {
	tests := []struct {
		name        string
		email       string
		setupMocks  func(*MockUserRepositoryForEmailVerification, *MockEmailServiceForVerification)
		expectError bool
		errorMsg    string
	}{
		{
			name:  "successful resend",
			email: "test@example.com",
			setupMocks: func(userRepo *MockUserRepositoryForEmailVerification, emailService *MockEmailServiceForVerification) {
				user := createTestUserForEmailVerification()
				user.UpdatedAt = time.Now().Add(-10 * time.Minute) // Old enough to allow resend
				userRepo.On("GetByEmail", "test@example.com").Return(user, nil)
				userRepo.On("SetVerificationToken", 1, mock.AnythingOfType("string")).Return(nil)
				emailService.On("SendVerificationEmail", "test@example.com", "Test User", mock.AnythingOfType("string")).Return(nil)
			},
			expectError: false,
		},
		{
			name:  "empty email",
			email: "",
			setupMocks: func(userRepo *MockUserRepositoryForEmailVerification, emailService *MockEmailServiceForVerification) {
				// No mocks needed for validation error
			},
			expectError: true,
			errorMsg:    "email is required",
		},
		{
			name:  "user not found",
			email: "notfound@example.com",
			setupMocks: func(userRepo *MockUserRepositoryForEmailVerification, emailService *MockEmailServiceForVerification) {
				userRepo.On("GetByEmail", "notfound@example.com").Return((*models.User)(nil), fmt.Errorf("user not found"))
			},
			expectError: true,
			errorMsg:    "user not found",
		},
		{
			name:  "already verified",
			email: "verified@example.com",
			setupMocks: func(userRepo *MockUserRepositoryForEmailVerification, emailService *MockEmailServiceForVerification) {
				user := createTestUserForEmailVerification()
				user.EmailVerified = true
				userRepo.On("GetByEmail", "verified@example.com").Return(user, nil)
			},
			expectError: true,
			errorMsg:    "email is already verified",
		},
		{
			name:  "rate limited",
			email: "ratelimited@example.com",
			setupMocks: func(userRepo *MockUserRepositoryForEmailVerification, emailService *MockEmailServiceForVerification) {
				user := createTestUserForEmailVerification()
				user.UpdatedAt = time.Now().Add(-2 * time.Minute) // Too recent
				userRepo.On("GetByEmail", "ratelimited@example.com").Return(user, nil)
			},
			expectError: true,
			errorMsg:    "wait at least 5 minutes",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mocks
			mockUserRepo := &MockUserRepositoryForEmailVerification{}
			mockEmailService := &MockEmailServiceForVerification{}
			tt.setupMocks(mockUserRepo, mockEmailService)

			// Create service
			authService := NewAuthService(mockUserRepo, mockEmailService)

			// Call method
			err := authService.ResendVerificationEmail(tt.email)

			// Check results
			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}

			// Verify mocks
			mockUserRepo.AssertExpectations(t)
			mockEmailService.AssertExpectations(t)
		})
	}
}

func TestAuthService_CompletePasswordReset(t *testing.T) {
	tests := []struct {
		name        string
		request     *PasswordResetCompleteRequest
		setupMocks  func(*MockUserRepositoryForEmailVerification, *MockEmailServiceForVerification)
		expectError bool
		errorMsg    string
	}{
		{
			name: "successful password reset",
			request: &PasswordResetCompleteRequest{
				Token:       "valid-reset-token",
				NewPassword: "newpassword123",
			},
			setupMocks: func(userRepo *MockUserRepositoryForEmailVerification, emailService *MockEmailServiceForVerification) {
				user := createTestUserForEmailVerification()
				userRepo.On("GetByPasswordResetToken", "valid-reset-token").Return(user, nil)
				userRepo.On("UpdatePassword", 1, mock.AnythingOfType("string")).Return(nil)
				userRepo.On("ClearPasswordResetToken", 1).Return(nil)
				userRepo.On("DeleteUserSessions", 1).Return(nil)
			},
			expectError: false,
		},
		{
			name: "empty token",
			request: &PasswordResetCompleteRequest{
				Token:       "",
				NewPassword: "newpassword123",
			},
			setupMocks: func(userRepo *MockUserRepositoryForEmailVerification, emailService *MockEmailServiceForVerification) {
				// No mocks needed for validation error
			},
			expectError: true,
			errorMsg:    "reset token is required",
		},
		{
			name: "empty password",
			request: &PasswordResetCompleteRequest{
				Token:       "valid-token",
				NewPassword: "",
			},
			setupMocks: func(userRepo *MockUserRepositoryForEmailVerification, emailService *MockEmailServiceForVerification) {
				// No mocks needed for validation error
			},
			expectError: true,
			errorMsg:    "new password is required",
		},
		{
			name: "password too short",
			request: &PasswordResetCompleteRequest{
				Token:       "valid-token",
				NewPassword: "short",
			},
			setupMocks: func(userRepo *MockUserRepositoryForEmailVerification, emailService *MockEmailServiceForVerification) {
				// No mocks needed for validation error
			},
			expectError: true,
			errorMsg:    "password must be at least 8 characters long",
		},
		{
			name: "invalid token",
			request: &PasswordResetCompleteRequest{
				Token:       "invalid-token",
				NewPassword: "newpassword123",
			},
			setupMocks: func(userRepo *MockUserRepositoryForEmailVerification, emailService *MockEmailServiceForVerification) {
				userRepo.On("GetByPasswordResetToken", "invalid-token").Return((*models.User)(nil), fmt.Errorf("user not found"))
			},
			expectError: true,
			errorMsg:    "invalid or expired reset token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mocks
			mockUserRepo := &MockUserRepositoryForEmailVerification{}
			mockEmailService := &MockEmailServiceForVerification{}
			tt.setupMocks(mockUserRepo, mockEmailService)

			// Create service
			authService := NewAuthService(mockUserRepo, mockEmailService)

			// Call method
			err := authService.CompletePasswordReset(tt.request)

			// Check results
			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}

			// Verify mocks
			mockUserRepo.AssertExpectations(t)
			mockEmailService.AssertExpectations(t)
		})
	}
}

func TestAuthService_ValidatePasswordResetToken(t *testing.T) {
	tests := []struct {
		name        string
		token       string
		setupMocks  func(*MockUserRepositoryForEmailVerification, *MockEmailServiceForVerification)
		expectError bool
		errorMsg    string
	}{
		{
			name:  "valid token",
			token: "valid-token",
			setupMocks: func(userRepo *MockUserRepositoryForEmailVerification, emailService *MockEmailServiceForVerification) {
				user := createTestUserForEmailVerification()
				userRepo.On("GetByPasswordResetToken", "valid-token").Return(user, nil)
			},
			expectError: false,
		},
		{
			name:  "empty token",
			token: "",
			setupMocks: func(userRepo *MockUserRepositoryForEmailVerification, emailService *MockEmailServiceForVerification) {
				// No mocks needed for validation error
			},
			expectError: true,
			errorMsg:    "reset token is required",
		},
		{
			name:  "invalid token",
			token: "invalid-token",
			setupMocks: func(userRepo *MockUserRepositoryForEmailVerification, emailService *MockEmailServiceForVerification) {
				userRepo.On("GetByPasswordResetToken", "invalid-token").Return((*models.User)(nil), fmt.Errorf("user not found"))
			},
			expectError: true,
			errorMsg:    "invalid or expired reset token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mocks
			mockUserRepo := &MockUserRepositoryForEmailVerification{}
			mockEmailService := &MockEmailServiceForVerification{}
			tt.setupMocks(mockUserRepo, mockEmailService)

			// Create service
			authService := NewAuthService(mockUserRepo, mockEmailService)

			// Call method
			user, err := authService.ValidatePasswordResetToken(tt.token)

			// Check results
			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
				assert.Nil(t, user)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, user)
			}

			// Verify mocks
			mockUserRepo.AssertExpectations(t)
			mockEmailService.AssertExpectations(t)
		})
	}
}

func TestAuthService_CleanupExpiredTokens(t *testing.T) {
	// Setup mocks
	mockUserRepo := &MockUserRepositoryForEmailVerification{}
	mockEmailService := &MockEmailServiceForVerification{}
	
	mockUserRepo.On("CleanupExpiredTokens").Return(nil)

	// Create service
	authService := NewAuthService(mockUserRepo, mockEmailService)

	// Call method
	err := authService.CleanupExpiredTokens()

	// Check results
	assert.NoError(t, err)

	// Verify mocks
	mockUserRepo.AssertExpectations(t)
	mockEmailService.AssertExpectations(t)
}