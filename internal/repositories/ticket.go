package repositories

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"event-ticketing-platform/internal/models"
)

// TicketRepository handles ticket and ticket type data operations
type TicketRepository struct {
	db *sql.DB
}

// NewTicketRepository creates a new ticket repository
func NewTicketRepository(db *sql.DB) *TicketRepository {
	return &TicketRepository{db: db}
}

// TicketReservation represents a temporary ticket reservation
type TicketReservation struct {
	ID           string    `json:"id"`
	TicketTypeID int       `json:"ticket_type_id"`
	Quantity     int       `json:"quantity"`
	UserID       int       `json:"user_id"`
	ExpiresAt    time.Time `json:"expires_at"`
	CreatedAt    time.Time `json:"created_at"`
}

// TicketSearchFilters represents filters for ticket search
type TicketSearchFilters struct {
	EventID      int                 // Filter by event
	OrderID      int                 // Filter by order
	UserID       int                 // Filter by user (through order)
	Status       models.TicketStatus // Filter by status
	Limit        int                 // Number of results to return
	Offset       int                 // Number of results to skip
	SortBy       string              // "created_at", "status"
	SortDesc     bool                // Sort in descending order
}

// TicketType operations

// CreateTicketType creates a new ticket type
func (r *TicketRepository) CreateTicketType(req *models.TicketTypeCreateRequest) (*models.TicketType, error) {
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	query := `
		INSERT INTO ticket_types (event_id, name, description, price, quantity, sold, sale_start, sale_end, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, event_id, name, description, price, quantity, sold, sale_start, sale_end, created_at`

	ticketType := &models.TicketType{}
	err := r.db.QueryRow(
		query,
		req.EventID,
		req.Name,
		req.Description,
		req.Price,
		req.Quantity,
		0, // Initial sold count
		req.SaleStart,
		req.SaleEnd,
		time.Now(),
	).Scan(
		&ticketType.ID,
		&ticketType.EventID,
		&ticketType.Name,
		&ticketType.Description,
		&ticketType.Price,
		&ticketType.Quantity,
		&ticketType.Sold,
		&ticketType.SaleStart,
		&ticketType.SaleEnd,
		&ticketType.CreatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create ticket type: %w", err)
	}

	return ticketType, nil
}

// GetTicketTypeByID retrieves a ticket type by ID
func (r *TicketRepository) GetTicketTypeByID(id int) (*models.TicketType, error) {
	query := `
		SELECT id, event_id, name, description, price, quantity, sold, sale_start, sale_end, created_at
		FROM ticket_types
		WHERE id = $1`

	ticketType := &models.TicketType{}
	err := r.db.QueryRow(query, id).Scan(
		&ticketType.ID,
		&ticketType.EventID,
		&ticketType.Name,
		&ticketType.Description,
		&ticketType.Price,
		&ticketType.Quantity,
		&ticketType.Sold,
		&ticketType.SaleStart,
		&ticketType.SaleEnd,
		&ticketType.CreatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("ticket type with id %d not found", id)
		}
		return nil, fmt.Errorf("failed to get ticket type: %w", err)
	}

	return ticketType, nil
}

// GetTicketTypesByEvent retrieves all ticket types for an event
func (r *TicketRepository) GetTicketTypesByEvent(eventID int) ([]*models.TicketType, error) {
	query := `
		SELECT id, event_id, name, description, price, quantity, sold, sale_start, sale_end, created_at
		FROM ticket_types
		WHERE event_id = $1
		ORDER BY price ASC, created_at ASC`

	rows, err := r.db.Query(query, eventID)
	if err != nil {
		return nil, fmt.Errorf("failed to get ticket types by event: %w", err)
	}
	defer rows.Close()

	var ticketTypes []*models.TicketType
	for rows.Next() {
		ticketType := &models.TicketType{}
		err := rows.Scan(
			&ticketType.ID,
			&ticketType.EventID,
			&ticketType.Name,
			&ticketType.Description,
			&ticketType.Price,
			&ticketType.Quantity,
			&ticketType.Sold,
			&ticketType.SaleStart,
			&ticketType.SaleEnd,
			&ticketType.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan ticket type: %w", err)
		}
		ticketTypes = append(ticketTypes, ticketType)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating ticket types: %w", err)
	}

	return ticketTypes, nil
}

