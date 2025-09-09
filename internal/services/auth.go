package services

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"event-ticketing-platform/internal/models"
	"event-ticketing-platform/internal/repositories"
	"event-ticketing-platform/internal/utils"
)

// UserRepository interface for user data operations
type UserRepository interface {
	Create(req *models.UserCreateRequest) (*models.User, error)
	GetByID(id int) (*models.User, error)
	GetByEmail(email string) (*models.User, error)
	Update(id int, req *models.UserUpdateRequest) (*models.User, error)
	UpdatePassword(id int, passwordHash string) error
	Delete(id int) error
	Search(filters repositories.UserSearchFilters) ([]*models.User, int, error)
	GetByRole(role models.UserRole) ([]*models.User, error)
	CreateSession(userID int, sessionID string, expiresAt time.Time) error
	GetUserBySession(sessionID string) (*models.User, error)
	DeleteSession(sessionID string) error
	DeleteExpiredSessions() error
	DeleteUserSessions(userID int) error
	ExtendSession(sessionID string, expiresAt time.Time) error
	SetVerificationToken(userID int, token string) error
	GetByVerificationToken(token string) (*models.User, error)
	VerifyEmail(userID int) error
	SetPasswordResetToken(userID int, token string, expiresAt time.Time) error
	GetByPasswordResetToken(token string) (*models.User, error)
	ClearPasswordResetToken(userID int) error
	CleanupExpiredTokens() error
}

// AuthService handles authentication-related business logic
type AuthService struct {
	userRepo     UserRepository
	emailService EmailService // Interface for email service
}

// EmailService interface for sending emails
type EmailService interface {
	SendPasswordResetEmail(email, token string) error
	SendWelcomeEmail(email, userName string) error
	SendVerificationEmail(email, userName, token string) error
	SendOrderConfirmationWithTickets(email, userName, subject, htmlContent, textContent string, order *models.Order, tickets []*models.Ticket) error
}



// NewAuthService creates a new authentication service
func NewAuthService(userRepo UserRepository, emailService EmailService) *AuthService {
	return &AuthService{
		userRepo:     userRepo,
		emailService: emailService,
	}
}

// RegisterRequest represents a user registration request
type RegisterRequest struct {
	Email     string           `json:"email"`
	Password  string           `json:"password"`
	FirstName string           `json:"first_name"`
	LastName  string           `json:"last_name"`
	Role      models.UserRole  `json:"role"`
}

// LoginRequest represents a user login request
type LoginRequest struct {
	Email      string `json:"email"`
	Password   string `json:"password"`
	RememberMe bool   `json:"remember_me"`
}

// PasswordResetRequest represents a password reset request
type PasswordResetRequest struct {
	Email string `json:"email"`
}

// PasswordChangeRequest represents a password change request
type PasswordChangeRequest struct {
	OldPassword string `json:"old_password"`
	NewPassword string `json:"new_password"`
}

// PasswordResetCompleteRequest represents a password reset completion request
type PasswordResetCompleteRequest struct {
	Token       string `json:"token"`
	NewPassword string `json:"new_password"`
}

// AuthResponse represents the response after successful authentication
type AuthResponse struct {
	User      *models.User `json:"user"`
	SessionID string       `json:"session_id"`
	ExpiresAt time.Time    `json:"expires_at"`
}

