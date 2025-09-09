package auth

import (
	"strconv"
	"time"

	"event-ticketing-platform/internal/models"
	"event-ticketing-platform/internal/utils"
)

// AuthbossUser extends the existing User model to implement Authboss interfaces
type AuthbossUser struct {
	*models.User

	// Authboss confirmation fields
	ConfirmedAt      *time.Time `db:"confirmed_at"`
	ConfirmSelector  string     `db:"confirm_selector"`
	ConfirmVerifier  string     `db:"confirm_verifier"`

	// Authboss locking fields
	LockedUntil      *time.Time `db:"locked_until"`
	AttemptCount     int        `db:"attempt_count"`
	LastAttempt      *time.Time `db:"last_attempt"`

	// Authboss password management
	PasswordChangedAt *time.Time `db:"password_changed_at"`

	// Authboss recovery fields
	RecoverSelector     string     `db:"recover_selector"`
	RecoverVerifier     string     `db:"recover_verifier"`
	RecoverTokenExpires *time.Time `db:"recover_token_expires"`
}

// NewAuthbossUser creates a new AuthbossUser from an existing User
func NewAuthbossUser(user *models.User) *AuthbossUser {
	return &AuthbossUser{
		User:         user,
		AttemptCount: 0,
	}
}

// Implement authboss.User interface

// GetPID returns the primary identifier for the user
func (u *AuthbossUser) GetPID() string {
	return strconv.Itoa(u.ID)
}

// PutPID sets the primary identifier for the user
func (u *AuthbossUser) PutPID(pid string) {
	id, _ := strconv.Atoi(pid)
	u.ID = id
}

// GetPassword returns the user's password hash
func (u *AuthbossUser) GetPassword() string {
	return u.PasswordHash
}

// PutPassword sets the user's password hash
func (u *AuthbossUser) PutPassword(password string) {
	u.PasswordHash = password
	now := time.Now()
	u.PasswordChangedAt = &now
}

// VerifyPassword verifies a plain text password against the stored hash
func (u *AuthbossUser) VerifyPassword(password string) bool {
	// Use the utils.VerifyPassword function
	valid, err := utils.VerifyPassword(password, u.PasswordHash)
	if err != nil {
		return false
	}
	return valid
}

// GetEmail returns the user's email
func (u *AuthbossUser) GetEmail() string {
	return u.Email
}

// PutEmail sets the user's email
func (u *AuthbossUser) PutEmail(email string) {
	u.Email = email
}

// Implement authboss.AuthableUser interface

// GetPassword is already implemented above

// Implement authboss.ConfirmableUser interface

// GetConfirmed returns whether the user's email is confirmed
func (u *AuthbossUser) GetConfirmed() bool {
	return u.ConfirmedAt != nil
}

// GetConfirmSelector returns the confirmation selector
func (u *AuthbossUser) GetConfirmSelector() string {
	return u.ConfirmSelector
}

// GetConfirmVerifier returns the confirmation verifier
func (u *AuthbossUser) GetConfirmVerifier() string {
	return u.ConfirmVerifier
}

// PutConfirmed sets the user's confirmation status
func (u *AuthbossUser) PutConfirmed(confirmed bool) {
	if confirmed {
		now := time.Now()
		u.ConfirmedAt = &now
		u.EmailVerified = true
		u.EmailVerifiedAt = &now
	} else {
		u.ConfirmedAt = nil
		u.EmailVerified = false
		u.EmailVerifiedAt = nil
	}
}

// PutConfirmSelector sets the confirmation selector
func (u *AuthbossUser) PutConfirmSelector(selector string) {
	u.ConfirmSelector = selector
}

// PutConfirmVerifier sets the confirmation verifier
func (u *AuthbossUser) PutConfirmVerifier(verifier string) {
	u.ConfirmVerifier = verifier
}

// Implement authboss.LockableUser interface

// GetAttemptCount returns the number of failed login attempts
func (u *AuthbossUser) GetAttemptCount() int {
	return u.AttemptCount
}

// PutAttemptCount sets the number of failed login attempts
func (u *AuthbossUser) PutAttemptCount(attempts int) {
	u.AttemptCount = attempts
}

