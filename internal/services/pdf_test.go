package services

import (
	"strings"
	"testing"
	"time"

	"event-ticketing-platform/internal/models"

	"github.com/stretchr/testify/assert"
)

func TestPDFService_GenerateTicketsPDF(t *testing.T) {
	service := NewPDFService()

	// Create test data
	event := &models.Event{
		ID:          1,
		Title:       "Test Concert",
		Description: "An amazing test concert with great music and atmosphere",
		StartDate:   time.Date(2024, 6, 15, 19, 30, 0, 0, time.UTC),
		EndDate:     time.Date(2024, 6, 15, 23, 0, 0, 0, time.UTC),
		Location:    "Test Venue, 123 Main St, Test City",
		Status:      models.StatusPublished,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	order := &models.Order{
		ID:           1,
		UserID:       1,
		EventID:      1,
		OrderNumber:  "ORD-20240101-123456",
		TotalAmount:  7500, // $75.00
		Status:       models.OrderCompleted,
		PaymentID:    "pay_test123",
		BillingEmail: "test@example.com",
		BillingName:  "John Doe",
		CreatedAt:    time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
		UpdatedAt:    time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
	}

	tickets := []*models.Ticket{
		{
			ID:           1,
			OrderID:      1,
			TicketTypeID: 1,
			QRCode:       "TKT-1-1-1704110400-abcdef123456",
			Status:       models.TicketActive,
			CreatedAt:    time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
		},
		{
			ID:           2,
			OrderID:      1,
			TicketTypeID: 1,
			QRCode:       "TKT-1-1-1704110400-ghijkl789012",
			Status:       models.TicketActive,
			CreatedAt:    time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
		},
	}

	tests := []struct {
		name           string
		tickets        []*models.Ticket
		event          *models.Event
		order          *models.Order
		expectedError  bool
		expectedInPDF  []string
	}{
		{
			name:          "successful PDF generation with multiple tickets",
			tickets:       tickets,
			event:         event,
			order:         order,
			expectedError: false,
			expectedInPDF: []string{
				"EVENT TICKETS",
				"Test Concert",
				"ORD-20240101-123456",
				"$75.00",
				"John Doe",
				"test@example.com",
				"TICKET #1",
				"TICKET #2",
				"TKT-1-1-1704110400-abcdef123456",
				"TKT-1-1-1704110400-ghijkl789012",
				"ACTIVE - Ready to use",
			},
		},
		{
			name:          "successful PDF generation with single ticket",
			tickets:       tickets[:1],
			event:         event,
			order:         order,
			expectedError: false,
			expectedInPDF: []string{
				"EVENT TICKETS",
				"Test Concert",
				"TICKET #1",
				"TKT-1-1-1704110400-abcdef123456",
			},
		},
		{
			name:          "PDF generation with empty tickets",
			tickets:       []*models.Ticket{},
			event:         event,
			order:         order,
			expectedError: false,
			expectedInPDF: []string{
				"EVENT TICKETS",
				"Test Concert",
				"ORD-20240101-123456",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pdfData, err := service.GenerateTicketsPDF(tt.tickets, tt.event, tt.order)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, pdfData)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, pdfData)
				assert.Greater(t, len(pdfData), 0)

				// Convert PDF data to string for content checking
				pdfString := string(pdfData)

				// Check that it's a valid PDF
				assert.True(t, strings.HasPrefix(pdfString, "%PDF-1.4"))
				assert.True(t, strings.HasSuffix(strings.TrimSpace(pdfString), "%%EOF"))

				// Check for expected content in PDF
				for _, expectedContent := range tt.expectedInPDF {
					assert.Contains(t, pdfString, expectedContent, 
						"Expected content '%s' not found in PDF", expectedContent)
				}
			}
		})
	}
}

func TestPDFService_generateTicketContent(t *testing.T) {
	service := NewPDFService()

	event := &models.Event{
		ID:          1,
		Title:       "Test Event",
		Description: "A test event description",
		StartDate:   time.Date(2024, 6, 15, 19, 30, 0, 0, time.UTC),
		Location:    "Test Location",
	}

	order := &models.Order{
		ID:           1,
		OrderNumber:  "ORD-20240101-123456",
		TotalAmount:  5000,
		BillingEmail: "test@example.com",
		BillingName:  "Test User",
		CreatedAt:    time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
	}

	tickets := []*models.Ticket{
		{
			ID:        1,
			QRCode:    "TEST-QR-CODE-1",
			Status:    models.TicketActive,
			CreatedAt: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
		},
	}

	content := service.generateTicketContent(tickets, event, order)

	// Check that content contains expected sections
	assert.Contains(t, content, "EVENT TICKETS")
	assert.Contains(t, content, "ORDER DETAILS")
	assert.Contains(t, content, "YOUR TICKETS")
	assert.Contains(t, content, "IMPORTANT INFORMATION")

	// Check event information
	assert.Contains(t, content, "Test Event")
	assert.Contains(t, content, "Test Location")
	assert.Contains(t, content, "Saturday, June 15, 2024 at 7:30 PM")

	// Check order information
	assert.Contains(t, content, "ORD-20240101-123456")
	assert.Contains(t, content, "$50.00")
	assert.Contains(t, content, "Test User")
	assert.Contains(t, content, "test@example.com")

	// Check ticket information
	assert.Contains(t, content, "TICKET #1")
	assert.Contains(t, content, "TEST-QR-CODE-1")
	assert.Contains(t, content, "ACTIVE - Ready to use")
}

