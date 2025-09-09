package services

import (
	"database/sql"
	"testing"
	"time"

	"event-ticketing-platform/internal/models"
	"event-ticketing-platform/internal/repositories"
	"event-ticketing-platform/internal/utils"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockUserRepository is a mock implementation of UserRepository
type MockUserRepository struct {
	mock.Mock
}

func (m *MockUserRepository) Create(req *models.UserCreateRequest) (*models.User, error) {
	args := m.Called(req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserRepository) GetByID(id int) (*models.User, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserRepository) GetByEmail(email string) (*models.User, error) {
	args := m.Called(email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserRepository) Update(id int, req *models.UserUpdateRequest) (*models.User, error) {
	args := m.Called(id, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserRepository) UpdatePassword(id int, passwordHash string) error {
	args := m.Called(id, passwordHash)
	return args.Error(0)
}

func (m *MockUserRepository) Delete(id int) error {
	args := m.Called(id)
	return args.Error(0)
}

func (m *MockUserRepository) Search(filters repositories.UserSearchFilters) ([]*models.User, int, error) {
	args := m.Called(filters)
	if args.Get(0) == nil {
		return nil, args.Int(1), args.Error(2)
	}
	return args.Get(0).([]*models.User), args.Int(1), args.Error(2)
}

func (m *MockUserRepository) GetByRole(role models.UserRole) ([]*models.User, error) {
	args := m.Called(role)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.User), args.Error(1)
}

func (m *MockUserRepository) CreateSession(userID int, sessionID string, expiresAt time.Time) error {
	args := m.Called(userID, sessionID, expiresAt)
	return args.Error(0)
}

func (m *MockUserRepository) GetUserBySession(sessionID string) (*models.User, error) {
	args := m.Called(sessionID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserRepository) DeleteSession(sessionID string) error {
	args := m.Called(sessionID)
	return args.Error(0)
}

func (m *MockUserRepository) DeleteExpiredSessions() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockUserRepository) DeleteUserSessions(userID int) error {
	args := m.Called(userID)
	return args.Error(0)
}

func (m *MockUserRepository) ExtendSession(sessionID string, expiresAt time.Time) error {
	args := m.Called(sessionID, expiresAt)
	return args.Error(0)
}

func (m *MockUserRepository) SetVerificationToken(userID int, token string) error {
	args := m.Called(userID, token)
	return args.Error(0)
}

func (m *MockUserRepository) GetByVerificationToken(token string) (*models.User, error) {
	args := m.Called(token)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserRepository) VerifyEmail(userID int) error {
	args := m.Called(userID)
	return args.Error(0)
}

func (m *MockUserRepository) SetPasswordResetToken(userID int, token string, expiresAt time.Time) error {
	args := m.Called(userID, token, expiresAt)
	return args.Error(0)
}

func (m *MockUserRepository) GetByPasswordResetToken(token string) (*models.User, error) {
	args := m.Called(token)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserRepository) ClearPasswordResetToken(userID int) error {
	args := m.Called(userID)
	return args.Error(0)
}

func (m *MockUserRepository) CleanupExpiredTokens() error {
	args := m.Called()
	return args.Error(0)
}

// MockEmailServiceForAuth is a mock implementation of EmailService for auth tests
type MockEmailServiceForAuth struct {
	mock.Mock
}

func (m *MockEmailServiceForAuth) SendPasswordResetEmail(email, token string) error {
	args := m.Called(email, token)
	return args.Error(0)
}

func (m *MockEmailServiceForAuth) SendWelcomeEmail(email, userName string) error {
	args := m.Called(email, userName)
	return args.Error(0)
}

func (m *MockEmailServiceForAuth) SendVerificationEmail(email, userName, token string) error {
	args := m.Called(email, userName, token)
	return args.Error(0)
}

func (m *MockEmailServiceForAuth) SendOrderConfirmationWithTickets(email, userName, subject, htmlContent, textContent string, order *models.Order, tickets []*models.Ticket) error {
	args := m.Called(email, userName, subject, htmlContent, textContent, order, tickets)
	return args.Error(0)
}

func TestAuthService_Register(t *testing.T) {
	tests := []struct {
		name          string
		request       *RegisterRequest
		setupMocks    func(*MockUserRepository, *MockEmailServiceForAuth)
		expectedError string
	}{
		{
			name: "successful registration",
			request: &RegisterRequest{
				Email:     "test@example.com",
				Password:  "password123",
				FirstName: "John",
				LastName:  "Doe",
				Role:      models.RoleAttendee,
			},
			setupMocks: func(userRepo *MockUserRepository, emailService *MockEmailServiceForAuth) {
				// User doesn't exist
				userRepo.On("GetByEmail", "test@example.com").Return(nil, sql.ErrNoRows)
				
				// Create user successfully
				user := &models.User{
					ID:        1,
					Email:     "test@example.com",
					FirstName: "John",
					LastName:  "Doe",
					Role:      models.RoleAttendee,
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				}
				userRepo.On("Create", mock.AnythingOfType("*models.UserCreateRequest")).Return(user, nil)
				userRepo.On("SetVerificationToken", 1, mock.AnythingOfType("string")).Return(nil)
				emailService.On("SendVerificationEmail", "test@example.com", "John Doe", mock.AnythingOfType("string")).Return(nil)
			},
		},
		{
			name: "user already exists",
			request: &RegisterRequest{
				Email:     "existing@example.com",
				Password:  "password123",
				FirstName: "John",
				LastName:  "Doe",
				Role:      models.RoleAttendee,
			},
			setupMocks: func(userRepo *MockUserRepository, emailService *MockEmailServiceForAuth) {
				// User already exists
				existingUser := &models.User{
					ID:    1,
					Email: "existing@example.com",
				}
				userRepo.On("GetByEmail", "existing@example.com").Return(existingUser, nil)
			},
			expectedError: "user with email existing@example.com already exists",
		},
		{
			name: "invalid email",
			request: &RegisterRequest{
				Email:     "invalid-email",
				Password:  "password123",
				FirstName: "John",
				LastName:  "Doe",
				Role:      models.RoleAttendee,
			},
			setupMocks:    func(userRepo *MockUserRepository, emailService *MockEmailServiceForAuth) {},
			expectedError: "validation failed",
		},
		{
			name: "password too short",
			request: &RegisterRequest{
				Email:     "test@example.com",
				Password:  "123",
				FirstName: "John",
				LastName:  "Doe",
				Role:      models.RoleAttendee,
			},
			setupMocks:    func(userRepo *MockUserRepository, emailService *MockEmailServiceForAuth) {},
			expectedError: "validation failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			userRepo := &MockUserRepository{}
			emailService := &MockEmailServiceForAuth{}
			tt.setupMocks(userRepo, emailService)

			authService := NewAuthService(userRepo, emailService)
			response, err := authService.Register(tt.request)

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
				assert.Nil(t, response)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, response)
				assert.Empty(t, response.SessionID) // No session for unverified users
				assert.Equal(t, tt.request.Email, response.User.Email)
			}

			userRepo.AssertExpectations(t)
			emailService.AssertExpectations(t)
		})
	}
}

