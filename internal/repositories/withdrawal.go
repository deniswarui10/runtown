package repositories

import (
	"database/sql"
	"fmt"
	"time"

	"event-ticketing-platform/internal/models"
)

// WithdrawalRepository handles withdrawal data operations
type WithdrawalRepository struct {
	db *sql.DB
}

// NewWithdrawalRepository creates a new withdrawal repository
func NewWithdrawalRepository(db *sql.DB) *WithdrawalRepository {
	return &WithdrawalRepository{db: db}
}

// Create creates a new withdrawal request
func (r *WithdrawalRepository) Create(organizerID int, req *models.WithdrawalCreateRequest) (*models.Withdrawal, error) {
	query := `
		INSERT INTO withdrawals (organizer_id, amount, reason, bank_details, requested_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, organizer_id, amount, status, reason, bank_details, notes, admin_notes, 
		          requested_at, processed_at, created_at, updated_at`

	withdrawal := &models.Withdrawal{}
	var processedAt sql.NullTime
	now := time.Now()

	err := r.db.QueryRow(query, organizerID, req.Amount, req.Reason, req.BankDetails, now).Scan(
		&withdrawal.ID,
		&withdrawal.OrganizerID,
		&withdrawal.Amount,
		&withdrawal.Status,
		&withdrawal.Reason,
		&withdrawal.BankDetails,
		&withdrawal.Notes,
		&withdrawal.AdminNotes,
		&withdrawal.RequestedAt,
		&processedAt,
		&withdrawal.CreatedAt,
		&withdrawal.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create withdrawal: %w", err)
	}

	if processedAt.Valid {
		withdrawal.ProcessedAt = &processedAt.Time
	}

	return withdrawal, nil
}

// GetByID retrieves a withdrawal by ID
func (r *WithdrawalRepository) GetByID(id int) (*models.Withdrawal, error) {
	query := `
		SELECT w.id, w.organizer_id, w.amount, w.status, w.reason, w.bank_details, 
		       w.notes, w.admin_notes, w.requested_at, w.processed_at, w.created_at, w.updated_at,
		       u.first_name, u.last_name, u.email
		FROM withdrawals w
		JOIN users u ON w.organizer_id = u.id
		WHERE w.id = $1`

	withdrawal := &models.Withdrawal{
		Organizer: &models.User{},
	}
	var processedAt sql.NullTime

	err := r.db.QueryRow(query, id).Scan(
		&withdrawal.ID,
		&withdrawal.OrganizerID,
		&withdrawal.Amount,
		&withdrawal.Status,
		&withdrawal.Reason,
		&withdrawal.BankDetails,
		&withdrawal.Notes,
		&withdrawal.AdminNotes,
		&withdrawal.RequestedAt,
		&processedAt,
		&withdrawal.CreatedAt,
		&withdrawal.UpdatedAt,
		&withdrawal.Organizer.FirstName,
		&withdrawal.Organizer.LastName,
		&withdrawal.Organizer.Email,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("withdrawal not found")
		}
		return nil, fmt.Errorf("failed to get withdrawal: %w", err)
	}

	if processedAt.Valid {
		withdrawal.ProcessedAt = &processedAt.Time
	}

	return withdrawal, nil
}

// GetByOrganizer retrieves withdrawals for a specific organizer
func (r *WithdrawalRepository) GetByOrganizer(organizerID int, limit, offset int) ([]*models.Withdrawal, int, error) {
	// Get total count
	countQuery := "SELECT COUNT(*) FROM withdrawals WHERE organizer_id = $1"
	var totalCount int
	err := r.db.QueryRow(countQuery, organizerID).Scan(&totalCount)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get withdrawal count: %w", err)
	}

	// Get withdrawals
	query := `
		SELECT id, organizer_id, amount, status, reason, bank_details, notes, admin_notes,
		       requested_at, processed_at, created_at, updated_at
		FROM withdrawals
		WHERE organizer_id = $1
		ORDER BY requested_at DESC
		LIMIT $2 OFFSET $3`

	rows, err := r.db.Query(query, organizerID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query withdrawals: %w", err)
	}
	defer rows.Close()

	var withdrawals []*models.Withdrawal
	for rows.Next() {
		withdrawal := &models.Withdrawal{}
		var processedAt sql.NullTime

		err := rows.Scan(
			&withdrawal.ID,
			&withdrawal.OrganizerID,
			&withdrawal.Amount,
			&withdrawal.Status,
			&withdrawal.Reason,
			&withdrawal.BankDetails,
			&withdrawal.Notes,
			&withdrawal.AdminNotes,
			&withdrawal.RequestedAt,
			&processedAt,
			&withdrawal.CreatedAt,
			&withdrawal.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan withdrawal: %w", err)
		}

		if processedAt.Valid {
			withdrawal.ProcessedAt = &processedAt.Time
		}

		withdrawals = append(withdrawals, withdrawal)
	}

	if err = rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating withdrawals: %w", err)
	}

	return withdrawals, totalCount, nil
}