// Register creates a new user account
func (s *AuthService) Register(req *RegisterRequest) (*AuthResponse, error) {
	// Validate the request
	createReq := &models.UserCreateRequest{
		Email:     req.Email,
		Password:  req.Password,
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Role:      req.Role,
	}
	
	if err := createReq.Validate(); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}
	
	// Check if user already exists
	existingUser, err := s.userRepo.GetByEmail(req.Email)
	if err == nil && existingUser != nil {
		return nil, fmt.Errorf("user with email %s already exists", req.Email)
	}
	
	// Hash the password
	hashedPassword, err := utils.HashPassword(req.Password)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}
	
	// Create the user
	createReq.Password = hashedPassword
	user, err := s.userRepo.Create(createReq)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}
	
	// Generate verification token
	verificationToken, err := s.generateVerificationToken()
	if err != nil {
		return nil, fmt.Errorf("failed to generate verification token: %w", err)
	}
	
	// Set verification token for the user
	err = s.userRepo.SetVerificationToken(user.ID, verificationToken)
	if err != nil {
		return nil, fmt.Errorf("failed to set verification token: %w", err)
	}
	
	// Send verification email instead of welcome email
	if s.emailService != nil {
		userName := fmt.Sprintf("%s %s", user.FirstName, user.LastName)
		err = s.emailService.SendVerificationEmail(user.Email, userName, verificationToken)
		if err != nil {
			// Log the error but don't fail registration
			fmt.Printf("Warning: failed to send verification email to %s: %v\n", user.Email, err)
		}
	}
	
	// Don't create a session - user needs to verify email first
	// Return response without session to indicate verification needed
	return &AuthResponse{
		User:      user,
		SessionID: "", // Empty session ID indicates verification needed
		ExpiresAt: time.Time{},
	}, nil
}

// Login authenticates a user and creates a session
func (s *AuthService) Login(req *LoginRequest) (*AuthResponse, error) {
	// Validate input
	if req.Email == "" {
		return nil, fmt.Errorf("email is required")
	}
	if req.Password == "" {
		return nil, fmt.Errorf("password is required")
	}
	
	// Get user by email
	user, err := s.userRepo.GetByEmail(req.Email)
	if err != nil {
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
	
	// Check if email is verified
	if !user.EmailVerified {
		return nil, fmt.Errorf("please verify your email address before logging in")
	}
	
	// Check if user account is active
	if !user.IsActive {
		return nil, fmt.Errorf("your account has been suspended. Please contact support")
	}
	
	// Create a session with appropriate duration
	sessionID, expiresAt, err := s.createSessionWithDuration(user.ID, req.RememberMe)
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
func (s *AuthService) ValidateSession(sessionID string) (*models.User, error) {
	if sessionID == "" {
		return nil, fmt.Errorf("session ID is required")
	}
	
	user, err := s.userRepo.GetUserBySession(sessionID)
	if err != nil {
		return nil, fmt.Errorf("invalid or expired session")
	}
	
	return user, nil
}

// Logout invalidates a user session
func (s *AuthService) Logout(sessionID string) error {
	if sessionID == "" {
		return fmt.Errorf("session ID is required")
	}
	
	err := s.userRepo.DeleteSession(sessionID)
	if err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}
	
	return nil
}

// RequestPasswordReset initiates a password reset process
func (s *AuthService) RequestPasswordReset(req *PasswordResetRequest) error {
	if req.Email == "" {
		return fmt.Errorf("email is required")
	}
	
	// Check if user exists
	user, err := s.userRepo.GetByEmail(req.Email)
	if err != nil {
		// Don't reveal whether the email exists or not for security
		return nil
	}
	
	// Generate a secure reset token
	token, err := s.generateResetToken()
	if err != nil {
		return fmt.Errorf("failed to generate reset token: %w", err)
	}
	
	// Store the reset token with expiration (24 hours)
	err = s.userRepo.SetPasswordResetToken(user.ID, token, time.Now().Add(24*time.Hour))
	if err != nil {
		return fmt.Errorf("failed to store reset token: %w", err)
	}
	
	// Send password reset email
	if s.emailService != nil {
		err = s.emailService.SendPasswordResetEmail(user.Email, token)
		if err != nil {
			return fmt.Errorf("failed to send password reset email: %w", err)
		}
	}
	
	return nil
}

// CompletePasswordReset completes the password reset process
func (s *AuthService) CompletePasswordReset(req *PasswordResetCompleteRequest) error {
	if req.Token == "" {
		return fmt.Errorf("reset token is required")
	}
	if req.NewPassword == "" {
		return fmt.Errorf("new password is required")
	}
	
	// Validate new password
	if len(req.NewPassword) < 8 {
		return fmt.Errorf("password must be at least 8 characters long")
	}
	if len(req.NewPassword) > 128 {
		return fmt.Errorf("password must be less than 128 characters")
	}
	
	// Get user by reset token
	user, err := s.userRepo.GetByPasswordResetToken(req.Token)
	if err != nil {
		return fmt.Errorf("invalid or expired reset token")
	}
	
	// Hash new password
	hashedPassword, err := utils.HashPassword(req.NewPassword)
	if err != nil {
		return fmt.Errorf("failed to hash new password: %w", err)
	}
	
	// Update password
	err = s.userRepo.UpdatePassword(user.ID, hashedPassword)
	if err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}
	
	// Clear the reset token
	err = s.userRepo.ClearPasswordResetToken(user.ID)
	if err != nil {
		// Log this error but don't fail the password reset
		fmt.Printf("Warning: failed to clear password reset token: %v\n", err)
	}
	
	// Invalidate all existing sessions for this user
	err = s.userRepo.DeleteUserSessions(user.ID)
	if err != nil {
		// Log this error but don't fail the password reset
		fmt.Printf("Warning: failed to delete user sessions after password reset: %v\n", err)
	}
	
	return nil
}