func TestAuthService_Login(t *testing.T) {
	// Create a test password hash
	testPassword := "password123"
	hashedPassword, err := utils.HashPassword(testPassword)
	require.NoError(t, err)

	tests := []struct {
		name          string
		request       *LoginRequest
		setupMocks    func(*MockUserRepository, *MockEmailServiceForAuth)
		expectedError string
	}{
		{
			name: "successful login",
			request: &LoginRequest{
				Email:    "test@example.com",
				Password: testPassword,
			},
			setupMocks: func(userRepo *MockUserRepository, emailService *MockEmailServiceForAuth) {
				user := &models.User{
					ID:            1,
					Email:         "test@example.com",
					PasswordHash:  hashedPassword,
					FirstName:     "John",
					LastName:      "Doe",
					Role:          models.RoleAttendee,
					EmailVerified: true,
				}
				userRepo.On("GetByEmail", "test@example.com").Return(user, nil)
				userRepo.On("CreateSession", 1, mock.AnythingOfType("string"), mock.AnythingOfType("time.Time")).Return(nil)
			},
		},
		{
			name: "user not found",
			request: &LoginRequest{
				Email:    "nonexistent@example.com",
				Password: testPassword,
			},
			setupMocks: func(userRepo *MockUserRepository, emailService *MockEmailServiceForAuth) {
				userRepo.On("GetByEmail", "nonexistent@example.com").Return(nil, sql.ErrNoRows)
			},
			expectedError: "invalid email or password",
		},
		{
			name: "wrong password",
			request: &LoginRequest{
				Email:    "test@example.com",
				Password: "wrongpassword",
			},
			setupMocks: func(userRepo *MockUserRepository, emailService *MockEmailServiceForAuth) {
				user := &models.User{
					ID:            1,
					Email:         "test@example.com",
					PasswordHash:  hashedPassword,
					EmailVerified: true,
				}
				userRepo.On("GetByEmail", "test@example.com").Return(user, nil)
			},
			expectedError: "invalid email or password",
		},
		{
			name: "email not verified",
			request: &LoginRequest{
				Email:    "test@example.com",
				Password: testPassword,
			},
			setupMocks: func(userRepo *MockUserRepository, emailService *MockEmailServiceForAuth) {
				user := &models.User{
					ID:            1,
					Email:         "test@example.com",
					PasswordHash:  hashedPassword,
					EmailVerified: false,
				}
				userRepo.On("GetByEmail", "test@example.com").Return(user, nil)
			},
			expectedError: "please verify your email address",
		},
		{
			name: "empty email",
			request: &LoginRequest{
				Email:    "",
				Password: testPassword,
			},
			setupMocks:    func(userRepo *MockUserRepository, emailService *MockEmailServiceForAuth) {},
			expectedError: "email is required",
		},
		{
			name: "empty password",
			request: &LoginRequest{
				Email:    "test@example.com",
				Password: "",
			},
			setupMocks:    func(userRepo *MockUserRepository, emailService *MockEmailServiceForAuth) {},
			expectedError: "password is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			userRepo := &MockUserRepository{}
			emailService := &MockEmailServiceForAuth{}
			tt.setupMocks(userRepo, emailService)

			authService := NewAuthService(userRepo, emailService)
			response, err := authService.Login(tt.request)

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
				assert.Nil(t, response)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, response)
				assert.NotEmpty(t, response.SessionID)
				assert.NotZero(t, response.ExpiresAt)
				assert.Equal(t, tt.request.Email, response.User.Email)
			}

			userRepo.AssertExpectations(t)
			emailService.AssertExpectations(t)
		})
	}
}

