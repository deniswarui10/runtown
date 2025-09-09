package services

import (
	"errors"
	"testing"
	"time"

	"event-ticketing-platform/internal/models"
	"event-ticketing-platform/internal/repositories"
)

// Mock implementations for testing

type mockTicketRepository struct {
	ticketTypes   map[int]*models.TicketType
	tickets       map[int]*models.Ticket
	reservations  map[string]*repositories.TicketReservation
	nextID        int
	shouldFailOps map[string]bool
}

func newMockTicketRepository() *mockTicketRepository {
	return &mockTicketRepository{
		ticketTypes:   make(map[int]*models.TicketType),
		tickets:       make(map[int]*models.Ticket),
		reservations:  make(map[string]*repositories.TicketReservation),
		nextID:        1,
		shouldFailOps: make(map[string]bool),
	}
}

func (m *mockTicketRepository) CreateTicketType(req *models.TicketTypeCreateRequest) (*models.TicketType, error) {
	if m.shouldFailOps["CreateTicketType"] {
		return nil, errors.New("mock error")
	}
	
	tt := &models.TicketType{
		ID:          m.nextID,
		EventID:     req.EventID,
		Name:        req.Name,
		Description: req.Description,
		Price:       req.Price,
		Quantity:    req.Quantity,
		Sold:        0,
		SaleStart:   req.SaleStart,
		SaleEnd:     req.SaleEnd,
		CreatedAt:   time.Now(),
	}
	
	m.ticketTypes[m.nextID] = tt
	m.nextID++
	return tt, nil
}

func (m *mockTicketRepository) GetTicketTypeByID(id int) (*models.TicketType, error) {
	if m.shouldFailOps["GetTicketTypeByID"] {
		return nil, errors.New("mock error")
	}
	
	tt, exists := m.ticketTypes[id]
	if !exists {
		return nil, errors.New("ticket type not found")
	}
	return tt, nil
}

func (m *mockTicketRepository) GetTicketTypesByEvent(eventID int) ([]*models.TicketType, error) {
	if m.shouldFailOps["GetTicketTypesByEvent"] {
		return nil, errors.New("mock error")
	}
	
	var result []*models.TicketType
	for _, tt := range m.ticketTypes {
		if tt.EventID == eventID {
			result = append(result, tt)
		}
	}
	return result, nil
}

func (m *mockTicketRepository) UpdateTicketType(id int, req *models.TicketTypeUpdateRequest) (*models.TicketType, error) {
	if m.shouldFailOps["UpdateTicketType"] {
		return nil, errors.New("mock error")
	}
	
	tt, exists := m.ticketTypes[id]
	if !exists {
		return nil, errors.New("ticket type not found")
	}
	
	tt.Name = req.Name
	tt.Description = req.Description
	tt.Price = req.Price
	tt.Quantity = req.Quantity
	tt.SaleStart = req.SaleStart
	tt.SaleEnd = req.SaleEnd
	
	return tt, nil
}

func (m *mockTicketRepository) DeleteTicketType(id int) error {
	if m.shouldFailOps["DeleteTicketType"] {
		return errors.New("mock error")
	}
	
	tt, exists := m.ticketTypes[id]
	if !exists {
		return errors.New("ticket type not found")
	}
	
	if tt.Sold > 0 {
		return errors.New("cannot delete ticket type with sold tickets")
	}
	
	delete(m.ticketTypes, id)
	return nil
}

func (m *mockTicketRepository) ReserveTickets(ticketTypeID, quantity, userID int, expirationMinutes int) (*repositories.TicketReservation, error) {
	if m.shouldFailOps["ReserveTickets"] {
		return nil, errors.New("mock error")
	}
	
	tt, exists := m.ticketTypes[ticketTypeID]
	if !exists {
		return nil, errors.New("ticket type not found")
	}
	
	if tt.Available() < quantity {
		return nil, errors.New("insufficient tickets available")
	}
	
	// Simulate reservation by increasing sold count temporarily
	tt.Sold += quantity
	
	reservation := &repositories.TicketReservation{
		ID:           "RES-123",
		TicketTypeID: ticketTypeID,
		Quantity:     quantity,
		UserID:       userID,
		ExpiresAt:    time.Now().Add(time.Duration(expirationMinutes) * time.Minute),
		CreatedAt:    time.Now(),
	}
	
	m.reservations[reservation.ID] = reservation
	return reservation, nil
}