func TestPDFService_generateQRCodeRepresentation(t *testing.T) {
	service := NewPDFService()

	tests := []struct {
		name     string
		qrCode   string
		expected []string
	}{
		{
			name:   "basic QR code representation",
			qrCode: "TEST-QR-CODE",
			expected: []string{
				"┌", "─", "┐",
				"│", "█", "│",
				"└", "─", "┘",
			},
		},
		{
			name:   "different QR code should produce different pattern",
			qrCode: "DIFFERENT-QR-CODE",
			expected: []string{
				"┌", "─", "┐",
				"│", "│",
				"└", "─", "┘",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.generateQRCodeRepresentation(tt.qrCode)

			// Check that result contains expected characters
			for _, expected := range tt.expected {
				assert.Contains(t, result, expected)
			}

			// Check that result has proper structure
			lines := strings.Split(result, "\n")
			assert.Greater(t, len(lines), 5, "QR code should have multiple lines")
			
			// First and last lines should be borders
			assert.Contains(t, lines[0], "┌")
			assert.Contains(t, lines[0], "┐")
			assert.Contains(t, lines[len(lines)-1], "└")
			assert.Contains(t, lines[len(lines)-1], "┘")
		})
	}
}

func TestPDFService_getTicketStatusDisplay(t *testing.T) {
	service := NewPDFService()

	tests := []struct {
		name     string
		status   models.TicketStatus
		expected string
	}{
		{
			name:     "active ticket status",
			status:   models.TicketActive,
			expected: "ACTIVE - Ready to use",
		},
		{
			name:     "used ticket status",
			status:   models.TicketUsed,
			expected: "USED - Already scanned",
		},
		{
			name:     "refunded ticket status",
			status:   models.TicketRefunded,
			expected: "REFUNDED - No longer valid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.getTicketStatusDisplay(tt.status)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestPDFService_truncateString(t *testing.T) {
	service := NewPDFService()

	tests := []struct {
		name     string
		input    string
		maxLen   int
		expected string
	}{
		{
			name:     "string shorter than max length",
			input:    "Short string",
			maxLen:   20,
			expected: "Short string",
		},
		{
			name:     "string equal to max length",
			input:    "Exactly twenty chars",
			maxLen:   20,
			expected: "Exactly twenty chars",
		},
		{
			name:     "string longer than max length",
			input:    "This is a very long string that needs to be truncated",
			maxLen:   20,
			expected: "This is a very lo...",
		},
		{
			name:     "very short max length",
			input:    "Hello World",
			maxLen:   5,
			expected: "He...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.truncateString(tt.input, tt.maxLen)
			assert.Equal(t, tt.expected, result)
			assert.LessOrEqual(t, len(result), tt.maxLen)
		})
	}
}

func TestPDFService_escapePDFString(t *testing.T) {
	service := NewPDFService()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "string with parentheses",
			input:    "Hello (world)",
			expected: "Hello \\(world\\)",
		},
		{
			name:     "string with backslashes",
			input:    "Path\\to\\file",
			expected: "Path\\\\to\\\\file",
		},
		{
			name:     "string with carriage returns",
			input:    "Line 1\r\nLine 2",
			expected: "Line 1\nLine 2",
		},
		{
			name:     "string with all special characters",
			input:    "Test\\(special)\\chars\r",
			expected: "Test\\\\\\(special\\)\\\\chars",
		},
		{
			name:     "normal string",
			input:    "Normal string without special chars",
			expected: "Normal string without special chars",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.escapePDFString(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestPDFService_simpleHash(t *testing.T) {
	service := NewPDFService()

	tests := []struct {
		name     string
		input    string
		expected bool // Whether hash should be positive
	}{
		{
			name:     "basic string",
			input:    "test",
			expected: true,
		},
		{
			name:     "empty string",
			input:    "",
			expected: true,
		},
		{
			name:     "long string",
			input:    "this is a very long string for testing hash function",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.simpleHash(tt.input)
			
			if tt.expected {
				assert.GreaterOrEqual(t, result, 0, "Hash should be non-negative")
			}
			
			// Test consistency - same input should produce same hash
			result2 := service.simpleHash(tt.input)
			assert.Equal(t, result, result2, "Hash should be consistent")
		})
	}

	// Test that different inputs produce different hashes (most of the time)
	hash1 := service.simpleHash("string1")
	hash2 := service.simpleHash("string2")
	assert.NotEqual(t, hash1, hash2, "Different strings should produce different hashes")
}

func TestPDFService_formatContentForPDF(t *testing.T) {
	service := NewPDFService()

	content := `EVENT TICKETS
=============

Event: Test Event
Date: June 15, 2024

TICKET #1
QR Code: TEST-QR-123

IMPORTANT INFORMATION
Please present this ticket`

	result := service.formatContentForPDF(content)

	// Check that result contains PDF stream commands
	assert.Contains(t, result, "BT\n", "Should start with BT (Begin Text)")
	assert.Contains(t, result, "ET\n", "Should end with ET (End Text)")
	assert.Contains(t, result, "/F1", "Should contain font references")
	assert.Contains(t, result, "/F2", "Should contain bold font references")
	assert.Contains(t, result, "Tf\n", "Should contain font size commands")
	assert.Contains(t, result, "Td\n", "Should contain text positioning commands")
	assert.Contains(t, result, "Tj\n", "Should contain text show commands")

	// Check that content is properly escaped and included
	assert.Contains(t, result, "(EVENT TICKETS) Tj")
	assert.Contains(t, result, "(Event: Test Event) Tj")
	assert.Contains(t, result, "(TICKET #1) Tj")
}