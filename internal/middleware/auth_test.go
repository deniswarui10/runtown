package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"event-ticketing-platform/internal/models"
	"event-ticketing-platform/internal/services"

	"github.com/gorilla/sessions"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockAuthService is a mock implementation of AuthServiceInterface
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

func (m *MockAuthService) ResendVerificationEmail(email string) error {
	args := m.Called(email)
	return args.Error(0)
}

func (m *MockAuthService) ExtendSession(sessionID string, duration time.Duration) error {
	args := m.Called(sessionID, duration)
	return args.Error(0)
}

func (m *MockAuthService) LogoutAllSessions(userID int) error {
	args := m.Called(userID)
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



func TestAuthMiddleware_LoadUser(t *testing.T) {
	tests := []struct {
		name           string
		sessionValues  map[interface{}]interface{}
		mockSetup      func(*MockAuthService)
		expectedUser   *models.User
		expectedCalled bool
	}{
		{
			name:          "no session",
			sessionValues: map[interface{}]interface{}{},
			mockSetup:     func(m *MockAuthService) {},
			expectedUser:  nil,
		},
		{
			name: "valid session",
			sessionValues: map[interface{}]interface{}{
				"user_id":    1,
				"session_id": "valid-session",
			},
			mockSetup: func(m *MockAuthService) {
				user := &models.User{
					ID:        1,
					Email:     "test@example.com",
					FirstName: "Test",
					LastName:  "User",
					Role:      models.RoleAttendee,
				}
				m.On("ValidateSession", "valid-session").Return(user, nil)
			},
			expectedUser: &models.User{
				ID:        1,
				Email:     "test@example.com",
				FirstName: "Test",
				LastName:  "User",
				Role:      models.RoleAttendee,
			},
			expectedCalled: true,
		},
		{
			name: "invalid session",
			sessionValues: map[interface{}]interface{}{
				"user_id":    1,
				"session_id": "invalid-session",
			},
			mockSetup: func(m *MockAuthService) {
				m.On("ValidateSession", "invalid-session").Return(nil, assert.AnError)
			},
			expectedUser:   nil,
			expectedCalled: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			mockAuth := new(MockAuthService)
			tt.mockSetup(mockAuth)

			store := sessions.NewCookieStore([]byte("test-key"))
			middleware := NewAuthMiddleware(mockAuth, store)

			// Create test handler
			var capturedUser *models.User
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				capturedUser = GetUserFromContext(r.Context())
				w.WriteHeader(http.StatusOK)
			})

			// Create request and response
			req := httptest.NewRequest("GET", "/test", nil)
			rr := httptest.NewRecorder()

			// Set up session
			session, _ := store.Get(req, "session")
			for key, value := range tt.sessionValues {
				session.Values[key] = value
			}
			session.Save(req, rr)

			// Execute middleware
			middleware.LoadUser(handler).ServeHTTP(rr, req)

			// Assertions
			assert.Equal(t, http.StatusOK, rr.Code)
			if tt.expectedUser != nil {
				assert.NotNil(t, capturedUser)
				assert.Equal(t, tt.expectedUser.ID, capturedUser.ID)
				assert.Equal(t, tt.expectedUser.Email, capturedUser.Email)
			} else {
				assert.Nil(t, capturedUser)
			}

			if tt.expectedCalled {
				mockAuth.AssertExpectations(t)
			}
		})
	}
}

