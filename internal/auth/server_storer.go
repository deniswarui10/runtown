package auth

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"time"

	"github.com/aarondl/authboss/v3"
	"event-ticketing-platform/internal/models"
)

// ServerStorer implements authboss.ServerStorer interface
type ServerStorer struct {
	db *sql.DB
}

// NewServerStorer creates a new server storer
func NewServerStorer(db *sql.DB) *ServerStorer {
	return &ServerStorer{db: db}
}

// Load retrieves a user from the database by ID or email
func (s *ServerStorer) Load(ctx context.Context, key string) (authboss.User, error) {
	var query string
	var queryParam interface{}

	// Try to parse as user ID first
	if userID, err := strconv.Atoi(key); err == nil {
		// Key is a user ID
		fmt.Printf("[DEBUG] Loading user by ID: %d\n", userID)
		query = `
			SELECT id, email, password_hash, first_name, last_name, role, 
			       created_at, updated_at, email_verified, email_verified_at,
			       confirmed_at, confirm_selector, confirm_verifier,
			       locked_until, attempt_count, last_attempt, password_changed_at,
			       recover_selector, recover_verifier, recover_token_expires
			FROM users 
			WHERE id = $1
		`
		queryParam = userID
	} else {
		// Key is an email address
		fmt.Printf("[DEBUG] Loading user by email: %s\n", key)
		query = `
			SELECT id, email, password_hash, first_name, last_name, role, 
			       created_at, updated_at, email_verified, email_verified_at,
			       confirmed_at, confirm_selector, confirm_verifier,
			       locked_until, attempt_count, last_attempt, password_changed_at,
			       recover_selector, recover_verifier, recover_token_expires
			FROM users 
			WHERE email = $1
		`
		queryParam = key
	}

	// Initialize the user with embedded models.User
	user := &AuthbossUser{
		User: &models.User{},
	}
	var confirmedAt, lockedUntil, lastAttempt, passwordChangedAt, recoverTokenExpires sql.NullTime
	var confirmSelector, confirmVerifier, recoverSelector, recoverVerifier sql.NullString

	err := s.db.QueryRowContext(ctx, query, queryParam).Scan(
		&user.ID, &user.Email, &user.PasswordHash, &user.FirstName, &user.LastName, &user.Role,
		&user.CreatedAt, &user.UpdatedAt, &user.EmailVerified, &user.EmailVerifiedAt,
		&confirmedAt, &confirmSelector, &confirmVerifier,
		&lockedUntil, &user.AttemptCount, &lastAttempt, &passwordChangedAt,
		&recoverSelector, &recoverVerifier, &recoverTokenExpires,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, authboss.ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to load user: %w", err)
	}

	// Convert nullable fields
	if confirmedAt.Valid {
		user.ConfirmedAt = &confirmedAt.Time
	}
	if lockedUntil.Valid {
		user.LockedUntil = &lockedUntil.Time
	}
	if lastAttempt.Valid {
		user.LastAttempt = &lastAttempt.Time
	}
	if passwordChangedAt.Valid {
		user.PasswordChangedAt = &passwordChangedAt.Time
	}
	if recoverTokenExpires.Valid {
		user.RecoverTokenExpires = &recoverTokenExpires.Time
	}
	if confirmSelector.Valid {
		user.ConfirmSelector = confirmSelector.String
	}
	if confirmVerifier.Valid {
		user.ConfirmVerifier = confirmVerifier.String
	}
	if recoverSelector.Valid {
		user.RecoverSelector = recoverSelector.String
	}
	if recoverVerifier.Valid {
		user.RecoverVerifier = recoverVerifier.String
	}

	return user, nil
}

