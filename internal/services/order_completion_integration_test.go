package services

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"event-ticketing-platform/internal/models"
	"event-ticketing-platform/internal/repositories"
)

// TestOrderCompletionIntegration tests the complete order completion workflow
func TestOrderCompletionIntegration(t *testing.T) {
	t.Run("complete order completion workflow", func(t *testing.T) {
		// This test demonstrates the complete order completion workflow
		// including email generation and ticket processing
		
		// Create test data
		user := &models.User{
			ID:        1,
			Email:     "test@example.com",
			FirstName: "John",
			LastName:  "Doe",
		}

		order := &models.Order{
			ID:           1,
			UserID:       1,
			EventID:      1,
			OrderNumber:  "ORD-20240101-123456",
			TotalAmount:  5000, // $50.00
			Status:       models.OrderCompleted,
			PaymentID:    "payment-123",
			BillingEmail: "test@example.com",
			BillingName:  "John Doe",
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}

		tickets := []*models.Ticket{
			{
				ID:           1,
				OrderID:      1,
				TicketTypeID: 1,
				QRCode:       "TKT-1-1-123456-abc123",
				Status:       models.TicketActive,
				CreatedAt:    time.Now(),
			},
			{
				ID:           2,
				OrderID:      1,
				TicketTypeID: 1,
				QRCode:       "TKT-1-1-123456-def456",
				Status:       models.TicketActive,
				CreatedAt:    time.Now(),
			},
		}

		// Setup service with mocks
		orderRepo := NewMockOrderRepository()
		ticketRepo := NewMockTicketRepository()
		userRepo := &MockUserRepository{}
		paymentService := NewMockPaymentService(nil, nil)
		emailService := NewMockEmailService(nil)

		service := NewOrderService(orderRepo, ticketRepo, userRepo, paymentService, emailService)

		// Test HTML email generation
		htmlContent := service.generateOrderConfirmationHTML(order, user, tickets)
		
		// Verify HTML contains expected elements
		expectedHTMLElements := []string{
			"John Doe",
			"ORD-20240101-123456",
			"$50.00",
			"Order Confirmed!",
			"2 tickets",
		}

		for _, element := range expectedHTMLElements {
			if !containsString(htmlContent, element) {
				t.Errorf("HTML content missing expected element: %s", element)
			}
		}
		
		// Print content for debugging
		t.Logf("Generated HTML content: %s", htmlContent)

		// Test text email generation
		textContent := service.generateOrderConfirmationText(order, user, tickets)
		
		// Verify text contains expected elements
		expectedTextElements := []string{
			"John Doe",
			"ORD-20240101-123456",
			"$50.00",
			"Order Confirmed!",
			"2 ticket(s)",
			"ORDER DETAILS",
			"YOUR TICKETS",
			"IMPORTANT INFORMATION",
		}

		for _, element := range expectedTextElements {
			if !containsString(textContent, element) {
				t.Errorf("Text content missing expected element: %s", element)
			}
		}

		t.Logf("Order completion integration test completed successfully")
		t.Logf("Generated HTML email content length: %d characters", len(htmlContent))
		t.Logf("Generated text email content length: %d characters", len(textContent))
	})
}



