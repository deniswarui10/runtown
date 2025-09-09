package services

import (
	"fmt"

	"event-ticketing-platform/internal/models"
	"event-ticketing-platform/internal/repositories"
)

// OrderService handles order-related business logic
type OrderService struct {
	orderRepo      OrderRepository
	ticketRepo     TicketRepository
	userRepo       UserRepository
	paymentService PaymentService
	emailService   EmailService
}

// OrderRepository interface for order data operations
type OrderRepository interface {
	Create(req *models.OrderCreateRequest) (*models.Order, error)
	GetByID(id int) (*models.Order, error)
	GetByOrderNumber(orderNumber string) (*models.Order, error)
	Update(id int, req *models.OrderUpdateRequest) (*models.Order, error)
	UpdateStatus(id int, status models.OrderStatus) error
	GetByUser(userID int, limit, offset int) ([]*models.Order, int, error)
	GetByEvent(eventID int, limit, offset int) ([]*models.Order, int, error)
	Search(filters repositories.OrderSearchFilters) ([]*models.Order, int, error)
	GetOrdersWithDetails(filters repositories.OrderSearchFilters) ([]*repositories.OrderWithDetails, int, error)
	ProcessOrderCompletion(orderID int, paymentID string, ticketData []struct {
		TicketTypeID int
		QRCode       string
	}) error
	GetOrderStatistics(eventID *int, userID *int) (map[string]interface{}, error)

	// Admin-specific methods
	GetOrderCount() (int, error)
	GetTotalRevenue() (float64, error)
}

// NewOrderService creates a new order service
func NewOrderService(
	orderRepo OrderRepository,
	ticketRepo TicketRepository,
	userRepo UserRepository,
	paymentService PaymentService,
	emailService EmailService,
) *OrderService {
	return &OrderService{
		orderRepo:      orderRepo,
		ticketRepo:     ticketRepo,
		userRepo:       userRepo,
		paymentService: paymentService,
		emailService:   emailService,
	}
}

// CreateOrder creates a new order
func (s *OrderService) CreateOrder(req *models.OrderCreateRequest) (*models.Order, error) {
	return s.orderRepo.Create(req)
}

