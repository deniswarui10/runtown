package repositories

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"event-ticketing-platform/internal/models"
)

// UserRepository handles user data operations
type UserRepository struct {
	db *sql.DB
}

// NewUserRepository creates a new user repository
func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

// UserSearchFilters represents filters for user search
type UserSearchFilters struct {
	Role     models.UserRole
	Email    string
	Name     string
	Limit    int
	Offset   int
	SortBy   string // "created_at", "email", "name"
	SortDesc bool
}

// Create creates a new user
func (r *UserRepository) Create(req *models.UserCreateRequest) (*models.User, error) {
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	query := `
		INSERT INTO users (email, password_hash, first_name, last_name, role, email_verified, verification_token, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, email, first_name, last_name, role, email_verified, email_verified_at, verification_token, created_at, updated_at`

	now := time.Now()
	user := &models.User{}

	var verificationToken sql.NullString
	
	err := r.db.QueryRow(
		query,
		req.Email,
		req.Password, // This should be hashed before calling this method
		req.FirstName,
		req.LastName,
		req.Role,
		false, // email_verified = false by default
		nil,   // verification_token will be set later
		now,
		now,
	).Scan(
		&user.ID,
		&user.Email,
		&user.FirstName,
		&user.LastName,
		&user.Role,
		&user.EmailVerified,
		&user.EmailVerifiedAt,
		&verificationToken,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	
	if verificationToken.Valid {
		user.VerificationToken = &verificationToken.String
	}

	if err != nil {
		if strings.Contains(err.Error(), "duplicate key") {
			return nil, fmt.Errorf("user with email %s already exists", req.Email)
		}
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return user, nil
}

// GetByID retrieves a user by ID
func (r *UserRepository) GetByID(id int) (*models.User, error) {
	query := `
		SELECT id, email, password_hash, first_name, last_name, role, email_verified, email_verified_at, verification_token, created_at, updated_at
		FROM users
		WHERE id = $1`

	user := &models.User{}
	var verificationToken sql.NullString
	
	err := r.db.QueryRow(query, id).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.FirstName,
		&user.LastName,
		&user.Role,
		&user.EmailVerified,
		&user.EmailVerifiedAt,
		&verificationToken,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	
	if verificationToken.Valid {
		user.VerificationToken = &verificationToken.String
	}

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user with id %d not found", id)
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return user, nil
}

// GetByEmail retrieves a user by email (for authentication)
func (r *UserRepository) GetByEmail(email string) (*models.User, error) {
	query := `
		SELECT id, email, password_hash, first_name, last_name, role, is_active, email_verified, email_verified_at, verification_token, created_at, updated_at
		FROM users
		WHERE email = $1`

	user := &models.User{}
	var verificationToken sql.NullString
	
	err := r.db.QueryRow(query, email).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.FirstName,
		&user.LastName,
		&user.Role,
		&user.IsActive,
		&user.EmailVerified,
		&user.EmailVerifiedAt,
		&verificationToken,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	
	if verificationToken.Valid {
		user.VerificationToken = &verificationToken.String
	}

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user with email %s not found", email)
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return user, nil
}

// Update updates a user's information
func (r *UserRepository) Update(id int, req *models.UserUpdateRequest) (*models.User, error) {
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	query := `
		UPDATE users
		SET first_name = $2, last_name = $3, role = $4, updated_at = $5
		WHERE id = $1
		RETURNING id, email, first_name, last_name, role, created_at, updated_at`

	user := &models.User{}
	err := r.db.QueryRow(
		query,
		id,
		req.FirstName,
		req.LastName,
		req.Role,
		time.Now(),
	).Scan(
		&user.ID,
		&user.Email,
		&user.FirstName,
		&user.LastName,
		&user.Role,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user with id %d not found", id)
		}
		return nil, fmt.Errorf("failed to update user: %w", err)
	}

	return user, nil
}

// UpdatePassword updates a user's password hash
func (r *UserRepository) UpdatePassword(id int, passwordHash string) error {
	query := `
		UPDATE users
		SET password_hash = $2, updated_at = $3
		WHERE id = $1`

	result, err := r.db.Exec(query, id, passwordHash, time.Now())
	if err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("user with id %d not found", id)
	}

	return nil
}

// Delete deletes a user by ID
func (r *UserRepository) Delete(id int) error {
	query := `DELETE FROM users WHERE id = $1`

	result, err := r.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("user with id %d not found", id)
	}

	return nil
}

