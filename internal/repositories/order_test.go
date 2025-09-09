package repositories

import (
	"database/sql"
	"testing"
	"time"

	"event-ticketing-platform/internal/models"
	_ "github.com/lib/pq"
)

func setupOrderTestDB(t *testing.T) *sql.DB {
	// This would typically use a test database
	// For now, we'll skip actual database tests and focus on the structure
	t.Skip("Database tests require test database setup")
	return nil
}

func TestOrderRepository_Create(t *testing.T) {
	db := setupOrderTestDB(t)
	if db == nil {
		return
	}
	defer db.Close()

	repo := NewOrderRepository(db)

	tests := []struct {
		name    string
		req     *models.OrderCreateRequest
		wantErr bool
	}{
		{
			name: "valid order",
			req: &models.OrderCreateRequest{
				UserID:       1,
				EventID:      1,
				TotalAmount:  5000, // $50.00
				BillingEmail: "test@example.com",
				BillingName:  "John Doe",
				Status:       models.OrderPending,
			},
			wantErr: false,
		},
		{
			name: "invalid total amount",
			req: &models.OrderCreateRequest{
				UserID:       1,
				EventID:      1,
				TotalAmount:  -100, // Invalid negative amount
				BillingEmail: "test@example.com",
				BillingName:  "John Doe",
				Status:       models.OrderPending,
			},
			wantErr: true,
		},
		{
			name: "invalid email",
			req: &models.OrderCreateRequest{
				UserID:       1,
				EventID:      1,
				TotalAmount:  5000,
				BillingEmail: "invalid-email", // Invalid email format
				BillingName:  "John Doe",
				Status:       models.OrderPending,
			},
			wantErr: true,
		},
		{
			name: "empty billing name",
			req: &models.OrderCreateRequest{
				UserID:       1,
				EventID:      1,
				TotalAmount:  5000,
				BillingEmail: "test@example.com",
				BillingName:  "", // Empty name
				Status:       models.OrderPending,
			},
			wantErr: true,
		},
		{
			name: "invalid status",
			req: &models.OrderCreateRequest{
				UserID:       1,
				EventID:      1,
				TotalAmount:  5000,
				BillingEmail: "test@example.com",
				BillingName:  "John Doe",
				Status:       "invalid_status", // Invalid status
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			order, err := repo.Create(tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("Create() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if order == nil {
					t.Error("Create() returned nil order")
				}
				if order.OrderNumber == "" {
					t.Error("Create() did not generate order number")
				}
			}
		})
	}
}

func TestOrderRepository_GetByID(t *testing.T) {
	db := setupOrderTestDB(t)
	if db == nil {
		return
	}
	defer db.Close()

	repo := NewOrderRepository(db)

	tests := []struct {
		name    string
		orderID int
		wantErr bool
	}{
		{
			name:    "valid order ID",
			orderID: 1,
			wantErr: false,
		},
		{
			name:    "non-existent order",
			orderID: 999,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			order, err := repo.GetByID(tt.orderID)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetByID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && order == nil {
				t.Error("GetByID() returned nil order")
			}
		})
	}
}

func TestOrderRepository_GetByOrderNumber(t *testing.T) {
	db := setupOrderTestDB(t)
	if db == nil {
		return
	}
	defer db.Close()

	repo := NewOrderRepository(db)

	tests := []struct {
		name        string
		orderNumber string
		wantErr     bool
	}{
		{
			name:        "valid order number",
			orderNumber: "ORD-20240101-123456",
			wantErr:     false,
		},
		{
			name:        "non-existent order number",
			orderNumber: "ORD-20240101-999999",
			wantErr:     true,
		},
		{
			name:        "invalid order number format",
			orderNumber: "INVALID",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			order, err := repo.GetByOrderNumber(tt.orderNumber)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetByOrderNumber() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && order == nil {
				t.Error("GetByOrderNumber() returned nil order")
			}
		})
	}
}

