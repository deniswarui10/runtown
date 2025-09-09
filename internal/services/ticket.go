package services

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"event-ticketing-platform/internal/models"
	"event-ticketing-platform/internal/repositories"
)

// TicketRepository interface for ticket data operations
type TicketRepository interface {
	CreateTicketType(req *models.TicketTypeCreateRequest) (*models.TicketType, error)
	GetTicketTypeByID(id int) (*models.TicketType, error)
	GetTicketTypesByEvent(eventID int) ([]*models.TicketType, error)
	UpdateTicketType(id int, req *models.TicketTypeUpdateRequest) (*models.TicketType, error)
	DeleteTicketType(id int) error
	ReserveTickets(ticketTypeID, quantity, userID int, expirationMinutes int) (*repositories.TicketReservation, error)
	ReleaseReservation(reservationID string, ticketTypeID, quantity int) error
	CreateTicket(orderID, ticketTypeID int, qrCode string) (*models.Ticket, error)
	GetTicketByID(id int) (*models.Ticket, error)
	GetTicketByQRCode(qrCode string) (*models.Ticket, error)
	GetTicketsByOrder(orderID int) ([]*models.Ticket, error)
	UpdateTicketStatus(id int, status models.TicketStatus) error
	SearchTickets(filters repositories.TicketSearchFilters) ([]*models.Ticket, int, error)
}



// PaymentService interface for payment processing
type PaymentService interface {
	ProcessPayment(amount int, paymentMethod string, billingInfo PaymentBillingInfo) (*PaymentResult, error)
	RefundPayment(paymentID string, amount int) (*RefundResult, error)
	GetPaymentStatus(paymentID string) (*PaymentStatus, error)
}

// PaymentBillingInfo represents billing information for payment processing
type PaymentBillingInfo struct {
	Email       string `json:"email"`
	Name        string `json:"name"`
	Address     string `json:"address"`
	City        string `json:"city"`
	State       string `json:"state"`
	ZipCode     string `json:"zip_code"`
	Country     string `json:"country"`
	CardToken   string `json:"card_token"`   // Tokenized card information
	PaymentType string `json:"payment_type"` // "card", "paypal", etc.
}

// PaymentResult represents the result of a payment processing attempt
type PaymentResult struct {
	PaymentID        string    `json:"payment_id"`
	Status           string    `json:"status"`        // "success", "failed", "pending"
	Amount           int       `json:"amount"`        // Amount in cents
	TransactionID    string    `json:"transaction_id"`
	ProcessedAt      time.Time `json:"processed_at"`
	ErrorMessage     string    `json:"error_message,omitempty"`
	AuthorizationURL string    `json:"authorization_url,omitempty"` // For redirect-based payments like Paystack
}

// RefundResult represents the result of a refund attempt
type RefundResult struct {
	RefundID      string    `json:"refund_id"`
	Status        string    `json:"status"`        // "success", "failed", "pending"
	Amount        int       `json:"amount"`        // Amount in cents
	ProcessedAt   time.Time `json:"processed_at"`
	ErrorMessage  string    `json:"error_message,omitempty"`
}