func (m *mockTicketRepository) ReleaseReservation(reservationID string, ticketTypeID, quantity int) error {
	if m.shouldFailOps["ReleaseReservation"] {
		return errors.New("mock error")
	}
	
	_, exists := m.reservations[reservationID]
	if !exists {
		return errors.New("reservation not found")
	}
	
	tt, exists := m.ticketTypes[ticketTypeID]
	if exists {
		tt.Sold -= quantity
		if tt.Sold < 0 {
			tt.Sold = 0
		}
	}
	
	delete(m.reservations, reservationID)
	return nil
}

func (m *mockTicketRepository) CreateTicket(orderID, ticketTypeID int, qrCode string) (*models.Ticket, error) {
	if m.shouldFailOps["CreateTicket"] {
		return nil, errors.New("mock error")
	}
	
	ticket := &models.Ticket{
		ID:           m.nextID,
		OrderID:      orderID,
		TicketTypeID: ticketTypeID,
		QRCode:       qrCode,
		Status:       models.TicketActive,
		CreatedAt:    time.Now(),
	}
	
	m.tickets[m.nextID] = ticket
	m.nextID++
	return ticket, nil
}

func (m *mockTicketRepository) GetTicketByID(id int) (*models.Ticket, error) {
	if m.shouldFailOps["GetTicketByID"] {
		return nil, errors.New("mock error")
	}
	
	ticket, exists := m.tickets[id]
	if !exists {
		return nil, errors.New("ticket not found")
	}
	return ticket, nil
}

func (m *mockTicketRepository) GetTicketByQRCode(qrCode string) (*models.Ticket, error) {
	if m.shouldFailOps["GetTicketByQRCode"] {
		return nil, errors.New("mock error")
	}
	
	for _, ticket := range m.tickets {
		if ticket.QRCode == qrCode {
			return ticket, nil
		}
	}
	return nil, errors.New("ticket not found")
}

func (m *mockTicketRepository) GetTicketsByOrder(orderID int) ([]*models.Ticket, error) {
	if m.shouldFailOps["GetTicketsByOrder"] {
		return nil, errors.New("mock error")
	}
	
	var result []*models.Ticket
	for _, ticket := range m.tickets {
		if ticket.OrderID == orderID {
			result = append(result, ticket)
		}
	}
	return result, nil
}

func (m *mockTicketRepository) UpdateTicketStatus(id int, status models.TicketStatus) error {
	if m.shouldFailOps["UpdateTicketStatus"] {
		return errors.New("mock error")
	}
	
	ticket, exists := m.tickets[id]
	if !exists {
		return errors.New("ticket not found")
	}
	
	ticket.Status = status
	return nil
}

func (m *mockTicketRepository) SearchTickets(filters repositories.TicketSearchFilters) ([]*models.Ticket, int, error) {
	if m.shouldFailOps["SearchTickets"] {
		return nil, 0, errors.New("mock error")
	}
	
	var result []*models.Ticket
	for _, ticket := range m.tickets {
		result = append(result, ticket)
	}
	return result, len(result), nil
}

type mockOrderRepository struct {
	orders        map[int]*models.Order
	nextID        int
	shouldFailOps map[string]bool
}

func newMockOrderRepository() *mockOrderRepository {
	return &mockOrderRepository{
		orders:        make(map[int]*models.Order),
		nextID:        1,
		shouldFailOps: make(map[string]bool),
	}
}

