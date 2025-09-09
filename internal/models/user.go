package models

import (
	"errors"
	"regexp"
	"strings"
	"time"
)

// UserRole represents the role of a user in the system
type UserRole string

const (
	UserRoleUser      UserRole = "user"
	UserRoleOrganizer UserRole = "organizer"
	UserRoleModerator UserRole = "moderator"
	UserRoleAdmin     UserRole = "admin"
	
	// Legacy aliases for backward compatibility
	RoleAttendee  UserRole = "user"
	RoleOrganizer UserRole = "organizer"
	RoleModerator UserRole = "moderator"
	RoleAdmin     UserRole = "admin"
)

// User represents a user in the system
type User struct {
	ID                int        `json:"id" db:"id"`
	Email             string     `json:"email" db:"email"`
	PasswordHash      string     `json:"-" db:"password_hash"`
	FirstName         string     `json:"first_name" db:"first_name"`
	LastName          string     `json:"last_name" db:"last_name"`
	Role              UserRole   `json:"role" db:"role"`
	IsActive          bool       `json:"is_active" db:"is_active"`
	EmailVerified        bool       `json:"email_verified" db:"email_verified"`
	EmailVerifiedAt      *time.Time `json:"email_verified_at,omitempty" db:"email_verified_at"`
	VerificationToken    *string    `json:"-" db:"verification_token"`
	PasswordResetToken   *string    `json:"-" db:"password_reset_token"`
	PasswordResetExpires *time.Time `json:"-" db:"password_reset_expires"`
	CreatedAt            time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt            time.Time  `json:"updated_at" db:"updated_at"`
}

// UserCreateRequest represents the data needed to create a new user
type UserCreateRequest struct {
	Email     string   `json:"email"`
	Password  string   `json:"password"`
	FirstName string   `json:"first_name"`
	LastName  string   `json:"last_name"`
	Role      UserRole `json:"role"`
}

// UserUpdateRequest represents the data that can be updated for a user
type UserUpdateRequest struct {
	FirstName string   `json:"first_name"`
	LastName  string   `json:"last_name"`
	Role      UserRole `json:"role"`
}

var (
	// Email validation regex
	emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
)

// Validate validates the user data
func (u *User) Validate() error {
	if err := u.validateEmail(); err != nil {
		return err
	}
	
	if err := u.validateName(); err != nil {
		return err
	}
	
	if err := u.validateRole(); err != nil {
		return err
	}
	
	return nil
}

// ValidateCreate validates user creation data
func (req *UserCreateRequest) Validate() error {
	if err := validateEmail(req.Email); err != nil {
		return err
	}
	
	if err := validatePassword(req.Password); err != nil {
		return err
	}
	
	if err := validateName(req.FirstName, req.LastName); err != nil {
		return err
	}
	
	if err := validateRole(req.Role); err != nil {
		return err
	}
	
	return nil
}

// ValidateUpdate validates user update data
func (req *UserUpdateRequest) Validate() error {
	if err := validateName(req.FirstName, req.LastName); err != nil {
		return err
	}
	
	if err := validateRole(req.Role); err != nil {
		return err
	}
	
	return nil
}

// validateEmail validates the user's email
func (u *User) validateEmail() error {
	return validateEmail(u.Email)
}

// validateName validates the user's name
func (u *User) validateName() error {
	return validateName(u.FirstName, u.LastName)
}

// validateRole validates the user's role
func (u *User) validateRole() error {
	return validateRole(u.Role)
}

// validateEmail validates an email address
func validateEmail(email string) error {
	if email == "" {
		return errors.New("email is required")
	}
	
	if len(email) > 255 {
		return errors.New("email must be less than 255 characters")
	}
	
	if !emailRegex.MatchString(email) {
		return errors.New("email format is invalid")
	}
	
	return nil
}

// validatePassword validates a password
func validatePassword(password string) error {
	if password == "" {
		return errors.New("password is required")
	}
	
	if len(password) < 8 {
		return errors.New("password must be at least 8 characters long")
	}
	
	if len(password) > 128 {
		return errors.New("password must be less than 128 characters")
	}
	
	return nil
}

// validateName validates first and last names
func validateName(firstName, lastName string) error {
	if firstName == "" {
		return errors.New("first name is required")
	}
	
	if lastName == "" {
		return errors.New("last name is required")
	}
	
	if len(firstName) > 100 {
		return errors.New("first name must be less than 100 characters")
	}
	
	if len(lastName) > 100 {
		return errors.New("last name must be less than 100 characters")
	}
	
	// Check for valid characters (letters, spaces, hyphens, apostrophes)
	nameRegex := regexp.MustCompile(`^[a-zA-Z\s\-']+$`)
	if !nameRegex.MatchString(firstName) {
		return errors.New("first name contains invalid characters")
	}
	
	if !nameRegex.MatchString(lastName) {
		return errors.New("last name contains invalid characters")
	}
	
	return nil
}

// validateRole validates a user role
func validateRole(role UserRole) error {
	switch role {
	case RoleAttendee, RoleOrganizer, RoleAdmin:
		return nil
	default:
		return errors.New("invalid user role")
	}
}

// FullName returns the user's full name
func (u *User) FullName() string {
	return strings.TrimSpace(u.FirstName + " " + u.LastName)
}

// IsAdmin returns true if the user is an admin
func (u *User) IsAdmin() bool {
	return u.Role == RoleAdmin
}

// IsOrganizer returns true if the user is an organizer
func (u *User) IsOrganizer() bool {
	return u.Role == RoleOrganizer
}

// IsAttendee returns true if the user is an attendee
func (u *User) IsAttendee() bool {
	return u.Role == RoleAttendee
}

// CanCreateEvents returns true if the user can create events
func (u *User) CanCreateEvents() bool {
	return u.Role == RoleOrganizer || u.Role == RoleAdmin
}

// CanManageUsers returns true if the user can manage other users
func (u *User) CanManageUsers() bool {
	return u.Role == RoleAdmin
}

// IsEmailVerified returns true if the user's email is verified
func (u *User) IsEmailVerified() bool {
	return u.EmailVerified
}

// CanLogin returns true if the user can log in (email must be verified)
func (u *User) CanLogin() bool {
	return u.EmailVerified
}

// NeedsEmailVerification returns true if the user needs to verify their email
func (u *User) NeedsEmailVerification() bool {
	return !u.EmailVerified
}

// Session represents a user session
type Session struct {
	ID        string    `json:"id" db:"id"`
	UserID    int       `json:"user_id" db:"user_id"`
	ExpiresAt time.Time `json:"expires_at" db:"expires_at"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// ErrNotFound represents a not found error
type ErrNotFound struct {
	Message string
}

func (e *ErrNotFound) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return "not found"
}