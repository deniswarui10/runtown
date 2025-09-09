package repositories

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"event-ticketing-platform/internal/models"
)

// OrderRepository handles order data operations
type OrderRepository struct {
	db *sql.DB
}

// NewOrderRepository creates a new order repository
func NewOrderRepository(db *sql.DB) *OrderRepository {
	return &OrderRepository{db: db}
}

// OrderSearchFilters represents filters for order search
type OrderSearchFilters struct {
	UserID      int                 // Filter by user
	EventID     int                 // Filter by event
	Status      models.OrderStatus  // Filter by status
	DateFrom    *time.Time          // Filter orders created from this date
	DateTo      *time.Time          // Filter orders created before this date
	AmountMin   *int                // Minimum amount filter (in cents)
	AmountMax   *int                // Maximum amount filter (in cents)
	Limit       int                 // Number of results to return
	Offset      int                 // Number of results to skip
	SortBy      string              // "created_at", "total_amount", "status"
	SortDesc    bool                // Sort in descending order
}

// OrderWithDetails represents an order with additional details
type OrderWithDetails struct {
	*models.Order
	EventTitle    string `json:"event_title" db:"event_title"`
	EventDate     time.Time `json:"event_date" db:"event_date"`
	TicketCount   int    `json:"ticket_count" db:"ticket_count"`
}

// Create creates a new order with transaction handling
func (r *OrderRepository) Create(req *models.OrderCreateRequest) (*models.Order, error) {
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Start transaction
	tx, err := r.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Generate unique order number
	orderNumber := models.GenerateOrderNumber()
	
	// Ensure order number is unique (retry if collision)
	for i := 0; i < 5; i++ {
		var exists bool
		err = tx.QueryRow("SELECT EXISTS(SELECT 1 FROM orders WHERE order_number = $1)", orderNumber).Scan(&exists)
		if err != nil {
			return nil, fmt.Errorf("failed to check order number uniqueness: %w", err)
		}
		if !exists {
			break
		}
		orderNumber = models.GenerateOrderNumber()
	}

	query := `
		INSERT INTO orders (user_id, event_id, order_number, total_amount, status, billing_email, billing_name, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, user_id, event_id, order_number, total_amount, status, payment_id, billing_email, billing_name, created_at, updated_at`

	now := time.Now()
	order := &models.Order{}

	err = tx.QueryRow(
		query,
		req.UserID,
		req.EventID,
		orderNumber,
		req.TotalAmount,
		req.Status,
		req.BillingEmail,
		req.BillingName,
		now,
		now,
	).Scan(
		&order.ID,
		&order.UserID,
		&order.EventID,
		&order.OrderNumber,
		&order.TotalAmount,
		&order.Status,
		&order.PaymentID,
		&order.BillingEmail,
		&order.BillingName,
		&order.CreatedAt,
		&order.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create order: %w", err)
	}

	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit order creation: %w", err)
	}

	return order, nil
}

// GetByID retrieves an order by ID
func (r *OrderRepository) GetByID(id int) (*models.Order, error) {
	query := `
		SELECT id, user_id, event_id, order_number, total_amount, status, payment_id, billing_email, billing_name, created_at, updated_at
		FROM orders
		WHERE id = $1`

	order := &models.Order{}
	err := r.db.QueryRow(query, id).Scan(
		&order.ID,
		&order.UserID,
		&order.EventID,
		&order.OrderNumber,
		&order.TotalAmount,
		&order.Status,
		&order.PaymentID,
		&order.BillingEmail,
		&order.BillingName,
		&order.CreatedAt,
		&order.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("order with id %d not found", id)
		}
		return nil, fmt.Errorf("failed to get order: %w", err)
	}

	return order, nil
}