func TestAuthMiddleware_RequireAuth(t *testing.T) {
	tests := []struct {
		name           string
		user           *models.User
		isHTMX         bool
		expectedStatus int
		expectedHeader string
	}{
		{
			name:           "authenticated user",
			user:           &models.User{ID: 1, Email: "test@example.com"},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "unauthenticated user - regular request",
			user:           nil,
			expectedStatus: http.StatusSeeOther,
		},
		{
			name:           "unauthenticated user - HTMX request",
			user:           nil,
			isHTMX:         true,
			expectedStatus: http.StatusUnauthorized,
			expectedHeader: "/auth/login",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			mockAuth := new(MockAuthService)
			store := sessions.NewCookieStore([]byte("test-key"))
			middleware := NewAuthMiddleware(mockAuth, store)

			// Create test handler
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})

			// Create request
			req := httptest.NewRequest("GET", "/protected", nil)
			if tt.isHTMX {
				req.Header.Set("HX-Request", "true")
			}

			// Add user to context if provided
			if tt.user != nil {
				ctx := context.WithValue(req.Context(), UserContextKey, tt.user)
				req = req.WithContext(ctx)
			}

			rr := httptest.NewRecorder()

			// Execute middleware
			middleware.RequireAuth(handler).ServeHTTP(rr, req)

			// Assertions
			assert.Equal(t, tt.expectedStatus, rr.Code)
			if tt.expectedHeader != "" {
				assert.Equal(t, tt.expectedHeader, rr.Header().Get("HX-Redirect"))
			}
		})
	}
}

func TestAuthMiddleware_RequireRole(t *testing.T) {
	tests := []struct {
		name           string
		user           *models.User
		requiredRole   models.UserRole
		expectedStatus int
	}{
		{
			name:           "user has required role",
			user:           &models.User{ID: 1, Role: models.RoleOrganizer},
			requiredRole:   models.RoleOrganizer,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "admin can access any role",
			user:           &models.User{ID: 1, Role: models.RoleAdmin},
			requiredRole:   models.RoleOrganizer,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "user lacks required role",
			user:           &models.User{ID: 1, Role: models.RoleAttendee},
			requiredRole:   models.RoleOrganizer,
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "no user",
			user:           nil,
			requiredRole:   models.RoleOrganizer,
			expectedStatus: http.StatusSeeOther,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			mockAuth := new(MockAuthService)
			store := sessions.NewCookieStore([]byte("test-key"))
			middleware := NewAuthMiddleware(mockAuth, store)

			// Create test handler
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})

			// Create request
			req := httptest.NewRequest("GET", "/protected", nil)

			// Add user to context if provided
			if tt.user != nil {
				ctx := context.WithValue(req.Context(), UserContextKey, tt.user)
				req = req.WithContext(ctx)
			}

			rr := httptest.NewRecorder()

			// Execute middleware
			middleware.RequireRole(tt.requiredRole)(handler).ServeHTTP(rr, req)

			// Assertions
			assert.Equal(t, tt.expectedStatus, rr.Code)
		})
	}
}

func TestGetUserFromContext(t *testing.T) {
	tests := []struct {
		name     string
		ctx      context.Context
		expected *models.User
	}{
		{
			name:     "no user in context",
			ctx:      context.Background(),
			expected: nil,
		},
		{
			name: "user in context",
			ctx: context.WithValue(context.Background(), UserContextKey, &models.User{
				ID:    1,
				Email: "test@example.com",
			}),
			expected: &models.User{
				ID:    1,
				Email: "test@example.com",
			},
		},
		{
			name:     "wrong type in context",
			ctx:      context.WithValue(context.Background(), UserContextKey, "not-a-user"),
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetUserFromContext(tt.ctx)
			if tt.expected == nil {
				assert.Nil(t, result)
			} else {
				assert.NotNil(t, result)
				assert.Equal(t, tt.expected.ID, result.ID)
				assert.Equal(t, tt.expected.Email, result.Email)
			}
		})
	}
}

func TestIsHTMXRequest(t *testing.T) {
	tests := []struct {
		name     string
		headers  map[string]string
		expected bool
	}{
		{
			name:     "no HTMX header",
			headers:  map[string]string{},
			expected: false,
		},
		{
			name:     "HTMX header present",
			headers:  map[string]string{"HX-Request": "true"},
			expected: true,
		},
		{
			name:     "HTMX header false",
			headers:  map[string]string{"HX-Request": "false"},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test", nil)
			for key, value := range tt.headers {
				req.Header.Set(key, value)
			}

			result := IsHTMXRequest(req)
			assert.Equal(t, tt.expected, result)
		})
	}
}