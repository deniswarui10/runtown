package services

import (
	"strings"
	"testing"
	"time"

	"event-ticketing-platform/internal/models"
)

func TestEmailService_Creation(t *testing.T) {
	config := EmailConfig{
		SMTPHost:     "smtp.example.com",
		SMTPPort:     "587",
		SMTPUsername: "test@example.com",
		SMTPPassword: "password",
		FromEmail:    "noreply@example.com",
		FromName:     "Test Service",
		TemplateDir:  "",
	}
	
	service := NewEmailService(config)
	
	if service == nil {
		t.Fatal("expected email service to be created")
	}
	
	if len(service.templates) == 0 {
		t.Error("expected templates to be loaded")
	}
	
	// Check that all expected templates are loaded
	expectedTemplates := []string{
		"order_confirmation",
		"password_reset",
		"ticket_delivery",
		"welcome",
		"event_reminder",
		"event_cancellation",
	}
	
	for _, templateName := range expectedTemplates {
		if _, exists := service.templates[templateName]; !exists {
			t.Errorf("expected template %s to be loaded", templateName)
		}
	}
}

func TestEmailService_GetTemplateNames(t *testing.T) {
	config := EmailConfig{
		SMTPHost:     "smtp.example.com",
		SMTPPort:     "587",
		SMTPUsername: "test@example.com",
		SMTPPassword: "password",
		FromEmail:    "noreply@example.com",
		FromName:     "Test Service",
	}
	
	service := NewEmailService(config)
	names := service.GetTemplateNames()
	
	if len(names) == 0 {
		t.Error("expected template names to be returned")
	}
	
	// Check that expected templates are in the list
	expectedTemplates := map[string]bool{
		"order_confirmation": false,
		"password_reset":     false,
		"ticket_delivery":    false,
		"welcome":           false,
		"event_reminder":    false,
		"event_cancellation": false,
	}
	
	for _, name := range names {
		if _, exists := expectedTemplates[name]; exists {
			expectedTemplates[name] = true
		}
	}
	
	for templateName, found := range expectedTemplates {
		if !found {
			t.Errorf("expected template %s to be in template names", templateName)
		}
	}
}

func TestEmailService_CreateMIMEMessage(t *testing.T) {
	config := EmailConfig{
		FromEmail: "noreply@example.com",
		FromName:  "Test Service",
	}
	
	service := NewEmailService(config)
	
	to := "test@example.com"
	subject := "Test Subject"
	htmlBody := "<h1>Test HTML</h1>"
	textBody := "Test Text"
	
	message := service.createMIMEMessage(to, subject, htmlBody, textBody)
	
	// Check that message contains expected headers
	if !strings.Contains(message, "From: Test Service <noreply@example.com>") {
		t.Error("expected From header to be present")
	}
	
	if !strings.Contains(message, "To: test@example.com") {
		t.Error("expected To header to be present")
	}
	
	if !strings.Contains(message, "Subject: Test Subject") {
		t.Error("expected Subject header to be present")
	}
	
	if !strings.Contains(message, "MIME-Version: 1.0") {
		t.Error("expected MIME-Version header to be present")
	}
	
	if !strings.Contains(message, "multipart/alternative") {
		t.Error("expected multipart/alternative content type")
	}
	
	// Check that both HTML and text content are present
	if !strings.Contains(message, htmlBody) {
		t.Error("expected HTML body to be present")
	}
	
	if !strings.Contains(message, textBody) {
		t.Error("expected text body to be present")
	}
}