// Search searches for users with filters and pagination
func (r *UserRepository) Search(filters UserSearchFilters) ([]*models.User, int, error) {
	// Build WHERE clause
	var conditions []string
	var args []interface{}
	argIndex := 1

	if filters.Role != "" {
		conditions = append(conditions, fmt.Sprintf("role = $%d", argIndex))
		args = append(args, filters.Role)
		argIndex++
	}

	if filters.Email != "" {
		conditions = append(conditions, fmt.Sprintf("email ILIKE $%d", argIndex))
		args = append(args, "%"+filters.Email+"%")
		argIndex++
	}

	if filters.Name != "" {
		conditions = append(conditions, fmt.Sprintf("(first_name ILIKE $%d OR last_name ILIKE $%d)", argIndex, argIndex))
		args = append(args, "%"+filters.Name+"%")
		argIndex++
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	// Build ORDER BY clause
	orderBy := "ORDER BY created_at DESC"
	if filters.SortBy != "" {
		direction := "ASC"
		if filters.SortDesc {
			direction = "DESC"
		}

		switch filters.SortBy {
		case "created_at", "email":
			orderBy = fmt.Sprintf("ORDER BY %s %s", filters.SortBy, direction)
		case "name":
			orderBy = fmt.Sprintf("ORDER BY first_name %s, last_name %s", direction, direction)
		}
	}

	// Set default pagination
	if filters.Limit <= 0 {
		filters.Limit = 20
	}
	if filters.Offset < 0 {
		filters.Offset = 0
	}

	// Get total count
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM users %s", whereClause)
	var total int
	err := r.db.QueryRow(countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get user count: %w", err)
	}

	// Get users
	query := fmt.Sprintf(`
		SELECT id, email, first_name, last_name, role, created_at, updated_at
		FROM users
		%s
		%s
		LIMIT $%d OFFSET $%d`,
		whereClause, orderBy, argIndex, argIndex+1)

	args = append(args, filters.Limit, filters.Offset)

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to search users: %w", err)
	}
	defer rows.Close()

	var users []*models.User
	for rows.Next() {
		user := &models.User{}
		err := rows.Scan(
			&user.ID,
			&user.Email,
			&user.FirstName,
			&user.LastName,
			&user.Role,
			&user.CreatedAt,
			&user.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan user: %w", err)
		}
		users = append(users, user)
	}

	if err = rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating users: %w", err)
	}

	return users, total, nil
}

// GetByRole retrieves users by role
func (r *UserRepository) GetByRole(role models.UserRole) ([]*models.User, error) {
	query := `
		SELECT id, email, first_name, last_name, role, created_at, updated_at
		FROM users
		WHERE role = $1
		ORDER BY created_at DESC`

	rows, err := r.db.Query(query, role)
	if err != nil {
		return nil, fmt.Errorf("failed to get users by role: %w", err)
	}
	defer rows.Close()

	var users []*models.User
	for rows.Next() {
		user := &models.User{}
		err := rows.Scan(
			&user.ID,
			&user.Email,
			&user.FirstName,
			&user.LastName,
			&user.Role,
			&user.CreatedAt,
			&user.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}
		users = append(users, user)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating users: %w", err)
	}

	return users, nil
}

// Session management methods

// CreateSession creates a new session for a user
func (r *UserRepository) CreateSession(userID int, sessionID string, expiresAt time.Time) error {
	query := `
		INSERT INTO sessions (id, user_id, expires_at, created_at)
		VALUES ($1, $2, $3, $4)`

	_, err := r.db.Exec(query, sessionID, userID, expiresAt, time.Now())
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}

	return nil
}

// GetUserBySession retrieves a user by session ID
func (r *UserRepository) GetUserBySession(sessionID string) (*models.User, error) {
	query := `
		SELECT u.id, u.email, u.password_hash, u.first_name, u.last_name, u.role, u.created_at, u.updated_at
		FROM users u
		JOIN sessions s ON u.id = s.user_id
		WHERE s.id = $1 AND s.expires_at > $2`

	user := &models.User{}
	err := r.db.QueryRow(query, sessionID, time.Now()).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.FirstName,
		&user.LastName,
		&user.Role,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("session not found or expired")
		}
		return nil, fmt.Errorf("failed to get user by session: %w", err)
	}

	return user, nil
}

// DeleteSession deletes a session
func (r *UserRepository) DeleteSession(sessionID string) error {
	query := `DELETE FROM sessions WHERE id = $1`

	_, err := r.db.Exec(query, sessionID)
	if err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}

	return nil
}

// DeleteExpiredSessions deletes all expired sessions
func (r *UserRepository) DeleteExpiredSessions() error {
	query := `DELETE FROM sessions WHERE expires_at <= $1`

	_, err := r.db.Exec(query, time.Now())
	if err != nil {
		return fmt.Errorf("failed to delete expired sessions: %w", err)
	}

	return nil
}

// DeleteUserSessions deletes all sessions for a specific user
func (r *UserRepository) DeleteUserSessions(userID int) error {
	query := `DELETE FROM sessions WHERE user_id = $1`

	_, err := r.db.Exec(query, userID)
	if err != nil {
		return fmt.Errorf("failed to delete user sessions: %w", err)
	}

	return nil
}