// GetByOrderNumber retrieves an order by order number
func (r *OrderRepository) GetByOrderNumber(orderNumber string) (*models.Order, error) {
	query := `
		SELECT id, user_id, event_id, order_number, total_amount, status, payment_id, billing_email, billing_name, created_at, updated_at
		FROM orders
		WHERE order_number = $1`

	order := &models.Order{}
	err := r.db.QueryRow(query, orderNumber).Scan(
		&order.ID,
		&order.UserID,
		&order.EventID,
		&order.OrderNumber,
		&order.TotalAmount,
		&order.Status,
		&order.PaymentID,
		&order.BillingEmail,
		&order.BillingName,
		&order.CreatedAt,
		&order.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("order with number %s not found", orderNumber)
		}
		return nil, fmt.Errorf("failed to get order by number: %w", err)
	}

	return order, nil
}

// Update updates an order
func (r *OrderRepository) Update(id int, req *models.OrderUpdateRequest) (*models.Order, error) {
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	query := `
		UPDATE orders
		SET status = $2, payment_id = $3, updated_at = $4
		WHERE id = $1
		RETURNING id, user_id, event_id, order_number, total_amount, status, payment_id, billing_email, billing_name, created_at, updated_at`

	order := &models.Order{}
	err := r.db.QueryRow(
		query,
		id,
		req.Status,
		req.PaymentID,
		time.Now(),
	).Scan(
		&order.ID,
		&order.UserID,
		&order.EventID,
		&order.OrderNumber,
		&order.TotalAmount,
		&order.Status,
		&order.PaymentID,
		&order.BillingEmail,
		&order.BillingName,
		&order.CreatedAt,
		&order.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("order with id %d not found", id)
		}
		return nil, fmt.Errorf("failed to update order: %w", err)
	}

	return order, nil
}

// UpdateStatus updates only the order status
func (r *OrderRepository) UpdateStatus(id int, status models.OrderStatus) error {
	// Validate status transition
	order, err := r.GetByID(id)
	if err != nil {
		return err
	}

	// Business rule validation for status transitions
	switch status {
	case models.OrderCompleted:
		if !order.CanBeCompleted() {
			return fmt.Errorf("order cannot be completed in current status: %s", order.Status)
		}
	case models.OrderCancelled:
		if !order.CanBeCancelled() {
			return fmt.Errorf("order cannot be cancelled in current status: %s", order.Status)
		}
	case models.OrderRefunded:
		if !order.CanBeRefunded() {
			return fmt.Errorf("order cannot be refunded in current status: %s", order.Status)
		}
	}

	query := `UPDATE orders SET status = $2, updated_at = $3 WHERE id = $1`

	result, err := r.db.Exec(query, id, status, time.Now())
	if err != nil {
		return fmt.Errorf("failed to update order status: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("order with id %d not found", id)
	}

	return nil
}

// Delete deletes an order (only if it's in pending status and has no tickets)
func (r *OrderRepository) Delete(id int) error {
	// Start transaction
	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Check if order exists and is deletable
	var status models.OrderStatus
	var ticketCount int
	err = tx.QueryRow(`
		SELECT o.status, COUNT(t.id)
		FROM orders o
		LEFT JOIN tickets t ON o.id = t.order_id
		WHERE o.id = $1
		GROUP BY o.status`, id).Scan(&status, &ticketCount)

	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("order with id %d not found", id)
		}
		return fmt.Errorf("failed to check order status: %w", err)
	}

	if status != models.OrderPending {
		return fmt.Errorf("can only delete pending orders")
	}

	if ticketCount > 0 {
		return fmt.Errorf("cannot delete order with existing tickets")
	}

	// Delete the order
	_, err = tx.Exec("DELETE FROM orders WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("failed to delete order: %w", err)
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit order deletion: %w", err)
	}

	return nil
}

// GetByUser retrieves orders for a specific user
func (r *OrderRepository) GetByUser(userID int, limit, offset int) ([]*models.Order, int, error) {
	filters := OrderSearchFilters{
		UserID: userID,
		Limit:  limit,
		Offset: offset,
		SortBy: "created_at",
		SortDesc: true,
	}

	return r.Search(filters)
}

