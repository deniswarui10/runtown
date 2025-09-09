package handlers

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"event-ticketing-platform/internal/models"
	"event-ticketing-platform/internal/services"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/sessions"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockAuthServiceForEmailVerification for testing email verification
type MockAuthServiceForEmailVerification struct {
	mock.Mock
}

func (m *MockAuthServiceForEmailVerification) Register(req *services.RegisterRequest) (*services.AuthResponse, error) {
	args := m.Called(req)
	return args.Get(0).(*services.AuthResponse), args.Error(1)
}

func (m *MockAuthServiceForEmailVerification) Login(req *services.LoginRequest) (*services.AuthResponse, error) {
	args := m.Called(req)
	return args.Get(0).(*services.AuthResponse), args.Error(1)
}

func (m *MockAuthServiceForEmailVerification) ValidateSession(sessionID string) (*models.User, error) {
	args := m.Called(sessionID)
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockAuthServiceForEmailVerification) Logout(sessionID string) error {
	args := m.Called(sessionID)
	return args.Error(0)
}

func (m *MockAuthServiceForEmailVerification) RequestPasswordReset(req *services.PasswordResetRequest) error {
	args := m.Called(req)
	return args.Error(0)
}

func (m *MockAuthServiceForEmailVerification) ChangePassword(userID int, req *services.PasswordChangeRequest) error {
	args := m.Called(userID, req)
	return args.Error(0)
}

func (m *MockAuthServiceForEmailVerification) RequireRole(user *models.User, requiredRole models.UserRole) error {
	args := m.Called(user, requiredRole)
	return args.Error(0)
}

func (m *MockAuthServiceForEmailVerification) RequireRoles(user *models.User, requiredRoles ...models.UserRole) error {
	args := m.Called(user, requiredRoles)
	return args.Error(0)
}

func (m *MockAuthServiceForEmailVerification) CleanupExpiredSessions() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockAuthServiceForEmailVerification) VerifyEmail(token string) (*models.User, error) {
	args := m.Called(token)
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockAuthServiceForEmailVerification) ExtendSession(sessionID string, duration time.Duration) error {
	args := m.Called(sessionID, duration)
	return args.Error(0)
}

func (m *MockAuthServiceForEmailVerification) LogoutAllSessions(userID int) error {
	args := m.Called(userID)
	return args.Error(0)
}

func (m *MockAuthServiceForEmailVerification) ResendVerificationEmail(email string) error {
	args := m.Called(email)
	return args.Error(0)
}

func (m *MockAuthServiceForEmailVerification) CompletePasswordReset(req *services.PasswordResetCompleteRequest) error {
	args := m.Called(req)
	return args.Error(0)
}

func (m *MockAuthServiceForEmailVerification) ValidatePasswordResetToken(token string) (*models.User, error) {
	args := m.Called(token)
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockAuthServiceForEmailVerification) CleanupExpiredTokens() error {
	args := m.Called()
	return args.Error(0)
}