// GetLastAttempt returns the time of the last login attempt
func (u *AuthbossUser) GetLastAttempt() time.Time {
	if u.LastAttempt != nil {
		return *u.LastAttempt
	}
	return time.Time{}
}

// PutLastAttempt sets the time of the last login attempt
func (u *AuthbossUser) PutLastAttempt(last time.Time) {
	u.LastAttempt = &last
}

// GetLocked returns when the user account is locked until
func (u *AuthbossUser) GetLocked() time.Time {
	if u.LockedUntil != nil {
		return *u.LockedUntil
	}
	return time.Time{}
}

// PutLocked sets when the user account is locked until
func (u *AuthbossUser) PutLocked(locked time.Time) {
	if locked.IsZero() {
		u.LockedUntil = nil
	} else {
		u.LockedUntil = &locked
	}
}

// Implement authboss.RecoverableUser interface

// GetRecoverSelector returns the recovery selector
func (u *AuthbossUser) GetRecoverSelector() string {
	return u.RecoverSelector
}

// GetRecoverVerifier returns the recovery verifier
func (u *AuthbossUser) GetRecoverVerifier() string {
	return u.RecoverVerifier
}

// GetRecoverExpiry returns when the recovery token expires
func (u *AuthbossUser) GetRecoverExpiry() time.Time {
	if u.RecoverTokenExpires != nil {
		return *u.RecoverTokenExpires
	}
	return time.Time{}
}

// PutRecoverSelector sets the recovery selector
func (u *AuthbossUser) PutRecoverSelector(selector string) {
	u.RecoverSelector = selector
}

// PutRecoverVerifier sets the recovery verifier
func (u *AuthbossUser) PutRecoverVerifier(verifier string) {
	u.RecoverVerifier = verifier
}

// PutRecoverExpiry sets when the recovery token expires
func (u *AuthbossUser) PutRecoverExpiry(expiry time.Time) {
	if expiry.IsZero() {
		u.RecoverTokenExpires = nil
	} else {
		u.RecoverTokenExpires = &expiry
	}
}

// Implement authboss.ArbitraryUser interface for custom fields

// GetArbitrary returns arbitrary user data (for role, name, etc.)
func (u *AuthbossUser) GetArbitrary() map[string]string {
	return map[string]string{
		"role":       string(u.Role),
		"first_name": u.FirstName,
		"last_name":  u.LastName,
		"name":       u.FirstName + " " + u.LastName,
	}
}

// PutArbitrary sets arbitrary user data
func (u *AuthbossUser) PutArbitrary(values map[string]string) {
	if role, ok := values["role"]; ok {
		u.Role = models.UserRole(role)
	}
	if firstName, ok := values["first_name"]; ok {
		u.FirstName = firstName
	}
	if lastName, ok := values["last_name"]; ok {
		u.LastName = lastName
	}
}

// Helper methods

// IsLocked returns whether the user account is currently locked
func (u *AuthbossUser) IsLocked() bool {
	if u.LockedUntil == nil {
		return false
	}
	return time.Now().Before(*u.LockedUntil)
}

// IsConfirmed returns whether the user's email is confirmed
func (u *AuthbossUser) IsConfirmed() bool {
	return u.GetConfirmed()
}

// HasRecoveryToken returns whether the user has a valid recovery token
func (u *AuthbossUser) HasRecoveryToken() bool {
	return u.RecoverSelector != "" && 
		   u.RecoverVerifier != "" && 
		   u.RecoverTokenExpires != nil && 
		   time.Now().Before(*u.RecoverTokenExpires)
}

// ClearRecoveryToken clears the recovery token
func (u *AuthbossUser) ClearRecoveryToken() {
	u.RecoverSelector = ""
	u.RecoverVerifier = ""
	u.RecoverTokenExpires = nil
}

// ClearConfirmationToken clears the confirmation token
func (u *AuthbossUser) ClearConfirmationToken() {
	u.ConfirmSelector = ""
	u.ConfirmVerifier = ""
}

// Unlock unlocks the user account
func (u *AuthbossUser) Unlock() {
	u.LockedUntil = nil
	u.AttemptCount = 0
}