// Email verification methods

// SetVerificationToken sets a verification token for a user
func (r *UserRepository) SetVerificationToken(userID int, token string) error {
	query := `
		UPDATE users
		SET verification_token = $2, updated_at = $3
		WHERE id = $1`

	result, err := r.db.Exec(query, userID, token, time.Now())
	if err != nil {
		return fmt.Errorf("failed to set verification token: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("user with id %d not found", userID)
	}

	return nil
}

// GetByVerificationToken retrieves a user by verification token
func (r *UserRepository) GetByVerificationToken(token string) (*models.User, error) {
	query := `
		SELECT id, email, password_hash, first_name, last_name, role, email_verified, email_verified_at, verification_token, created_at, updated_at
		FROM users
		WHERE verification_token = $1`

	user := &models.User{}
	var verificationToken sql.NullString
	
	err := r.db.QueryRow(query, token).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.FirstName,
		&user.LastName,
		&user.Role,
		&user.EmailVerified,
		&user.EmailVerifiedAt,
		&verificationToken,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	
	if verificationToken.Valid {
		user.VerificationToken = &verificationToken.String
	}

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user with verification token not found")
		}
		return nil, fmt.Errorf("failed to get user by verification token: %w", err)
	}

	return user, nil
}

// VerifyEmail marks a user's email as verified
func (r *UserRepository) VerifyEmail(userID int) error {
	now := time.Now()
	query := `
		UPDATE users
		SET email_verified = TRUE, email_verified_at = $2, verification_token = NULL, updated_at = $3
		WHERE id = $1`

	result, err := r.db.Exec(query, userID, now, now)
	if err != nil {
		return fmt.Errorf("failed to verify email: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("user with id %d not found", userID)
	}

	return nil
}

// ExtendSession extends the expiration time of a session
func (r *UserRepository) ExtendSession(sessionID string, expiresAt time.Time) error {
	query := `
		UPDATE sessions
		SET expires_at = $2
		WHERE id = $1`

	result, err := r.db.Exec(query, sessionID, expiresAt)
	if err != nil {
		return fmt.Errorf("failed to extend session: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("session with id %s not found", sessionID)
	}

	return nil
}

// SetPasswordResetToken sets a password reset token for a user
func (r *UserRepository) SetPasswordResetToken(userID int, token string, expiresAt time.Time) error {
	query := `UPDATE users SET password_reset_token = $2, password_reset_expires = $3, updated_at = $4 WHERE id = $1`
	
	result, err := r.db.Exec(query, userID, token, expiresAt, time.Now())
	if err != nil {
		return fmt.Errorf("failed to set password reset token: %w", err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	
	if rowsAffected == 0 {
		return fmt.Errorf("user with id %d not found", userID)
	}
	
	return nil
}

// GetByPasswordResetToken retrieves a user by password reset token
func (r *UserRepository) GetByPasswordResetToken(token string) (*models.User, error) {
	query := `
		SELECT id, email, password_hash, first_name, last_name, role, email_verified, email_verified_at, 
		       verification_token, password_reset_token, password_reset_expires, created_at, updated_at
		FROM users 
		WHERE password_reset_token = $1 AND password_reset_expires > $2`
	
	user := &models.User{}
	var emailVerifiedAt sql.NullTime
	var verificationToken sql.NullString
	var passwordResetToken sql.NullString
	var passwordResetExpires sql.NullTime
	
	err := r.db.QueryRow(query, token, time.Now()).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.FirstName,
		&user.LastName,
		&user.Role,
		&user.EmailVerified,
		&emailVerifiedAt,
		&verificationToken,
		&passwordResetToken,
		&passwordResetExpires,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("invalid or expired password reset token")
		}
		return nil, fmt.Errorf("failed to get user by password reset token: %w", err)
	}
	
	// Set nullable fields
	if emailVerifiedAt.Valid {
		user.EmailVerifiedAt = &emailVerifiedAt.Time
	}
	if verificationToken.Valid {
		user.VerificationToken = &verificationToken.String
	}
	if passwordResetToken.Valid {
		user.PasswordResetToken = &passwordResetToken.String
	}
	if passwordResetExpires.Valid {
		user.PasswordResetExpires = &passwordResetExpires.Time
	}
	
	return user, nil
}

// ClearPasswordResetToken clears the password reset token for a user
func (r *UserRepository) ClearPasswordResetToken(userID int) error {
	query := `UPDATE users SET password_reset_token = NULL, password_reset_expires = NULL, updated_at = $2 WHERE id = $1`
	
	result, err := r.db.Exec(query, userID, time.Now())
	if err != nil {
		return fmt.Errorf("failed to clear password reset token: %w", err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	
	if rowsAffected == 0 {
		return fmt.Errorf("user with id %d not found", userID)
	}
	
	return nil
}

// CleanupExpiredTokens removes expired verification and password reset tokens
func (r *UserRepository) CleanupExpiredTokens() error {
	// Clean up expired password reset tokens
	query1 := `UPDATE users SET password_reset_token = NULL, password_reset_expires = NULL, updated_at = $1 
	           WHERE password_reset_expires IS NOT NULL AND password_reset_expires < $1`
	
	_, err := r.db.Exec(query1, time.Now())
	if err != nil {
		return fmt.Errorf("failed to cleanup expired password reset tokens: %w", err)
	}
	
	// Clean up expired verification tokens (older than 24 hours)
	query2 := `UPDATE users SET verification_token = NULL, updated_at = $1 
	           WHERE verification_token IS NOT NULL AND email_verified = false AND created_at < $2`
	
	expirationTime := time.Now().Add(-24 * time.Hour)
	_, err = r.db.Exec(query2, time.Now(), expirationTime)
	if err != nil {
		return fmt.Errorf("failed to cleanup expired verification tokens: %w", err)
	}
	
	return nil
}

// Admin-specific methods

// GetUsersWithPagination retrieves users with pagination and filtering
func (r *UserRepository) GetUsersWithPagination(page, limit int, search, roleFilter string) ([]*models.User, int, error) {
	offset := (page - 1) * limit
	
	// Build the WHERE clause
	var whereConditions []string
	var args []interface{}
	argIndex := 1
	
	if search != "" {
		whereConditions = append(whereConditions, fmt.Sprintf("(first_name ILIKE $%d OR last_name ILIKE $%d OR email ILIKE $%d)", argIndex, argIndex+1, argIndex+2))
		searchPattern := "%" + search + "%"
		args = append(args, searchPattern, searchPattern, searchPattern)
		argIndex += 3
	}
	
	if roleFilter != "" {
		whereConditions = append(whereConditions, fmt.Sprintf("role = $%d", argIndex))
		args = append(args, roleFilter)
		argIndex++
	}
	
	whereClause := ""
	if len(whereConditions) > 0 {
		whereClause = "WHERE " + strings.Join(whereConditions, " AND ")
	}
	
	// Get total count
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM users %s", whereClause)
	var totalCount int
	err := r.db.QueryRow(countQuery, args...).Scan(&totalCount)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get user count: %w", err)
	}
	
	// Get users with pagination
	query := fmt.Sprintf(`
		SELECT id, email, first_name, last_name, role, email_verified, email_verified_at, 
		       is_active, created_at, updated_at
		FROM users %s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d`, whereClause, argIndex, argIndex+1)
	
	args = append(args, limit, offset)
	
	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query users: %w", err)
	}
	defer rows.Close()
	
	var users []*models.User
	for rows.Next() {
		user := &models.User{}
		var emailVerifiedAt sql.NullTime
		
		err := rows.Scan(
			&user.ID,
			&user.Email,
			&user.FirstName,
			&user.LastName,
			&user.Role,
			&user.EmailVerified,
			&emailVerifiedAt,
			&user.IsActive,
			&user.CreatedAt,
			&user.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan user: %w", err)
		}
		
		if emailVerifiedAt.Valid {
			user.EmailVerifiedAt = &emailVerifiedAt.Time
		}
		
		users = append(users, user)
	}
	
	if err = rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating users: %w", err)
	}
	
	return users, totalCount, nil
}

// GetUserCount returns the total number of users
func (r *UserRepository) GetUserCount() (int, error) {
	var count int
	err := r.db.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get user count: %w", err)
	}
	return count, nil
}

// GetActiveUserCount returns the number of active users
func (r *UserRepository) GetActiveUserCount() (int, error) {
	var count int
	err := r.db.QueryRow("SELECT COUNT(*) FROM users WHERE is_active = true").Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get active user count: %w", err)
	}
	return count, nil
}

// UpdateUserRole updates a user's role
func (r *UserRepository) UpdateUserRole(userID int, role models.UserRole) error {
	query := "UPDATE users SET role = $1, updated_at = $2 WHERE id = $3"
	_, err := r.db.Exec(query, role, time.Now(), userID)
	if err != nil {
		return fmt.Errorf("failed to update user role: %w", err)
	}
	return nil
}

// UpdateUserStatus updates a user's active status
func (r *UserRepository) UpdateUserStatus(userID int, isActive bool) error {
	query := "UPDATE users SET is_active = $1, updated_at = $2 WHERE id = $3"
	_, err := r.db.Exec(query, isActive, time.Now(), userID)
	if err != nil {
		return fmt.Errorf("failed to update user status: %w", err)
	}
	return nil
}