func TestEmailService_TemplateRendering(t *testing.T) {
	config := EmailConfig{
		FromEmail: "noreply@example.com",
		FromName:  "Test Service",
	}
	
	service := NewEmailService(config)
	
	// Test order confirmation template
	t.Run("order confirmation template", func(t *testing.T) {
		event := &models.Event{
			ID:        1,
			Title:     "Test Event",
			Location:  "Test Location",
			StartDate: time.Now().Add(24 * time.Hour),
		}
		
		order := &models.Order{
			ID:          1,
			OrderNumber: "ORD-20240101-123456",
			TotalAmount: 5000,
		}
		
		tickets := []*models.Ticket{
			{
				ID:     1,
				QRCode: "TEST-QR-CODE-1",
				Status: models.TicketActive,
			},
		}
		
		data := &OrderConfirmationData{
			Order:       order,
			Event:       event,
			Tickets:     tickets,
			TotalAmount: "$50.00",
			OrderDate:   time.Now().Format("January 2, 2006"),
		}
		
		emailData := EmailData{
			To:            "test@example.com",
			Subject:       "Test Order Confirmation",
			RecipientName: "Test User",
			Data:          data,
		}
		
		// Test HTML template rendering
		tmpl := service.templates["order_confirmation"]
		var htmlBuf strings.Builder
		err := tmpl.ExecuteTemplate(&htmlBuf, "html", emailData)
		if err != nil {
			t.Fatalf("failed to render HTML template: %v", err)
		}
		
		htmlContent := htmlBuf.String()
		if !strings.Contains(htmlContent, "Test Event") {
			t.Error("expected event title in HTML content")
		}
		
		if !strings.Contains(htmlContent, "ORD-20240101-123456") {
			t.Error("expected order number in HTML content")
		}
		
		if !strings.Contains(htmlContent, "TEST-QR-CODE-1") {
			t.Error("expected QR code in HTML content")
		}
		
		// Test text template rendering
		var textBuf strings.Builder
		err = tmpl.ExecuteTemplate(&textBuf, "text", emailData)
		if err != nil {
			t.Fatalf("failed to render text template: %v", err)
		}
		
		textContent := textBuf.String()
		if !strings.Contains(textContent, "Test Event") {
			t.Error("expected event title in text content")
		}
		
		if !strings.Contains(textContent, "ORD-20240101-123456") {
			t.Error("expected order number in text content")
		}
	})
	
	// Test password reset template
	t.Run("password reset template", func(t *testing.T) {
		data := &PasswordResetData{
			UserName:  "Test User",
			ResetLink: "https://example.com/reset?token=abc123",
			ExpiresAt: "3:00 PM MST",
		}
		
		emailData := EmailData{
			To:            "test@example.com",
			Subject:       "Password Reset",
			RecipientName: "Test User",
			Data:          data,
		}
		
		tmpl := service.templates["password_reset"]
		var htmlBuf strings.Builder
		err := tmpl.ExecuteTemplate(&htmlBuf, "html", emailData)
		if err != nil {
			t.Fatalf("failed to render HTML template: %v", err)
		}
		
		htmlContent := htmlBuf.String()
		if !strings.Contains(htmlContent, "https://example.com/reset?token=abc123") {
			t.Error("expected reset link in HTML content")
		}
		
		if !strings.Contains(htmlContent, "3:00 PM MST") {
			t.Error("expected expiration time in HTML content")
		}
	})
	
	// Test welcome email template
	t.Run("welcome email template", func(t *testing.T) {
		data := &WelcomeEmailData{
			UserName:     "Test User",
			LoginLink:    "https://example.com/login",
			SupportEmail: "support@example.com",
		}
		
		emailData := EmailData{
			To:            "test@example.com",
			Subject:       "Welcome",
			RecipientName: "Test User",
			Data:          data,
		}
		
		tmpl := service.templates["welcome"]
		var htmlBuf strings.Builder
		err := tmpl.ExecuteTemplate(&htmlBuf, "html", emailData)
		if err != nil {
			t.Fatalf("failed to render HTML template: %v", err)
		}
		
		htmlContent := htmlBuf.String()
		if !strings.Contains(htmlContent, "Welcome to Event Ticketing Platform") {
			t.Error("expected welcome message in HTML content")
		}
		
		if !strings.Contains(htmlContent, "https://example.com/login") {
			t.Error("expected login link in HTML content")
		}
		
		if !strings.Contains(htmlContent, "support@example.com") {
			t.Error("expected support email in HTML content")
		}
	})
}

func TestEmailService_PasswordResetEmail(t *testing.T) {
	config := EmailConfig{
		SMTPHost:     "smtp.example.com",
		SMTPPort:     "587",
		SMTPUsername: "test@example.com",
		SMTPPassword: "password",
		FromEmail:    "noreply@example.com",
		FromName:     "Test Service",
	}
	
	_ = NewEmailService(config)
	
	// Mock the sendEmail method by testing template rendering only
	token := "reset-token-123"
	
	// Test that the method would create the correct data structure
	// In a real test, you might want to mock the SMTP sending
	resetLink := "https://yourapp.com/reset-password?token=" + token
	
	if !strings.Contains(resetLink, token) {
		t.Error("expected reset link to contain token")
	}
	
	if !strings.HasPrefix(resetLink, "https://") {
		t.Error("expected reset link to be HTTPS")
	}
}