// Save persists a user to the database (handles both INSERT and UPDATE)
func (s *ServerStorer) Save(ctx context.Context, user authboss.User) error {
	authUser, ok := user.(*AuthbossUser)
	if !ok {
		return fmt.Errorf("invalid user type: expected *AuthbossUser")
	}

	if authUser.ID == 0 {
		// New user - INSERT
		query := `
			INSERT INTO users (
				email, password_hash, first_name, last_name, role,
				created_at, updated_at, email_verified, email_verified_at,
				confirmed_at, confirm_selector, confirm_verifier,
				locked_until, attempt_count, last_attempt, password_changed_at,
				recover_selector, recover_verifier, recover_token_expires
			) VALUES (
				$1, $2, $3, $4, $5,
				CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, $6, $7,
				$8, $9, $10,
				$11, $12, $13, $14,
				$15, $16, $17
			) RETURNING id, created_at, updated_at
		`

		err := s.db.QueryRowContext(ctx, query,
			authUser.Email, authUser.PasswordHash, authUser.FirstName, authUser.LastName, authUser.Role,
			authUser.EmailVerified, authUser.EmailVerifiedAt,
			authUser.ConfirmedAt, authUser.ConfirmSelector, authUser.ConfirmVerifier,
			authUser.LockedUntil, authUser.AttemptCount, authUser.LastAttempt, authUser.PasswordChangedAt,
			authUser.RecoverSelector, authUser.RecoverVerifier, authUser.RecoverTokenExpires,
		).Scan(&authUser.ID, &authUser.CreatedAt, &authUser.UpdatedAt)

		if err != nil {
			return fmt.Errorf("failed to create user: %w", err)
		}
	} else {
		// Existing user - UPDATE
		query := `
			UPDATE users SET 
				email = $2, password_hash = $3, first_name = $4, last_name = $5, role = $6,
				updated_at = CURRENT_TIMESTAMP, email_verified = $7, email_verified_at = $8,
				confirmed_at = $9, confirm_selector = $10, confirm_verifier = $11,
				locked_until = $12, attempt_count = $13, last_attempt = $14, password_changed_at = $15,
				recover_selector = $16, recover_verifier = $17, recover_token_expires = $18
			WHERE id = $1
		`

		_, err := s.db.ExecContext(ctx, query,
			authUser.ID, authUser.Email, authUser.PasswordHash, authUser.FirstName, authUser.LastName, authUser.Role,
			authUser.EmailVerified, authUser.EmailVerifiedAt,
			authUser.ConfirmedAt, authUser.ConfirmSelector, authUser.ConfirmVerifier,
			authUser.LockedUntil, authUser.AttemptCount, authUser.LastAttempt, authUser.PasswordChangedAt,
			authUser.RecoverSelector, authUser.RecoverVerifier, authUser.RecoverTokenExpires,
		)

		if err != nil {
			return fmt.Errorf("failed to update user: %w", err)
		}
	}

	return nil
}

// New creates a new user in the database
func (s *ServerStorer) New(ctx context.Context) authboss.User {
	return &AuthbossUser{
		User: &models.User{
			Role: models.UserRoleUser, // Default role
		},
		AttemptCount: 0,
	}
}