func (m *mockOrderRepository) Create(req *models.OrderCreateRequest) (*models.Order, error) {
	if m.shouldFailOps["Create"] {
		return nil, errors.New("mock error")
	}
	
	order := &models.Order{
		ID:           m.nextID,
		UserID:       req.UserID,
		EventID:      req.EventID,
		OrderNumber:  models.GenerateOrderNumber(),
		TotalAmount:  req.TotalAmount,
		Status:       req.Status,
		BillingEmail: req.BillingEmail,
		BillingName:  req.BillingName,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	
	m.orders[m.nextID] = order
	m.nextID++
	return order, nil
}

func (m *mockOrderRepository) GetByID(id int) (*models.Order, error) {
	if m.shouldFailOps["GetByID"] {
		return nil, errors.New("mock error")
	}
	
	order, exists := m.orders[id]
	if !exists {
		return nil, errors.New("order not found")
	}
	return order, nil
}

func (m *mockOrderRepository) GetByOrderNumber(orderNumber string) (*models.Order, error) {
	if m.shouldFailOps["GetByOrderNumber"] {
		return nil, errors.New("mock error")
	}
	
	for _, order := range m.orders {
		if order.OrderNumber == orderNumber {
			return order, nil
		}
	}
	return nil, errors.New("order not found")
}

func (m *mockOrderRepository) Update(id int, req *models.OrderUpdateRequest) (*models.Order, error) {
	if m.shouldFailOps["Update"] {
		return nil, errors.New("mock error")
	}
	
	order, exists := m.orders[id]
	if !exists {
		return nil, errors.New("order not found")
	}
	
	order.Status = req.Status
	order.PaymentID = req.PaymentID
	order.UpdatedAt = time.Now()
	
	return order, nil
}

func (m *mockOrderRepository) UpdateStatus(id int, status models.OrderStatus) error {
	if m.shouldFailOps["UpdateStatus"] {
		return errors.New("mock error")
	}
	
	order, exists := m.orders[id]
	if !exists {
		return errors.New("order not found")
	}
	
	order.Status = status
	order.UpdatedAt = time.Now()
	return nil
}

func (m *mockOrderRepository) ProcessOrderCompletion(orderID int, paymentID string, ticketData []struct {
	TicketTypeID int
	QRCode       string
}) error {
	if m.shouldFailOps["ProcessOrderCompletion"] {
		return errors.New("mock error")
	}
	
	order, exists := m.orders[orderID]
	if !exists {
		return errors.New("order not found")
	}
	
	order.Status = models.OrderCompleted
	order.PaymentID = paymentID
	order.UpdatedAt = time.Now()
	
	return nil
}

func (m *mockOrderRepository) GetByEvent(eventID int, limit, offset int) ([]*models.Order, int, error) {
	if m.shouldFailOps["GetByEvent"] {
		return nil, 0, errors.New("mock error")
	}
	
	var orders []*models.Order
	for _, order := range m.orders {
		if order.EventID == eventID {
			orders = append(orders, order)
		}
	}
	
	// Apply pagination
	total := len(orders)
	if offset >= total {
		return []*models.Order{}, total, nil
	}
	
	end := offset + limit
	if end > total {
		end = total
	}
	
	return orders[offset:end], total, nil
}

func (m *mockOrderRepository) GetByUser(userID int, limit, offset int) ([]*models.Order, int, error) {
	if m.shouldFailOps["GetByUser"] {
		return nil, 0, errors.New("mock error")
	}
	
	var orders []*models.Order
	for _, order := range m.orders {
		if order.UserID == userID {
			orders = append(orders, order)
		}
	}
	
	// Apply pagination
	total := len(orders)
	if offset >= total {
		return []*models.Order{}, total, nil
	}
	
	end := offset + limit
	if end > total {
		end = total
	}
	
	return orders[offset:end], total, nil
}

func (m *mockOrderRepository) Search(filters repositories.OrderSearchFilters) ([]*models.Order, int, error) {
	// Simple mock implementation
	var orders []*models.Order
	for _, order := range m.orders {
		orders = append(orders, order)
	}
	return orders, len(orders), nil
}

func (m *mockOrderRepository) GetOrdersWithDetails(filters repositories.OrderSearchFilters) ([]*repositories.OrderWithDetails, int, error) {
	// Simple mock implementation
	return []*repositories.OrderWithDetails{}, 0, nil
}

func (m *mockOrderRepository) GetOrderStatistics(eventID *int, userID *int) (map[string]interface{}, error) {
	if m.shouldFailOps["GetOrderStatistics"] {
		return nil, errors.New("mock error")
	}
	
	// Simple mock implementation
	stats := map[string]interface{}{
		"total_orders": len(m.orders),
		"total_revenue": 0,
		"completed_orders": 0,
	}
	
	return stats, nil
}

func (m *mockOrderRepository) GetOrderCount() (int, error) {
	return len(m.orders), nil
}

func (m *mockOrderRepository) GetTotalRevenue() (float64, error) {
	return 0.0, nil
}

type mockPaymentService struct {
	shouldFailOps map[string]bool
}

func newMockPaymentService() *mockPaymentService {
	return &mockPaymentService{
		shouldFailOps: make(map[string]bool),
	}
}

func (m *mockPaymentService) ProcessPayment(amount int, paymentMethod string, billingInfo PaymentBillingInfo) (*PaymentResult, error) {
	if m.shouldFailOps["ProcessPayment"] {
		return nil, errors.New("payment processing failed")
	}
	
	return &PaymentResult{
		PaymentID:     "PAY-123",
		Status:        "success",
		Amount:        amount,
		TransactionID: "TXN-123",
		ProcessedAt:   time.Now(),
	}, nil
}

func (m *mockPaymentService) RefundPayment(paymentID string, amount int) (*RefundResult, error) {
	if m.shouldFailOps["RefundPayment"] {
		return nil, errors.New("refund processing failed")
	}
	
	return &RefundResult{
		RefundID:    "REF-123",
		Status:      "success",
		Amount:      amount,
		ProcessedAt: time.Now(),
	}, nil
}

func (m *mockPaymentService) GetPaymentStatus(paymentID string) (*PaymentStatus, error) {
	if m.shouldFailOps["GetPaymentStatus"] {
		return nil, errors.New("mock error")
	}
	
	return &PaymentStatus{
		PaymentID:     paymentID,
		Status:        "success",
		Amount:        1000,
		TransactionID: "TXN-123",
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}, nil
}

// Note: mockUserRepository is defined in event_test.go to avoid duplication

// Test helper functions

func createTestTicketService() (*TicketService, *mockTicketRepository, *mockOrderRepository, *mockPaymentService, *mockUserRepository) {
	ticketRepo := newMockTicketRepository()
	orderRepo := newMockOrderRepository()
	paymentService := newMockPaymentService()
	userRepo := newMockUserRepository()
	
	// Add test user
	userRepo.users[1] = &models.User{
		ID:    1,
		Email: "test@example.com",
		Role:  models.RoleAttendee,
	}
	
	authService := &AuthService{userRepo: userRepo}
	pdfService := NewPDFService() // Add PDF service
	ticketService := NewTicketService(ticketRepo, orderRepo, paymentService, authService, pdfService, 15)
	
	return ticketService, ticketRepo, orderRepo, paymentService, userRepo
}

func createTestTicketType(repo *mockTicketRepository, eventID int) *models.TicketType {
	now := time.Now()
	req := &models.TicketTypeCreateRequest{
		EventID:     eventID,
		Name:        "General Admission",
		Description: "General admission ticket",
		Price:       2500, // $25.00
		Quantity:    100,
		SaleStart:   now.Add(-time.Hour),
		SaleEnd:     now.Add(24 * time.Hour),
	}
	
	ticketType, _ := repo.CreateTicketType(req)
	return ticketType
}

// Tests

func TestTicketService_ReserveTickets(t *testing.T) {
	service, ticketRepo, _, _, _ := createTestTicketService()
	ticketType := createTestTicketType(ticketRepo, 1)
	
	tests := []struct {
		name        string
		req         *TicketReservationRequest
		expectError bool
		errorMsg    string
	}{
		{
			name: "successful reservation",
			req: &TicketReservationRequest{
				TicketTypeID: ticketType.ID,
				Quantity:     2,
				UserID:       1,
			},
			expectError: false,
		},
		{
			name: "user not found",
			req: &TicketReservationRequest{
				TicketTypeID: ticketType.ID,
				Quantity:     2,
				UserID:       999,
			},
			expectError: true,
			errorMsg:    "user not found: user not found",
		},
		{
			name: "ticket type not found",
			req: &TicketReservationRequest{
				TicketTypeID: 999,
				Quantity:     2,
				UserID:       1,
			},
			expectError: true,
			errorMsg:    "ticket type not found: ticket type not found",
		},
		{
			name: "invalid quantity",
			req: &TicketReservationRequest{
				TicketTypeID: ticketType.ID,
				Quantity:     0,
				UserID:       1,
			},
			expectError: true,
			errorMsg:    "quantity must be greater than 0",
		},
		{
			name: "too many tickets",
			req: &TicketReservationRequest{
				TicketTypeID: ticketType.ID,
				Quantity:     15, // More than max allowed (10)
				UserID:       1,
			},
			expectError: true,
			errorMsg:    "cannot reserve more than 10 tickets at once",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reservation, err := service.ReserveTickets(tt.req)
			
			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
					return
				}
				if tt.errorMsg != "" && err.Error() != tt.errorMsg {
					t.Errorf("expected error message '%s', got '%s'", tt.errorMsg, err.Error())
				}
				return
			}
			
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			
			if reservation == nil {
				t.Errorf("expected reservation but got nil")
				return
			}
			
			if reservation.TicketTypeID != tt.req.TicketTypeID {
				t.Errorf("expected ticket type ID %d, got %d", tt.req.TicketTypeID, reservation.TicketTypeID)
			}
			
			if reservation.Quantity != tt.req.Quantity {
				t.Errorf("expected quantity %d, got %d", tt.req.Quantity, reservation.Quantity)
			}
		})
	}
}

