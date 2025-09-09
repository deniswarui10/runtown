package services

import (
	"bytes"
	"fmt"
	"html/template"
	"net/smtp"
	"time"

	"event-ticketing-platform/internal/models"
)

// EmailConfig represents email service configuration
type EmailConfig struct {
	SMTPHost     string
	SMTPPort     string
	SMTPUsername string
	SMTPPassword string
	FromEmail    string
	FromName     string
	TemplateDir  string
}

// EmailServiceImpl handles email sending functionality
type EmailServiceImpl struct {
	config    EmailConfig
	templates map[string]*template.Template
}

// NewEmailService creates a new email service
func NewEmailService(config EmailConfig) *EmailServiceImpl {
	service := &EmailServiceImpl{
		config:    config,
		templates: make(map[string]*template.Template),
	}
	
	// Load email templates
	service.loadTemplates()
	
	return service
}

// EmailData represents data for email templates
type EmailData struct {
	To          string
	Subject     string
	RecipientName string
	Data        interface{}
}

// OrderConfirmationData represents data for order confirmation emails
type OrderConfirmationData struct {
	Order       *models.Order
	Event       *models.Event
	Tickets     []*models.Ticket
	TicketTypes map[int]*models.TicketType
	TotalAmount string
	OrderDate   string
}

// PasswordResetData represents data for password reset emails
type PasswordResetData struct {
	UserName  string
	ResetLink string
	ExpiresAt string
}

// TicketDeliveryData represents data for ticket delivery emails
type TicketDeliveryData struct {
	Order       *models.Order
	Event       *models.Event
	Tickets     []*models.Ticket
	TicketTypes map[int]*models.TicketType
	QRCodes     map[int]string // ticket ID to QR code mapping
}

// WelcomeEmailData represents data for welcome emails
type WelcomeEmailData struct {
	UserName    string
	LoginLink   string
	SupportEmail string
}

// SendOrderConfirmation sends an order confirmation email
func (s *EmailServiceImpl) SendOrderConfirmation(to, recipientName string, data *OrderConfirmationData) error {
	emailData := EmailData{
		To:            to,
		Subject:       fmt.Sprintf("Order Confirmation - %s", data.Event.Title),
		RecipientName: recipientName,
		Data:          data,
	}
	
	return s.sendTemplatedEmail("order_confirmation", emailData)
}

// SendPasswordResetEmail sends a password reset email
func (s *EmailServiceImpl) SendPasswordResetEmail(email, token string) error {
	// This would typically get user info from database
	// For now, we'll use a simple implementation
	resetLink := fmt.Sprintf("https://yourapp.com/reset-password?token=%s", token)
	
	data := &PasswordResetData{
		UserName:  email, // In real implementation, get actual name
		ResetLink: resetLink,
		ExpiresAt: time.Now().Add(1 * time.Hour).Format("3:04 PM MST"),
	}
	
	emailData := EmailData{
		To:            email,
		Subject:       "Password Reset Request",
		RecipientName: email,
		Data:          data,
	}
	
	return s.sendTemplatedEmail("password_reset", emailData)
}

// SendTicketDelivery sends tickets via email
func (s *EmailServiceImpl) SendTicketDelivery(to, recipientName string, data *TicketDeliveryData) error {
	emailData := EmailData{
		To:            to,
		Subject:       fmt.Sprintf("Your Tickets for %s", data.Event.Title),
		RecipientName: recipientName,
		Data:          data,
	}
	
	return s.sendTemplatedEmail("ticket_delivery", emailData)
}

// SendWelcomeEmail sends a welcome email to new users
func (s *EmailServiceImpl) SendWelcomeEmail(to, recipientName string, data *WelcomeEmailData) error {
	emailData := EmailData{
		To:            to,
		Subject:       "Welcome to Event Ticketing Platform!",
		RecipientName: recipientName,
		Data:          data,
	}
	
	return s.sendTemplatedEmail("welcome", emailData)
}