// GetByEvent retrieves orders for a specific event
func (r *OrderRepository) GetByEvent(eventID int, limit, offset int) ([]*models.Order, int, error) {
	filters := OrderSearchFilters{
		EventID: eventID,
		Limit:   limit,
		Offset:  offset,
		SortBy:  "created_at",
		SortDesc: true,
	}

	return r.Search(filters)
}

// Search searches for orders with filters and pagination
func (r *OrderRepository) Search(filters OrderSearchFilters) ([]*models.Order, int, error) {
	// Build WHERE clause
	var conditions []string
	var args []interface{}
	argIndex := 1

	if filters.UserID > 0 {
		conditions = append(conditions, fmt.Sprintf("user_id = $%d", argIndex))
		args = append(args, filters.UserID)
		argIndex++
	}

	if filters.EventID > 0 {
		conditions = append(conditions, fmt.Sprintf("event_id = $%d", argIndex))
		args = append(args, filters.EventID)
		argIndex++
	}

	if filters.Status != "" {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argIndex))
		args = append(args, filters.Status)
		argIndex++
	}

	if filters.DateFrom != nil {
		conditions = append(conditions, fmt.Sprintf("created_at >= $%d", argIndex))
		args = append(args, *filters.DateFrom)
		argIndex++
	}

	if filters.DateTo != nil {
		conditions = append(conditions, fmt.Sprintf("created_at <= $%d", argIndex))
		args = append(args, *filters.DateTo)
		argIndex++
	}

	if filters.AmountMin != nil {
		conditions = append(conditions, fmt.Sprintf("total_amount >= $%d", argIndex))
		args = append(args, *filters.AmountMin)
		argIndex++
	}

	if filters.AmountMax != nil {
		conditions = append(conditions, fmt.Sprintf("total_amount <= $%d", argIndex))
		args = append(args, *filters.AmountMax)
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
		case "created_at", "total_amount", "status":
			orderBy = fmt.Sprintf("ORDER BY %s %s", filters.SortBy, direction)
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
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM orders %s", whereClause)
	var total int
	err := r.db.QueryRow(countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get order count: %w", err)
	}

	// Get orders
	query := fmt.Sprintf(`
		SELECT id, user_id, event_id, order_number, total_amount, status, payment_id, billing_email, billing_name, created_at, updated_at
		FROM orders
		%s
		%s
		LIMIT $%d OFFSET $%d`,
		whereClause, orderBy, argIndex, argIndex+1)

	args = append(args, filters.Limit, filters.Offset)

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to search orders: %w", err)
	}
	defer rows.Close()

	var orders []*models.Order
	for rows.Next() {
		order := &models.Order{}
		err := rows.Scan(
			&order.ID,
			&order.UserID,
			&order.EventID,
			&order.OrderNumber,
			&order.TotalAmount,
			&order.Status,
			&order.PaymentID,
			&order.BillingEmail,
			&order.BillingName,
			&order.CreatedAt,
			&order.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan order: %w", err)
		}
		orders = append(orders, order)
	}

	if err = rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating orders: %w", err)
	}

	return orders, total, nil
}