// ChangePassword changes a user's password
func (s *AuthService) ChangePassword(userID int, req *PasswordChangeRequest) error {
	if req.OldPassword == "" {
		return fmt.Errorf("old password is required")
	}
	if req.NewPassword == "" {
		return fmt.Errorf("new password is required")
	}
	
	// Validate new password
	if len(req.NewPassword) < 8 {
		return fmt.Errorf("new password must be at least 8 characters long")
	}
	if len(req.NewPassword) > 128 {
		return fmt.Errorf("new password must be less than 128 characters")
	}
	
	// Get the user
	user, err := s.userRepo.GetByID(userID)
	if err != nil {
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
	err = s.userRepo.UpdatePassword(userID, hashedPassword)
	if err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}
	
	// Invalidate all existing sessions for this user
	err = s.userRepo.DeleteUserSessions(userID)
	if err != nil {
		// Log this error but don't fail the password change
		// In a real implementation, you'd use a proper logger
		fmt.Printf("Warning: failed to delete user sessions after password change: %v\n", err)
	}
	
	return nil
}

// RequireRole checks if a user has the required role
func (s *AuthService) RequireRole(user *models.User, requiredRole models.UserRole) error {
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
func (s *AuthService) RequireRoles(user *models.User, requiredRoles ...models.UserRole) error {
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
func (s *AuthService) CleanupExpiredSessions() error {
	err := s.userRepo.DeleteExpiredSessions()
	if err != nil {
		return fmt.Errorf("failed to cleanup expired sessions: %w", err)
	}
	
	return nil
}

// createSession creates a new session for a user with default duration
func (s *AuthService) createSession(userID int) (string, time.Time, error) {
	return s.createSessionWithDuration(userID, false)
}

// createSessionWithDuration creates a new session for a user with specified duration
func (s *AuthService) createSessionWithDuration(userID int, rememberMe bool) (string, time.Time, error) {
	// Generate a secure session ID
	sessionBytes := make([]byte, 32)
	if _, err := rand.Read(sessionBytes); err != nil {
		return "", time.Time{}, fmt.Errorf("failed to generate session ID: %w", err)
	}
	sessionID := hex.EncodeToString(sessionBytes)
	
	// Set session expiration based on remember me option
	var expiresAt time.Time
	if rememberMe {
		// 30 days for remember me
		expiresAt = time.Now().Add(30 * 24 * time.Hour)
	} else {
		// 24 hours for regular sessions
		expiresAt = time.Now().Add(24 * time.Hour)
	}
	
	// Store the session
	err := s.userRepo.CreateSession(userID, sessionID, expiresAt)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to store session: %w", err)
	}
	
	return sessionID, expiresAt, nil
}

// generateResetToken generates a secure token for password reset
func (s *AuthService) generateResetToken() (string, error) {
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", fmt.Errorf("failed to generate reset token: %w", err)
	}
	return hex.EncodeToString(tokenBytes), nil
}

// generateVerificationToken generates a secure token for email verification
func (s *AuthService) generateVerificationToken() (string, error) {
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", fmt.Errorf("failed to generate verification token: %w", err)
	}
	return hex.EncodeToString(tokenBytes), nil
}

