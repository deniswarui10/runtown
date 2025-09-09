package handlers

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"event-ticketing-platform/internal/models"
	"event-ticketing-platform/internal/services"

	"github.com/gorilla/sessions"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockAuthService for testing
type MockAuthService struct {
	mock.Mock
}

func (m *MockAuthService) Register(req *services.RegisterRequest) (*services.AuthResponse, error) {
	args := m.Called(req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*services.AuthResponse), args.Error(1)
}

func (m *MockAuthService) Login(req *services.LoginRequest) (*services.AuthResponse, error) {
	args := m.Called(req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*services.AuthResponse), args.Error(1)
}

func (m *MockAuthService) ValidateSession(sessionID string) (*models.User, error) {
	args := m.Called(sessionID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockAuthService) Logout(sessionID string) error {
	args := m.Called(sessionID)
	return args.Error(0)
}

func (m *MockAuthService) RequestPasswordReset(req *services.PasswordResetRequest) error {
	args := m.Called(req)
	return args.Error(0)
}

func (m *MockAuthService) ChangePassword(userID int, req *services.PasswordChangeRequest) error {
	args := m.Called(userID, req)
	return args.Error(0)
}

func (m *MockAuthService) RequireRole(user *models.User, requiredRole models.UserRole) error {
	args := m.Called(user, requiredRole)
	return args.Error(0)
}

func (m *MockAuthService) RequireRoles(user *models.User, requiredRoles ...models.UserRole) error {
	args := m.Called(user, requiredRoles)
	return args.Error(0)
}

func (m *MockAuthService) CleanupExpiredSessions() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockAuthService) VerifyEmail(token string) (*models.User, error) {
	args := m.Called(token)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockAuthService) ExtendSession(sessionID string, duration time.Duration) error {
	args := m.Called(sessionID, duration)
	return args.Error(0)
}

func (m *MockAuthService) LogoutAllSessions(userID int) error {
	args := m.Called(userID)
	return args.Error(0)
}

func (m *MockAuthService) ResendVerificationEmail(email string) error {
	args := m.Called(email)
	return args.Error(0)
}

func (m *MockAuthService) CompletePasswordReset(req *services.PasswordResetCompleteRequest) error {
	args := m.Called(req)
	return args.Error(0)
}

func (m *MockAuthService) ValidatePasswordResetToken(token string) (*models.User, error) {
	args := m.Called(token)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockAuthService) CleanupExpiredTokens() error {
	args := m.Called()
	return args.Error(0)
}

func TestAuthHandler_LoginPage(t *testing.T) {
	mockAuthService := new(MockAuthService)
	store := sessions.NewCookieStore([]byte("test-secret"))
	handler := NewAuthHandler(mockAuthService, store)

	req, err := http.NewRequest("GET", "/auth/login", nil)
	assert.NoError(t, err)

	rr := httptest.NewRecorder()
	handler.LoginPage(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), "Sign in to your account")
}

func TestAuthHandler_LoginSubmit_Success(t *testing.T) {
	mockAuthService := new(MockAuthService)
	store := sessions.NewCookieStore([]byte("test-secret"))
	handler := NewAuthHandler(mockAuthService, store)

	// Mock successful login
	authResponse := &services.AuthResponse{
		User: &models.User{
			ID:        1,
			Email:     "test@example.com",
			FirstName: "John",
			LastName:  "Doe",
			Role:      models.RoleAttendee,
		},
		SessionID: "session123",
	}
	mockAuthService.On("Login", mock.AnythingOfType("*services.LoginRequest")).Return(authResponse, nil)

	// Create form data
	form := url.Values{}
	form.Add("email", "test@example.com")
	form.Add("password", "password123")

	req, err := http.NewRequest("POST", "/auth/login", strings.NewReader(form.Encode()))
	assert.NoError(t, err)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	rr := httptest.NewRecorder()
	handler.LoginSubmit(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "/dashboard", rr.Header().Get("HX-Redirect"))
	mockAuthService.AssertExpectations(t)
}

func TestAuthHandler_LoginSubmit_InvalidCredentials(t *testing.T) {
	mockAuthService := new(MockAuthService)
	store := sessions.NewCookieStore([]byte("test-secret"))
	handler := NewAuthHandler(mockAuthService, store)

	// Mock failed login
	mockAuthService.On("Login", mock.AnythingOfType("*services.LoginRequest")).Return(nil, assert.AnError)

	// Create form data
	form := url.Values{}
	form.Add("email", "test@example.com")
	form.Add("password", "wrongpassword")

	req, err := http.NewRequest("POST", "/auth/login", strings.NewReader(form.Encode()))
	assert.NoError(t, err)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	rr := httptest.NewRecorder()
	handler.LoginSubmit(rr, req)

	assert.Equal(t, http.StatusUnprocessableEntity, rr.Code)
	body := rr.Body.String()
	assert.Contains(t, body, "Invalid email or password")
	mockAuthService.AssertExpectations(t)
}

