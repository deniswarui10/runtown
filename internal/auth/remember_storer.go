package auth

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/aarondl/authboss/v3"
)

// RememberStorer implements authboss.RememberingServerStorer interface
type RememberStorer struct {
	db *sql.DB
}

// NewRememberStorer creates a new remember storer
func NewRememberStorer(db *sql.DB) *RememberStorer {
	return &RememberStorer{db: db}
}

// AddRememberToken adds a remember token for a user
func (s *RememberStorer) AddRememberToken(ctx context.Context, pid, token string) error {
	// Parse user ID
	userID := pid

	// Insert remember token
	query := `
		INSERT INTO authboss_remember_tokens (selector, verifier, user_id, expires_at)
		VALUES ($1, $2, $3, $4)
	`

	// Token expires in 30 days
	expiresAt := time.Now().Add(30 * 24 * time.Hour)

	_, err := s.db.ExecContext(ctx, query, token, token, userID, expiresAt)
	if err != nil {
		return fmt.Errorf("failed to add remember token: %w", err)
	}

	return nil
}

// DelRememberTokens removes all remember tokens for a user
func (s *RememberStorer) DelRememberTokens(ctx context.Context, pid string) error {
	query := `DELETE FROM authboss_remember_tokens WHERE user_id = $1`

	_, err := s.db.ExecContext(ctx, query, pid)
	if err != nil {
		return fmt.Errorf("failed to delete remember tokens: %w", err)
	}

	return nil
}

// UseRememberToken validates and uses a remember token
func (s *RememberStorer) UseRememberToken(ctx context.Context, pid, token string) error {
	// Check if token exists and is valid
	query := `
		SELECT user_id FROM authboss_remember_tokens 
		WHERE selector = $1 AND user_id = $2 AND expires_at > NOW()
	`

	var foundUserID string
	err := s.db.QueryRowContext(ctx, query, token, pid).Scan(&foundUserID)
	if err != nil {
		if err == sql.ErrNoRows {
			return authboss.ErrTokenNotFound
		}
		return fmt.Errorf("failed to validate remember token: %w", err)
	}

	// Token is valid, remove it (single use)
	deleteQuery := `DELETE FROM authboss_remember_tokens WHERE selector = $1 AND user_id = $2`
	_, err = s.db.ExecContext(ctx, deleteQuery, token, pid)
	if err != nil {
		return fmt.Errorf("failed to remove used remember token: %w", err)
	}

	return nil
}

// CleanupExpiredRememberTokens removes expired remember tokens
func (s *RememberStorer) CleanupExpiredRememberTokens(ctx context.Context) error {
	query := `DELETE FROM authboss_remember_tokens WHERE expires_at < NOW()`

	result, err := s.db.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to cleanup expired remember tokens: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	fmt.Printf("Cleaned up %d expired remember tokens\n", rowsAffected)

	return nil
}