// TestTicketGeneration tests ticket generation and QR code functionality
func TestTicketGeneration(t *testing.T) {
	t.Run("ticket QR code generation", func(t *testing.T) {
		tickets := []*models.Ticket{
			{
				ID:           1,
				OrderID:      1,
				TicketTypeID: 1,
				QRCode:       "TKT-1-1-123456-abc123",
				Status:       models.TicketActive,
				CreatedAt:    time.Now(),
			},
			{
				ID:           2,
				OrderID:      1,
				TicketTypeID: 1,
				QRCode:       "TKT-1-1-123456-def456",
				Status:       models.TicketActive,
				CreatedAt:    time.Now(),
			},
		}

		// Test ticket status methods
		for i, ticket := range tickets {
			if !ticket.IsActive() {
				t.Errorf("Ticket %d should be active", i+1)
			}

			if ticket.IsUsed() {
				t.Errorf("Ticket %d should not be used", i+1)
			}

			if ticket.IsRefunded() {
				t.Errorf("Ticket %d should not be refunded", i+1)
			}

			if !ticket.CanBeUsed() {
				t.Errorf("Ticket %d should be able to be used", i+1)
			}

			if !ticket.CanBeRefunded() {
				t.Errorf("Ticket %d should be able to be refunded", i+1)
			}

			// Test QR code format
			if len(ticket.QRCode) == 0 {
				t.Errorf("Ticket %d should have a QR code", i+1)
			}

			// QR codes should be unique
			for j, otherTicket := range tickets {
				if i != j && ticket.QRCode == otherTicket.QRCode {
					t.Errorf("Tickets %d and %d have duplicate QR codes", i+1, j+1)
				}
			}
		}

		t.Logf("Ticket generation test completed successfully with %d tickets", len(tickets))
	})
}

// TestEmailContentGeneration tests email content generation with various scenarios
func TestEmailContentGeneration(t *testing.T) {
	t.Run("email content with different ticket counts", func(t *testing.T) {
		// Setup service
		orderRepo := NewMockOrderRepository()
		ticketRepo := NewMockTicketRepository()
		userRepo := &MockUserRepository{}
		paymentService := NewMockPaymentService(nil, nil)
		emailService := NewMockEmailService(nil)

		service := NewOrderService(orderRepo, ticketRepo, userRepo, paymentService, emailService)

		user := &models.User{
			ID:        1,
			FirstName: "Jane",
			LastName:  "Smith",
		}

		// Test with single ticket
		singleTicketOrder := &models.Order{
			ID:           1,
			OrderNumber:  "ORD-20240101-111111",
			TotalAmount:  2500, // $25.00
			Status:       models.OrderCompleted,
			BillingEmail: "jane@example.com",
			CreatedAt:    time.Now(),
		}

		singleTicket := []*models.Ticket{
			{ID: 1, QRCode: "TKT-SINGLE-123"},
		}

		htmlSingle := service.generateOrderConfirmationHTML(singleTicketOrder, user, singleTicket)
		textSingle := service.generateOrderConfirmationText(singleTicketOrder, user, singleTicket)

		if !containsString(htmlSingle, "1 tickets") {
			t.Error("Single ticket HTML should mention '1 tickets'")
		}

		if !containsString(textSingle, "1 ticket(s)") {
			t.Error("Single ticket text should mention '1 ticket(s)'")
		}

		// Test with multiple tickets
		multiTicketOrder := &models.Order{
			ID:           2,
			OrderNumber:  "ORD-20240101-222222",
			TotalAmount:  7500, // $75.00
			Status:       models.OrderCompleted,
			BillingEmail: "jane@example.com",
			CreatedAt:    time.Now(),
		}

		multiTickets := []*models.Ticket{
			{ID: 1, QRCode: "TKT-MULTI-123"},
			{ID: 2, QRCode: "TKT-MULTI-456"},
			{ID: 3, QRCode: "TKT-MULTI-789"},
		}

		htmlMulti := service.generateOrderConfirmationHTML(multiTicketOrder, user, multiTickets)
		textMulti := service.generateOrderConfirmationText(multiTicketOrder, user, multiTickets)

		if !containsString(htmlMulti, "3 tickets") {
			t.Error("Multi ticket HTML should mention '3 tickets'")
		}

		if !containsString(textMulti, "3 ticket(s)") {
			t.Error("Multi ticket text should mention '3 ticket(s)'")
		}

		t.Log("Email content generation test completed successfully")
	})
}