func TestAuthService_ValidateSession(t *testing.T) {
	tests := []struct {
		name          string
		sessionID     string
		setupMocks    func(*MockUserRepository, *MockEmailServiceForAuth)
		expectedError string
	}{
		{
			name:      "valid session",
			sessionID: "valid-session-id",
			setupMocks: func(userRepo *MockUserRepository, emailService *MockEmailServiceForAuth) {
				user := &models.User{
					ID:        1,
					Email:     "test@example.com",
					FirstName: "John",
					LastName:  "Doe",
					Role:      models.RoleAttendee,
				}
				userRepo.On("GetUserBySession", "valid-session-id").Return(user, nil)
			},
		},
		{
			name:      "invalid session",
			sessionID: "invalid-session-id",
			setupMocks: func(userRepo *MockUserRepository, emailService *MockEmailServiceForAuth) {
				userRepo.On("GetUserBySession", "invalid-session-id").Return(nil, sql.ErrNoRows)
			},
			expectedError: "invalid or expired session",
		},
		{
			name:          "empty session ID",
			sessionID:     "",
			setupMocks:    func(userRepo *MockUserRepository, emailService *MockEmailServiceForAuth) {},
			expectedError: "session ID is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			userRepo := &MockUserRepository{}
			emailService := &MockEmailServiceForAuth{}
			tt.setupMocks(userRepo, emailService)

			authService := NewAuthService(userRepo, emailService)
			user, err := authService.ValidateSession(tt.sessionID)

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
				assert.Nil(t, user)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, user)
			}

			userRepo.AssertExpectations(t)
			emailService.AssertExpectations(t)
		})
	}
}

func TestAuthService_Logout(t *testing.T) {
	tests := []struct {
		name          string
		sessionID     string
		setupMocks    func(*MockUserRepository, *MockEmailServiceForAuth)
		expectedError string
	}{
		{
			name:      "successful logout",
			sessionID: "valid-session-id",
			setupMocks: func(userRepo *MockUserRepository, emailService *MockEmailServiceForAuth) {
				userRepo.On("DeleteSession", "valid-session-id").Return(nil)
			},
		},
		{
			name:          "empty session ID",
			sessionID:     "",
			setupMocks:    func(userRepo *MockUserRepository, emailService *MockEmailServiceForAuth) {},
			expectedError: "session ID is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			userRepo := &MockUserRepository{}
			emailService := &MockEmailServiceForAuth{}
			tt.setupMocks(userRepo, emailService)

			authService := NewAuthService(userRepo, emailService)
			err := authService.Logout(tt.sessionID)

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
			}

			userRepo.AssertExpectations(t)
			emailService.AssertExpectations(t)
		})
	}
}

