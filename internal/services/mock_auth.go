package services

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"event-ticketing-platform/internal/models"
	"event-ticketing-platform/internal/utils"
)

// MockAuthService provides a working mock authentication service for testing/demo
type MockAuthService struct {
	users    map[string]*models.User // email -> user
	sessions map[string]*models.User // sessionID -> user
	userID   int
	mutex    sync.RWMutex
}

// NewMockAuthService creates a new mock authentication service
func NewMockAuthService() *MockAuthService {
	service := &MockAuthService{
		users:    make(map[string]*models.User),
		sessions: make(map[string]*models.User),
		userID:   1,
	}
	
	// Add some default users for testing
	service.addDefaultUsers()
	
	return service
}

// addDefaultUsers adds some default users for testing
func (s *MockAuthService) addDefaultUsers() {
	// Add a test user
	hashedPassword, _ := utils.HashPassword("password123")
	testUser := &models.User{
		ID:           1,
		Email:        "test@example.com",
		PasswordHash: hashedPassword,
		FirstName:    "Test",
		LastName:     "User",
		Role:         models.RoleAttendee,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	s.users["test@example.com"] = testUser
	
	// Add an organizer user
	hashedPassword2, _ := utils.HashPassword("organizer123")
	organizerUser := &models.User{
		ID:           2,
		Email:        "organizer@example.com",
		PasswordHash: hashedPassword2,
		FirstName:    "Event",
		LastName:     "Organizer",
		Role:         models.RoleOrganizer,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	s.users["organizer@example.com"] = organizerUser
	s.userID = 3 // Next user ID
}

// Register creates a new user account
func (s *MockAuthService) Register(req *RegisterRequest) (*AuthResponse, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	
	// Validate the request
	if req.Email == "" {
		return nil, fmt.Errorf("email is required")
	}
	if req.Password == "" {
		return nil, fmt.Errorf("password is required")
	}
	if req.FirstName == "" {
		return nil, fmt.Errorf("first name is required")
	}
	if req.LastName == "" {
		return nil, fmt.Errorf("last name is required")
	}
	
	// Check if user already exists
	if _, exists := s.users[req.Email]; exists {
		return nil, fmt.Errorf("user with email %s already exists", req.Email)
	}
	
	// Hash the password
	hashedPassword, err := utils.HashPassword(req.Password)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}
	
	// Create the user
	user := &models.User{
		ID:           s.userID,
		Email:        req.Email,
		PasswordHash: hashedPassword,
		FirstName:    req.FirstName,
		LastName:     req.LastName,
		Role:         req.Role,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	
	s.users[req.Email] = user
	s.userID++
	
	// Create a session
	sessionID, expiresAt, err := s.createSession(user)
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}
	
	return &AuthResponse{
		User:      user,
		SessionID: sessionID,
		ExpiresAt: expiresAt,
	}, nil
}

// Login authenticates a user and creates a session
func (s *MockAuthService) Login(req *LoginRequest) (*AuthResponse, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	
	// Validate input
	if req.Email == "" {
		return nil, fmt.Errorf("email is required")
	}
	if req.Password == "" {
		return nil, fmt.Errorf("password is required")
	}
	
	// Get user by email
	user, exists := s.users[req.Email]
	if !exists {
		return nil, fmt.Errorf("invalid email or password")
	}
	
	// Verify password
	valid, err := utils.VerifyPassword(req.Password, user.PasswordHash)
	if err != nil {
		return nil, fmt.Errorf("failed to verify password: %w", err)
	}
	
	if !valid {
		return nil, fmt.Errorf("invalid email or password")
	}
	
	// Create a session
	sessionID, expiresAt, err := s.createSession(user)
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}
	
	return &AuthResponse{
		User:      user,
		SessionID: sessionID,
		ExpiresAt: expiresAt,
	}, nil
}

// ValidateSession validates a session and returns the associated user
func (s *MockAuthService) ValidateSession(sessionID string) (*models.User, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	
	if sessionID == "" {
		return nil, fmt.Errorf("session ID is required")
	}
	
	user, exists := s.sessions[sessionID]
	if !exists {
		return nil, fmt.Errorf("invalid or expired session")
	}
	
	return user, nil
}