// UpdateTicketType updates a ticket type
func (r *TicketRepository) UpdateTicketType(id int, req *models.TicketTypeUpdateRequest) (*models.TicketType, error) {
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// First get the existing ticket type to validate quantity update
	existing, err := r.GetTicketTypeByID(id)
	if err != nil {
		return nil, err
	}

	// Validate that new quantity is not less than sold tickets
	if !existing.CanUpdateQuantity(req.Quantity) {
		return nil, fmt.Errorf("cannot reduce quantity below sold tickets (%d)", existing.Sold)
	}

	query := `
		UPDATE ticket_types
		SET name = $2, description = $3, price = $4, quantity = $5, sale_start = $6, sale_end = $7
		WHERE id = $1
		RETURNING id, event_id, name, description, price, quantity, sold, sale_start, sale_end, created_at`

	ticketType := &models.TicketType{}
	err = r.db.QueryRow(
		query,
		id,
		req.Name,
		req.Description,
		req.Price,
		req.Quantity,
		req.SaleStart,
		req.SaleEnd,
	).Scan(
		&ticketType.ID,
		&ticketType.EventID,
		&ticketType.Name,
		&ticketType.Description,
		&ticketType.Price,
		&ticketType.Quantity,
		&ticketType.Sold,
		&ticketType.SaleStart,
		&ticketType.SaleEnd,
		&ticketType.CreatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("ticket type with id %d not found", id)
		}
		return nil, fmt.Errorf("failed to update ticket type: %w", err)
	}

	return ticketType, nil
}

// DeleteTicketType deletes a ticket type (only if no tickets sold)
func (r *TicketRepository) DeleteTicketType(id int) error {
	// First check if any tickets have been sold
	ticketType, err := r.GetTicketTypeByID(id)
	if err != nil {
		return err
	}

	if ticketType.Sold > 0 {
		return fmt.Errorf("cannot delete ticket type with sold tickets")
	}

	query := `DELETE FROM ticket_types WHERE id = $1`

	result, err := r.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete ticket type: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("ticket type with id %d not found", id)
	}

	return nil
}

// ReserveTickets creates a temporary reservation for tickets
func (r *TicketRepository) ReserveTickets(ticketTypeID, quantity, userID int, expirationMinutes int) (*TicketReservation, error) {
	// Start transaction
	tx, err := r.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Check ticket availability with row locking
	var available int
	err = tx.QueryRow(`
		SELECT (quantity - sold) 
		FROM ticket_types 
		WHERE id = $1 
		FOR UPDATE`, ticketTypeID).Scan(&available)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("ticket type not found")
		}
		return nil, fmt.Errorf("failed to check ticket availability: %w", err)
	}

	if available < quantity {
		return nil, fmt.Errorf("insufficient tickets available (requested: %d, available: %d)", quantity, available)
	}

	// Check if ticket type is currently on sale
	var saleStart, saleEnd time.Time
	err = tx.QueryRow(`
		SELECT sale_start, sale_end 
		FROM ticket_types 
		WHERE id = $1`, ticketTypeID).Scan(&saleStart, &saleEnd)

	if err != nil {
		return nil, fmt.Errorf("failed to get sale period: %w", err)
	}

	now := time.Now()
	if now.Before(saleStart) {
		return nil, fmt.Errorf("ticket sales have not started yet")
	}
	if now.After(saleEnd) {
		return nil, fmt.Errorf("ticket sales have ended")
	}

	// Create reservation record (using a simple table for reservations)
	// In a production system, you might use Redis or a dedicated reservations table
	reservationID := fmt.Sprintf("RES-%d-%d-%d", ticketTypeID, userID, time.Now().Unix())
	expiresAt := time.Now().Add(time.Duration(expirationMinutes) * time.Minute)

	// For now, we'll temporarily increase the sold count to reserve tickets
	// This is a simplified approach - in production you'd want a proper reservations system
	_, err = tx.Exec(`
		UPDATE ticket_types 
		SET sold = sold + $2 
		WHERE id = $1`, ticketTypeID, quantity)

	if err != nil {
		return nil, fmt.Errorf("failed to reserve tickets: %w", err)
	}

	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit reservation: %w", err)
	}

	reservation := &TicketReservation{
		ID:           reservationID,
		TicketTypeID: ticketTypeID,
		Quantity:     quantity,
		UserID:       userID,
		ExpiresAt:    expiresAt,
		CreatedAt:    now,
	}

	return reservation, nil
}