// VerifyEmail verifies a user's email address using a verification token
func (s *AuthService) VerifyEmail(token string) (*models.User, error) {
	if token == "" {
		return nil, fmt.Errorf("verification token is required")
	}
	
	// Get user by verification token
	user, err := s.userRepo.GetByVerificationToken(token)
	if err != nil {
		return nil, fmt.Errorf("invalid or expired verification token")
	}
	
	// Check if email is already verified
	if user.EmailVerified {
		return user, nil // Already verified, return success
	}
	
	// Mark email as verified
	err = s.userRepo.VerifyEmail(user.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to verify email: %w", err)
	}
	
	// Send welcome email after successful verification
	if s.emailService != nil {
		userName := fmt.Sprintf("%s %s", user.FirstName, user.LastName)
		err = s.emailService.SendWelcomeEmail(user.Email, userName)
		if err != nil {
			// Log the error but don't fail verification
			fmt.Printf("Warning: failed to send welcome email to %s: %v\n", user.Email, err)
		}
	}
	
	// Get updated user data
	verifiedUser, err := s.userRepo.GetByID(user.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get updated user data: %w", err)
	}
	
	return verifiedUser, nil
}

// ExtendSession extends the duration of an existing session
func (s *AuthService) ExtendSession(sessionID string, duration time.Duration) error {
	if sessionID == "" {
		return fmt.Errorf("session ID is required")
	}
	
	// Validate that the session exists
	_, err := s.ValidateSession(sessionID)
	if err != nil {
		return fmt.Errorf("invalid session: %w", err)
	}
	
	// Extend the session
	err = s.userRepo.ExtendSession(sessionID, time.Now().Add(duration))
	if err != nil {
		return fmt.Errorf("failed to extend session: %w", err)
	}
	
	return nil
}

// LogoutAllSessions logs out all sessions for a user
func (s *AuthService) LogoutAllSessions(userID int) error {
	err := s.userRepo.DeleteUserSessions(userID)
	if err != nil {
		return fmt.Errorf("failed to logout all sessions: %w", err)
	}
	
	return nil
}

// ResendVerificationEmail resends the verification email to a user with rate limiting
func (s *AuthService) ResendVerificationEmail(email string) error {
	if email == "" {
		return fmt.Errorf("email is required")
	}
	
	// Get user by email
	user, err := s.userRepo.GetByEmail(email)
	if err != nil {
		return fmt.Errorf("user not found")
	}
	
	// Check if email is already verified
	if user.EmailVerified {
		return fmt.Errorf("email is already verified")
	}
	
	// Check rate limiting - allow resend only if last verification token was created more than 5 minutes ago
	if user.UpdatedAt.After(time.Now().Add(-5 * time.Minute)) {
		return fmt.Errorf("please wait at least 5 minutes before requesting another verification email")
	}
	
	// Generate new verification token
	verificationToken, err := s.generateVerificationToken()
	if err != nil {
		return fmt.Errorf("failed to generate verification token: %w", err)
	}
	
	// Update verification token
	err = s.userRepo.SetVerificationToken(user.ID, verificationToken)
	if err != nil {
		return fmt.Errorf("failed to set verification token: %w", err)
	}
	
	// Send verification email
	if s.emailService != nil {
		userName := fmt.Sprintf("%s %s", user.FirstName, user.LastName)
		err = s.emailService.SendVerificationEmail(user.Email, userName, verificationToken)
		if err != nil {
			return fmt.Errorf("failed to send verification email: %w", err)
		}
	}
	
	return nil
}

// CleanupExpiredTokens removes expired verification and reset tokens
func (s *AuthService) CleanupExpiredTokens() error {
	err := s.userRepo.CleanupExpiredTokens()
	if err != nil {
		return fmt.Errorf("failed to cleanup expired tokens: %w", err)
	}
	
	return nil
}

// ValidatePasswordResetToken validates a password reset token without using it
func (s *AuthService) ValidatePasswordResetToken(token string) (*models.User, error) {
	if token == "" {
		return nil, fmt.Errorf("reset token is required")
	}
	
	// Get user by reset token
	user, err := s.userRepo.GetByPasswordResetToken(token)
	if err != nil {
		return nil, fmt.Errorf("invalid or expired reset token")
	}
	
	return user, nil
}