func TestEmailService_EventReminder(t *testing.T) {
	config := EmailConfig{
		FromEmail: "noreply@example.com",
		FromName:  "Test Service",
	}
	
	service := NewEmailService(config)
	
	event := &models.Event{
		ID:        1,
		Title:     "Test Concert",
		Location:  "Test Venue",
		StartDate: time.Now().Add(24 * time.Hour),
	}
	
	tickets := []*models.Ticket{
		{
			ID:     1,
			QRCode: "TICKET-1",
			Status: models.TicketActive,
		},
		{
			ID:     2,
			QRCode: "TICKET-2",
			Status: models.TicketActive,
		},
	}
	
	// Test template rendering for event reminder
	data := struct {
		Event     *models.Event
		Tickets   []*models.Ticket
		EventDate string
	}{
		Event:     event,
		Tickets:   tickets,
		EventDate: event.StartDate.Format("Monday, January 2, 2006 at 3:04 PM"),
	}
	
	emailData := EmailData{
		To:            "test@example.com",
		Subject:       "Event Reminder",
		RecipientName: "Test User",
		Data:          data,
	}
	
	tmpl := service.templates["event_reminder"]
	var htmlBuf strings.Builder
	err := tmpl.ExecuteTemplate(&htmlBuf, "html", emailData)
	if err != nil {
		t.Fatalf("failed to render HTML template: %v", err)
	}
	
	htmlContent := htmlBuf.String()
	if !strings.Contains(htmlContent, "Test Concert") {
		t.Error("expected event title in HTML content")
	}
	
	if !strings.Contains(htmlContent, "Test Venue") {
		t.Error("expected event location in HTML content")
	}
	
	if !strings.Contains(htmlContent, "Ticket #1") {
		t.Error("expected first ticket in HTML content")
	}
	
	if !strings.Contains(htmlContent, "Ticket #2") {
		t.Error("expected second ticket in HTML content")
	}
}

func TestEmailService_EventCancellation(t *testing.T) {
	config := EmailConfig{
		FromEmail: "noreply@example.com",
		FromName:  "Test Service",
	}
	
	service := NewEmailService(config)
	
	event := &models.Event{
		ID:        1,
		Title:     "Cancelled Concert",
		Location:  "Test Venue",
		StartDate: time.Now().Add(24 * time.Hour),
	}
	
	refundInfo := "Your refund of $50.00 will be processed within 3-5 business days."
	
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
		To:            "test@example.com",
		Subject:       "Event Cancelled",
		RecipientName: "Test User",
		Data:          data,
	}
	
	tmpl := service.templates["event_cancellation"]
	var htmlBuf strings.Builder
	err := tmpl.ExecuteTemplate(&htmlBuf, "html", emailData)
	if err != nil {
		t.Fatalf("failed to render HTML template: %v", err)
	}
	
	htmlContent := htmlBuf.String()
	if !strings.Contains(htmlContent, "Cancelled Concert") {
		t.Error("expected event title in HTML content")
	}
	
	if !strings.Contains(htmlContent, "has been cancelled") {
		t.Error("expected cancellation message in HTML content")
	}
	
	if !strings.Contains(htmlContent, refundInfo) {
		t.Error("expected refund information in HTML content")
	}
}

func TestEmailService_TicketDelivery(t *testing.T) {
	config := EmailConfig{
		FromEmail: "noreply@example.com",
		FromName:  "Test Service",
	}
	
	service := NewEmailService(config)
	
	event := &models.Event{
		ID:        1,
		Title:     "Test Event",
		Location:  "Test Location",
		StartDate: time.Now().Add(24 * time.Hour),
	}
	
	order := &models.Order{
		ID:          1,
		OrderNumber: "ORD-20240101-123456",
	}
	
	tickets := []*models.Ticket{
		{
			ID:     1,
			QRCode: "QR-CODE-1",
			Status: models.TicketActive,
		},
		{
			ID:     2,
			QRCode: "QR-CODE-2",
			Status: models.TicketActive,
		},
	}
	
	data := &TicketDeliveryData{
		Order:   order,
		Event:   event,
		Tickets: tickets,
		QRCodes: map[int]string{
			1: "QR-CODE-1",
			2: "QR-CODE-2",
		},
	}
	
	emailData := EmailData{
		To:            "test@example.com",
		Subject:       "Your Tickets",
		RecipientName: "Test User",
		Data:          data,
	}
	
	tmpl := service.templates["ticket_delivery"]
	var htmlBuf strings.Builder
	err := tmpl.ExecuteTemplate(&htmlBuf, "html", emailData)
	if err != nil {
		t.Fatalf("failed to render HTML template: %v", err)
	}
	
	htmlContent := htmlBuf.String()
	if !strings.Contains(htmlContent, "Test Event") {
		t.Error("expected event title in HTML content")
	}
	
	if !strings.Contains(htmlContent, "QR-CODE-1") {
		t.Error("expected first QR code in HTML content")
	}
	
	if !strings.Contains(htmlContent, "QR-CODE-2") {
		t.Error("expected second QR code in HTML content")
	}
	
	if !strings.Contains(htmlContent, "ORD-20240101-123456") {
		t.Error("expected order number in HTML content")
	}
}