// SendEventReminder sends event reminder emails
func (s *EmailServiceImpl) SendEventReminder(to, recipientName string, event *models.Event, tickets []*models.Ticket) error {
	data := struct {
		Event   *models.Event
		Tickets []*models.Ticket
		EventDate string
	}{
		Event:     event,
		Tickets:   tickets,
		EventDate: event.StartDate.Format("Monday, January 2, 2006 at 3:04 PM"),
	}
	
	emailData := EmailData{
		To:            to,
		Subject:       fmt.Sprintf("Reminder: %s is Tomorrow!", event.Title),
		RecipientName: recipientName,
		Data:          data,
	}
	
	return s.sendTemplatedEmail("event_reminder", emailData)
}

// SendEventCancellation sends event cancellation notification
func (s *EmailServiceImpl) SendEventCancellation(to, recipientName string, event *models.Event, refundInfo string) error {
	data := struct {
		Event      *models.Event
		RefundInfo string
		EventDate  string
	}{
		Event:      event,
		RefundInfo: refundInfo,
		EventDate:  event.StartDate.Format("Monday, January 2, 2006 at 3:04 PM"),
	}
	
	emailData := EmailData{
		To:            to,
		Subject:       fmt.Sprintf("Event Cancelled: %s", event.Title),
		RecipientName: recipientName,
		Data:          data,
	}
	
	return s.sendTemplatedEmail("event_cancellation", emailData)
}

// sendTemplatedEmail sends an email using a template
func (s *EmailServiceImpl) sendTemplatedEmail(templateName string, data EmailData) error {
	tmpl, exists := s.templates[templateName]
	if !exists {
		return fmt.Errorf("template %s not found", templateName)
	}
	
	// Render HTML content
	var htmlBuf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&htmlBuf, "html", data); err != nil {
		return fmt.Errorf("failed to render HTML template: %w", err)
	}
	
	// Render text content
	var textBuf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&textBuf, "text", data); err != nil {
		return fmt.Errorf("failed to render text template: %w", err)
	}
	
	// Create email message
	message := s.createMIMEMessage(data.To, data.Subject, htmlBuf.String(), textBuf.String())
	
	// Send email
	return s.sendEmail(data.To, message)
}

// sendEmail sends an email via SMTP
func (s *EmailServiceImpl) sendEmail(to, message string) error {
	// Set up authentication
	auth := smtp.PlainAuth("", s.config.SMTPUsername, s.config.SMTPPassword, s.config.SMTPHost)
	
	// Send email
	addr := fmt.Sprintf("%s:%s", s.config.SMTPHost, s.config.SMTPPort)
	err := smtp.SendMail(addr, auth, s.config.FromEmail, []string{to}, []byte(message))
	if err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}
	
	return nil
}

// createMIMEMessage creates a MIME email message with both HTML and text parts
func (s *EmailServiceImpl) createMIMEMessage(to, subject, htmlBody, textBody string) string {
	boundary := "boundary123456789"
	
	message := fmt.Sprintf(`From: %s <%s>
To: %s
Subject: %s
MIME-Version: 1.0
Content-Type: multipart/alternative; boundary="%s"

--%s
Content-Type: text/plain; charset=UTF-8
Content-Transfer-Encoding: 7bit

%s

--%s
Content-Type: text/html; charset=UTF-8
Content-Transfer-Encoding: 7bit

%s

--%s--
`, s.config.FromName, s.config.FromEmail, to, subject, boundary, boundary, textBody, boundary, htmlBody, boundary)
	
	return message
}

// loadTemplates loads email templates from the template directory
func (s *EmailServiceImpl) loadTemplates() {
	if s.config.TemplateDir == "" {
		s.loadDefaultTemplates()
		return
	}
	
	// In a real implementation, you would load templates from files
	// For now, we'll use the default templates
	s.loadDefaultTemplates()
}

