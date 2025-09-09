package services

import (
	"event-ticketing-platform/internal/config"
	"event-ticketing-platform/internal/models"
	"log"
)

// MockEmailService provides a mock email service that can optionally use Resend
type MockEmailService struct {
	resendService *ResendEmailService
	useResend     bool
}

// NewMockEmailService creates a new mock email service
func NewMockEmailService(resendConfig *config.ResendConfig) *MockEmailService {
	service := &MockEmailService{
		useResend: false,
	}

	// If Resend config is provided and has API key, use Resend
	if resendConfig != nil && resendConfig.APIKey != "" {
		// Convert config types
		resendServiceConfig := ResendConfig{
			APIKey:    resendConfig.APIKey,
			FromEmail: resendConfig.FromEmail,
			FromName:  resendConfig.FromName,
		}
		service.resendService = NewResendEmailService(resendServiceConfig)
		service.useResend = true
		log.Println("Email service: Using Resend API")
	} else {
		log.Println("Email service: Using mock (no Resend API key provided)")
	}

	return service
}

// SendPasswordResetEmail sends a password reset email
func (s *MockEmailService) SendPasswordResetEmail(email, token string) error {
	if s.useResend && s.resendService != nil {
		return s.resendService.SendPasswordResetEmail(email, token)
	}

	// Mock implementation - just log
	log.Printf("Mock Email: Password reset email sent to %s with token %s", email, token)
	return nil
}

// SendWelcomeEmail sends a welcome email to new users
func (s *MockEmailService) SendWelcomeEmail(email, userName string) error {
	if s.useResend && s.resendService != nil {
		return s.resendService.SendWelcomeEmail(email, userName)
	}

	// Mock implementation - just log
	log.Printf("Mock Email: Welcome email sent to %s (%s)", email, userName)
	return nil
}

// SendVerificationEmail sends an email verification link to new users
func (s *MockEmailService) SendVerificationEmail(email, userName, token string) error {
	if s.useResend && s.resendService != nil {
		return s.resendService.SendVerificationEmail(email, userName, token)
	}

	// Mock implementation - just log
	log.Printf("Mock Email: Verification email sent to %s (%s) with token: %s", email, userName, token)
	log.Printf("Mock Email: Verification link: http://localhost:8080/auth/verify?token=%s", token)
	return nil
}

// SendOrderConfirmation sends an order confirmation email
func (s *MockEmailService) SendOrderConfirmation(email, userName, orderNumber, eventTitle, eventDate, totalAmount string) error {
	if s.useResend && s.resendService != nil {
		return s.resendService.SendOrderConfirmation(email, userName, orderNumber, eventTitle, eventDate, totalAmount)
	}

	// Mock implementation - just log
	log.Printf("Mock Email: Order confirmation sent to %s for order %s", email, orderNumber)
	return nil
}

// SendOrderConfirmationWithTickets sends an order confirmation email with ticket attachments
func (s *MockEmailService) SendOrderConfirmationWithTickets(email, userName, subject, htmlContent, textContent string, order *models.Order, tickets []*models.Ticket) error {
	if s.useResend && s.resendService != nil {
		return s.resendService.SendOrderConfirmationWithTickets(email, userName, subject, htmlContent, textContent, order, tickets)
	}

	// Enhanced mock implementation - show detailed email content
	log.Printf("\n"+
		"========================================\n"+
		"📧 ORDER CONFIRMATION EMAIL SENT\n"+
		"========================================\n"+
		"To: %s (%s)\n"+
		"Subject: %s\n"+
		"Order: %s\n"+
		"Tickets: %d\n"+
		"Amount: $%.2f\n"+
		"========================================\n",
		email, userName, subject, order.OrderNumber, len(tickets), float64(order.TotalAmount)/100)

	// Log ticket details
	for i, ticket := range tickets {
		log.Printf("🎫 Ticket %d: %s (Status: %s)\n", i+1, ticket.QRCode, ticket.Status)
	}

	log.Printf("📱 View tickets: http://localhost:8080/dashboard/orders/%d\n", order.ID)
	log.Printf("========================================\n")

	return nil
}

// TestConnection tests the email service connection
func (s *MockEmailService) TestConnection() error {
	if s.useResend && s.resendService != nil {
		return s.resendService.TestConnection()
	}

	// Mock always works
	return nil
}