func TestAuthService_ChangePassword(t *testing.T) {
	// Create test password hashes
	oldPassword := "oldpassword123"
	newPassword := "newpassword123"
	hashedOldPassword, err := utils.HashPassword(oldPassword)
	require.NoError(t, err)

	tests := []struct {
		name          string
		userID        int
		request       *PasswordChangeRequest
		setupMocks    func(*MockUserRepository, *MockEmailServiceForAuth)
		expectedError string
	}{
		{
			name:   "successful password change",
			userID: 1,
			request: &PasswordChangeRequest{
				OldPassword: oldPassword,
				NewPassword: newPassword,
			},
			setupMocks: func(userRepo *MockUserRepository, emailService *MockEmailServiceForAuth) {
				user := &models.User{
					ID:           1,
					PasswordHash: hashedOldPassword,
				}
				userRepo.On("GetByID", 1).Return(user, nil)
				userRepo.On("UpdatePassword", 1, mock.AnythingOfType("string")).Return(nil)
				userRepo.On("DeleteUserSessions", 1).Return(nil)
			},
		},
		{
			name:   "wrong old password",
			userID: 1,
			request: &PasswordChangeRequest{
				OldPassword: "wrongpassword",
				NewPassword: newPassword,
			},
			setupMocks: func(userRepo *MockUserRepository, emailService *MockEmailServiceForAuth) {
				user := &models.User{
					ID:           1,
					PasswordHash: hashedOldPassword,
				}
				userRepo.On("GetByID", 1).Return(user, nil)
			},
			expectedError: "old password is incorrect",
		},
		{
			name:   "user not found",
			userID: 999,
			request: &PasswordChangeRequest{
				OldPassword: oldPassword,
				NewPassword: newPassword,
			},
			setupMocks: func(userRepo *MockUserRepository, emailService *MockEmailServiceForAuth) {
				userRepo.On("GetByID", 999).Return(nil, sql.ErrNoRows)
			},
			expectedError: "user not found",
		},
		{
			name:   "empty old password",
			userID: 1,
			request: &PasswordChangeRequest{
				OldPassword: "",
				NewPassword: newPassword,
			},
			setupMocks:    func(userRepo *MockUserRepository, emailService *MockEmailServiceForAuth) {},
			expectedError: "old password is required",
		},
		{
			name:   "empty new password",
			userID: 1,
			request: &PasswordChangeRequest{
				OldPassword: oldPassword,
				NewPassword: "",
			},
			setupMocks:    func(userRepo *MockUserRepository, emailService *MockEmailServiceForAuth) {},
			expectedError: "new password is required",
		},
		{
			name:   "new password too short",
			userID: 1,
			request: &PasswordChangeRequest{
				OldPassword: oldPassword,
				NewPassword: "123",
			},
			setupMocks:    func(userRepo *MockUserRepository, emailService *MockEmailServiceForAuth) {},
			expectedError: "new password must be at least 8 characters long",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			userRepo := &MockUserRepository{}
			emailService := &MockEmailServiceForAuth{}
			tt.setupMocks(userRepo, emailService)

			authService := NewAuthService(userRepo, emailService)
			err := authService.ChangePassword(tt.userID, tt.request)

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
			}

			userRepo.AssertExpectations(t)
			emailService.AssertExpectations(t)
		})
	}
}