// Create inserts a new user into the database
func (s *ServerStorer) Create(ctx context.Context, user authboss.User) error {
	authUser, ok := user.(*AuthbossUser)
	if !ok {
		return fmt.Errorf("invalid user type: expected *AuthbossUser")
	}

	// Insert new user
	query := `
		INSERT INTO users (
			email, password_hash, first_name, last_name, role,
			email_verified, email_verified_at, confirmed_at, 
			confirm_selector, confirm_verifier, attempt_count, password_changed_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id, created_at, updated_at
	`

	err := s.db.QueryRowContext(ctx, query,
		authUser.Email, authUser.PasswordHash, authUser.FirstName, authUser.LastName, authUser.Role,
		authUser.EmailVerified, authUser.EmailVerifiedAt, authUser.ConfirmedAt,
		authUser.ConfirmSelector, authUser.ConfirmVerifier, authUser.AttemptCount, 
		time.Now(), // password_changed_at
	).Scan(&authUser.ID, &authUser.CreatedAt, &authUser.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	return nil
}

// LoadByConfirmSelector loads a user by confirmation selector
func (s *ServerStorer) LoadByConfirmSelector(ctx context.Context, selector string) (authboss.ConfirmableUser, error) {
	query := `
		SELECT id, email, password_hash, first_name, last_name, role, 
		       created_at, updated_at, email_verified, email_verified_at,
		       confirmed_at, confirm_selector, confirm_verifier,
		       locked_until, attempt_count, last_attempt, password_changed_at,
		       recover_selector, recover_verifier, recover_token_expires
		FROM users 
		WHERE confirm_selector = $1
	`

	var user AuthbossUser
	var confirmedAt, lockedUntil, lastAttempt, passwordChangedAt, recoverTokenExpires sql.NullTime
	var confirmSelector, confirmVerifier, recoverSelector, recoverVerifier sql.NullString

	err := s.db.QueryRowContext(ctx, query, selector).Scan(
		&user.ID, &user.Email, &user.PasswordHash, &user.FirstName, &user.LastName, &user.Role,
		&user.CreatedAt, &user.UpdatedAt, &user.EmailVerified, &user.EmailVerifiedAt,
		&confirmedAt, &confirmSelector, &confirmVerifier,
		&lockedUntil, &user.AttemptCount, &lastAttempt, &passwordChangedAt,
		&recoverSelector, &recoverVerifier, &recoverTokenExpires,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, authboss.ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to load user by confirm selector: %w", err)
	}

	// Convert nullable fields
	if confirmedAt.Valid {
		user.ConfirmedAt = &confirmedAt.Time
	}
	if lockedUntil.Valid {
		user.LockedUntil = &lockedUntil.Time
	}
	if lastAttempt.Valid {
		user.LastAttempt = &lastAttempt.Time
	}
	if passwordChangedAt.Valid {
		user.PasswordChangedAt = &passwordChangedAt.Time
	}
	if recoverTokenExpires.Valid {
		user.RecoverTokenExpires = &recoverTokenExpires.Time
	}
	if confirmSelector.Valid {
		user.ConfirmSelector = confirmSelector.String
	}
	if confirmVerifier.Valid {
		user.ConfirmVerifier = confirmVerifier.String
	}
	if recoverSelector.Valid {
		user.RecoverSelector = recoverSelector.String
	}
	if recoverVerifier.Valid {
		user.RecoverVerifier = recoverVerifier.String
	}

	return &user, nil
}

// LoadByRecoverSelector loads a user by recovery selector
func (s *ServerStorer) LoadByRecoverSelector(ctx context.Context, selector string) (authboss.RecoverableUser, error) {
	query := `
		SELECT id, email, password_hash, first_name, last_name, role, 
		       created_at, updated_at, email_verified, email_verified_at,
		       confirmed_at, confirm_selector, confirm_verifier,
		       locked_until, attempt_count, last_attempt, password_changed_at,
		       recover_selector, recover_verifier, recover_token_expires
		FROM users 
		WHERE recover_selector = $1 AND recover_token_expires > NOW()
	`

	var user AuthbossUser
	var confirmedAt, lockedUntil, lastAttempt, passwordChangedAt, recoverTokenExpires sql.NullTime
	var confirmSelector, confirmVerifier, recoverSelector, recoverVerifier sql.NullString

	err := s.db.QueryRowContext(ctx, query, selector).Scan(
		&user.ID, &user.Email, &user.PasswordHash, &user.FirstName, &user.LastName, &user.Role,
		&user.CreatedAt, &user.UpdatedAt, &user.EmailVerified, &user.EmailVerifiedAt,
		&confirmedAt, &confirmSelector, &confirmVerifier,
		&lockedUntil, &user.AttemptCount, &lastAttempt, &passwordChangedAt,
		&recoverSelector, &recoverVerifier, &recoverTokenExpires,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, authboss.ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to load user by recover selector: %w", err)
	}

	// Convert nullable fields
	if confirmedAt.Valid {
		user.ConfirmedAt = &confirmedAt.Time
	}
	if lockedUntil.Valid {
		user.LockedUntil = &lockedUntil.Time
	}
	if lastAttempt.Valid {
		user.LastAttempt = &lastAttempt.Time
	}
	if passwordChangedAt.Valid {
		user.PasswordChangedAt = &passwordChangedAt.Time
	}
	if recoverTokenExpires.Valid {
		user.RecoverTokenExpires = &recoverTokenExpires.Time
	}
	if confirmSelector.Valid {
		user.ConfirmSelector = confirmSelector.String
	}
	if confirmVerifier.Valid {
		user.ConfirmVerifier = confirmVerifier.String
	}
	if recoverSelector.Valid {
		user.RecoverSelector = recoverSelector.String
	}
	if recoverVerifier.Valid {
		user.RecoverVerifier = recoverVerifier.String
	}

	return &user, nil
}