func TestTicketService_PurchaseTickets(t *testing.T) {
	service, ticketRepo, _, paymentService, _ := createTestTicketService()
	ticketType := createTestTicketType(ticketRepo, 1)
	
	tests := []struct {
		name        string
		req         *TicketPurchaseRequest
		setupMocks  func()
		expectError bool
		errorMsg    string
	}{
		{
			name: "successful purchase",
			req: &TicketPurchaseRequest{
				EventID: 1,
				TicketSelections: []TicketSelection{
					{
						TicketTypeID: ticketType.ID,
						Quantity:     2,
					},
				},
				BillingInfo: PaymentBillingInfo{
					Email: "test@example.com",
					Name:  "Test User",
				},
				PaymentMethod: "card",
				UserID:        1,
			},
			setupMocks:  func() {},
			expectError: false,
		},
		{
			name: "payment failure",
			req: &TicketPurchaseRequest{
				EventID: 1,
				TicketSelections: []TicketSelection{
					{
						TicketTypeID: ticketType.ID,
						Quantity:     2,
					},
				},
				BillingInfo: PaymentBillingInfo{
					Email: "test@example.com",
					Name:  "Test User",
				},
				PaymentMethod: "card",
				UserID:        1,
			},
			setupMocks: func() {
				paymentService.shouldFailOps["ProcessPayment"] = true
			},
			expectError: true,
			errorMsg:    "payment processing failed: payment processing failed",
		},
		{
			name: "no ticket selections",
			req: &TicketPurchaseRequest{
				EventID:          1,
				TicketSelections: []TicketSelection{},
				BillingInfo: PaymentBillingInfo{
					Email: "test@example.com",
					Name:  "Test User",
				},
				PaymentMethod: "card",
				UserID:        1,
			},
			setupMocks:  func() {},
			expectError: true,
			errorMsg:    "invalid ticket selection: no tickets selected",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset mocks
			paymentService.shouldFailOps = make(map[string]bool)
			tt.setupMocks()
			
			result, err := service.PurchaseTickets(tt.req)
			
			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
					return
				}
				if tt.errorMsg != "" && err.Error() != tt.errorMsg {
					t.Errorf("expected error message '%s', got '%s'", tt.errorMsg, err.Error())
				}
				return
			}
			
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			
			if result == nil {
				t.Errorf("expected purchase result but got nil")
				return
			}
			
			if result.Order == nil {
				t.Errorf("expected order in result but got nil")
			}
			
			if result.PaymentInfo == nil {
				t.Errorf("expected payment info in result but got nil")
			}
		})
	}
}