func createTestUserForVerification() *models.User {
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

func createVerifiedTestUser() *models.User {
	user := createTestUserForVerification()
	user.EmailVerified = true
	verifiedAt := time.Now()
	user.EmailVerifiedAt = &verifiedAt
	return user
}

func TestAuthHandler_VerifyEmail(t *testing.T) {
	tests := []struct {
		name           string
		token          string
		setupMocks     func(*MockAuthServiceForEmailVerification)
		expectedStatus int
		expectedBody   string
	}{
		{
			name:  "successful email verification",
			token: "valid-token-123",
			setupMocks: func(authService *MockAuthServiceForEmailVerification) {
				user := createVerifiedTestUser()
				authService.On("VerifyEmail", "valid-token-123").Return(user, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   "Email Verified!",
		},
		{
			name:  "missing token",
			token: "",
			setupMocks: func(authService *MockAuthServiceForEmailVerification) {
				// No mocks needed for missing token
			},
			expectedStatus: http.StatusOK,
			expectedBody:   "Verification Failed",
		},
		{
			name:  "invalid token",
			token: "invalid-token",
			setupMocks: func(authService *MockAuthServiceForEmailVerification) {
				authService.On("VerifyEmail", "invalid-token").Return((*models.User)(nil), fmt.Errorf("invalid or expired verification token"))
			},
			expectedStatus: http.StatusOK,
			expectedBody:   "Verification Failed",
		},
		{
			name:  "expired token",
			token: "expired-token",
			setupMocks: func(authService *MockAuthServiceForEmailVerification) {
				authService.On("VerifyEmail", "expired-token").Return((*models.User)(nil), fmt.Errorf("invalid or expired verification token"))
			},
			expectedStatus: http.StatusOK,
			expectedBody:   "Verification Failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mocks
			mockAuthService := &MockAuthServiceForEmailVerification{}
			tt.setupMocks(mockAuthService)

			// Create handler
			store := sessions.NewCookieStore([]byte("test-secret"))
			handler := NewAuthHandler(mockAuthService, store)

			// Create router
			r := chi.NewRouter()
			r.Get("/auth/verify", handler.VerifyEmail)

			// Create request
			reqURL := "/auth/verify"
			if tt.token != "" {
				reqURL += "?token=" + tt.token
			}
			req := httptest.NewRequest("GET", reqURL, nil)

			// Create response recorder
			rr := httptest.NewRecorder()

			// Call handler
			r.ServeHTTP(rr, req)

			// Check status
			assert.Equal(t, tt.expectedStatus, rr.Code)

			// Check body contains expected content
			if tt.expectedBody != "" {
				assert.Contains(t, rr.Body.String(), tt.expectedBody)
			}

			// Verify mocks
			mockAuthService.AssertExpectations(t)
		})
	}
}

func TestAuthHandler_ResendVerificationEmail(t *testing.T) {
	tests := []struct {
		name           string
		email          string
		setupMocks     func(*MockAuthServiceForEmailVerification)
		expectedStatus int
		expectedBody   string
	}{
		{
			name:  "successful resend",
			email: "test@example.com",
			setupMocks: func(authService *MockAuthServiceForEmailVerification) {
				authService.On("ResendVerificationEmail", "test@example.com").Return(nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   "Email Sent!",
		},
		{
			name:  "missing email",
			email: "",
			setupMocks: func(authService *MockAuthServiceForEmailVerification) {
				// No mocks needed for validation error
			},
			expectedStatus: http.StatusUnprocessableEntity,
			expectedBody:   "Email is required",
		},
		{
			name:  "invalid email format",
			email: "invalid-email",
			setupMocks: func(authService *MockAuthServiceForEmailVerification) {
				// No mocks needed for validation error
			},
			expectedStatus: http.StatusUnprocessableEntity,
			expectedBody:   "Please enter a valid email address",
		},
		{
			name:  "user not found",
			email: "notfound@example.com",
			setupMocks: func(authService *MockAuthServiceForEmailVerification) {
				authService.On("ResendVerificationEmail", "notfound@example.com").Return(fmt.Errorf("user not found"))
			},
			expectedStatus: http.StatusUnprocessableEntity,
			expectedBody:   "No account found with this email address",
		},
		{
			name:  "already verified",
			email: "verified@example.com",
			setupMocks: func(authService *MockAuthServiceForEmailVerification) {
				authService.On("ResendVerificationEmail", "verified@example.com").Return(fmt.Errorf("email is already verified"))
			},
			expectedStatus: http.StatusUnprocessableEntity,
			expectedBody:   "This email is already verified",
		},
		{
			name:  "rate limited",
			email: "ratelimited@example.com",
			setupMocks: func(authService *MockAuthServiceForEmailVerification) {
				authService.On("ResendVerificationEmail", "ratelimited@example.com").Return(fmt.Errorf("please wait at least 5 minutes before requesting another verification email"))
			},
			expectedStatus: http.StatusUnprocessableEntity,
			expectedBody:   "Please wait at least 5 minutes",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mocks
			mockAuthService := &MockAuthServiceForEmailVerification{}
			tt.setupMocks(mockAuthService)

			// Create handler
			store := sessions.NewCookieStore([]byte("test-secret"))
			handler := NewAuthHandler(mockAuthService, store)

			// Create router
			r := chi.NewRouter()
			r.Post("/auth/resend-verification", handler.ResendVerificationSubmit)

			// Create form data
			formData := url.Values{}
			formData.Set("email", tt.email)

			// Create request
			req := httptest.NewRequest("POST", "/auth/resend-verification", strings.NewReader(formData.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

			// Create response recorder
			rr := httptest.NewRecorder()

			// Call handler
			r.ServeHTTP(rr, req)

			// Check status
			assert.Equal(t, tt.expectedStatus, rr.Code)

			// Check body contains expected content
			if tt.expectedBody != "" {
				assert.Contains(t, rr.Body.String(), tt.expectedBody)
			}

			// Verify mocks
			mockAuthService.AssertExpectations(t)
		})
	}
}

func TestAuthHandler_ResetPasswordPage(t *testing.T) {
	tests := []struct {
		name           string
		token          string
		setupMocks     func(*MockAuthServiceForEmailVerification)
		expectedStatus int
		expectedBody   string
	}{
		{
			name:  "valid token shows reset form",
			token: "valid-reset-token",
			setupMocks: func(authService *MockAuthServiceForEmailVerification) {
				user := createTestUserForVerification()
				authService.On("ValidatePasswordResetToken", "valid-reset-token").Return(user, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   "Set New Password",
		},
		{
			name:  "missing token shows error",
			token: "",
			setupMocks: func(authService *MockAuthServiceForEmailVerification) {
				// No mocks needed for missing token
			},
			expectedStatus: http.StatusOK,
			expectedBody:   "Reset token is required",
		},
		{
			name:  "invalid token shows error",
			token: "invalid-token",
			setupMocks: func(authService *MockAuthServiceForEmailVerification) {
				authService.On("ValidatePasswordResetToken", "invalid-token").Return((*models.User)(nil), fmt.Errorf("invalid or expired reset token"))
			},
			expectedStatus: http.StatusOK,
			expectedBody:   "Invalid or expired reset token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mocks
			mockAuthService := &MockAuthServiceForEmailVerification{}
			tt.setupMocks(mockAuthService)

			// Create handler
			store := sessions.NewCookieStore([]byte("test-secret"))
			handler := NewAuthHandler(mockAuthService, store)

			// Create router
			r := chi.NewRouter()
			r.Get("/auth/reset-password", handler.ResetPasswordPage)

			// Create request
			reqURL := "/auth/reset-password"
			if tt.token != "" {
				reqURL += "?token=" + tt.token
			}
			req := httptest.NewRequest("GET", reqURL, nil)

			// Create response recorder
			rr := httptest.NewRecorder()

			// Call handler
			r.ServeHTTP(rr, req)

			// Check status
			assert.Equal(t, tt.expectedStatus, rr.Code)

			// Check body contains expected content
			if tt.expectedBody != "" {
				assert.Contains(t, rr.Body.String(), tt.expectedBody)
			}

			// Verify mocks
			mockAuthService.AssertExpectations(t)
		})
	}
}

func TestAuthHandler_ResetPasswordSubmit(t *testing.T) {
	tests := []struct {
		name            string
		token           string
		newPassword     string
		confirmPassword string
		setupMocks      func(*MockAuthServiceForEmailVerification)
		expectedStatus  int
		expectedBody    string
	}{
		{
			name:            "successful password reset",
			token:           "valid-token",
			newPassword:     "newpassword123",
			confirmPassword: "newpassword123",
			setupMocks: func(authService *MockAuthServiceForEmailVerification) {
				req := &services.PasswordResetCompleteRequest{
					Token:       "valid-token",
					NewPassword: "newpassword123",
				}
				authService.On("CompletePasswordReset", req).Return(nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   "Password Reset Successful!",
		},
		{
			name:            "missing token",
			token:           "",
			newPassword:     "newpassword123",
			confirmPassword: "newpassword123",
			setupMocks: func(authService *MockAuthServiceForEmailVerification) {
				// No mocks needed for validation error
			},
			expectedStatus: http.StatusUnprocessableEntity,
			expectedBody:   "Reset token is required",
		},
		{
			name:            "password too short",
			token:           "valid-token",
			newPassword:     "short",
			confirmPassword: "short",
			setupMocks: func(authService *MockAuthServiceForEmailVerification) {
				// No mocks needed for validation error
			},
			expectedStatus: http.StatusUnprocessableEntity,
			expectedBody:   "Password must be at least 8 characters long",
		},
		{
			name:            "passwords don't match",
			token:           "valid-token",
			newPassword:     "newpassword123",
			confirmPassword: "differentpassword",
			setupMocks: func(authService *MockAuthServiceForEmailVerification) {
				// No mocks needed for validation error
			},
			expectedStatus: http.StatusUnprocessableEntity,
			expectedBody:   "Passwords do not match",
		},
		{
			name:            "invalid token during reset",
			token:           "invalid-token",
			newPassword:     "newpassword123",
			confirmPassword: "newpassword123",
			setupMocks: func(authService *MockAuthServiceForEmailVerification) {
				req := &services.PasswordResetCompleteRequest{
					Token:       "invalid-token",
					NewPassword: "newpassword123",
				}
				authService.On("CompletePasswordReset", req).Return(fmt.Errorf("invalid or expired reset token"))
			},
			expectedStatus: http.StatusUnprocessableEntity,
			expectedBody:   "Invalid or expired reset token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mocks
			mockAuthService := &MockAuthServiceForEmailVerification{}
			tt.setupMocks(mockAuthService)

			// Create handler
			store := sessions.NewCookieStore([]byte("test-secret"))
			handler := NewAuthHandler(mockAuthService, store)

			// Create router
			r := chi.NewRouter()
			r.Post("/auth/reset-password", handler.ResetPasswordSubmit)

			// Create form data
			formData := url.Values{}
			formData.Set("token", tt.token)
			formData.Set("new_password", tt.newPassword)
			formData.Set("confirm_password", tt.confirmPassword)

			// Create request
			req := httptest.NewRequest("POST", "/auth/reset-password", strings.NewReader(formData.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

			// Create response recorder
			rr := httptest.NewRecorder()

			// Call handler
			r.ServeHTTP(rr, req)

			// Check status
			assert.Equal(t, tt.expectedStatus, rr.Code)

			// Check body contains expected content
			if tt.expectedBody != "" {
				assert.Contains(t, rr.Body.String(), tt.expectedBody)
			}

			// Verify mocks
			mockAuthService.AssertExpectations(t)
		})
	}
}

func TestAuthHandler_LoginWithEmailVerificationRequired(t *testing.T) {
	// Test that login fails when email is not verified
	mockAuthService := &MockAuthServiceForEmailVerification{}
	
	loginReq := &services.LoginRequest{
		Email:      "unverified@example.com",
		Password:   "password123",
		RememberMe: false,
	}
	
	mockAuthService.On("Login", loginReq).Return((*services.AuthResponse)(nil), fmt.Errorf("please verify your email address before logging in"))

	// Create handler
	store := sessions.NewCookieStore([]byte("test-secret"))
	handler := NewAuthHandler(mockAuthService, store)

	// Create router
	r := chi.NewRouter()
	r.Post("/auth/login", handler.LoginSubmit)

	// Create form data
	formData := url.Values{}
	formData.Set("email", "unverified@example.com")
	formData.Set("password", "password123")

	// Create request
	req := httptest.NewRequest("POST", "/auth/login", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// Create response recorder
	rr := httptest.NewRecorder()

	// Call handler
	r.ServeHTTP(rr, req)

	// Check status
	assert.Equal(t, http.StatusUnprocessableEntity, rr.Code)

	// Check body contains expected content
	assert.Contains(t, rr.Body.String(), "Please verify your email address before logging in")

	// Verify mocks
	mockAuthService.AssertExpectations(t)
}