// ReleaseReservation releases a ticket reservation
func (r *TicketRepository) ReleaseReservation(reservationID string, ticketTypeID, quantity int) error {
	// In a simplified approach, we just reduce the sold count
	// In production, you'd have a proper reservations table to track this
	query := `
		UPDATE ticket_types 
		SET sold = sold - $2 
		WHERE id = $1 AND sold >= $2`

	result, err := r.db.Exec(query, ticketTypeID, quantity)
	if err != nil {
		return fmt.Errorf("failed to release reservation: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("reservation not found or already released")
	}

	return nil
}

// Ticket operations

// CreateTicket creates a new ticket
func (r *TicketRepository) CreateTicket(orderID, ticketTypeID int, qrCode string) (*models.Ticket, error) {
	query := `
		INSERT INTO tickets (order_id, ticket_type_id, qr_code, status, created_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, order_id, ticket_type_id, qr_code, status, created_at`

	ticket := &models.Ticket{}
	err := r.db.QueryRow(
		query,
		orderID,
		ticketTypeID,
		qrCode,
		models.TicketActive,
		time.Now(),
	).Scan(
		&ticket.ID,
		&ticket.OrderID,
		&ticket.TicketTypeID,
		&ticket.QRCode,
		&ticket.Status,
		&ticket.CreatedAt,
	)

	if err != nil {
		if strings.Contains(err.Error(), "duplicate key") {
			return nil, fmt.Errorf("ticket with QR code already exists")
		}
		return nil, fmt.Errorf("failed to create ticket: %w", err)
	}

	return ticket, nil
}

// GetTicketByID retrieves a ticket by ID
func (r *TicketRepository) GetTicketByID(id int) (*models.Ticket, error) {
	query := `
		SELECT id, order_id, ticket_type_id, qr_code, status, created_at
		FROM tickets
		WHERE id = $1`

	ticket := &models.Ticket{}
	err := r.db.QueryRow(query, id).Scan(
		&ticket.ID,
		&ticket.OrderID,
		&ticket.TicketTypeID,
		&ticket.QRCode,
		&ticket.Status,
		&ticket.CreatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("ticket with id %d not found", id)
		}
		return nil, fmt.Errorf("failed to get ticket: %w", err)
	}

	return ticket, nil
}

// GetTicketByQRCode retrieves a ticket by QR code
func (r *TicketRepository) GetTicketByQRCode(qrCode string) (*models.Ticket, error) {
	query := `
		SELECT id, order_id, ticket_type_id, qr_code, status, created_at
		FROM tickets
		WHERE qr_code = $1`

	ticket := &models.Ticket{}
	err := r.db.QueryRow(query, qrCode).Scan(
		&ticket.ID,
		&ticket.OrderID,
		&ticket.TicketTypeID,
		&ticket.QRCode,
		&ticket.Status,
		&ticket.CreatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("ticket with QR code not found")
		}
		return nil, fmt.Errorf("failed to get ticket by QR code: %w", err)
	}

	return ticket, nil
}

// GetTicketsByOrder retrieves all tickets for an order
func (r *TicketRepository) GetTicketsByOrder(orderID int) ([]*models.Ticket, error) {
	query := `
		SELECT id, order_id, ticket_type_id, qr_code, status, created_at
		FROM tickets
		WHERE order_id = $1
		ORDER BY created_at ASC`

	rows, err := r.db.Query(query, orderID)
	if err != nil {
		return nil, fmt.Errorf("failed to get tickets by order: %w", err)
	}
	defer rows.Close()

	var tickets []*models.Ticket
	for rows.Next() {
		ticket := &models.Ticket{}
		err := rows.Scan(
			&ticket.ID,
			&ticket.OrderID,
			&ticket.TicketTypeID,
			&ticket.QRCode,
			&ticket.Status,
			&ticket.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan ticket: %w", err)
		}
		tickets = append(tickets, ticket)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating tickets: %w", err)
	}

	return tickets, nil
}