// TestOrderStatusTracking tests comprehensive order status tracking functionality
func TestOrderStatusTracking(t *testing.T) {
	t.Run("order status transitions and notifications", func(t *testing.T) {
		// Setup service
		orderRepo := NewMockOrderRepository()
		ticketRepo := NewMockTicketRepository()
		userRepo := &MockUserRepository{}
		paymentService := NewMockPaymentService(nil, nil)
		emailService := NewMockEmailService(nil)

		service := NewOrderService(orderRepo, ticketRepo, userRepo, paymentService, emailService)

		// Test valid status transitions
		validTransitions := []struct {
			from models.OrderStatus
			to   models.OrderStatus
			valid bool
		}{
			{models.OrderPending, models.OrderCompleted, true},
			{models.OrderPending, models.OrderCancelled, true},
			{models.OrderCompleted, models.OrderRefunded, true},
			{models.OrderCancelled, models.OrderCompleted, false},
			{models.OrderRefunded, models.OrderCompleted, false},
			{models.OrderCompleted, models.OrderPending, false},
		}

		for _, transition := range validTransitions {
			isValid := service.isValidStatusTransition(transition.from, transition.to)
			if isValid != transition.valid {
				t.Errorf("Status transition %s -> %s: expected %v, got %v", 
					transition.from, transition.to, transition.valid, isValid)
			}
		}

		// Test notification triggers
		notificationTests := []struct {
			from         models.OrderStatus
			to           models.OrderStatus
			shouldNotify bool
		}{
			{models.OrderPending, models.OrderCompleted, true},
			{models.OrderCompleted, models.OrderRefunded, true},
			{models.OrderPending, models.OrderCancelled, true},
			{models.OrderCompleted, models.OrderCompleted, false}, // Same status
		}

		for _, test := range notificationTests {
			shouldNotify := service.shouldSendStatusNotification(test.from, test.to)
			if shouldNotify != test.shouldNotify {
				t.Errorf("Notification for %s -> %s: expected %v, got %v", 
					test.from, test.to, test.shouldNotify, shouldNotify)
			}
		}

		t.Log("Order status tracking test completed successfully")
	})
}

// TestEnhancedEmailGeneration tests enhanced email generation with ticket details
func TestEnhancedEmailGeneration(t *testing.T) {
	t.Run("enhanced email content with ticket details", func(t *testing.T) {
		// Create test order and tickets
		order := &models.Order{
			ID:           1,
			OrderNumber:  "ORD-20240101-123456",
			TotalAmount:  5000,
			Status:       models.OrderCompleted,
			BillingEmail: "test@example.com",
			CreatedAt:    time.Now(),
		}

		tickets := []*models.Ticket{
			{
				ID:           1,
				OrderID:      1,
				TicketTypeID: 1,
				QRCode:       "TKT-1-1-123456-abc123",
				Status:       models.TicketActive,
				CreatedAt:    time.Now(),
			},
			{
				ID:           2,
				OrderID:      1,
				TicketTypeID: 1,
				QRCode:       "TKT-1-1-123456-def456",
				Status:       models.TicketActive,
				CreatedAt:    time.Now(),
			},
		}

		// Create Resend service for testing
		config := ResendConfig{
			APIKey:    "test-key",
			FromEmail: "test@example.com",
			FromName:  "Test Platform",
		}
		resendService := NewResendEmailService(config)

		// Test HTML enhancement
		originalHTML := "<div>Original content</div><div class=\"footer\">Footer</div>"
		enhancedHTML := resendService.enhanceOrderConfirmationHTML(originalHTML, order, tickets)

		// Verify enhanced content contains ticket details
		expectedHTMLElements := []string{
			"Ticket Details",
			"TKT-1-1-123456-abc123",
			"TKT-1-1-123456-def456",
			"Mobile Access",
			"dashboard",
		}

		for _, element := range expectedHTMLElements {
			if !containsString(enhancedHTML, element) {
				t.Errorf("Enhanced HTML missing expected element: %s", element)
			}
		}

		// Test text enhancement
		originalText := "Original content\nEvent Ticketing Platform"
		enhancedText := resendService.enhanceOrderConfirmationText(originalText, order, tickets)

		// Verify enhanced content contains ticket details
		expectedTextElements := []string{
			"TICKET DETAILS",
			"TKT-1-1-123456-abc123",
			"TKT-1-1-123456-def456",
			"MOBILE ACCESS",
			"NEXT STEPS",
		}

		for _, element := range expectedTextElements {
			if !containsString(enhancedText, element) {
				t.Errorf("Enhanced text missing expected element: %s", element)
			}
		}

		t.Log("Enhanced email generation test completed successfully")
	})
}