// GetAll retrieves all withdrawals with pagination (for admin)
func (r *WithdrawalRepository) GetAll(limit, offset int, status string) ([]*models.Withdrawal, int, error) {
	// Build WHERE clause
	whereClause := ""
	args := []interface{}{}
	argIndex := 1

	if status != "" {
		whereClause = "WHERE w.status = $1"
		args = append(args, status)
		argIndex++
	}

	// Get total count
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM withdrawals w %s", whereClause)
	var totalCount int
	err := r.db.QueryRow(countQuery, args...).Scan(&totalCount)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get withdrawal count: %w", err)
	}

	// Get withdrawals
	query := fmt.Sprintf(`
		SELECT w.id, w.organizer_id, w.amount, w.status, w.reason, w.bank_details,
		       w.notes, w.admin_notes, w.requested_at, w.processed_at, w.created_at, w.updated_at,
		       u.first_name, u.last_name, u.email
		FROM withdrawals w
		JOIN users u ON w.organizer_id = u.id
		%s
		ORDER BY w.requested_at DESC
		LIMIT $%d OFFSET $%d`, whereClause, argIndex, argIndex+1)

	args = append(args, limit, offset)

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query withdrawals: %w", err)
	}
	defer rows.Close()

	var withdrawals []*models.Withdrawal
	for rows.Next() {
		withdrawal := &models.Withdrawal{
			Organizer: &models.User{},
		}
		var processedAt sql.NullTime

		err := rows.Scan(
			&withdrawal.ID,
			&withdrawal.OrganizerID,
			&withdrawal.Amount,
			&withdrawal.Status,
			&withdrawal.Reason,
			&withdrawal.BankDetails,
			&withdrawal.Notes,
			&withdrawal.AdminNotes,
			&withdrawal.RequestedAt,
			&processedAt,
			&withdrawal.CreatedAt,
			&withdrawal.UpdatedAt,
			&withdrawal.Organizer.FirstName,
			&withdrawal.Organizer.LastName,
			&withdrawal.Organizer.Email,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan withdrawal: %w", err)
		}

		if processedAt.Valid {
			withdrawal.ProcessedAt = &processedAt.Time
		}

		withdrawals = append(withdrawals, withdrawal)
	}

	if err = rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating withdrawals: %w", err)
	}

	return withdrawals, totalCount, nil
}

// UpdateStatus updates the status of a withdrawal
func (r *WithdrawalRepository) UpdateStatus(id int, status models.WithdrawalStatus, adminNotes string) error {
	query := `
		UPDATE withdrawals 
		SET status = $1, admin_notes = $2, processed_at = $3, updated_at = $4
		WHERE id = $5`

	now := time.Now()
	_, err := r.db.Exec(query, status, adminNotes, now, now, id)
	if err != nil {
		return fmt.Errorf("failed to update withdrawal status: %w", err)
	}

	return nil
}

// GetOrganizerBalance calculates available balance for an organizer
func (r *WithdrawalRepository) GetOrganizerBalance(organizerID int) (float64, error) {
	// Get total earnings from completed orders (convert from cents to dollars)
	query := `
		SELECT COALESCE(SUM(total_amount), 0) as total_earnings_cents
		FROM orders o
		JOIN events e ON o.event_id = e.id
		WHERE e.organizer_id = $1 AND o.status = 'completed'`

	var totalEarningsCents int64
	err := r.db.QueryRow(query, organizerID).Scan(&totalEarningsCents)
	if err != nil {
		return 0, fmt.Errorf("failed to get total earnings: %w", err)
	}

	// Convert cents to dollars
	totalEarnings := float64(totalEarningsCents) / 100.0

	// Subtract previous withdrawals
	withdrawalQuery := `
		SELECT COALESCE(SUM(amount), 0) as total_withdrawn
		FROM withdrawals
		WHERE organizer_id = $1 AND status IN ('approved', 'completed')`

	var totalWithdrawn float64
	err = r.db.QueryRow(withdrawalQuery, organizerID).Scan(&totalWithdrawn)
	if err != nil {
		return 0, fmt.Errorf("failed to get total withdrawn: %w", err)
	}

	// Calculate available balance (subtract platform fee of 5%)
	platformFee := totalEarnings * 0.05
	availableBalance := totalEarnings - platformFee - totalWithdrawn
	if availableBalance < 0 {
		availableBalance = 0
	}

	return availableBalance, nil
}