func TestTicketService_ValidateTicket(t *testing.T) {
	service, ticketRepo, orderRepo, _, _ := createTestTicketService()
	
	// Create test data
	order, _ := orderRepo.Create(&models.OrderCreateRequest{
		UserID:       1,
		EventID:      1,
		TotalAmount:  2500,
		BillingEmail: "test@example.com",
		BillingName:  "Test User",
		Status:       models.OrderCompleted,
	})
	
	ticket, _ := ticketRepo.CreateTicket(order.ID, 1, "TEST-QR-CODE")
	
	tests := []struct {
		name        string
		qrCode      string
		eventID     int
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid ticket",
			qrCode:      "TEST-QR-CODE",
			eventID:     1,
			expectError: false,
		},
		{
			name:        "ticket not found",
			qrCode:      "INVALID-QR-CODE",
			eventID:     1,
			expectError: true,
			errorMsg:    "invalid ticket: ticket not found",
		},
		{
			name:        "wrong event",
			qrCode:      "TEST-QR-CODE",
			eventID:     2,
			expectError: true,
			errorMsg:    "ticket is not valid for this event",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validatedTicket, err := service.ValidateTicket(tt.qrCode, tt.eventID)
			
			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
					return
				}
				if tt.errorMsg != "" && err.Error() != tt.errorMsg {
					t.Errorf("expected error message '%s', got '%s'", tt.errorMsg, err.Error())
				}
				return
			}
			
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			
			if validatedTicket == nil {
				t.Errorf("expected ticket but got nil")
				return
			}
			
			if validatedTicket.ID != ticket.ID {
				t.Errorf("expected ticket ID %d, got %d", ticket.ID, validatedTicket.ID)
			}
		})
	}
}

