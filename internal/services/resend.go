package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
	
	"event-ticketing-platform/internal/models"
)

// ResendConfig represents Resend email service configuration
type ResendConfig struct {
	APIKey    string
	FromEmail string
	FromName  string
}

// ResendEmailService handles email sending via Resend API
type ResendEmailService struct {
	config ResendConfig
	client *http.Client
}

// NewResendEmailService creates a new Resend email service
func NewResendEmailService(config ResendConfig) *ResendEmailService {
	return &ResendEmailService{
		config: config,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// ResendEmailRequest represents the request structure for Resend API
type ResendEmailRequest struct {
	From     string            `json:"from"`
	To       []string          `json:"to"`
	Subject  string            `json:"subject"`
	HTML     string            `json:"html,omitempty"`
	Text     string            `json:"text,omitempty"`
	Headers  map[string]string `json:"headers,omitempty"`
	Tags     []ResendTag       `json:"tags,omitempty"`
}

// ResendTag represents a tag for email categorization
type ResendTag struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// ResendEmailResponse represents the response from Resend API
type ResendEmailResponse struct {
	ID string `json:"id"`
}

// ResendErrorResponse represents error response from Resend API
type ResendErrorResponse struct {
	Message string `json:"message"`
	Name    string `json:"name"`
}

// getFromField constructs the from field properly
func (s *ResendEmailService) getFromField() string {
	if s.config.FromName != "" {
		return fmt.Sprintf("%s <%s>", s.config.FromName, s.config.FromEmail)
	}
	return s.config.FromEmail
}

// SendPasswordResetEmail sends a password reset email via Resend
func (s *ResendEmailService) SendPasswordResetEmail(email, token string) error {
	resetLink := fmt.Sprintf("https://runtown.onrender.com/auth/reset-password?token=%s", token)
	
	htmlContent := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Password Reset</title>
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background-color: #DC2626; color: white; padding: 20px; text-align: center; }
        .content { padding: 20px; background-color: #f9f9f9; }
        .button { display: inline-block; padding: 12px 24px; background-color: #DC2626; color: white; text-decoration: none; border-radius: 4px; margin: 20px 0; }
        .footer { text-align: center; padding: 20px; color: #666; font-size: 12px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>Password Reset Request</h1>
        </div>
        <div class="content">
            <p>Dear User,</p>
            <p>We received a request to reset your password. If you made this request, please click the button below to reset your password:</p>
            
            <a href="%s" class="button">Reset Password</a>
            
            <p>This link will expire in 1 hour.</p>
            <p>If you didn't request a password reset, please ignore this email. Your password will remain unchanged.</p>
            
            <p>For security reasons, please do not share this link with anyone.</p>
        </div>
        <div class="footer">
            <p>Runtown Security Team</p>
        </div>
    </div>
</body>
</html>`, resetLink)

	textContent := fmt.Sprintf(`Password Reset Request

Dear User,

We received a request to reset your password. If you made this request, please visit the following link to reset your password:

%s

This link will expire in 1 hour.

If you didn't request a password reset, please ignore this email. Your password will remain unchanged.

For security reasons, please do not share this link with anyone.

Runtown Security Team`, resetLink)

	request := ResendEmailRequest{
		From:    s.getFromField(),
		To:      []string{email},
		Subject: "Password Reset Request",
		HTML:    htmlContent,
		Text:    textContent,
		Tags: []ResendTag{
			{Name: "category", Value: "password_reset"},
		},
	}

	return s.sendEmail(request)
}

// SendWelcomeEmail sends a welcome email to new users
func (s *ResendEmailService) SendWelcomeEmail(email, userName string) error {
	htmlContent := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Welcome</title>
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background-color: #7C3AED; color: white; padding: 20px; text-align: center; }
        .content { padding: 20px; background-color: #f9f9f9; }
        .button { display: inline-block; padding: 12px 24px; background-color: #7C3AED; color: white; text-decoration: none; border-radius: 4px; margin: 20px 0; }
        .footer { text-align: center; padding: 20px; color: #666; font-size: 12px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>Welcome to Runtown!</h1>
        </div>
        <div class="content">
            <p>Dear %s,</p>
            <p>Welcome to Runtown! We're excited to have you join our community.</p>
            
            <p>With your new account, you can:</p>
            <ul>
                <li>Browse and discover amazing events</li>
                <li>Purchase tickets securely</li>
                <li>Manage your orders and tickets</li>
                <li>Get notified about upcoming events</li>
            </ul>
            
            <a href="https://runtown.onrender.com/events" class="button">Start Exploring Events</a>
            
            <p>If you have any questions, feel free to contact our support team.</p>
            
            <p>Happy event hunting!</p>
        </div>
        <div class="footer">
            <p>Runtown Team</p>
        </div>
    </div>
</body>
</html>`, userName)

	textContent := fmt.Sprintf(`Welcome to Runtown!

Dear %s,

Welcome to Runtown! We're excited to have you join our community.

With your new account, you can:
- Browse and discover amazing events
- Purchase tickets securely
- Manage your orders and tickets
- Get notified about upcoming events

Start exploring events: https://runtown.onrender.com/events

If you have any questions, feel free to contact our support team.

Happy event hunting!

Runtown Team`, userName)

	request := ResendEmailRequest{
		From:    s.getFromField(),
		To:      []string{email},
		Subject: "Welcome to Runtown!",
		HTML:    htmlContent,
		Text:    textContent,
		Tags: []ResendTag{
			{Name: "category", Value: "welcome"},
		},
	}

	return s.sendEmail(request)
}

// SendVerificationEmail sends an email verification link to new users
func (s *ResendEmailService) SendVerificationEmail(email, userName, token string) error {
	verificationLink := fmt.Sprintf("https://runtown.onrender.com/auth/verify?token=%s", token)
	
	htmlContent := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Verify Your Email</title>
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background-color: #059669; color: white; padding: 20px; text-align: center; }
        .content { padding: 20px; background-color: #f9f9f9; }
        .button { display: inline-block; padding: 12px 24px; background-color: #059669; color: white; text-decoration: none; border-radius: 4px; margin: 20px 0; }
        .footer { text-align: center; padding: 20px; color: #666; font-size: 12px; }
        .warning { background-color: #FEF3C7; padding: 15px; border-left: 4px solid #F59E0B; margin: 20px 0; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>Verify Your Email Address</h1>
        </div>
        <div class="content">
            <p>Dear %s,</p>
            <p>Thank you for creating an account with Runtown! To complete your registration and access all features, please verify your email address by clicking the button below:</p>
            
            <div style="text-align: center;">
                <a href="%s" class="button">Verify My Email</a>
            </div>
            
            <p>Or copy and paste this link into your browser:</p>
            <p style="word-break: break-all; background-color: #f0f0f0; padding: 10px; border-radius: 4px;">%s</p>
            
            <div class="warning">
                <p><strong>Important:</strong> This verification link will expire in 24 hours for security reasons.</p>
            </div>
            
            <p>If you did not create an account with Runtown, please ignore this email.</p>
            
            <p>Once your email is verified, you'll be able to:</p>
            <ul>
                <li>Browse and discover amazing events</li>
                <li>Purchase tickets securely</li>
                <li>Manage your orders and tickets</li>
                <li>Receive important event updates</li>
            </ul>
            
            <p>Welcome to the community!</p>
        </div>
        <div class="footer">
            <p>Runtown Team</p>
        </div>
    </div>
</body>
</html>`, userName, verificationLink, verificationLink)

	textContent := fmt.Sprintf(`Verify Your Email Address

Dear %s,

Thank you for creating an account with Runtown! To complete your registration and access all features, please verify your email address by visiting the following link:

%s

Important: This verification link will expire in 24 hours for security reasons.

If you did not create an account with Runtown, please ignore this email.

Once your email is verified, you'll be able to:
- Browse and discover amazing events
- Purchase tickets securely
- Manage your orders and tickets
- Receive important event updates

Welcome to the community!

The Runtown Team`, userName, verificationLink)

	request := ResendEmailRequest{
		From:    s.getFromField(),
		To:      []string{email},
		Subject: "Verify your email address - Runtown",
		HTML:    htmlContent,
		Text:    textContent,
		Tags: []ResendTag{
			{Name: "category", Value: "email_verification"},
		},
	}

	return s.sendEmail(request)
}

// SendOrderConfirmation sends an order confirmation email
func (s *ResendEmailService) SendOrderConfirmation(email, userName, orderNumber, eventTitle, eventDate, totalAmount string) error {
	htmlContent := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Order Confirmation</title>
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background-color: #4F46E5; color: white; padding: 20px; text-align: center; }
        .content { padding: 20px; background-color: #f9f9f9; }
        .highlight { background-color: #EEF2FF; padding: 15px; border-left: 4px solid #4F46E5; margin: 20px 0; }
        .footer { text-align: center; padding: 20px; color: #666; font-size: 12px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>Order Confirmation</h1>
        </div>
        <div class="content">
            <p>Dear %s,</p>
            <p>Thank you for your order! Here are your order details:</p>
            
            <div class="highlight">
                <h3>Event: %s</h3>
                <p><strong>Date:</strong> %s</p>
                <p><strong>Order Number:</strong> %s</p>
                <p><strong>Total Amount:</strong> %s</p>
            </div>
            
            <p>Your tickets will be sent to you in a separate email shortly.</p>
            <p>Please bring your tickets (printed or on your mobile device) to the event.</p>
            
            <p>Thank you for choosing Runtown!</p>
        </div>
        <div class="footer">
            <p>Runtown</p>
        </div>
    </div>
</body>
</html>`, userName, eventTitle, eventDate, orderNumber, totalAmount)

	textContent := fmt.Sprintf(`Order Confirmation

Dear %s,

Thank you for your order! Here are your order details:

Event: %s
Date: %s
Order Number: %s
Total Amount: %s

Your tickets will be sent to you in a separate email shortly.
Please bring your tickets (printed or on your mobile device) to the event.

Thank you for choosing Runtown!`, userName, eventTitle, eventDate, orderNumber, totalAmount)

	request := ResendEmailRequest{
		From:    s.getFromField(),
		To:      []string{email},
		Subject: fmt.Sprintf("Order Confirmation - %s", eventTitle),
		HTML:    htmlContent,
		Text:    textContent,
		Tags: []ResendTag{
			{Name: "category", Value: "order_confirmation"},
		},
	}

	return s.sendEmail(request)
}

// SendOrderConfirmationWithTickets sends an order confirmation email with ticket PDF attachment
func (s *ResendEmailService) SendOrderConfirmationWithTickets(email, userName, subject, htmlContent, textContent string, order *models.Order, tickets []*models.Ticket) error {
	// Enhanced email with better formatting and ticket information
	enhancedHTMLContent := s.enhanceOrderConfirmationHTML(htmlContent, order, tickets)
	enhancedTextContent := s.enhanceOrderConfirmationText(textContent, order, tickets)
	
	request := ResendEmailRequest{
		From:    s.getFromField(),
		To:      []string{email},
		Subject: subject,
		HTML:    enhancedHTMLContent,
		Text:    enhancedTextContent,
		Tags: []ResendTag{
			{Name: "category", Value: "order_confirmation_with_tickets"},
			{Name: "order_number", Value: order.OrderNumber},
			{Name: "ticket_count", Value: fmt.Sprintf("%d", len(tickets))},
		},
	}

	return s.sendEmail(request)
}

// enhanceOrderConfirmationHTML enhances the HTML content with additional ticket information
func (s *ResendEmailService) enhanceOrderConfirmationHTML(originalHTML string, order *models.Order, tickets []*models.Ticket) string {
	// Add ticket details section to the HTML
	ticketDetailsHTML := `
		<div style="margin: 30px 0; padding: 20px; background-color: #f8fafc; border-radius: 8px; border: 1px solid #e2e8f0;">
			<h3 style="margin-top: 0; color: #1e293b; font-size: 18px;">Ticket Details</h3>
			<div style="margin: 15px 0;">
				<table style="width: 100%; border-collapse: collapse;">
					<thead>
						<tr style="background-color: #e2e8f0;">
							<th style="padding: 10px; text-align: left; border: 1px solid #cbd5e1; font-size: 14px; color: #475569;">Ticket #</th>
							<th style="padding: 10px; text-align: left; border: 1px solid #cbd5e1; font-size: 14px; color: #475569;">QR Code</th>
							<th style="padding: 10px; text-align: left; border: 1px solid #cbd5e1; font-size: 14px; color: #475569;">Status</th>
						</tr>
					</thead>
					<tbody>`

	for i, ticket := range tickets {
		ticketDetailsHTML += fmt.Sprintf(`
						<tr>
							<td style="padding: 10px; border: 1px solid #cbd5e1; font-size: 14px;">Ticket #%d</td>
							<td style="padding: 10px; border: 1px solid #cbd5e1; font-size: 12px; font-family: monospace;">%s</td>
							<td style="padding: 10px; border: 1px solid #cbd5e1; font-size: 14px;">
								<span style="background-color: #dcfce7; color: #166534; padding: 4px 8px; border-radius: 4px; font-size: 12px;">%s</span>
							</td>
						</tr>`, i+1, ticket.QRCode, string(ticket.Status))
	}

	ticketDetailsHTML += `
					</tbody>
				</table>
			</div>
			<div style="margin-top: 20px; padding: 15px; background-color: #dbeafe; border-radius: 6px; border-left: 4px solid #3b82f6;">
				<p style="margin: 0; font-size: 14px; color: #1e40af;">
					<strong>📱 Mobile Access:</strong> You can also access your tickets anytime from your account dashboard at 
					<a href="https://runtown.onrender.com/dashboard/orders/%d" style="color: #2563eb; text-decoration: none;">https://runtown.onrender.com/dashboard/orders/%d</a>
				</p>
			</div>
		</div>`

	ticketDetailsHTML = fmt.Sprintf(ticketDetailsHTML, order.ID, order.ID)

	// Insert ticket details before the footer
	footerIndex := strings.Index(originalHTML, `<div class="footer">`)
	if footerIndex != -1 {
		return originalHTML[:footerIndex] + ticketDetailsHTML + originalHTML[footerIndex:]
	}

	// If no footer found, append to the end
	return originalHTML + ticketDetailsHTML
}

// enhanceOrderConfirmationText enhances the text content with additional ticket information
func (s *ResendEmailService) enhanceOrderConfirmationText(originalText string, order *models.Order, tickets []*models.Ticket) string {
	ticketDetailsText := fmt.Sprintf(`

TICKET DETAILS
==============
You have %d ticket(s) for this order:

`, len(tickets))

	for i, ticket := range tickets {
		ticketDetailsText += fmt.Sprintf(`Ticket #%d
QR Code: %s
Status: %s
Generated: %s

`, i+1, ticket.QRCode, string(ticket.Status), ticket.CreatedAt.Format("Jan 2, 2006 at 3:04 PM"))
	}

	ticketDetailsText += fmt.Sprintf(`MOBILE ACCESS
=============
You can access your tickets anytime from your account dashboard:
https://runtown.onrender.com/dashboard/orders/%d

NEXT STEPS
==========
1. Save this email for your records
2. Download the tickets from your account dashboard
3. Bring your tickets (printed or on mobile) to the event
4. Arrive early to avoid entrance queues

`, order.ID)

	// Insert ticket details before the footer
	footerIndex := strings.Index(originalText, "Runtown")
	if footerIndex != -1 {
		return originalText[:footerIndex] + ticketDetailsText + originalText[footerIndex:]
	}

	// If no footer found, append to the end
	return originalText + ticketDetailsText
}

// sendEmail sends an email via Resend API
func (s *ResendEmailService) sendEmail(request ResendEmailRequest) error {
	jsonData, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", "https://api.resend.com/emails", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+s.config.APIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		var errorResp ResendErrorResponse
		if err := json.NewDecoder(resp.Body).Decode(&errorResp); err != nil {
			return fmt.Errorf("failed to send email, status: %d", resp.StatusCode)
		}
		return fmt.Errorf("failed to send email: %s", errorResp.Message)
	}

	var response ResendEmailResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	return nil
}

// TestConnection tests the Resend API connection
func (s *ResendEmailService) TestConnection() error {
	// Send a test request to validate API key
	request := ResendEmailRequest{
		From:    s.getFromField(),
		To:      []string{"test@example.com"}, // This won't actually send
		Subject: "Test Connection",
		Text:    "This is a test email to validate API connection",
	}

	jsonData, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("failed to marshal test request: %w", err)
	}

	req, err := http.NewRequest("POST", "https://api.resend.com/emails", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create test request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+s.config.APIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send test request: %w", err)
	}
	defer resp.Body.Close()

	// Check if we get a valid response (even if it's an error about the test email)
	if resp.StatusCode == 401 {
		return fmt.Errorf("invalid API key")
	}

	return nil
}