func TestAuthService_RequireRole(t *testing.T) {
	tests := []struct {
		name          string
		user          *models.User
		requiredRole  models.UserRole
		expectedError string
	}{
		{
			name: "user has required role",
			user: &models.User{
				Role: models.RoleOrganizer,
			},
			requiredRole: models.RoleOrganizer,
		},
		{
			name: "admin can access everything",
			user: &models.User{
				Role: models.RoleAdmin,
			},
			requiredRole: models.RoleOrganizer,
		},
		{
			name: "user doesn't have required role",
			user: &models.User{
				Role: models.RoleAttendee,
			},
			requiredRole:  models.RoleOrganizer,
			expectedError: "insufficient permissions",
		},
		{
			name:          "nil user",
			user:          nil,
			requiredRole:  models.RoleOrganizer,
			expectedError: "user is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			userRepo := &MockUserRepository{}
			emailService := &MockEmailServiceForAuth{}
			authService := NewAuthService(userRepo, emailService)

			err := authService.RequireRole(tt.user, tt.requiredRole)

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestAuthService_RequireRoles(t *testing.T) {
	tests := []struct {
		name          string
		user          *models.User
		requiredRoles []models.UserRole
		expectedError string
	}{
		{
			name: "user has one of required roles",
			user: &models.User{
				Role: models.RoleOrganizer,
			},
			requiredRoles: []models.UserRole{models.RoleOrganizer, models.RoleAdmin},
		},
		{
			name: "admin can access everything",
			user: &models.User{
				Role: models.RoleAdmin,
			},
			requiredRoles: []models.UserRole{models.RoleOrganizer},
		},
		{
			name: "user doesn't have any required role",
			user: &models.User{
				Role: models.RoleAttendee,
			},
			requiredRoles: []models.UserRole{models.RoleOrganizer, models.RoleAdmin},
			expectedError: "insufficient permissions",
		},
		{
			name:          "nil user",
			user:          nil,
			requiredRoles: []models.UserRole{models.RoleOrganizer},
			expectedError: "user is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			userRepo := &MockUserRepository{}
			emailService := &MockEmailServiceForAuth{}
			authService := NewAuthService(userRepo, emailService)

			err := authService.RequireRoles(tt.user, tt.requiredRoles...)

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestAuthService_RequestPasswordReset(t *testing.T) {
	tests := []struct {
		name          string
		request       *PasswordResetRequest
		setupMocks    func(*MockUserRepository, *MockEmailServiceForAuth)
		expectedError string
	}{
		{
			name: "successful password reset request",
			request: &PasswordResetRequest{
				Email: "test@example.com",
			},
			setupMocks: func(userRepo *MockUserRepository, emailService *MockEmailServiceForAuth) {
				user := &models.User{
					ID:    1,
					Email: "test@example.com",
				}
				userRepo.On("GetByEmail", "test@example.com").Return(user, nil)
				userRepo.On("SetPasswordResetToken", 1, mock.AnythingOfType("string"), mock.AnythingOfType("time.Time")).Return(nil)
				emailService.On("SendPasswordResetEmail", "test@example.com", mock.AnythingOfType("string")).Return(nil)
			},
		},
		{
			name: "user not found (should not error for security)",
			request: &PasswordResetRequest{
				Email: "nonexistent@example.com",
			},
			setupMocks: func(userRepo *MockUserRepository, emailService *MockEmailServiceForAuth) {
				userRepo.On("GetByEmail", "nonexistent@example.com").Return(nil, sql.ErrNoRows)
			},
		},
		{
			name: "empty email",
			request: &PasswordResetRequest{
				Email: "",
			},
			setupMocks:    func(userRepo *MockUserRepository, emailService *MockEmailServiceForAuth) {},
			expectedError: "email is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			userRepo := &MockUserRepository{}
			emailService := &MockEmailServiceForAuth{}
			tt.setupMocks(userRepo, emailService)

			authService := NewAuthService(userRepo, emailService)
			err := authService.RequestPasswordReset(tt.request)

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
			}

			userRepo.AssertExpectations(t)
			emailService.AssertExpectations(t)
		})
	}
}

func TestAuthService_CleanupExpiredSessions(t *testing.T) {
	tests := []struct {
		name          string
		setupMocks    func(*MockUserRepository, *MockEmailServiceForAuth)
		expectedError string
	}{
		{
			name: "successful cleanup",
			setupMocks: func(userRepo *MockUserRepository, emailService *MockEmailServiceForAuth) {
				userRepo.On("DeleteExpiredSessions").Return(nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			userRepo := &MockUserRepository{}
			emailService := &MockEmailServiceForAuth{}
			tt.setupMocks(userRepo, emailService)

			authService := NewAuthService(userRepo, emailService)
			err := authService.CleanupExpiredSessions()

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
			}

			userRepo.AssertExpectations(t)
			emailService.AssertExpectations(t)
		})
	}
}