func TestTicketService_UseTicket(t *testing.T) {
	service, ticketRepo, orderRepo, _, _ := createTestTicketService()
	
	// Create test data
	order, _ := orderRepo.Create(&models.OrderCreateRequest{
		UserID:       1,
		EventID:      1,
		TotalAmount:  2500,
		BillingEmail: "test@example.com",
		BillingName:  "Test User",
		Status:       models.OrderCompleted,
	})
	
	ticketRepo.CreateTicket(order.ID, 1, "TEST-QR-CODE")
	
	tests := []struct {
		name        string
		qrCode      string
		eventID     int
		expectError bool
	}{
		{
			name:        "successful ticket use",
			qrCode:      "TEST-QR-CODE",
			eventID:     1,
			expectError: false,
		},
		{
			name:        "ticket not found",
			qrCode:      "INVALID-QR-CODE",
			eventID:     1,
			expectError: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.UseTicket(tt.qrCode, tt.eventID)
			
			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}
			
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			
			// Verify ticket status was updated
			ticket, _ := ticketRepo.GetTicketByQRCode(tt.qrCode)
			if ticket.Status != models.TicketUsed {
				t.Errorf("expected ticket status to be 'used', got '%s'", ticket.Status)
			}
		})
	}
}

func TestTicketService_RefundTickets(t *testing.T) {
	tests := []struct {
		name            string
		requestingUser  int
		setupMocks      func(*mockPaymentService)
		expectError     bool
		errorMsg        string
	}{
		{
			name:           "successful refund by owner",
			requestingUser: 1,
			setupMocks:     func(ps *mockPaymentService) {},
			expectError:    false,
		},
		{
			name:           "successful refund by admin",
			requestingUser: 2,
			setupMocks:     func(ps *mockPaymentService) {},
			expectError:    false,
		},
		{
			name:           "refund failure",
			requestingUser: 1,
			setupMocks: func(ps *mockPaymentService) {
				ps.shouldFailOps["RefundPayment"] = true
			},
			expectError: true,
			errorMsg:    "refund processing failed: refund processing failed",
		},
		{
			name:           "insufficient permissions",
			requestingUser: 999, // Non-existent user
			setupMocks:     func(ps *mockPaymentService) {},
			expectError:    true,
			errorMsg:       "user not found: user not found",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create fresh service and mocks for each test
			service, ticketRepo, orderRepo, paymentService, userRepo := createTestTicketService()
			
			// Add admin user
			userRepo.users[2] = &models.User{
				ID:   2,
				Role: models.RoleAdmin,
			}
			
			// Create fresh test data for each test
			order, _ := orderRepo.Create(&models.OrderCreateRequest{
				UserID:       1,
				EventID:      1,
				TotalAmount:  2500,
				BillingEmail: "test@example.com",
				BillingName:  "Test User",
				Status:       models.OrderCompleted,
			})
			order.PaymentID = "PAY-123"
			
			ticketRepo.CreateTicket(order.ID, 1, "TEST-QR-CODE")
			
			// Setup mocks
			tt.setupMocks(paymentService)
			
			result, err := service.RefundTickets(order.ID, tt.requestingUser)
			
			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
					return
				}
				if tt.errorMsg != "" && err.Error() != tt.errorMsg {
					t.Errorf("expected error message '%s', got '%s'", tt.errorMsg, err.Error())
				}
				return
			}
			
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			
			if result == nil {
				t.Errorf("expected refund result but got nil")
			}
		})
	}
}