// TestCompleteOrderWorkflow tests the complete order completion workflow
func TestCompleteOrderWorkflow(t *testing.T) {
	t.Run("complete order completion with email and status tracking", func(t *testing.T) {
		// Setup service with mocks
		orderRepo := NewMockOrderRepository()
		ticketRepo := NewMockTicketRepository()
		userRepo := &SimpleUserRepository{}
		paymentService := NewMockPaymentService(nil, nil)
		emailService := NewMockEmailService(nil)

		service := NewOrderService(orderRepo, ticketRepo, userRepo, paymentService, emailService)

		// Test data
		orderID := 1
		paymentID := "payment-123"
		ticketData := []struct {
			TicketTypeID int
			QRCode       string
		}{
			{TicketTypeID: 1, QRCode: "TKT-1-1-123456-abc123"},
			{TicketTypeID: 1, QRCode: "TKT-1-1-123456-def456"},
		}

		// Create a test order first
		orderRepo.orders[orderID] = &models.Order{
			ID:           orderID,
			UserID:       1,
			EventID:      1,
			OrderNumber:  "ORD-20240101-123456",
			TotalAmount:  5000,
			Status:       models.OrderPending,
			BillingEmail: "test@example.com",
			BillingName:  "John Doe",
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}

		// Test order completion
		err := service.CompleteOrder(orderID, paymentID, ticketData)
		if err != nil {
			t.Errorf("Order completion failed: %v", err)
		}

		// Verify order was processed
		// In a real test, you would verify the mock was called correctly
		t.Log("Order completion workflow test completed successfully")
	})
}

// TestQRCodeGeneration tests QR code generation and uniqueness
func TestQRCodeGeneration(t *testing.T) {
	t.Run("QR code uniqueness and format", func(t *testing.T) {
		// Generate multiple QR codes and verify uniqueness
		qrCodes := make(map[string]bool)
		orderID := 1
		ticketTypeID := 1

		// Create ticket service for QR code generation
		ticketRepo := NewMockTicketRepository()
		orderRepo := NewMockOrderRepository()
		paymentService := NewMockPaymentService(nil, nil)
		authService := &AuthService{} // Mock auth service
		pdfService := NewPDFService()

		service := NewTicketService(ticketRepo, orderRepo, paymentService, authService, pdfService, 15)

		// Generate 100 QR codes and check for uniqueness
		for i := 0; i < 100; i++ {
			qrCode, err := service.generateQRCode(orderID, ticketTypeID)
			if err != nil {
				t.Errorf("QR code generation failed: %v", err)
				continue
			}

			// Check format
			if len(qrCode) == 0 {
				t.Error("QR code should not be empty")
			}

			if !containsString(qrCode, "TKT-") {
				t.Error("QR code should start with 'TKT-'")
			}

			// Check uniqueness
			if qrCodes[qrCode] {
				t.Errorf("Duplicate QR code generated: %s", qrCode)
			}
			qrCodes[qrCode] = true
		}

		t.Logf("Generated %d unique QR codes successfully", len(qrCodes))
	})
}