// GetOrdersWithDetails retrieves orders with additional event details
func (r *OrderRepository) GetOrdersWithDetails(filters OrderSearchFilters) ([]*OrderWithDetails, int, error) {
	// Build WHERE clause
	var conditions []string
	var args []interface{}
	argIndex := 1

	if filters.UserID > 0 {
		conditions = append(conditions, fmt.Sprintf("o.user_id = $%d", argIndex))
		args = append(args, filters.UserID)
		argIndex++
	}

	if filters.EventID > 0 {
		conditions = append(conditions, fmt.Sprintf("o.event_id = $%d", argIndex))
		args = append(args, filters.EventID)
		argIndex++
	}

	if filters.Status != "" {
		conditions = append(conditions, fmt.Sprintf("o.status = $%d", argIndex))
		args = append(args, filters.Status)
		argIndex++
	}

	if filters.DateFrom != nil {
		conditions = append(conditions, fmt.Sprintf("o.created_at >= $%d", argIndex))
		args = append(args, *filters.DateFrom)
		argIndex++
	}

	if filters.DateTo != nil {
		conditions = append(conditions, fmt.Sprintf("o.created_at <= $%d", argIndex))
		args = append(args, *filters.DateTo)
		argIndex++
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	// Build ORDER BY clause
	orderBy := "ORDER BY o.created_at DESC"
	if filters.SortBy != "" {
		direction := "ASC"
		if filters.SortDesc {
			direction = "DESC"
		}

		switch filters.SortBy {
		case "created_at", "total_amount", "status":
			orderBy = fmt.Sprintf("ORDER BY o.%s %s", filters.SortBy, direction)
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
	countQuery := fmt.Sprintf(`
		SELECT COUNT(*)
		FROM orders o
		JOIN events e ON o.event_id = e.id
		%s`, whereClause)
	
	var total int
	err := r.db.QueryRow(countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get order count: %w", err)
	}

	// Get orders with details
	query := fmt.Sprintf(`
		SELECT 
			o.id, o.user_id, o.event_id, o.order_number, o.total_amount, o.status, 
			o.payment_id, o.billing_email, o.billing_name, o.created_at, o.updated_at,
			e.title as event_title, e.start_date as event_date,
			COUNT(t.id) as ticket_count
		FROM orders o
		JOIN events e ON o.event_id = e.id
		LEFT JOIN tickets t ON o.id = t.order_id
		%s
		GROUP BY o.id, e.title, e.start_date
		%s
		LIMIT $%d OFFSET $%d`,
		whereClause, orderBy, argIndex, argIndex+1)

	args = append(args, filters.Limit, filters.Offset)

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to search orders with details: %w", err)
	}
	defer rows.Close()

	var orders []*OrderWithDetails
	for rows.Next() {
		orderDetail := &OrderWithDetails{
			Order: &models.Order{},
		}
		err := rows.Scan(
			&orderDetail.Order.ID,
			&orderDetail.Order.UserID,
			&orderDetail.Order.EventID,
			&orderDetail.Order.OrderNumber,
			&orderDetail.Order.TotalAmount,
			&orderDetail.Order.Status,
			&orderDetail.Order.PaymentID,
			&orderDetail.Order.BillingEmail,
			&orderDetail.Order.BillingName,
			&orderDetail.Order.CreatedAt,
			&orderDetail.Order.UpdatedAt,
			&orderDetail.EventTitle,
			&orderDetail.EventDate,
			&orderDetail.TicketCount,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan order with details: %w", err)
		}
		orders = append(orders, orderDetail)
	}

	if err = rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating orders with details: %w", err)
	}

	return orders, total, nil
}

// GetExpiredOrders retrieves orders that have expired (pending orders older than specified duration)
func (r *OrderRepository) GetExpiredOrders(expirationDuration time.Duration) ([]*models.Order, error) {
	expirationTime := time.Now().Add(-expirationDuration)
	
	query := `
		SELECT id, user_id, event_id, order_number, total_amount, status, payment_id, billing_email, billing_name, created_at, updated_at
		FROM orders
		WHERE status = $1 AND created_at < $2
		ORDER BY created_at ASC`

	rows, err := r.db.Query(query, models.OrderPending, expirationTime)
	if err != nil {
		return nil, fmt.Errorf("failed to get expired orders: %w", err)
	}
	defer rows.Close()

	var orders []*models.Order
	for rows.Next() {
		order := &models.Order{}
		err := rows.Scan(
			&order.ID,
			&order.UserID,
			&order.EventID,
			&order.OrderNumber,
			&order.TotalAmount,
			&order.Status,
			&order.PaymentID,
			&order.BillingEmail,
			&order.BillingName,
			&order.CreatedAt,
			&order.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan expired order: %w", err)
		}
		orders = append(orders, order)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating expired orders: %w", err)
	}

	return orders, nil
}

// ProcessOrderCompletion completes an order and creates tickets with transaction handling
func (r *OrderRepository) ProcessOrderCompletion(orderID int, paymentID string, ticketData []struct {
	TicketTypeID int
	QRCode       string
}) error {
	// Start transaction
	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Update order status to completed
	_, err = tx.Exec(`
		UPDATE orders 
		SET status = $2, payment_id = $3, updated_at = $4 
		WHERE id = $1 AND status = $5`,
		orderID, models.OrderCompleted, paymentID, time.Now(), models.OrderPending)

	if err != nil {
		return fmt.Errorf("failed to update order status: %w", err)
	}

	// Create tickets
	for _, ticket := range ticketData {
		_, err = tx.Exec(`
			INSERT INTO tickets (order_id, ticket_type_id, qr_code, status, created_at)
			VALUES ($1, $2, $3, $4, $5)`,
			orderID, ticket.TicketTypeID, ticket.QRCode, models.TicketActive, time.Now())

		if err != nil {
			return fmt.Errorf("failed to create ticket: %w", err)
		}
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit order completion: %w", err)
	}

	return nil
}

// GetOrderStatistics retrieves order statistics for an event or user
func (r *OrderRepository) GetOrderStatistics(eventID *int, userID *int) (map[string]interface{}, error) {
	var conditions []string
	var args []interface{}
	argIndex := 1

	if eventID != nil {
		conditions = append(conditions, fmt.Sprintf("event_id = $%d", argIndex))
		args = append(args, *eventID)
		argIndex++
	}

	if userID != nil {
		conditions = append(conditions, fmt.Sprintf("user_id = $%d", argIndex))
		args = append(args, *userID)
		argIndex++
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	query := fmt.Sprintf(`
		SELECT 
			COUNT(*) as total_orders,
			COUNT(CASE WHEN status = 'completed' THEN 1 END) as completed_orders,
			COUNT(CASE WHEN status = 'pending' THEN 1 END) as pending_orders,
			COUNT(CASE WHEN status = 'cancelled' THEN 1 END) as cancelled_orders,
			COUNT(CASE WHEN status = 'refunded' THEN 1 END) as refunded_orders,
			COALESCE(SUM(CASE WHEN status = 'completed' THEN total_amount END), 0) as total_revenue
		FROM orders
		%s`, whereClause)

	var stats struct {
		TotalOrders     int `db:"total_orders"`
		CompletedOrders int `db:"completed_orders"`
		PendingOrders   int `db:"pending_orders"`
		CancelledOrders int `db:"cancelled_orders"`
		RefundedOrders  int `db:"refunded_orders"`
		TotalRevenue    int `db:"total_revenue"`
	}

	err := r.db.QueryRow(query, args...).Scan(
		&stats.TotalOrders,
		&stats.CompletedOrders,
		&stats.PendingOrders,
		&stats.CancelledOrders,
		&stats.RefundedOrders,
		&stats.TotalRevenue,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get order statistics: %w", err)
	}

	result := map[string]interface{}{
		"total_orders":     stats.TotalOrders,
		"completed_orders": stats.CompletedOrders,
		"pending_orders":   stats.PendingOrders,
		"cancelled_orders": stats.CancelledOrders,
		"refunded_orders":  stats.RefundedOrders,
		"total_revenue":    stats.TotalRevenue,
		"revenue_dollars":  float64(stats.TotalRevenue) / 100.0,
	}

	return result, nil
}

// Admin-specific methods

// GetOrderCount returns the total number of orders
func (r *OrderRepository) GetOrderCount() (int, error) {
	var count int
	err := r.db.QueryRow("SELECT COUNT(*) FROM orders").Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get order count: %w", err)
	}
	return count, nil
}

// GetTotalRevenue returns the total revenue from all completed orders
func (r *OrderRepository) GetTotalRevenue() (float64, error) {
	var revenue float64
	err := r.db.QueryRow("SELECT COALESCE(SUM(total_amount), 0) FROM orders WHERE status = 'completed'").Scan(&revenue)
	if err != nil {
		return 0, fmt.Errorf("failed to get total revenue: %w", err)
	}
	return revenue, nil
}