func TestAuthHandler_LoginSubmit_ValidationErrors(t *testing.T) {
	mockAuthService := new(MockAuthService)
	store := sessions.NewCookieStore([]byte("test-secret"))
	handler := NewAuthHandler(mockAuthService, store)

	// Create form data with missing fields
	form := url.Values{}
	form.Add("email", "")
	form.Add("password", "")

	req, err := http.NewRequest("POST", "/auth/login", strings.NewReader(form.Encode()))
	assert.NoError(t, err)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	rr := httptest.NewRecorder()
	handler.LoginSubmit(rr, req)

	assert.Equal(t, http.StatusUnprocessableEntity, rr.Code)
	assert.Contains(t, rr.Body.String(), "Email is required")
	assert.Contains(t, rr.Body.String(), "Password is required")
}

func TestAuthHandler_RegisterPage(t *testing.T) {
	mockAuthService := new(MockAuthService)
	store := sessions.NewCookieStore([]byte("test-secret"))
	handler := NewAuthHandler(mockAuthService, store)

	req, err := http.NewRequest("GET", "/auth/register", nil)
	assert.NoError(t, err)

	rr := httptest.NewRecorder()
	handler.RegisterPage(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), "Create your account")
}

func TestAuthHandler_RegisterSubmit_Success(t *testing.T) {
	mockAuthService := new(MockAuthService)
	store := sessions.NewCookieStore([]byte("test-secret"))
	handler := NewAuthHandler(mockAuthService, store)

	// Mock successful registration
	authResponse := &services.AuthResponse{
		User: &models.User{
			ID:        1,
			Email:     "test@example.com",
			FirstName: "John",
			LastName:  "Doe",
			Role:      models.RoleAttendee,
		},
		SessionID: "session123",
	}
	mockAuthService.On("Register", mock.AnythingOfType("*services.RegisterRequest")).Return(authResponse, nil)

	// Create form data
	form := url.Values{}
	form.Add("email", "test@example.com")
	form.Add("password", "password123")
	form.Add("password_confirm", "password123")
	form.Add("first_name", "John")
	form.Add("last_name", "Doe")
	form.Add("terms", "on")

	req, err := http.NewRequest("POST", "/auth/register", strings.NewReader(form.Encode()))
	assert.NoError(t, err)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	rr := httptest.NewRecorder()
	handler.RegisterSubmit(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "/dashboard", rr.Header().Get("HX-Redirect"))
	mockAuthService.AssertExpectations(t)
}

func TestAuthHandler_RegisterSubmit_ValidationErrors(t *testing.T) {
	mockAuthService := new(MockAuthService)
	store := sessions.NewCookieStore([]byte("test-secret"))
	handler := NewAuthHandler(mockAuthService, store)

	// Create form data with validation errors
	form := url.Values{}
	form.Add("email", "")
	form.Add("password", "123") // Too short
	form.Add("password_confirm", "456") // Doesn't match
	form.Add("first_name", "")
	form.Add("last_name", "")

	req, err := http.NewRequest("POST", "/auth/register", strings.NewReader(form.Encode()))
	assert.NoError(t, err)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	rr := httptest.NewRecorder()
	handler.RegisterSubmit(rr, req)

	assert.Equal(t, http.StatusUnprocessableEntity, rr.Code)
	body := rr.Body.String()
	assert.Contains(t, body, "Email is required")
	assert.Contains(t, body, "Password must be at least 8 characters long")
	assert.Contains(t, body, "Passwords do not match")
	assert.Contains(t, body, "First name is required")
	assert.Contains(t, body, "Last name is required")
}

func TestAuthHandler_ForgotPasswordPage(t *testing.T) {
	mockAuthService := new(MockAuthService)
	store := sessions.NewCookieStore([]byte("test-secret"))
	handler := NewAuthHandler(mockAuthService, store)

	req, err := http.NewRequest("GET", "/auth/forgot-password", nil)
	assert.NoError(t, err)

	rr := httptest.NewRecorder()
	handler.ForgotPasswordPage(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), "Reset your password")
}

func TestAuthHandler_ForgotPasswordSubmit_Success(t *testing.T) {
	mockAuthService := new(MockAuthService)
	store := sessions.NewCookieStore([]byte("test-secret"))
	handler := NewAuthHandler(mockAuthService, store)

	// Mock successful password reset request
	mockAuthService.On("RequestPasswordReset", mock.AnythingOfType("*services.PasswordResetRequest")).Return(nil)

	// Create form data
	form := url.Values{}
	form.Add("email", "test@example.com")

	req, err := http.NewRequest("POST", "/auth/forgot-password", strings.NewReader(form.Encode()))
	assert.NoError(t, err)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	rr := httptest.NewRecorder()
	handler.ForgotPasswordSubmit(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), "Password reset instructions have been sent")
	mockAuthService.AssertExpectations(t)
}

func TestAuthHandler_Logout(t *testing.T) {
	mockAuthService := new(MockAuthService)
	store := sessions.NewCookieStore([]byte("test-secret"))
	handler := NewAuthHandler(mockAuthService, store)

	// Mock successful logout
	mockAuthService.On("Logout", "session123").Return(nil)

	req, err := http.NewRequest("POST", "/auth/logout", nil)
	assert.NoError(t, err)

	// Create a session with session ID
	session, _ := store.Get(req, "session")
	session.Values["session_id"] = "session123"
	session.Values["user_id"] = 1

	rr := httptest.NewRecorder()
	handler.Logout(rr, req)

	assert.Equal(t, http.StatusSeeOther, rr.Code)
	assert.Equal(t, "/", rr.Header().Get("Location"))
	mockAuthService.AssertExpectations(t)
}