// CompleteOrder completes an order and sends confirmation email with tickets
func (s *OrderService) CompleteOrder(orderID int, paymentID string, ticketData []struct {
	TicketTypeID int
	QRCode       string
}) error {
	// Complete the order in the repository
	err := s.orderRepo.ProcessOrderCompletion(orderID, paymentID, ticketData)
	if err != nil {
		return fmt.Errorf("failed to complete order: %w", err)
	}

	// Get the completed order
	order, err := s.orderRepo.GetByID(orderID)
	if err != nil {
		return fmt.Errorf("failed to get completed order: %w", err)
	}

	// Get the user
	user, err := s.userRepo.GetByID(order.UserID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	// Get tickets for the order
	tickets, err := s.ticketRepo.GetTicketsByOrder(orderID)
	if err != nil {
		return fmt.Errorf("failed to get order tickets: %w", err)
	}

	// Send order confirmation email with tickets
	err = s.sendOrderConfirmationEmail(order, user, tickets)
	if err != nil {
		// Log error but don't fail the order completion
		fmt.Printf("Warning: failed to send order confirmation email for order %s: %v\n", order.OrderNumber, err)
	}

	return nil
}

// sendOrderConfirmationEmail sends order confirmation email with ticket attachments
func (s *OrderService) sendOrderConfirmationEmail(order *models.Order, user *models.User, tickets []*models.Ticket) error {
	if s.emailService == nil {
		return fmt.Errorf("email service not available")
	}

	// Create email content with order and ticket details
	subject := fmt.Sprintf("Order Confirmation - %s", order.OrderNumber)

	// Generate HTML content
	htmlContent := s.generateOrderConfirmationHTML(order, user, tickets)

	// Generate text content
	textContent := s.generateOrderConfirmationText(order, user, tickets)

	// Send email
	err := s.emailService.SendOrderConfirmationWithTickets(
		order.BillingEmail,
		user.FullName(),
		subject,
		htmlContent,
		textContent,
		order,
		tickets,
	)

	if err != nil {
		return fmt.Errorf("failed to send order confirmation email: %w", err)
	}

	return nil
}

// generateOrderConfirmationHTML generates HTML content for order confirmation email
func (s *OrderService) generateOrderConfirmationHTML(order *models.Order, user *models.User, tickets []*models.Ticket) string {
	html := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Order Confirmation</title>
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background-color: #4F46E5; color: white; padding: 20px; text-align: center; border-radius: 8px 8px 0 0; }
        .content { padding: 20px; background-color: #f9f9f9; }
        .order-details { background-color: #EEF2FF; padding: 15px; border-left: 4px solid #4F46E5; margin: 20px 0; border-radius: 4px; }
        .ticket-item { background-color: white; padding: 15px; margin: 10px 0; border-radius: 4px; border: 1px solid #e5e7eb; }
        .button { display: inline-block; padding: 12px 24px; background-color: #4F46E5; color: white; text-decoration: none; border-radius: 4px; margin: 20px 0; }
        .footer { text-align: center; padding: 20px; color: #666; font-size: 12px; }
        .success-icon { color: #10B981; font-size: 48px; text-align: center; margin-bottom: 20px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <div class="success-icon">✓</div>
            <h1>Order Confirmed!</h1>
            <p>Thank you for your purchase</p>
        </div>
        <div class="content">
            <p>Dear %s,</p>
            <p>Your order has been successfully processed and your tickets are ready!</p>
            
            <div class="order-details">
                <h3>Order Details</h3>
                <p><strong>Order Number:</strong> %s</p>
                <p><strong>Order Date:</strong> %s</p>
                <p><strong>Total Amount:</strong> KSh %.2f</p>
                <p><strong>Payment Status:</strong> %s</p>
            </div>
            
            <h3>Your Tickets (%d tickets)</h3>
            <p>Please find your tickets attached to this email as a PDF. You can also download them from your account dashboard.</p>
            
            <div style="text-align: center; margin: 30px 0;">
                <a href="http://localhost:8080/dashboard/orders/%d" class="button">View Order Details</a>
            </div>
            
            <div style="background-color: #FEF3C7; padding: 15px; border-left: 4px solid #F59E0B; margin: 20px 0; border-radius: 4px;">
                <h4 style="margin-top: 0; color: #92400E;">Important Information:</h4>
                <ul style="color: #92400E; margin-bottom: 0;">
                    <li>Please bring your tickets (printed or on mobile) to the event</li>
                    <li>Arrive early to avoid queues at the entrance</li>
                    <li>Each ticket contains a unique QR code for entry</li>
                    <li>Tickets are non-transferable and non-refundable</li>
                </ul>
            </div>
            
            <p>If you have any questions about your order or need assistance, please don't hesitate to contact our support team.</p>
            
            <p>Thank you for choosing Event Ticketing Platform!</p>
        </div>
        <div class="footer">
            <p>Event Ticketing Platform</p>
            <p>This email was sent to %s</p>
        </div>
    </div>
</body>
</html>`,
		user.FullName(),
		order.OrderNumber,
		order.CreatedAt.Format("January 2, 2006 at 3:04 PM"),
		order.TotalAmountInCurrency(),
		order.GetStatusDisplayName(),
		len(tickets),
		order.ID,
		order.BillingEmail,
	)

	return html
}

// generateOrderConfirmationText generates text content for order confirmation email
func (s *OrderService) generateOrderConfirmationText(order *models.Order, user *models.User, tickets []*models.Ticket) string {
	text := fmt.Sprintf(`Order Confirmed!

Dear %s,

Your order has been successfully processed and your tickets are ready!

ORDER DETAILS
=============
Order Number: %s
Order Date: %s
Total Amount: KSh %.2f
Payment Status: %s

YOUR TICKETS
============
You have %d ticket(s) for this order.
Please find your tickets attached to this email as a PDF.
You can also download them from your account dashboard at:
http://localhost:8080/dashboard/orders/%d

IMPORTANT INFORMATION
====================
• Please bring your tickets (printed or on mobile) to the event
• Arrive early to avoid queues at the entrance
• Each ticket contains a unique QR code for entry
• Tickets are non-transferable and non-refundable

If you have any questions about your order or need assistance, please don't hesitate to contact our support team.

Thank you for choosing Event Ticketing Platform!

Event Ticketing Platform
This email was sent to %s`,
		user.FullName(),
		order.OrderNumber,
		order.CreatedAt.Format("January 2, 2006 at 3:04 PM"),
		order.TotalAmountInCurrency(),
		order.GetStatusDisplayName(),
		len(tickets),
		order.ID,
		order.BillingEmail,
	)

	return text
}

// GetUserOrders retrieves orders for a user with pagination
func (s *OrderService) GetUserOrders(userID int, limit, offset int) ([]*repositories.OrderWithDetails, int, error) {
	filters := repositories.OrderSearchFilters{
		UserID:   userID,
		Limit:    limit,
		Offset:   offset,
		SortBy:   "created_at",
		SortDesc: true,
	}

	return s.orderRepo.GetOrdersWithDetails(filters)
}

// GetOrderByID retrieves an order by ID with permission checking
func (s *OrderService) GetOrderByID(orderID int, requestingUserID int) (*models.Order, error) {
	// Get the order
	order, err := s.orderRepo.GetByID(orderID)
	if err != nil {
		return nil, fmt.Errorf("order not found: %w", err)
	}

	// Check permissions
	user, err := s.userRepo.GetByID(requestingUserID)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	// Users can only view their own orders unless they're admin
	if user.Role != models.RoleAdmin && order.UserID != requestingUserID {
		return nil, fmt.Errorf("insufficient permissions to view this order")
	}

	return order, nil
}

// GetOrderWithTickets retrieves an order with its tickets
func (s *OrderService) GetOrderWithTickets(orderID int, requestingUserID int) (*models.Order, []*models.Ticket, error) {
	// Get the order with permission checking
	order, err := s.GetOrderByID(orderID, requestingUserID)
	if err != nil {
		return nil, nil, err
	}

	// Get tickets for the order
	tickets, err := s.ticketRepo.GetTicketsByOrder(orderID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get order tickets: %w", err)
	}

	return order, tickets, nil
}

// CancelOrder cancels a pending order
func (s *OrderService) CancelOrder(orderID int, requestingUserID int) error {
	// Get the order with permission checking
	order, err := s.GetOrderByID(orderID, requestingUserID)
	if err != nil {
		return err
	}

	// Check if order can be cancelled
	if !order.CanBeCancelled() {
		return fmt.Errorf("order cannot be cancelled in current status: %s", order.Status)
	}

	// Update order status
	err = s.orderRepo.UpdateStatus(orderID, models.OrderCancelled)
	if err != nil {
		return fmt.Errorf("failed to cancel order: %w", err)
	}

	return nil
}

// GetEventOrders retrieves orders for an event (for organizers)
func (s *OrderService) GetEventOrders(eventID int, requestingUserID int, limit, offset int) ([]*repositories.OrderWithDetails, int, error) {
	// Check permissions - user must be admin or event organizer
	user, err := s.userRepo.GetByID(requestingUserID)
	if err != nil {
		return nil, 0, fmt.Errorf("user not found: %w", err)
	}

	// For now, allow organizers and admins to view event orders
	// In a real implementation, you'd check if the user owns the event
	if user.Role != models.RoleAdmin && user.Role != models.RoleOrganizer {
		return nil, 0, fmt.Errorf("insufficient permissions to view event orders")
	}

	filters := repositories.OrderSearchFilters{
		EventID:  eventID,
		Limit:    limit,
		Offset:   offset,
		SortBy:   "created_at",
		SortDesc: true,
	}

	return s.orderRepo.GetOrdersWithDetails(filters)
}

// GetOrderStatistics retrieves order statistics
func (s *OrderService) GetOrderStatistics(eventID *int, userID *int, requestingUserID int) (map[string]interface{}, error) {
	// Check permissions
	user, err := s.userRepo.GetByID(requestingUserID)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	// If requesting user stats, must be the same user or admin
	if userID != nil && user.Role != models.RoleAdmin && *userID != requestingUserID {
		return nil, fmt.Errorf("insufficient permissions to view user statistics")
	}

	// If requesting event stats, must be admin or organizer
	if eventID != nil && user.Role != models.RoleAdmin && user.Role != models.RoleOrganizer {
		return nil, fmt.Errorf("insufficient permissions to view event statistics")
	}

	return s.orderRepo.GetOrderStatistics(eventID, userID)
}

// SearchUserOrders searches orders for a user with filters and pagination
func (s *OrderService) SearchUserOrders(userID int, filters repositories.OrderSearchFilters, requestingUserID int) ([]*repositories.OrderWithDetails, int, error) {
	// Check permissions
	user, err := s.userRepo.GetByID(requestingUserID)
	if err != nil {
		return nil, 0, fmt.Errorf("user not found: %w", err)
	}

	// Users can only view their own orders unless they're admin
	if user.Role != models.RoleAdmin && userID != requestingUserID {
		return nil, 0, fmt.Errorf("insufficient permissions to view these orders")
	}

	// Set the user ID in filters
	filters.UserID = userID

	return s.orderRepo.GetOrdersWithDetails(filters)
}

// GetUpcomingEventsForUser retrieves upcoming events that the user has tickets for
func (s *OrderService) GetUpcomingEventsForUser(userID int, requestingUserID int) ([]*models.Event, error) {
	// Check permissions
	user, err := s.userRepo.GetByID(requestingUserID)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	// Users can only view their own upcoming events unless they're admin
	if user.Role != models.RoleAdmin && userID != requestingUserID {
		return nil, fmt.Errorf("insufficient permissions to view these events")
	}

	// Get user's completed orders
	filters := repositories.OrderSearchFilters{
		UserID:   userID,
		Status:   models.OrderCompleted,
		Limit:    100,
		SortBy:   "created_at",
		SortDesc: true,
	}

	ordersWithDetails, _, err := s.orderRepo.GetOrdersWithDetails(filters)
	if err != nil {
		return nil, fmt.Errorf("failed to get user orders: %w", err)
	}

	// Get unique event IDs from completed orders
	eventIDs := make(map[int]bool)
	for _, orderDetail := range ordersWithDetails {
		eventIDs[orderDetail.Order.EventID] = true
	}

	// This would require an event service dependency, but since we don't have it in the order service,
	// we'll return the event IDs and let the handler resolve the events
	// For now, return empty slice - this method should be moved to a higher-level service
	return []*models.Event{}, nil
}

// TrackOrderStatusUpdate tracks and logs order status changes
func (s *OrderService) TrackOrderStatusUpdate(orderID int, newStatus models.OrderStatus, requestingUserID int) error {
	// Get current order
	order, err := s.orderRepo.GetByID(orderID)
	if err != nil {
		return fmt.Errorf("order not found: %w", err)
	}

	// Check permissions
	user, err := s.userRepo.GetByID(requestingUserID)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}

	// Only admins or order owners can update status
	if user.Role != models.RoleAdmin && order.UserID != requestingUserID {
		return fmt.Errorf("insufficient permissions to update order status")
	}

	// Validate status transition
	if !s.isValidStatusTransition(order.Status, newStatus) {
		return fmt.Errorf("invalid status transition from %s to %s", order.Status, newStatus)
	}

	// Update status
	err = s.orderRepo.UpdateStatus(orderID, newStatus)
	if err != nil {
		return fmt.Errorf("failed to update order status: %w", err)
	}

	// Log status change (in a real implementation, this would go to an audit log)
	fmt.Printf("Order %s status changed from %s to %s by user %d\n",
		order.OrderNumber, order.Status, newStatus, requestingUserID)

	// Send notification email for certain status changes
	if s.shouldSendStatusNotification(order.Status, newStatus) {
		err = s.sendStatusUpdateNotification(order, newStatus)
		if err != nil {
			// Log error but don't fail the status update
			fmt.Printf("Warning: failed to send status update notification for order %s: %v\n",
				order.OrderNumber, err)
		}
	}

	return nil
}

// isValidStatusTransition checks if a status transition is valid
func (s *OrderService) isValidStatusTransition(currentStatus, newStatus models.OrderStatus) bool {
	// Define valid transitions
	validTransitions := map[models.OrderStatus][]models.OrderStatus{
		models.OrderPending:   {models.OrderCompleted, models.OrderCancelled},
		models.OrderCompleted: {models.OrderRefunded},
		models.OrderCancelled: {}, // No transitions from cancelled
		models.OrderRefunded:  {}, // No transitions from refunded
	}

	allowedStatuses, exists := validTransitions[currentStatus]
	if !exists {
		return false
	}

	for _, allowedStatus := range allowedStatuses {
		if newStatus == allowedStatus {
			return true
		}
	}

	return false
}

// shouldSendStatusNotification determines if a status change should trigger an email notification
func (s *OrderService) shouldSendStatusNotification(oldStatus, newStatus models.OrderStatus) bool {
	// Send notifications for these transitions
	notificationTransitions := map[string]bool{
		string(models.OrderPending) + "->" + string(models.OrderCompleted):  true,
		string(models.OrderCompleted) + "->" + string(models.OrderRefunded): true,
		string(models.OrderPending) + "->" + string(models.OrderCancelled):  true,
	}

	transitionKey := string(oldStatus) + "->" + string(newStatus)
	return notificationTransitions[transitionKey]
}

// sendStatusUpdateNotification sends an email notification for status updates
func (s *OrderService) sendStatusUpdateNotification(order *models.Order, newStatus models.OrderStatus) error {
	if s.emailService == nil {
		return fmt.Errorf("email service not available")
	}

	// For now, just log the status change notification
	// In a real implementation, you'd send specific notification emails
	fmt.Printf("Status update notification for order %s: %s -> %s\n",
		order.OrderNumber, order.Status, newStatus)

	return nil
}

// generateCompletionNotificationHTML generates HTML for order completion notification
func (s *OrderService) generateCompletionNotificationHTML(order *models.Order, user *models.User) string {
	return fmt.Sprintf(`
<div style="font-family: Arial, sans-serif; max-width: 600px; margin: 0 auto;">
	<h2 style="color: #10B981;">Order Completed Successfully!</h2>
	<p>Dear %s,</p>
	<p>Great news! Your order <strong>%s</strong> has been completed and your tickets are ready.</p>
	<p>You can download your tickets from your account dashboard.</p>
	<p>Thank you for your purchase!</p>
</div>`, user.FullName(), order.OrderNumber)
}

// generateCompletionNotificationText generates text for order completion notification
func (s *OrderService) generateCompletionNotificationText(order *models.Order, user *models.User) string {
	return fmt.Sprintf(`Order Completed Successfully!

Dear %s,

Great news! Your order %s has been completed and your tickets are ready.

You can download your tickets from your account dashboard.

Thank you for your purchase!`, user.FullName(), order.OrderNumber)
}

// generateRefundNotificationHTML generates HTML for refund notification
func (s *OrderService) generateRefundNotificationHTML(order *models.Order, user *models.User) string {
	return fmt.Sprintf(`
<div style="font-family: Arial, sans-serif; max-width: 600px; margin: 0 auto;">
	<h2 style="color: #EF4444;">Order Refunded</h2>
	<p>Dear %s,</p>
	<p>Your order <strong>%s</strong> has been refunded.</p>
	<p>The refund amount of <strong>KSh %.2f</strong> will be processed back to your original payment method within 3-5 business days.</p>
	<p>If you have any questions, please contact our support team.</p>
</div>`, user.FullName(), order.OrderNumber, order.TotalAmountInCurrency())
}

// generateRefundNotificationText generates text for refund notification
func (s *OrderService) generateRefundNotificationText(order *models.Order, user *models.User) string {
	return fmt.Sprintf(`Order Refunded

Dear %s,

Your order %s has been refunded.

The refund amount of KSh %.2f will be processed back to your original payment method within 3-5 business days.

If you have any questions, please contact our support team.`,
		user.FullName(), order.OrderNumber, order.TotalAmountInCurrency())
}

// generateCancellationNotificationHTML generates HTML for cancellation notification
func (s *OrderService) generateCancellationNotificationHTML(order *models.Order, user *models.User) string {
	return fmt.Sprintf(`
<div style="font-family: Arial, sans-serif; max-width: 600px; margin: 0 auto;">
	<h2 style="color: #F59E0B;">Order Cancelled</h2>
	<p>Dear %s,</p>
	<p>Your order <strong>%s</strong> has been cancelled.</p>
	<p>If this was not requested by you, please contact our support team immediately.</p>
	<p>You can place a new order anytime from our website.</p>
</div>`, user.FullName(), order.OrderNumber)
}

// generateCancellationNotificationText generates text for cancellation notification
func (s *OrderService) generateCancellationNotificationText(order *models.Order, user *models.User) string {
	return fmt.Sprintf(`Order Cancelled

Dear %s,

Your order %s has been cancelled.

If this was not requested by you, please contact our support team immediately.

You can place a new order anytime from our website.`, user.FullName(), order.OrderNumber)
}

// Admin-specific methods

// GetOrderCount returns the total number of orders
func (s *OrderService) GetOrderCount() (int, error) {
	return s.orderRepo.GetOrderCount()
}

// GetTotalRevenue returns the total revenue from all orders
func (s *OrderService) GetTotalRevenue() (float64, error) {
	return s.orderRepo.GetTotalRevenue()
}