// Logout invalidates a user session
func (s *MockAuthService) Logout(sessionID string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	
	if sessionID == "" {
		return fmt.Errorf("session ID is required")
	}
	
	delete(s.sessions, sessionID)
	return nil
}

// RequestPasswordReset initiates a password reset process
func (s *MockAuthService) RequestPasswordReset(req *PasswordResetRequest) error {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	
	if req.Email == "" {
		return fmt.Errorf("email is required")
	}
	
	// Check if user exists (but don't reveal if they don't for security)
	_, exists := s.users[req.Email]
	if !exists {
		// Don't reveal whether the email exists or not for security
		return nil
	}
	
	// In a real implementation, you'd send an email here
	// For the mock, we'll just return success
	return nil
}

// ChangePassword changes a user's password
func (s *MockAuthService) ChangePassword(userID int, req *PasswordChangeRequest) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	
	if req.OldPassword == "" {
		return fmt.Errorf("old password is required")
	}
	if req.NewPassword == "" {
		return fmt.Errorf("new password is required")
	}
	
	// Find user by ID
	var user *models.User
	for _, u := range s.users {
		if u.ID == userID {
			user = u
			break
		}
	}
	
	if user == nil {
		return fmt.Errorf("user not found")
	}
	
	// Verify old password
	valid, err := utils.VerifyPassword(req.OldPassword, user.PasswordHash)
	if err != nil {
		return fmt.Errorf("failed to verify old password: %w", err)
	}
	
	if !valid {
		return fmt.Errorf("old password is incorrect")
	}
	
	// Hash new password
	hashedPassword, err := utils.HashPassword(req.NewPassword)
	if err != nil {
		return fmt.Errorf("failed to hash new password: %w", err)
	}
	
	// Update password
	user.PasswordHash = hashedPassword
	user.UpdatedAt = time.Now()
	
	// Invalidate all sessions for this user
	for sessionID, sessionUser := range s.sessions {
		if sessionUser.ID == userID {
			delete(s.sessions, sessionID)
		}
	}
	
	return nil
}

// RequireRole checks if a user has the required role
func (s *MockAuthService) RequireRole(user *models.User, requiredRole models.UserRole) error {
	if user == nil {
		return fmt.Errorf("user is required")
	}
	
	// Admin can access everything
	if user.Role == models.RoleAdmin {
		return nil
	}
	
	if user.Role != requiredRole {
		return fmt.Errorf("insufficient permissions: required role %s, user has role %s", requiredRole, user.Role)
	}
	
	return nil
}

// RequireRoles checks if a user has any of the required roles
func (s *MockAuthService) RequireRoles(user *models.User, requiredRoles ...models.UserRole) error {
	if user == nil {
		return fmt.Errorf("user is required")
	}
	
	// Admin can access everything
	if user.Role == models.RoleAdmin {
		return nil
	}
	
	for _, role := range requiredRoles {
		if user.Role == role {
			return nil
		}
	}
	
	return fmt.Errorf("insufficient permissions: user role %s not in required roles", user.Role)
}

// CleanupExpiredSessions removes expired sessions from the database
func (s *MockAuthService) CleanupExpiredSessions() error {
	// In a real implementation, you'd check expiration times
	// For the mock, we'll just return success
	return nil
}

// createSession creates a new session for a user
func (s *MockAuthService) createSession(user *models.User) (string, time.Time, error) {
	// Generate a secure session ID
	sessionBytes := make([]byte, 32)
	if _, err := rand.Read(sessionBytes); err != nil {
		return "", time.Time{}, fmt.Errorf("failed to generate session ID: %w", err)
	}
	sessionID := hex.EncodeToString(sessionBytes)
	
	// Set session expiration (24 hours from now)
	expiresAt := time.Now().Add(24 * time.Hour)
	
	// Store the session
	s.sessions[sessionID] = user
	
	return sessionID, expiresAt, nil
}