// PaymentStatus represents the current status of a payment
type PaymentStatus struct {
	PaymentID     string    `json:"payment_id"`
	Status        string    `json:"status"`
	Amount        int       `json:"amount"`
	TransactionID string    `json:"transaction_id"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// TicketService handles ticket-related business logic
type TicketService struct {
	ticketRepo     TicketRepository
	orderRepo      OrderRepository
	paymentService PaymentService
	authService    *AuthService
	pdfService     *PDFService
	reservationTTL int // Reservation time-to-live in minutes
}

// NewTicketService creates a new ticket service
func NewTicketService(
	ticketRepo TicketRepository,
	orderRepo OrderRepository,
	paymentService PaymentService,
	authService *AuthService,
	pdfService *PDFService,
	reservationTTL int,
) *TicketService {
	if reservationTTL <= 0 {
		reservationTTL = 15 // Default 15 minutes
	}
	
	return &TicketService{
		ticketRepo:     ticketRepo,
		orderRepo:      orderRepo,
		paymentService: paymentService,
		authService:    authService,
		pdfService:     pdfService,
		reservationTTL: reservationTTL,
	}
}

// TicketReservationRequest represents a request to reserve tickets
type TicketReservationRequest struct {
	TicketTypeID int `json:"ticket_type_id"`
	Quantity     int `json:"quantity"`
	UserID       int `json:"user_id"`
}

// TicketPurchaseRequest represents a request to purchase tickets
type TicketPurchaseRequest struct {
	ReservationID   string             `json:"reservation_id"`
	EventID         int                `json:"event_id"`
	TicketSelections []TicketSelection `json:"ticket_selections"`
	BillingInfo     PaymentBillingInfo `json:"billing_info"`
	PaymentMethod   string             `json:"payment_method"`
	UserID          int                `json:"user_id"`
}

// TicketSelection represents a selection of tickets to purchase
type TicketSelection struct {
	TicketTypeID int `json:"ticket_type_id"`
	Quantity     int `json:"quantity"`
}

// PurchaseResult represents the result of a ticket purchase
type PurchaseResult struct {
	Order       *models.Order    `json:"order"`
	Tickets     []*models.Ticket `json:"tickets"`
	PaymentInfo *PaymentResult   `json:"payment_info"`
}

// ReserveTickets creates a time-limited reservation for tickets
func (s *TicketService) ReserveTickets(req *TicketReservationRequest) (*repositories.TicketReservation, error) {
	// Validate user permissions
	_, err := s.authService.userRepo.GetByID(req.UserID)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	// Validate ticket type exists and is available
	ticketType, err := s.ticketRepo.GetTicketTypeByID(req.TicketTypeID)
	if err != nil {
		return nil, fmt.Errorf("ticket type not found: %w", err)
	}

	// Check if tickets are available for purchase
	if !ticketType.IsAvailable() {
		if ticketType.IsSoldOut() {
			return nil, fmt.Errorf("tickets are sold out")
		}
		if ticketType.SaleNotStarted() {
			return nil, fmt.Errorf("ticket sales have not started yet")
		}
		if ticketType.SaleEnded() {
			return nil, fmt.Errorf("ticket sales have ended")
		}
		return nil, fmt.Errorf("tickets are not available for purchase")
	}

	// Validate quantity
	if req.Quantity <= 0 {
		return nil, fmt.Errorf("quantity must be greater than 0")
	}

	if req.Quantity > ticketType.Available() {
		return nil, fmt.Errorf("insufficient tickets available (requested: %d, available: %d)", 
			req.Quantity, ticketType.Available())
	}

	// Limit maximum tickets per reservation (business rule)
	maxTicketsPerReservation := 10
	if req.Quantity > maxTicketsPerReservation {
		return nil, fmt.Errorf("cannot reserve more than %d tickets at once", maxTicketsPerReservation)
	}

	// Create the reservation
	reservation, err := s.ticketRepo.ReserveTickets(
		req.TicketTypeID, 
		req.Quantity, 
		req.UserID, 
		s.reservationTTL,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to reserve tickets: %w", err)
	}

	return reservation, nil
}

// ReleaseReservation releases a ticket reservation
func (s *TicketService) ReleaseReservation(reservationID string, ticketTypeID, quantity int) error {
	err := s.ticketRepo.ReleaseReservation(reservationID, ticketTypeID, quantity)
	if err != nil {
		return fmt.Errorf("failed to release reservation: %w", err)
	}

	return nil
}

// PurchaseTickets processes a ticket purchase with payment
func (s *TicketService) PurchaseTickets(req *TicketPurchaseRequest) (*PurchaseResult, error) {
	// Validate user permissions
	_, err := s.authService.userRepo.GetByID(req.UserID)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	// Validate ticket selections and calculate total
	totalAmount, ticketDetails, err := s.validateAndCalculateTotal(req.TicketSelections)
	if err != nil {
		return nil, fmt.Errorf("invalid ticket selection: %w", err)
	}

	// Create pending order
	orderReq := &models.OrderCreateRequest{
		UserID:       req.UserID,
		EventID:      req.EventID,
		TotalAmount:  totalAmount,
		BillingEmail: req.BillingInfo.Email,
		BillingName:  req.BillingInfo.Name,
		Status:       models.OrderPending,
	}

	order, err := s.orderRepo.Create(orderReq)
	if err != nil {
		return nil, fmt.Errorf("failed to create order: %w", err)
	}

	// Process payment
	paymentResult, err := s.paymentService.ProcessPayment(
		totalAmount,
		req.PaymentMethod,
		req.BillingInfo,
	)
	if err != nil {
		// Cancel the order if payment fails
		s.orderRepo.UpdateStatus(order.ID, models.OrderCancelled)
		return nil, fmt.Errorf("payment processing failed: %w", err)
	}

	if paymentResult.Status != "success" {
		// Cancel the order if payment is not successful
		s.orderRepo.UpdateStatus(order.ID, models.OrderCancelled)
		return nil, fmt.Errorf("payment failed: %s", paymentResult.ErrorMessage)
	}

	// Generate tickets with QR codes
	var ticketData []struct {
		TicketTypeID int
		QRCode       string
	}

	for _, detail := range ticketDetails {
		for i := 0; i < detail.Quantity; i++ {
			qrCode, err := s.generateQRCode(order.ID, detail.TicketTypeID)
			if err != nil {
				// Refund payment and cancel order
				s.paymentService.RefundPayment(paymentResult.PaymentID, totalAmount)
				s.orderRepo.UpdateStatus(order.ID, models.OrderCancelled)
				return nil, fmt.Errorf("failed to generate QR code: %w", err)
			}

			ticketData = append(ticketData, struct {
				TicketTypeID int
				QRCode       string
			}{
				TicketTypeID: detail.TicketTypeID,
				QRCode:       qrCode,
			})
		}
	}

	// Complete the order and create tickets in a transaction
	err = s.orderRepo.ProcessOrderCompletion(order.ID, paymentResult.PaymentID, ticketData)
	if err != nil {
		// Refund payment and cancel order
		s.paymentService.RefundPayment(paymentResult.PaymentID, totalAmount)
		s.orderRepo.UpdateStatus(order.ID, models.OrderCancelled)
		return nil, fmt.Errorf("failed to complete order: %w", err)
	}

	// Get the updated order
	completedOrder, err := s.orderRepo.GetByID(order.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get completed order: %w", err)
	}

	// Get the created tickets
	tickets, err := s.ticketRepo.GetTicketsByOrder(order.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get created tickets: %w", err)
	}

	return &PurchaseResult{
		Order:       completedOrder,
		Tickets:     tickets,
		PaymentInfo: paymentResult,
	}, nil
}

// RefundTickets processes a ticket refund
func (s *TicketService) RefundTickets(orderID int, requestingUserID int) (*RefundResult, error) {
	// Get the order
	order, err := s.orderRepo.GetByID(orderID)
	if err != nil {
		return nil, fmt.Errorf("order not found: %w", err)
	}

	// Validate user permissions (user must own the order or be admin)
	user, err := s.authService.userRepo.GetByID(requestingUserID)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	if user.Role != models.RoleAdmin && order.UserID != requestingUserID {
		return nil, fmt.Errorf("insufficient permissions to refund this order")
	}

	// Check if order can be refunded
	if !order.CanBeRefunded() {
		return nil, fmt.Errorf("order cannot be refunded in current status: %s", order.Status)
	}

	// Get tickets for the order
	tickets, err := s.ticketRepo.GetTicketsByOrder(orderID)
	if err != nil {
		return nil, fmt.Errorf("failed to get order tickets: %w", err)
	}

	// Check if any tickets have been used
	for _, ticket := range tickets {
		if ticket.IsUsed() {
			return nil, fmt.Errorf("cannot refund order with used tickets")
		}
	}

	// Process refund
	refundResult, err := s.paymentService.RefundPayment(order.PaymentID, order.TotalAmount)
	if err != nil {
		return nil, fmt.Errorf("refund processing failed: %w", err)
	}

	if refundResult.Status != "success" {
		return nil, fmt.Errorf("refund failed: %s", refundResult.ErrorMessage)
	}

	// Update order status to refunded
	err = s.orderRepo.UpdateStatus(orderID, models.OrderRefunded)
	if err != nil {
		return nil, fmt.Errorf("failed to update order status: %w", err)
	}

	// Update all tickets to refunded status
	for _, ticket := range tickets {
		err = s.ticketRepo.UpdateTicketStatus(ticket.ID, models.TicketRefunded)
		if err != nil {
			// Log error but don't fail the refund
			fmt.Printf("Warning: failed to update ticket %d status to refunded: %v\n", ticket.ID, err)
		}
	}

	return refundResult, nil
}

// ValidateTicket validates a ticket by QR code (for event entry)
func (s *TicketService) ValidateTicket(qrCode string, eventID int) (*models.Ticket, error) {
	// Get ticket by QR code
	ticket, err := s.ticketRepo.GetTicketByQRCode(qrCode)
	if err != nil {
		return nil, fmt.Errorf("invalid ticket: %w", err)
	}

	// Get the order to verify event
	order, err := s.orderRepo.GetByID(ticket.OrderID)
	if err != nil {
		return nil, fmt.Errorf("failed to get ticket order: %w", err)
	}

	// Verify ticket is for the correct event
	if order.EventID != eventID {
		return nil, fmt.Errorf("ticket is not valid for this event")
	}

	// Check if ticket can be used
	if !ticket.CanBeUsed() {
		return nil, fmt.Errorf("ticket cannot be used (status: %s)", ticket.Status)
	}

	return ticket, nil
}

// UseTicket marks a ticket as used (for event entry)
func (s *TicketService) UseTicket(qrCode string, eventID int) error {
	// Validate the ticket first
	ticket, err := s.ValidateTicket(qrCode, eventID)
	if err != nil {
		return err
	}

	// Mark ticket as used
	err = s.ticketRepo.UpdateTicketStatus(ticket.ID, models.TicketUsed)
	if err != nil {
		return fmt.Errorf("failed to mark ticket as used: %w", err)
	}

	return nil
}

// GetUserTickets retrieves tickets for a user
func (s *TicketService) GetUserTickets(userID int, requestingUserID int) ([]*models.Ticket, error) {
	// Validate user permissions
	user, err := s.authService.userRepo.GetByID(requestingUserID)
	if err != nil {
		return nil, fmt.Errorf("requesting user not found: %w", err)
	}

	// Users can only see their own tickets unless they're admin
	if user.Role != models.RoleAdmin && requestingUserID != userID {
		return nil, fmt.Errorf("insufficient permissions to view user tickets")
	}

	// Search for user's tickets
	filters := repositories.TicketSearchFilters{
		UserID:   userID,
		Limit:    100, // Reasonable limit
		Offset:   0,
		SortBy:   "created_at",
		SortDesc: true,
	}

	tickets, _, err := s.ticketRepo.SearchTickets(filters)
	if err != nil {
		return nil, fmt.Errorf("failed to get user tickets: %w", err)
	}

	return tickets, nil
}

// GetEventTickets retrieves tickets for an event (for organizers)
func (s *TicketService) GetEventTickets(eventID int, requestingUserID int) ([]*models.Ticket, error) {
	// Check if user can view event tickets
	canView, err := s.canUserViewEventTickets(eventID, requestingUserID)
	if err != nil {
		return nil, err
	}

	if !canView {
		return nil, fmt.Errorf("insufficient permissions to view event tickets")
	}

	// Search for event tickets
	filters := repositories.TicketSearchFilters{
		EventID:  eventID,
		Limit:    1000, // Higher limit for organizers
		Offset:   0,
		SortBy:   "created_at",
		SortDesc: true,
	}

	tickets, _, err := s.ticketRepo.SearchTickets(filters)
	if err != nil {
		return nil, fmt.Errorf("failed to get event tickets: %w", err)
	}

	return tickets, nil
}

// validateAndCalculateTotal validates ticket selections and calculates total amount
func (s *TicketService) validateAndCalculateTotal(selections []TicketSelection) (int, []TicketSelection, error) {
	if len(selections) == 0 {
		return 0, nil, fmt.Errorf("no tickets selected")
	}

	var totalAmount int
	var validSelections []TicketSelection

	for _, selection := range selections {
		if selection.Quantity <= 0 {
			continue // Skip invalid quantities
		}

		// Get ticket type
		ticketType, err := s.ticketRepo.GetTicketTypeByID(selection.TicketTypeID)
		if err != nil {
			return 0, nil, fmt.Errorf("ticket type %d not found", selection.TicketTypeID)
		}

		// Check availability
		if !ticketType.IsAvailable() {
			return 0, nil, fmt.Errorf("ticket type '%s' is not available", ticketType.Name)
		}

		if selection.Quantity > ticketType.Available() {
			return 0, nil, fmt.Errorf("insufficient tickets available for '%s' (requested: %d, available: %d)", 
				ticketType.Name, selection.Quantity, ticketType.Available())
		}

		// Calculate amount for this selection
		selectionAmount := ticketType.Price * selection.Quantity
		totalAmount += selectionAmount

		validSelections = append(validSelections, selection)
	}

	if len(validSelections) == 0 {
		return 0, nil, fmt.Errorf("no valid ticket selections")
	}

	if totalAmount <= 0 {
		return 0, nil, fmt.Errorf("total amount must be greater than 0")
	}

	return totalAmount, validSelections, nil
}

// generateQRCode generates a unique QR code for a ticket
func (s *TicketService) generateQRCode(orderID, ticketTypeID int) (string, error) {
	// Generate random bytes for uniqueness
	randomBytes := make([]byte, 16)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}

	// Create QR code with order ID, ticket type ID, and random component
	timestamp := time.Now().Unix()
	qrData := fmt.Sprintf("TKT-%d-%d-%d-%s", orderID, ticketTypeID, timestamp, hex.EncodeToString(randomBytes))

	return qrData, nil
}

// GenerateTicketsPDF generates a PDF containing tickets
func (s *TicketService) GenerateTicketsPDF(tickets []*models.Ticket, event *models.Event, order *models.Order) ([]byte, error) {
	if s.pdfService == nil {
		return nil, fmt.Errorf("PDF service not available")
	}

	return s.pdfService.GenerateTicketsPDF(tickets, event, order)
}

// GetTicketTypesByEventID retrieves ticket types for an event
func (s *TicketService) GetTicketTypesByEventID(eventID int) ([]*models.TicketType, error) {
	return s.ticketRepo.GetTicketTypesByEvent(eventID)
}

// GetTicketTypeByID retrieves a ticket type by ID
func (s *TicketService) GetTicketTypeByID(id int) (*models.TicketType, error) {
	return s.ticketRepo.GetTicketTypeByID(id)
}



// CreateTicketType creates a new ticket type
func (s *TicketService) CreateTicketType(req *models.TicketTypeCreateRequest) (*models.TicketType, error) {
	return s.ticketRepo.CreateTicketType(req)
}

// UpdateTicketType updates an existing ticket type
func (s *TicketService) UpdateTicketType(id int, req *models.TicketTypeUpdateRequest) (*models.TicketType, error) {
	return s.ticketRepo.UpdateTicketType(id, req)
}

// DeleteTicketType deletes a ticket type
func (s *TicketService) DeleteTicketType(id int) error {
	return s.ticketRepo.DeleteTicketType(id)
}

// GetTicketByID retrieves a ticket by ID
func (s *TicketService) GetTicketByID(id int) (*models.Ticket, error) {
	return s.ticketRepo.GetTicketByID(id)
}

// GetTicketsByOrderID retrieves tickets for an order
func (s *TicketService) GetTicketsByOrderID(orderID int) ([]*models.Ticket, error) {
	return s.ticketRepo.GetTicketsByOrder(orderID)
}

// canUserViewEventTickets checks if a user can view tickets for an event
func (s *TicketService) canUserViewEventTickets(eventID int, userID int) (bool, error) {
	user, err := s.authService.userRepo.GetByID(userID)
	if err != nil {
		return false, fmt.Errorf("user not found: %w", err)
	}

	// Admins can view any event tickets
	if user.Role == models.RoleAdmin {
		return true, nil
	}

	// For organizers, check if they own the event
	if user.Role == models.RoleOrganizer {
		// This would require an event service or repository to check ownership
		// For now, we'll assume organizers can view tickets for events they organize
		// In a real implementation, you'd check event ownership here
		return true, nil
	}

	return false, nil
}