// loadDefaultTemplates loads default email templates
func (s *EmailServiceImpl) loadDefaultTemplates() {
	// Order confirmation template
	s.templates["order_confirmation"] = template.Must(template.New("order_confirmation").Parse(`
{{define "html"}}
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
        .ticket { border: 1px solid #ddd; margin: 10px 0; padding: 15px; background-color: white; }
        .footer { text-align: center; padding: 20px; color: #666; font-size: 12px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>Order Confirmation</h1>
        </div>
        <div class="content">
            <p>Dear {{.RecipientName}},</p>
            <p>Thank you for your order! Here are your order details:</p>
            
            <h3>Event: {{.Data.Event.Title}}</h3>
            <p><strong>Date:</strong> {{.Data.Event.StartDate.Format "Monday, January 2, 2006 at 3:04 PM"}}</p>
            <p><strong>Location:</strong> {{.Data.Event.Location}}</p>
            
            <h3>Order Details</h3>
            <p><strong>Order Number:</strong> {{.Data.Order.OrderNumber}}</p>
            <p><strong>Order Date:</strong> {{.Data.OrderDate}}</p>
            <p><strong>Total Amount:</strong> {{.Data.TotalAmount}}</p>
            
            <h3>Your Tickets</h3>
            {{range .Data.Tickets}}
            <div class="ticket">
                <p><strong>Ticket ID:</strong> {{.ID}}</p>
                <p><strong>QR Code:</strong> {{.QRCode}}</p>
                <p><strong>Status:</strong> {{.Status}}</p>
            </div>
            {{end}}
            
            <p>Your tickets will be sent to you in a separate email shortly.</p>
            <p>Please bring your tickets (printed or on your mobile device) to the event.</p>
        </div>
        <div class="footer">
            <p>Thank you for using Event Ticketing Platform!</p>
        </div>
    </div>
</body>
</html>
{{end}}

{{define "text"}}
Order Confirmation

Dear {{.RecipientName}},

Thank you for your order! Here are your order details:

Event: {{.Data.Event.Title}}
Date: {{.Data.Event.StartDate.Format "Monday, January 2, 2006 at 3:04 PM"}}
Location: {{.Data.Event.Location}}

Order Details:
Order Number: {{.Data.Order.OrderNumber}}
Order Date: {{.Data.OrderDate}}
Total Amount: {{.Data.TotalAmount}}

Your Tickets:
{{range .Data.Tickets}}
- Ticket ID: {{.ID}}, QR Code: {{.QRCode}}, Status: {{.Status}}
{{end}}

Your tickets will be sent to you in a separate email shortly.
Please bring your tickets (printed or on your mobile device) to the event.

Thank you for using Event Ticketing Platform!
{{end}}
`))

	// Password reset template
	s.templates["password_reset"] = template.Must(template.New("password_reset").Parse(`
{{define "html"}}
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
            <p>Dear {{.Data.UserName}},</p>
            <p>We received a request to reset your password. If you made this request, please click the button below to reset your password:</p>
            
            <a href="{{.Data.ResetLink}}" class="button">Reset Password</a>
            
            <p>This link will expire at {{.Data.ExpiresAt}}.</p>
            <p>If you didn't request a password reset, please ignore this email. Your password will remain unchanged.</p>
            
            <p>For security reasons, please do not share this link with anyone.</p>
        </div>
        <div class="footer">
            <p>Event Ticketing Platform Security Team</p>
        </div>
    </div>
</body>
</html>
{{end}}

{{define "text"}}
Password Reset Request

Dear {{.Data.UserName}},

We received a request to reset your password. If you made this request, please visit the following link to reset your password:

{{.Data.ResetLink}}

This link will expire at {{.Data.ExpiresAt}}.

If you didn't request a password reset, please ignore this email. Your password will remain unchanged.

For security reasons, please do not share this link with anyone.

Event Ticketing Platform Security Team
{{end}}
`))

	// Ticket delivery template
	s.templates["ticket_delivery"] = template.Must(template.New("ticket_delivery").Parse(`
{{define "html"}}
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Your Tickets</title>
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background-color: #059669; color: white; padding: 20px; text-align: center; }
        .content { padding: 20px; background-color: #f9f9f9; }
        .ticket { border: 2px solid #059669; margin: 20px 0; padding: 20px; background-color: white; border-radius: 8px; }
        .qr-code { font-family: monospace; font-size: 14px; background-color: #f0f0f0; padding: 10px; margin: 10px 0; }
        .footer { text-align: center; padding: 20px; color: #666; font-size: 12px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>Your Event Tickets</h1>
        </div>
        <div class="content">
            <p>Dear {{.RecipientName}},</p>
            <p>Here are your tickets for <strong>{{.Data.Event.Title}}</strong>:</p>
            
            <p><strong>Event Date:</strong> {{.Data.Event.StartDate.Format "Monday, January 2, 2006 at 3:04 PM"}}</p>
            <p><strong>Location:</strong> {{.Data.Event.Location}}</p>
            
            {{range .Data.Tickets}}
            <div class="ticket">
                <h3>Ticket #{{.ID}}</h3>
                <p><strong>Order:</strong> {{$.Data.Order.OrderNumber}}</p>
                <p><strong>Status:</strong> {{.Status}}</p>
                <div class="qr-code">
                    <strong>QR Code:</strong> {{.QRCode}}
                </div>
                <p><em>Present this QR code at the event entrance</em></p>
            </div>
            {{end}}
            
            <p><strong>Important Instructions:</strong></p>
            <ul>
                <li>Bring these tickets (printed or on your mobile device) to the event</li>
                <li>Arrive early to avoid queues at the entrance</li>
                <li>Each ticket is valid for one person only</li>
                <li>Tickets cannot be transferred or resold</li>
            </ul>
        </div>
        <div class="footer">
            <p>Enjoy your event! - Event Ticketing Platform</p>
        </div>
    </div>
</body>
</html>
{{end}}

{{define "text"}}
Your Event Tickets

Dear {{.RecipientName}},

Here are your tickets for {{.Data.Event.Title}}:

Event Date: {{.Data.Event.StartDate.Format "Monday, January 2, 2006 at 3:04 PM"}}
Location: {{.Data.Event.Location}}

Your Tickets:
{{range .Data.Tickets}}
Ticket #{{.ID}}
Order: {{$.Data.Order.OrderNumber}}
Status: {{.Status}}
QR Code: {{.QRCode}}
---
{{end}}

Important Instructions:
- Bring these tickets (printed or on your mobile device) to the event
- Arrive early to avoid queues at the entrance
- Each ticket is valid for one person only
- Tickets cannot be transferred or resold

Enjoy your event!
Event Ticketing Platform
{{end}}
`))

	// Welcome email template
	s.templates["welcome"] = template.Must(template.New("welcome").Parse(`
{{define "html"}}
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
            <h1>Welcome to Event Ticketing Platform!</h1>
        </div>
        <div class="content">
            <p>Dear {{.Data.UserName}},</p>
            <p>Welcome to Event Ticketing Platform! We're excited to have you join our community.</p>
            
            <p>With your new account, you can:</p>
            <ul>
                <li>Browse and discover amazing events</li>
                <li>Purchase tickets securely</li>
                <li>Manage your orders and tickets</li>
                <li>Get notified about upcoming events</li>
            </ul>
            
            <a href="{{.Data.LoginLink}}" class="button">Start Exploring Events</a>
            
            <p>If you have any questions, feel free to contact our support team at {{.Data.SupportEmail}}.</p>
            
            <p>Happy event hunting!</p>
        </div>
        <div class="footer">
            <p>The Event Ticketing Platform Team</p>
        </div>
    </div>
</body>
</html>
{{end}}

{{define "text"}}
Welcome to Event Ticketing Platform!

Dear {{.Data.UserName}},

Welcome to Event Ticketing Platform! We're excited to have you join our community.

With your new account, you can:
- Browse and discover amazing events
- Purchase tickets securely
- Manage your orders and tickets
- Get notified about upcoming events

Start exploring events: {{.Data.LoginLink}}

If you have any questions, feel free to contact our support team at {{.Data.SupportEmail}}.

Happy event hunting!

The Event Ticketing Platform Team
{{end}}
`))

	// Event reminder template
	s.templates["event_reminder"] = template.Must(template.New("event_reminder").Parse(`
{{define "html"}}
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Event Reminder</title>
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background-color: #F59E0B; color: white; padding: 20px; text-align: center; }
        .content { padding: 20px; background-color: #f9f9f9; }
        .highlight { background-color: #FEF3C7; padding: 15px; border-left: 4px solid #F59E0B; margin: 20px 0; }
        .footer { text-align: center; padding: 20px; color: #666; font-size: 12px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>Event Reminder</h1>
        </div>
        <div class="content">
            <p>Dear {{.RecipientName}},</p>
            
            <div class="highlight">
                <h2>{{.Data.Event.Title}} is Tomorrow!</h2>
                <p><strong>Date:</strong> {{.Data.EventDate}}</p>
                <p><strong>Location:</strong> {{.Data.Event.Location}}</p>
            </div>
            
            <p>Don't forget about your upcoming event! Here's what you need to know:</p>
            
            <h3>Your Tickets:</h3>
            <ul>
            {{range .Data.Tickets}}
                <li>Ticket #{{.ID}} - {{.Status}}</li>
            {{end}}
            </ul>
            
            <h3>Reminders:</h3>
            <ul>
                <li>Bring your tickets (printed or on mobile)</li>
                <li>Arrive early to avoid entrance queues</li>
                <li>Check the weather and dress appropriately</li>
                <li>Bring a valid ID if required</li>
            </ul>
            
            <p>We hope you have a fantastic time at the event!</p>
        </div>
        <div class="footer">
            <p>Event Ticketing Platform</p>
        </div>
    </div>
</body>
</html>
{{end}}

{{define "text"}}
Event Reminder

Dear {{.RecipientName}},

{{.Data.Event.Title}} is Tomorrow!

Date: {{.Data.EventDate}}
Location: {{.Data.Event.Location}}

Don't forget about your upcoming event! Here's what you need to know:

Your Tickets:
{{range .Data.Tickets}}
- Ticket #{{.ID}} - {{.Status}}
{{end}}

Reminders:
- Bring your tickets (printed or on mobile)
- Arrive early to avoid entrance queues
- Check the weather and dress appropriately
- Bring a valid ID if required

We hope you have a fantastic time at the event!

Event Ticketing Platform
{{end}}
`))

	// Event cancellation template
	s.templates["event_cancellation"] = template.Must(template.New("event_cancellation").Parse(`
{{define "html"}}
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Event Cancelled</title>
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background-color: #DC2626; color: white; padding: 20px; text-align: center; }
        .content { padding: 20px; background-color: #f9f9f9; }
        .alert { background-color: #FEE2E2; border: 1px solid #FECACA; padding: 15px; border-radius: 4px; margin: 20px 0; }
        .footer { text-align: center; padding: 20px; color: #666; font-size: 12px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>Event Cancellation Notice</h1>
        </div>
        <div class="content">
            <p>Dear {{.RecipientName}},</p>
            
            <div class="alert">
                <h2>{{.Data.Event.Title}} has been cancelled</h2>
                <p><strong>Originally scheduled for:</strong> {{.Data.EventDate}}</p>
                <p><strong>Location:</strong> {{.Data.Event.Location}}</p>
            </div>
            
            <p>We regret to inform you that the above event has been cancelled by the organizer.</p>
            
            <h3>Refund Information:</h3>
            <p>{{.Data.RefundInfo}}</p>
            
            <p>We sincerely apologize for any inconvenience this may cause. If you have any questions about your refund or need assistance, please don't hesitate to contact our support team.</p>
            
            <p>Thank you for your understanding.</p>
        </div>
        <div class="footer">
            <p>Event Ticketing Platform Support Team</p>
        </div>
    </div>
</body>
</html>
{{end}}

{{define "text"}}
Event Cancellation Notice

Dear {{.RecipientName}},

{{.Data.Event.Title}} has been cancelled

Originally scheduled for: {{.Data.EventDate}}
Location: {{.Data.Event.Location}}

We regret to inform you that the above event has been cancelled by the organizer.

Refund Information:
{{.Data.RefundInfo}}

We sincerely apologize for any inconvenience this may cause. If you have any questions about your refund or need assistance, please don't hesitate to contact our support team.

Thank you for your understanding.

Event Ticketing Platform Support Team
{{end}}
`))
}

// TestConnection tests the email service connection
func (s *EmailServiceImpl) TestConnection() error {
	// Try to connect to SMTP server
	addr := fmt.Sprintf("%s:%s", s.config.SMTPHost, s.config.SMTPPort)
	conn, err := smtp.Dial(addr)
	if err != nil {
		return fmt.Errorf("failed to connect to SMTP server: %w", err)
	}
	defer conn.Close()
	
	// Test authentication
	auth := smtp.PlainAuth("", s.config.SMTPUsername, s.config.SMTPPassword, s.config.SMTPHost)
	if err := conn.Auth(auth); err != nil {
		return fmt.Errorf("SMTP authentication failed: %w", err)
	}
	
	return nil
}

// GetTemplateNames returns the names of all loaded templates
func (s *EmailServiceImpl) GetTemplateNames() []string {
	var names []string
	for name := range s.templates {
		names = append(names, name)
	}
	return names
}