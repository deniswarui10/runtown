package repositories

import (
	"database/sql"
	"testing"
	"time"

	"event-ticketing-platform/internal/models"
	_ "github.com/lib/pq"
)

func setupTicketTestDB(t *testing.T) *sql.DB {
	// This would typically use a test database
	// For now, we'll skip actual database tests and focus on the structure
	t.Skip("Database tests require test database setup")
	return nil
}

func TestTicketRepository_CreateTicketType(t *testing.T) {
	db := setupTicketTestDB(t)
	if db == nil {
		return
	}
	defer db.Close()

	repo := NewTicketRepository(db)

	tests := []struct {
		name    string
		req     *models.TicketTypeCreateRequest
		wantErr bool
	}{
		{
			name: "valid ticket type",
			req: &models.TicketTypeCreateRequest{
				EventID:     1,
				Name:        "General Admission",
				Description: "Standard entry ticket",
				Price:       2500, // $25.00
				Quantity:    100,
				SaleStart:   time.Now().Add(time.Hour),
				SaleEnd:     time.Now().Add(24 * time.Hour),
			},
			wantErr: false,
		},
		{
			name: "invalid price",
			req: &models.TicketTypeCreateRequest{
				EventID:     1,
				Name:        "General Admission",
				Description: "Standard entry ticket",
				Price:       -100, // Invalid negative price
				Quantity:    100,
				SaleStart:   time.Now().Add(time.Hour),
				SaleEnd:     time.Now().Add(24 * time.Hour),
			},
			wantErr: true,
		},
		{
			name: "invalid quantity",
			req: &models.TicketTypeCreateRequest{
				EventID:     1,
				Name:        "General Admission",
				Description: "Standard entry ticket",
				Price:       2500,
				Quantity:    0, // Invalid zero quantity
				SaleStart:   time.Now().Add(time.Hour),
				SaleEnd:     time.Now().Add(24 * time.Hour),
			},
			wantErr: true,
		},
		{
			name: "invalid sale period",
			req: &models.TicketTypeCreateRequest{
				EventID:     1,
				Name:        "General Admission",
				Description: "Standard entry ticket",
				Price:       2500,
				Quantity:    100,
				SaleStart:   time.Now().Add(24 * time.Hour),
				SaleEnd:     time.Now().Add(time.Hour), // End before start
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ticketType, err := repo.CreateTicketType(tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateTicketType() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && ticketType == nil {
				t.Error("CreateTicketType() returned nil ticket type")
			}
		})
	}
}

func TestTicketRepository_ReserveTickets(t *testing.T) {
	db := setupTicketTestDB(t)
	if db == nil {
		return
	}
	defer db.Close()

	repo := NewTicketRepository(db)

	tests := []struct {
		name              string
		ticketTypeID      int
		quantity          int
		userID            int
		expirationMinutes int
		wantErr           bool
	}{
		{
			name:              "valid reservation",
			ticketTypeID:      1,
			quantity:          2,
			userID:            1,
			expirationMinutes: 15,
			wantErr:           false,
		},
		{
			name:              "insufficient tickets",
			ticketTypeID:      1,
			quantity:          1000, // More than available
			userID:            1,
			expirationMinutes: 15,
			wantErr:           true,
		},
		{
			name:              "invalid ticket type",
			ticketTypeID:      999, // Non-existent
			quantity:          2,
			userID:            1,
			expirationMinutes: 15,
			wantErr:           true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reservation, err := repo.ReserveTickets(tt.ticketTypeID, tt.quantity, tt.userID, tt.expirationMinutes)
			if (err != nil) != tt.wantErr {
				t.Errorf("ReserveTickets() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && reservation == nil {
				t.Error("ReserveTickets() returned nil reservation")
			}
		})
	}
}

func TestTicketRepository_CreateTicket(t *testing.T) {
	db := setupTicketTestDB(t)
	if db == nil {
		return
	}
	defer db.Close()

	repo := NewTicketRepository(db)

	tests := []struct {
		name         string
		orderID      int
		ticketTypeID int
		qrCode       string
		wantErr      bool
	}{
		{
			name:         "valid ticket",
			orderID:      1,
			ticketTypeID: 1,
			qrCode:       "QR123456789",
			wantErr:      false,
		},
		{
			name:         "duplicate QR code",
			orderID:      1,
			ticketTypeID: 1,
			qrCode:       "QR123456789", // Same as above
			wantErr:      true,
		},
		{
			name:         "empty QR code",
			orderID:      1,
			ticketTypeID: 1,
			qrCode:       "",
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ticket, err := repo.CreateTicket(tt.orderID, tt.ticketTypeID, tt.qrCode)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateTicket() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && ticket == nil {
				t.Error("CreateTicket() returned nil ticket")
			}
		})
	}
}