// TestPDFGeneration tests PDF ticket generation
func TestPDFGeneration(t *testing.T) {
	t.Run("PDF ticket generation with multiple tickets", func(t *testing.T) {
		// Create test data
		tickets := []*models.Ticket{
			{
				ID:           1,
				OrderID:      1,
				TicketTypeID: 1,
				QRCode:       "TKT-1-1-123456-abc123",
				Status:       models.TicketActive,
				CreatedAt:    time.Now(),
			},
			{
				ID:           2,
				OrderID:      1,
				TicketTypeID: 1,
				QRCode:       "TKT-1-1-123456-def456",
				Status:       models.TicketActive,
				CreatedAt:    time.Now(),
			},
		}

		event := &models.Event{
			ID:          1,
			Title:       "Test Event",
			Description: "A test event for PDF generation",
			StartDate:   time.Now().Add(24 * time.Hour),
			Location:    "Test Venue",
		}

		order := &models.Order{
			ID:           1,
			OrderNumber:  "ORD-20240101-123456",
			TotalAmount:  5000,
			Status:       models.OrderCompleted,
			BillingEmail: "test@example.com",
			BillingName:  "John Doe",
			CreatedAt:    time.Now(),
		}

		// Generate PDF
		pdfService := NewPDFService()
		pdfData, err := pdfService.GenerateTicketsPDF(tickets, event, order)
		if err != nil {
			t.Errorf("PDF generation failed: %v", err)
		}

		// Verify PDF data
		if len(pdfData) == 0 {
			t.Error("PDF data should not be empty")
		}

		// Check PDF header
		if !containsBytes(pdfData, []byte("%PDF-")) {
			t.Error("Generated data should be a valid PDF")
		}

		t.Logf("Generated PDF with %d bytes for %d tickets", len(pdfData), len(tickets))
	})
}

// Helper functions for tests

// containsString checks if a string contains a substring
func containsString(s, substr string) bool {
	return strings.Contains(s, substr)
}

// containsBytes checks if byte slice contains another byte slice
func containsBytes(data, pattern []byte) bool {
	return bytes.Contains(data, pattern)
}

// SimpleUserRepository provides a simple mock implementation without testify
type SimpleUserRepository struct{}

func (m *SimpleUserRepository) Create(req *models.UserCreateRequest) (*models.User, error) { return nil, nil }
func (m *SimpleUserRepository) GetByID(id int) (*models.User, error) {
	return &models.User{
		ID:        id,
		Email:     "test@example.com",
		FirstName: "John",
		LastName:  "Doe",
		Role:      models.RoleAttendee,
	}, nil
}
func (m *SimpleUserRepository) GetByEmail(email string) (*models.User, error) { return nil, nil }
func (m *SimpleUserRepository) Update(id int, req *models.UserUpdateRequest) (*models.User, error) { return nil, nil }
func (m *SimpleUserRepository) UpdatePassword(id int, passwordHash string) error { return nil }
func (m *SimpleUserRepository) Delete(id int) error { return nil }
func (m *SimpleUserRepository) Search(filters repositories.UserSearchFilters) ([]*models.User, int, error) { return nil, 0, nil }
func (m *SimpleUserRepository) GetByRole(role models.UserRole) ([]*models.User, error) { return nil, nil }
func (m *SimpleUserRepository) CreateSession(userID int, sessionID string, expiresAt time.Time) error { return nil }
func (m *SimpleUserRepository) GetUserBySession(sessionID string) (*models.User, error) { return nil, nil }
func (m *SimpleUserRepository) DeleteSession(sessionID string) error { return nil }
func (m *SimpleUserRepository) DeleteExpiredSessions() error { return nil }
func (m *SimpleUserRepository) DeleteUserSessions(userID int) error { return nil }
func (m *SimpleUserRepository) ExtendSession(sessionID string, expiresAt time.Time) error { return nil }
func (m *SimpleUserRepository) SetVerificationToken(userID int, token string) error { return nil }
func (m *SimpleUserRepository) GetByVerificationToken(token string) (*models.User, error) { return nil, nil }
func (m *SimpleUserRepository) VerifyEmail(userID int) error { return nil }
func (m *SimpleUserRepository) SetPasswordResetToken(userID int, token string, expiresAt time.Time) error { return nil }
func (m *SimpleUserRepository) GetByPasswordResetToken(token string) (*models.User, error) { return nil, nil }
func (m *SimpleUserRepository) ClearPasswordResetToken(userID int) error { return nil }
func (m *SimpleUserRepository) CleanupExpiredTokens() error { return nil }