// UpdateTicketStatus updates a ticket's status
func (r *TicketRepository) UpdateTicketStatus(id int, status models.TicketStatus) error {
	// Validate status transition
	ticket, err := r.GetTicketByID(id)
	if err != nil {
		return err
	}

	// Business rule validation for status transitions
	switch status {
	case models.TicketUsed:
		if !ticket.CanBeUsed() {
			return fmt.Errorf("ticket cannot be used in current status: %s", ticket.Status)
		}
	case models.TicketRefunded:
		if !ticket.CanBeRefunded() {
			return fmt.Errorf("ticket cannot be refunded in current status: %s", ticket.Status)
		}
	}

	query := `UPDATE tickets SET status = $2 WHERE id = $1`

	result, err := r.db.Exec(query, id, status)
	if err != nil {
		return fmt.Errorf("failed to update ticket status: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("ticket with id %d not found", id)
	}

	return nil
}

// SearchTickets searches for tickets with filters
func (r *TicketRepository) SearchTickets(filters TicketSearchFilters) ([]*models.Ticket, int, error) {
	// Build WHERE clause
	var conditions []string
	var args []interface{}
	argIndex := 1

	if filters.OrderID > 0 {
		conditions = append(conditions, fmt.Sprintf("order_id = $%d", argIndex))
		args = append(args, filters.OrderID)
		argIndex++
	}

	if filters.Status != "" {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argIndex))
		args = append(args, filters.Status)
		argIndex++
	}

	// Join with orders and events for additional filtering
	var joinClause string
	if filters.EventID > 0 || filters.UserID > 0 {
		joinClause = "JOIN orders o ON tickets.order_id = o.id"
		
		if filters.EventID > 0 {
			conditions = append(conditions, fmt.Sprintf("o.event_id = $%d", argIndex))
			args = append(args, filters.EventID)
			argIndex++
		}

		if filters.UserID > 0 {
			conditions = append(conditions, fmt.Sprintf("o.user_id = $%d", argIndex))
			args = append(args, filters.UserID)
			argIndex++
		}
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	// Build ORDER BY clause
	orderBy := "ORDER BY tickets.created_at DESC"
	if filters.SortBy != "" {
		direction := "ASC"
		if filters.SortDesc {
			direction = "DESC"
		}

		switch filters.SortBy {
		case "created_at", "status":
			orderBy = fmt.Sprintf("ORDER BY tickets.%s %s", filters.SortBy, direction)
		}
	}

	// Set default pagination
	if filters.Limit <= 0 {
		filters.Limit = 50
	}
	if filters.Offset < 0 {
		filters.Offset = 0
	}

	// Build the base query
	baseQuery := "FROM tickets"
	if joinClause != "" {
		baseQuery = fmt.Sprintf("FROM tickets %s", joinClause)
	}

	// Get total count
	countQuery := fmt.Sprintf("SELECT COUNT(*) %s %s", baseQuery, whereClause)
	var total int
	err := r.db.QueryRow(countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get ticket count: %w", err)
	}

	// Get tickets
	selectClause := "SELECT tickets.id, tickets.order_id, tickets.ticket_type_id, tickets.qr_code, tickets.status, tickets.created_at"
	query := fmt.Sprintf(`
		%s
		%s
		%s
		%s
		LIMIT $%d OFFSET $%d`,
		selectClause, baseQuery, whereClause, orderBy, argIndex, argIndex+1)

	args = append(args, filters.Limit, filters.Offset)

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to search tickets: %w", err)
	}
	defer rows.Close()

	var tickets []*models.Ticket
	for rows.Next() {
		ticket := &models.Ticket{}
		err := rows.Scan(
			&ticket.ID,
			&ticket.OrderID,
			&ticket.TicketTypeID,
			&ticket.QRCode,
			&ticket.Status,
			&ticket.CreatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan ticket: %w", err)
		}
		tickets = append(tickets, ticket)
	}

	if err = rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating tickets: %w", err)
	}

	return tickets, total, nil
}