func TestTicketRepository_UpdateTicketStatus(t *testing.T) {
	db := setupTicketTestDB(t)
	if db == nil {
		return
	}
	defer db.Close()

	repo := NewTicketRepository(db)

	tests := []struct {
		name     string
		ticketID int
		status   models.TicketStatus
		wantErr  bool
	}{
		{
			name:     "mark ticket as used",
			ticketID: 1,
			status:   models.TicketUsed,
			wantErr:  false,
		},
		{
			name:     "refund ticket",
			ticketID: 2,
			status:   models.TicketRefunded,
			wantErr:  false,
		},
		{
			name:     "invalid ticket ID",
			ticketID: 999,
			status:   models.TicketUsed,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := repo.UpdateTicketStatus(tt.ticketID, tt.status)
			if (err != nil) != tt.wantErr {
				t.Errorf("UpdateTicketStatus() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestTicketRepository_SearchTickets(t *testing.T) {
	db := setupTicketTestDB(t)
	if db == nil {
		return
	}
	defer db.Close()

	repo := NewTicketRepository(db)

	tests := []struct {
		name    string
		filters TicketSearchFilters
		wantErr bool
	}{
		{
			name: "search by order",
			filters: TicketSearchFilters{
				OrderID: 1,
				Limit:   10,
				Offset:  0,
			},
			wantErr: false,
		},
		{
			name: "search by status",
			filters: TicketSearchFilters{
				Status: models.TicketActive,
				Limit:  10,
				Offset: 0,
			},
			wantErr: false,
		},
		{
			name: "search by event",
			filters: TicketSearchFilters{
				EventID: 1,
				Limit:   10,
				Offset:  0,
			},
			wantErr: false,
		},
		{
			name: "search with sorting",
			filters: TicketSearchFilters{
				SortBy:   "created_at",
				SortDesc: true,
				Limit:    10,
				Offset:   0,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tickets, total, err := repo.SearchTickets(tt.filters)
			if (err != nil) != tt.wantErr {
				t.Errorf("SearchTickets() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if tickets == nil {
					t.Error("SearchTickets() returned nil tickets")
				}
				if total < 0 {
					t.Error("SearchTickets() returned negative total")
				}
			}
		})
	}
}

func TestTicketRepository_GetTicketTypesByEvent(t *testing.T) {
	db := setupTicketTestDB(t)
	if db == nil {
		return
	}
	defer db.Close()

	repo := NewTicketRepository(db)

	tests := []struct {
		name    string
		eventID int
		wantErr bool
	}{
		{
			name:    "valid event ID",
			eventID: 1,
			wantErr: false,
		},
		{
			name:    "non-existent event",
			eventID: 999,
			wantErr: false, // Should return empty slice, not error
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ticketTypes, err := repo.GetTicketTypesByEvent(tt.eventID)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetTicketTypesByEvent() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && ticketTypes == nil {
				t.Error("GetTicketTypesByEvent() returned nil ticket types")
			}
		})
	}
}

func TestTicketRepository_UpdateTicketType(t *testing.T) {
	db := setupTicketTestDB(t)
	if db == nil {
		return
	}
	defer db.Close()

	repo := NewTicketRepository(db)

	tests := []struct {
		name           string
		ticketTypeID   int
		req            *models.TicketTypeUpdateRequest
		wantErr        bool
	}{
		{
			name:         "valid update",
			ticketTypeID: 1,
			req: &models.TicketTypeUpdateRequest{
				Name:        "Updated General Admission",
				Description: "Updated description",
				Price:       3000, // $30.00
				Quantity:    150,
				SaleStart:   time.Now().Add(time.Hour),
				SaleEnd:     time.Now().Add(48 * time.Hour),
			},
			wantErr: false,
		},
		{
			name:         "invalid quantity reduction",
			ticketTypeID: 1,
			req: &models.TicketTypeUpdateRequest{
				Name:        "Updated General Admission",
				Description: "Updated description",
				Price:       3000,
				Quantity:    5, // Less than sold tickets
				SaleStart:   time.Now().Add(time.Hour),
				SaleEnd:     time.Now().Add(48 * time.Hour),
			},
			wantErr: true,
		},
		{
			name:         "non-existent ticket type",
			ticketTypeID: 999,
			req: &models.TicketTypeUpdateRequest{
				Name:        "Updated General Admission",
				Description: "Updated description",
				Price:       3000,
				Quantity:    150,
				SaleStart:   time.Now().Add(time.Hour),
				SaleEnd:     time.Now().Add(48 * time.Hour),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ticketType, err := repo.UpdateTicketType(tt.ticketTypeID, tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("UpdateTicketType() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && ticketType == nil {
				t.Error("UpdateTicketType() returned nil ticket type")
			}
		})
	}
}

func TestTicketRepository_DeleteTicketType(t *testing.T) {
	db := setupTicketTestDB(t)
	if db == nil {
		return
	}
	defer db.Close()

	repo := NewTicketRepository(db)

	tests := []struct {
		name           string
		ticketTypeID   int
		wantErr        bool
	}{
		{
			name:         "delete ticket type with no sales",
			ticketTypeID: 2, // Assuming this has no sold tickets
			wantErr:      false,
		},
		{
			name:         "delete ticket type with sales",
			ticketTypeID: 1, // Assuming this has sold tickets
			wantErr:      true,
		},
		{
			name:         "non-existent ticket type",
			ticketTypeID: 999,
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := repo.DeleteTicketType(tt.ticketTypeID)
			if (err != nil) != tt.wantErr {
				t.Errorf("DeleteTicketType() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestTicketRepository_GetTicketsByOrder(t *testing.T) {
	db := setupTicketTestDB(t)
	if db == nil {
		return
	}
	defer db.Close()

	repo := NewTicketRepository(db)

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
			wantErr: false, // Should return empty slice, not error
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tickets, err := repo.GetTicketsByOrder(tt.orderID)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetTicketsByOrder() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && tickets == nil {
				t.Error("GetTicketsByOrder() returned nil tickets")
			}
		})
	}
}

func TestTicketRepository_GetTicketByQRCode(t *testing.T) {
	db := setupTicketTestDB(t)
	if db == nil {
		return
	}
	defer db.Close()

	repo := NewTicketRepository(db)

	tests := []struct {
		name    string
		qrCode  string
		wantErr bool
	}{
		{
			name:    "valid QR code",
			qrCode:  "QR123456789",
			wantErr: false,
		},
		{
			name:    "non-existent QR code",
			qrCode:  "INVALID_QR",
			wantErr: true,
		},
		{
			name:    "empty QR code",
			qrCode:  "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ticket, err := repo.GetTicketByQRCode(tt.qrCode)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetTicketByQRCode() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && ticket == nil {
				t.Error("GetTicketByQRCode() returned nil ticket")
			}
		})
	}
}

func TestTicketRepository_ReleaseReservation(t *testing.T) {
	db := setupTicketTestDB(t)
	if db == nil {
		return
	}
	defer db.Close()

	repo := NewTicketRepository(db)

	tests := []struct {
		name           string
		reservationID  string
		ticketTypeID   int
		quantity       int
		wantErr        bool
	}{
		{
			name:          "valid reservation release",
			reservationID: "RES-1-1-123456",
			ticketTypeID:  1,
			quantity:      2,
			wantErr:       false,
		},
		{
			name:          "invalid reservation",
			reservationID: "INVALID_RES",
			ticketTypeID:  999,
			quantity:      2,
			wantErr:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := repo.ReleaseReservation(tt.reservationID, tt.ticketTypeID, tt.quantity)
			if (err != nil) != tt.wantErr {
				t.Errorf("ReleaseReservation() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}