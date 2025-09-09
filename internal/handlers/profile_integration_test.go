package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"event-ticketing-platform/internal/middleware"
	"event-ticketing-platform/internal/models"
	"event-ticketing-platform/internal/services"

	"github.com/gorilla/sessions"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockUserService implements UserServiceInterface for testing
type MockUserService struct {
	mock.Mock
}

func (m *MockUserService) UpdateProfile(userID int, req *services.UpdateProfileRequest) (*models.User, error) {
	args := m.Called(userID, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserService) UpdatePreferences(userID int, preferences *services.UserPreferences) error {
	args := m.Called(userID, preferences)
	return args.Error(0)
}

func (m *MockUserService) DeleteAccount(userID int) error {
	args := m.Called(userID)
	return args.Error(0)
}

func (m *MockUserService) GetUserPreferences(userID int) (*services.UserPreferences, error) {
	args := m.Called(userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*services.UserPreferences), args.Error(1)
}

func createProfileHandler() (*ProfileHandler, *MockAuthService, *MockUserService) {
	mockAuthService := &MockAuthService{}
	mockUserService := &MockUserService{}
	store := sessions.NewCookieStore([]byte("test-secret"))
	
	handler := NewProfileHandler(mockAuthService, mockUserService, store)
	return handler, mockAuthService, mockUserService
}

func TestProfileHandler_ProfilePage(t *testing.T) {
	handler, _, _ := createProfileHandler()
	user := createTestUser()

	req := httptest.NewRequest("GET", "/dashboard/profile", nil)
	w := httptest.NewRecorder()

	// Add user to context
	ctx := context.WithValue(req.Context(), middleware.UserContextKey, user)
	req = req.WithContext(ctx)

	handler.ProfilePage(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Profile Settings")
	assert.Contains(t, w.Body.String(), user.FirstName)
	assert.Contains(t, w.Body.String(), user.Email)
}

func TestProfileHandler_UpdateProfile_Success(t *testing.T) {
	handler, _, mockUserService := createProfileHandler()
	user := createTestUser()

	// Setup mock expectations
	updatedUser := &models.User{
		ID:        user.ID,
		Email:     "newemail@example.com",
		FirstName: "Jane",
		LastName:  "Smith",
		Role:      user.Role,
	}
	
	mockUserService.On("UpdateProfile", user.ID, mock.MatchedBy(func(req *services.UpdateProfileRequest) bool {
		return req.FirstName == "Jane" && req.LastName == "Smith" && req.Email == "newemail@example.com"
	})).Return(updatedUser, nil)

	// Create form data
	formData := url.Values{}
	formData.Set("first_name", "Jane")
	formData.Set("last_name", "Smith")
	formData.Set("email", "newemail@example.com")

	req := httptest.NewRequest("POST", "/dashboard/profile", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	// Add user to context
	ctx := context.WithValue(req.Context(), middleware.UserContextKey, user)
	req = req.WithContext(ctx)

	handler.UpdateProfile(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Profile updated successfully")
	mockUserService.AssertExpectations(t)
}

func TestProfileHandler_UpdateProfile_ValidationError(t *testing.T) {
	handler, _, _ := createProfileHandler()
	user := createTestUser()

	// Create form data with missing required fields
	formData := url.Values{}
	formData.Set("first_name", "")
	formData.Set("last_name", "Smith")
	formData.Set("email", "newemail@example.com")

	req := httptest.NewRequest("POST", "/dashboard/profile", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	// Add user to context
	ctx := context.WithValue(req.Context(), middleware.UserContextKey, user)
	req = req.WithContext(ctx)

	handler.UpdateProfile(w, req)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	assert.Contains(t, w.Body.String(), "First name is required")
}

func TestProfileHandler_SecurityPage(t *testing.T) {
	handler, _, _ := createProfileHandler()
	user := createTestUser()

	req := httptest.NewRequest("GET", "/dashboard/security", nil)
	w := httptest.NewRecorder()

	// Add user to context
	ctx := context.WithValue(req.Context(), middleware.UserContextKey, user)
	req = req.WithContext(ctx)

	handler.SecurityPage(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Security Settings")
	assert.Contains(t, w.Body.String(), "Change Password")
}

func TestProfileHandler_ChangePassword_Success(t *testing.T) {
	handler, mockAuthService, _ := createProfileHandler()
	user := createTestUser()

	// Setup mock expectations
	mockAuthService.On("ChangePassword", user.ID, mock.MatchedBy(func(req *services.PasswordChangeRequest) bool {
		return req.OldPassword == "oldpass123" && req.NewPassword == "newpass123"
	})).Return(nil)

	// Create form data
	formData := url.Values{}
	formData.Set("current_password", "oldpass123")
	formData.Set("new_password", "newpass123")
	formData.Set("confirm_password", "newpass123")

	req := httptest.NewRequest("POST", "/dashboard/security/change-password", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	// Add user to context
	ctx := context.WithValue(req.Context(), middleware.UserContextKey, user)
	req = req.WithContext(ctx)

	handler.ChangePassword(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Password changed successfully")
	mockAuthService.AssertExpectations(t)
}

func TestProfileHandler_ChangePassword_ValidationError(t *testing.T) {
	handler, _, _ := createProfileHandler()
	user := createTestUser()

	// Create form data with mismatched passwords
	formData := url.Values{}
	formData.Set("current_password", "oldpass123")
	formData.Set("new_password", "newpass123")
	formData.Set("confirm_password", "differentpass")

	req := httptest.NewRequest("POST", "/dashboard/security/change-password", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	// Add user to context
	ctx := context.WithValue(req.Context(), middleware.UserContextKey, user)
	req = req.WithContext(ctx)

	handler.ChangePassword(w, req)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	assert.Contains(t, w.Body.String(), "Passwords do not match")
}

func TestProfileHandler_SettingsPage(t *testing.T) {
	handler, _, _ := createProfileHandler()
	user := createTestUser()

	req := httptest.NewRequest("GET", "/dashboard/settings", nil)
	w := httptest.NewRecorder()

	// Add user to context
	ctx := context.WithValue(req.Context(), middleware.UserContextKey, user)
	req = req.WithContext(ctx)

	handler.SettingsPage(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Account Settings")
	assert.Contains(t, w.Body.String(), "Notification Preferences")
}

func TestProfileHandler_UpdateSettings_Success(t *testing.T) {
	handler, _, mockUserService := createProfileHandler()
	user := createTestUser()

	// Setup mock expectations
	mockUserService.On("UpdatePreferences", user.ID, mock.MatchedBy(func(prefs *services.UserPreferences) bool {
		return prefs.EmailNotifications == true && prefs.EventReminders == false
	})).Return(nil)

	// Create form data
	formData := url.Values{}
	formData.Set("email_notifications", "on")
	// event_reminders not set (should be false)

	req := httptest.NewRequest("POST", "/dashboard/settings", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	// Add user to context
	ctx := context.WithValue(req.Context(), middleware.UserContextKey, user)
	req = req.WithContext(ctx)

	handler.UpdateSettings(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Settings updated successfully")
	mockUserService.AssertExpectations(t)
}

func TestProfileHandler_DeleteAccountPage(t *testing.T) {
	handler, _, _ := createProfileHandler()
	user := createTestUser()

	req := httptest.NewRequest("GET", "/dashboard/delete-account", nil)
	w := httptest.NewRecorder()

	// Add user to context
	ctx := context.WithValue(req.Context(), middleware.UserContextKey, user)
	req = req.WithContext(ctx)

	handler.DeleteAccountPage(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Delete Account")
	assert.Contains(t, w.Body.String(), "Warning: Account Deletion")
	assert.Contains(t, w.Body.String(), user.Email)
}

func TestProfileHandler_DeleteAccount_Success(t *testing.T) {
	handler, mockAuthService, mockUserService := createProfileHandler()
	user := createTestUser()

	// Setup mock expectations
	mockAuthService.On("Login", mock.MatchedBy(func(req *services.LoginRequest) bool {
		return req.Email == user.Email && req.Password == "password123"
	})).Return(&services.AuthResponse{User: user}, nil)
	
	mockUserService.On("DeleteAccount", user.ID).Return(nil)

	// Create form data
	formData := url.Values{}
	formData.Set("password", "password123")
	formData.Set("confirmation", "DELETE")

	req := httptest.NewRequest("POST", "/dashboard/delete-account", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	// Add user to context
	ctx := context.WithValue(req.Context(), middleware.UserContextKey, user)
	req = req.WithContext(ctx)

	handler.DeleteAccount(w, req)

	assert.Equal(t, http.StatusSeeOther, w.Code)
	assert.Equal(t, "/?deleted=1", w.Header().Get("Location"))
	mockAuthService.AssertExpectations(t)
	mockUserService.AssertExpectations(t)
}

func TestProfileHandler_DeleteAccount_ValidationError(t *testing.T) {
	handler, _, _ := createProfileHandler()
	user := createTestUser()

	// Create form data with wrong confirmation text
	formData := url.Values{}
	formData.Set("password", "password123")
	formData.Set("confirmation", "delete") // should be "DELETE"

	req := httptest.NewRequest("POST", "/dashboard/delete-account", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	// Add user to context
	ctx := context.WithValue(req.Context(), middleware.UserContextKey, user)
	req = req.WithContext(ctx)

	handler.DeleteAccount(w, req)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	assert.Contains(t, w.Body.String(), "Please type 'DELETE' to confirm")
}

func TestProfileHandler_RequiresAuthentication(t *testing.T) {
	handler, _, _ := createProfileHandler()

	testCases := []struct {
		name    string
		method  string
		path    string
		handler http.HandlerFunc
	}{
		{"ProfilePage", "GET", "/dashboard/profile", handler.ProfilePage},
		{"UpdateProfile", "POST", "/dashboard/profile", handler.UpdateProfile},
		{"SecurityPage", "GET", "/dashboard/security", handler.SecurityPage},
		{"ChangePassword", "POST", "/dashboard/security/change-password", handler.ChangePassword},
		{"SettingsPage", "GET", "/dashboard/settings", handler.SettingsPage},
		{"UpdateSettings", "POST", "/dashboard/settings", handler.UpdateSettings},
		{"DeleteAccountPage", "GET", "/dashboard/delete-account", handler.DeleteAccountPage},
		{"DeleteAccount", "POST", "/dashboard/delete-account", handler.DeleteAccount},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var req *http.Request
			if tc.method == "POST" {
				req = httptest.NewRequest(tc.method, tc.path, strings.NewReader(""))
			} else {
				req = httptest.NewRequest(tc.method, tc.path, nil)
			}
			w := httptest.NewRecorder()

			// Don't add user to context (simulating unauthenticated request)
			tc.handler(w, req)

			assert.Equal(t, http.StatusSeeOther, w.Code)
			assert.Equal(t, "/auth/login", w.Header().Get("Location"))
		})
	}
}