func TestOrderRepository_Update(t *testing.T) {
	db := setupOrderTestDB(t)
	if db == nil {
		return
	}
	defer db.Close()

	repo := NewOrderRepository(db)

	tests := []struct {
		name    string
		orderID int
		req     *models.OrderUpdateRequest
		wantErr bool
	}{
		{
			name:    "valid update",
			orderID: 1,
			req: &models.OrderUpdateRequest{
				Status:    models.OrderCompleted,
				PaymentID: "pay_123456789",
			},
			wantErr: false,
		},
		{
			name:    "invalid status",
			orderID: 1,
			req: &models.OrderUpdateRequest{
				Status:    "invalid_status",
				PaymentID: "pay_123456789",
			},
			wantErr: true,
		},
		{
			name:    "non-existent order",
			orderID: 999,
			req: &models.OrderUpdateRequest{
				Status:    models.OrderCompleted,
				PaymentID: "pay_123456789",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			order, err := repo.Update(tt.orderID, tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("Update() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && order == nil {
				t.Error("Update() returned nil order")
			}
		})
	}
}

func TestOrderRepository_UpdateStatus(t *testing.T) {
	db := setupOrderTestDB(t)
	if db == nil {
		return
	}
	defer db.Close()

	repo := NewOrderRepository(db)

	tests := []struct {
		name    string
		orderID int
		status  models.OrderStatus
		wantErr bool
	}{
		{
			name:    "complete pending order",
			orderID: 1, // Assuming this is a pending order
			status:  models.OrderCompleted,
			wantErr: false,
		},
		{
			name:    "cancel pending order",
			orderID: 2, // Assuming this is a pending order
			status:  models.OrderCancelled,
			wantErr: false,
		},
		{
			name:    "refund completed order",
			orderID: 3, // Assuming this is a completed order
			status:  models.OrderRefunded,
			wantErr: false,
		},
		{
			name:    "invalid status transition",
			orderID: 4, // Assuming this is a cancelled order
			status:  models.OrderCompleted,
			wantErr: true,
		},
		{
			name:    "non-existent order",
			orderID: 999,
			status:  models.OrderCompleted,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := repo.UpdateStatus(tt.orderID, tt.status)
			if (err != nil) != tt.wantErr {
				t.Errorf("UpdateStatus() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestOrderRepository_Delete(t *testing.T) {
	db := setupOrderTestDB(t)
	if db == nil {
		return
	}
	defer db.Close()

	repo := NewOrderRepository(db)

	tests := []struct {
		name    string
		orderID int
		wantErr bool
	}{
		{
			name:    "delete pending order with no tickets",
			orderID: 1, // Assuming this is a pending order with no tickets
			wantErr: false,
		},
		{
			name:    "delete completed order",
			orderID: 2, // Assuming this is a completed order
			wantErr: true,
		},
		{
			name:    "delete order with tickets",
			orderID: 3, // Assuming this order has tickets
			wantErr: true,
		},
		{
			name:    "non-existent order",
			orderID: 999,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := repo.Delete(tt.orderID)
			if (err != nil) != tt.wantErr {
				t.Errorf("Delete() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestOrderRepository_Search(t *testing.T) {
	db := setupOrderTestDB(t)
	if db == nil {
		return
	}
	defer db.Close()

	repo := NewOrderRepository(db)

	tests := []struct {
		name    string
		filters OrderSearchFilters
		wantErr bool
	}{
		{
			name: "search by user",
			filters: OrderSearchFilters{
				UserID: 1,
				Limit:  10,
				Offset: 0,
			},
			wantErr: false,
		},
		{
			name: "search by event",
			filters: OrderSearchFilters{
				EventID: 1,
				Limit:   10,
				Offset:  0,
			},
			wantErr: false,
		},
		{
			name: "search by status",
			filters: OrderSearchFilters{
				Status: models.OrderCompleted,
				Limit:  10,
				Offset: 0,
			},
			wantErr: false,
		},
		{
			name: "search with date range",
			filters: OrderSearchFilters{
				DateFrom: &[]time.Time{time.Now().AddDate(0, -1, 0)}[0], // 1 month ago
				DateTo:   &[]time.Time{time.Now()}[0],                   // Now
				Limit:    10,
				Offset:   0,
			},
			wantErr: false,
		},
		{
			name: "search with amount range",
			filters: OrderSearchFilters{
				AmountMin: &[]int{1000}[0], // $10.00
				AmountMax: &[]int{10000}[0], // $100.00
				Limit:     10,
				Offset:    0,
			},
			wantErr: false,
		},
		{
			name: "search with sorting",
			filters: OrderSearchFilters{
				SortBy:   "total_amount",
				SortDesc: true,
				Limit:    10,
				Offset:   0,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			orders, total, err := repo.Search(tt.filters)
			if (err != nil) != tt.wantErr {
				t.Errorf("Search() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if orders == nil {
					t.Error("Search() returned nil orders")
				}
				if total < 0 {
					t.Error("Search() returned negative total")
				}
			}
		})
	}
}

func TestOrderRepository_GetByUser(t *testing.T) {
	db := setupOrderTestDB(t)
	if db == nil {
		return
	}
	defer db.Close()

	repo := NewOrderRepository(db)

	tests := []struct {
		name    string
		userID  int
		limit   int
		offset  int
		wantErr bool
	}{
		{
			name:    "valid user ID",
			userID:  1,
			limit:   10,
			offset:  0,
			wantErr: false,
		},
		{
			name:    "non-existent user",
			userID:  999,
			limit:   10,
			offset:  0,
			wantErr: false, // Should return empty slice, not error
		},
		{
			name:    "with pagination",
			userID:  1,
			limit:   5,
			offset:  10,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			orders, total, err := repo.GetByUser(tt.userID, tt.limit, tt.offset)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetByUser() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if orders == nil {
					t.Error("GetByUser() returned nil orders")
				}
				if total < 0 {
					t.Error("GetByUser() returned negative total")
				}
			}
		})
	}
}

func TestOrderRepository_GetByEvent(t *testing.T) {
	db := setupOrderTestDB(t)
	if db == nil {
		return
	}
	defer db.Close()

	repo := NewOrderRepository(db)

	tests := []struct {
		name    string
		eventID int
		limit   int
		offset  int
		wantErr bool
	}{
		{
			name:    "valid event ID",
			eventID: 1,
			limit:   10,
			offset:  0,
			wantErr: false,
		},
		{
			name:    "non-existent event",
			eventID: 999,
			limit:   10,
			offset:  0,
			wantErr: false, // Should return empty slice, not error
		},
		{
			name:    "with pagination",
			eventID: 1,
			limit:   5,
			offset:  10,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			orders, total, err := repo.GetByEvent(tt.eventID, tt.limit, tt.offset)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetByEvent() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if orders == nil {
					t.Error("GetByEvent() returned nil orders")
				}
				if total < 0 {
					t.Error("GetByEvent() returned negative total")
				}
			}
		})
	}
}

func TestOrderRepository_GetOrdersWithDetails(t *testing.T) {
	db := setupOrderTestDB(t)
	if db == nil {
		return
	}
	defer db.Close()

	repo := NewOrderRepository(db)

	tests := []struct {
		name    string
		filters OrderSearchFilters
		wantErr bool
	}{
		{
			name: "get orders with details",
			filters: OrderSearchFilters{
				UserID: 1,
				Limit:  10,
				Offset: 0,
			},
			wantErr: false,
		},
		{
			name: "get orders with details by event",
			filters: OrderSearchFilters{
				EventID: 1,
				Limit:   10,
				Offset:  0,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			orders, total, err := repo.GetOrdersWithDetails(tt.filters)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetOrdersWithDetails() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if orders == nil {
					t.Error("GetOrdersWithDetails() returned nil orders")
				}
				if total < 0 {
					t.Error("GetOrdersWithDetails() returned negative total")
				}
			}
		})
	}
}

func TestOrderRepository_GetExpiredOrders(t *testing.T) {
	db := setupOrderTestDB(t)
	if db == nil {
		return
	}
	defer db.Close()

	repo := NewOrderRepository(db)

	tests := []struct {
		name               string
		expirationDuration time.Duration
		wantErr            bool
	}{
		{
			name:               "get orders expired 15 minutes ago",
			expirationDuration: 15 * time.Minute,
			wantErr:            false,
		},
		{
			name:               "get orders expired 1 hour ago",
			expirationDuration: time.Hour,
			wantErr:            false,
		},
		{
			name:               "get orders expired 1 day ago",
			expirationDuration: 24 * time.Hour,
			wantErr:            false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			orders, err := repo.GetExpiredOrders(tt.expirationDuration)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetExpiredOrders() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && orders == nil {
				t.Error("GetExpiredOrders() returned nil orders")
			}
		})
	}
}

func TestOrderRepository_ProcessOrderCompletion(t *testing.T) {
	db := setupOrderTestDB(t)
	if db == nil {
		return
	}
	defer db.Close()

	repo := NewOrderRepository(db)

	tests := []struct {
		name       string
		orderID    int
		paymentID  string
		ticketData []struct {
			TicketTypeID int
			QRCode       string
		}
		wantErr bool
	}{
		{
			name:      "valid order completion",
			orderID:   1,
			paymentID: "pay_123456789",
			ticketData: []struct {
				TicketTypeID int
				QRCode       string
			}{
				{TicketTypeID: 1, QRCode: "QR123456789"},
				{TicketTypeID: 1, QRCode: "QR987654321"},
			},
			wantErr: false,
		},
		{
			name:      "non-existent order",
			orderID:   999,
			paymentID: "pay_123456789",
			ticketData: []struct {
				TicketTypeID int
				QRCode       string
			}{
				{TicketTypeID: 1, QRCode: "QR123456789"},
			},
			wantErr: true,
		},
		{
			name:       "empty ticket data",
			orderID:    1,
			paymentID:  "pay_123456789",
			ticketData: []struct {
				TicketTypeID int
				QRCode       string
			}{},
			wantErr: false, // Should still complete the order
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := repo.ProcessOrderCompletion(tt.orderID, tt.paymentID, tt.ticketData)
			if (err != nil) != tt.wantErr {
				t.Errorf("ProcessOrderCompletion() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestOrderRepository_GetOrderStatistics(t *testing.T) {
	db := setupOrderTestDB(t)
	if db == nil {
		return
	}
	defer db.Close()

	repo := NewOrderRepository(db)

	tests := []struct {
		name    string
		eventID *int
		userID  *int
		wantErr bool
	}{
		{
			name:    "statistics for specific event",
			eventID: &[]int{1}[0],
			userID:  nil,
			wantErr: false,
		},
		{
			name:    "statistics for specific user",
			eventID: nil,
			userID:  &[]int{1}[0],
			wantErr: false,
		},
		{
			name:    "overall statistics",
			eventID: nil,
			userID:  nil,
			wantErr: false,
		},
		{
			name:    "statistics for both event and user",
			eventID: &[]int{1}[0],
			userID:  &[]int{1}[0],
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stats, err := repo.GetOrderStatistics(tt.eventID, tt.userID)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetOrderStatistics() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if stats == nil {
					t.Error("GetOrderStatistics() returned nil statistics")
				}
				// Check that expected keys exist
				expectedKeys := []string{
					"total_orders", "completed_orders", "pending_orders",
					"cancelled_orders", "refunded_orders", "total_revenue", "revenue_dollars",
				}
				for _, key := range expectedKeys {
					if _, exists := stats[key]; !exists {
						t.Errorf("GetOrderStatistics() missing key: %s", key)
					}
				}
			}
		})
	}
}

// Test helper functions
func TestGenerateOrderNumber(t *testing.T) {
	orderNumber := models.GenerateOrderNumber()
	
	if orderNumber == "" {
		t.Error("GenerateOrderNumber() returned empty string")
	}
	
	// Check format: ORD-YYYYMMDD-XXXXXX
	if len(orderNumber) != 19 {
		t.Errorf("GenerateOrderNumber() returned wrong length: got %d, want 19", len(orderNumber))
	}
	
	if orderNumber[:4] != "ORD-" {
		t.Errorf("GenerateOrderNumber() wrong prefix: got %s, want ORD-", orderNumber[:4])
	}
	
	// Generate multiple order numbers to check uniqueness
	numbers := make(map[string]bool)
	for i := 0; i < 100; i++ {
		num := models.GenerateOrderNumber()
		if numbers[num] {
			t.Errorf("GenerateOrderNumber() generated duplicate: %s", num)
		}
		numbers[num] = true
	}
}