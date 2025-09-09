package services

import (
	"bytes"
	"fmt"
	"strings"
	"time"

	"event-ticketing-platform/internal/models"
)

// PDFService handles PDF generation for tickets
type PDFService struct{}

// NewPDFService creates a new PDF service
func NewPDFService() *PDFService {
	return &PDFService{}
}

// GenerateTicketsPDF generates a PDF containing tickets with enhanced formatting and QR codes
func (s *PDFService) GenerateTicketsPDF(tickets []*models.Ticket, event *models.Event, order *models.Order) ([]byte, error) {
	// Enhanced PDF generation with better formatting
	var buffer bytes.Buffer

	// Generate PDF header
	buffer.WriteString("%PDF-1.4\n")

	// Object 1: Catalog
	buffer.WriteString("1 0 obj\n<<\n/Type /Catalog\n/Pages 2 0 R\n>>\nendobj\n\n")

	// Object 2: Pages
	buffer.WriteString("2 0 obj\n<<\n/Type /Pages\n/Kids [3 0 R]\n/Count 1\n>>\nendobj\n\n")

	// Generate enhanced content with better formatting
	content := s.generateTicketContent(tickets, event, order)
	contentStream := s.formatContentForPDF(content)

	// Object 3: Page
	buffer.WriteString("3 0 obj\n<<\n/Type /Page\n/Parent 2 0 R\n/MediaBox [0 0 612 792]\n")
	buffer.WriteString("/Contents 4 0 R\n/Resources <<\n/Font <<\n/F1 5 0 R\n/F2 6 0 R\n>>\n>>\n>>\nendobj\n\n")

	// Object 4: Content stream
	buffer.WriteString(fmt.Sprintf("4 0 obj\n<<\n/Length %d\n>>\nstream\n", len(contentStream)))
	buffer.WriteString(contentStream)
	buffer.WriteString("\nendstream\nendobj\n\n")

	// Object 5: Font (Helvetica)
	buffer.WriteString("5 0 obj\n<<\n/Type /Font\n/Subtype /Type1\n/BaseFont /Helvetica\n>>\nendobj\n\n")

	// Object 6: Font (Helvetica-Bold)
	buffer.WriteString("6 0 obj\n<<\n/Type /Font\n/Subtype /Type1\n/BaseFont /Helvetica-Bold\n>>\nendobj\n\n")

	// Write xref table
	buffer.WriteString("xref\n0 7\n")
	buffer.WriteString("0000000000 65535 f \n")
	buffer.WriteString("0000000010 00000 n \n")
	buffer.WriteString("0000000079 00000 n \n")
	buffer.WriteString("0000000136 00000 n \n")
	buffer.WriteString("0000000301 00000 n \n")
	buffer.WriteString("0000000380 00000 n \n")
	buffer.WriteString("0000000459 00000 n \n")

	// Write trailer
	buffer.WriteString("trailer\n<<\n/Size 7\n/Root 1 0 R\n>>\nstartxref\n538\n%%EOF\n")

	return buffer.Bytes(), nil
}

// generateTicketContent creates the formatted content for tickets
func (s *PDFService) generateTicketContent(tickets []*models.Ticket, event *models.Event, order *models.Order) string {
	var content strings.Builder

	// Header
	content.WriteString("EVENT TICKETS\n")
	content.WriteString("=============\n\n")

	// Event information
	content.WriteString(fmt.Sprintf("Event: %s\n", event.Title))
	content.WriteString(fmt.Sprintf("Date: %s\n", event.StartDate.Format("Monday, January 2, 2006 at 3:04 PM")))
	content.WriteString(fmt.Sprintf("Location: %s\n", event.Location))
	if event.Description != "" {
		content.WriteString(fmt.Sprintf("Description: %s\n", s.truncateString(event.Description, 200)))
	}
	content.WriteString("\n")

	// Order information
	content.WriteString("ORDER DETAILS\n")
	content.WriteString("-------------\n")
	content.WriteString(fmt.Sprintf("Order Number: %s\n", order.OrderNumber))
	content.WriteString(fmt.Sprintf("Purchase Date: %s\n", order.CreatedAt.Format("January 2, 2006 at 3:04 PM")))
	content.WriteString(fmt.Sprintf("Total Amount: KSh %.2f\n", order.TotalAmountInCurrency()))
	content.WriteString(fmt.Sprintf("Billing Name: %s\n", order.BillingName))
	content.WriteString(fmt.Sprintf("Billing Email: %s\n", order.BillingEmail))
	content.WriteString("\n")

	// Tickets section
	content.WriteString("YOUR TICKETS\n")
	content.WriteString("------------\n\n")

	for i, ticket := range tickets {
		content.WriteString(fmt.Sprintf("TICKET #%d\n", i+1))
		content.WriteString(fmt.Sprintf("Ticket ID: %d\n", ticket.ID))
		content.WriteString(fmt.Sprintf("QR Code: %s\n", ticket.QRCode))
		content.WriteString(fmt.Sprintf("Status: %s\n", s.getTicketStatusDisplay(ticket.Status)))
		content.WriteString(fmt.Sprintf("Generated: %s\n", ticket.CreatedAt.Format("Jan 2, 2006 at 3:04 PM")))

		// Generate QR code representation (simplified)
		qrRepresentation := s.generateQRCodeRepresentation(ticket.QRCode)
		content.WriteString(fmt.Sprintf("QR Code:\n%s\n", qrRepresentation))

		if i < len(tickets)-1 {
			content.WriteString("\n" + strings.Repeat("-", 40) + "\n\n")
		}
	}

	// Footer
	content.WriteString("\n\nIMPORTANT INFORMATION\n")
	content.WriteString("====================\n")
	content.WriteString("• Please present this ticket at the event entrance\n")
	content.WriteString("• Each ticket is valid for one person only\n")
	content.WriteString("• Tickets are non-transferable\n")
	content.WriteString("• Keep this ticket safe - you can re-download from your account\n")
	content.WriteString("• For support, contact us with your order number\n")
	content.WriteString(fmt.Sprintf("• Generated on: %s\n", time.Now().Format("January 2, 2006 at 3:04 PM")))

	return content.String()
}

// formatContentForPDF formats content for PDF stream
func (s *PDFService) formatContentForPDF(content string) string {
	var stream strings.Builder

	stream.WriteString("BT\n")
	stream.WriteString("/F2 16 Tf\n") // Bold font for header
	stream.WriteString("50 750 Td\n")

	lines := strings.Split(content, "\n")
	currentFont := "F2"
	currentSize := 16

	for _, line := range lines {
		// Determine font and size based on content
		if strings.Contains(line, "EVENT TICKETS") ||
			strings.Contains(line, "ORDER DETAILS") ||
			strings.Contains(line, "YOUR TICKETS") ||
			strings.Contains(line, "IMPORTANT INFORMATION") {
			if currentFont != "F2" || currentSize != 14 {
				stream.WriteString("/F2 14 Tf\n")
				currentFont = "F2"
				currentSize = 14
			}
		} else if strings.HasPrefix(line, "TICKET #") {
			if currentFont != "F2" || currentSize != 12 {
				stream.WriteString("/F2 12 Tf\n")
				currentFont = "F2"
				currentSize = 12
			}
		} else {
			if currentFont != "F1" || currentSize != 10 {
				stream.WriteString("/F1 10 Tf\n")
				currentFont = "F1"
				currentSize = 10
			}
		}

		// Escape special characters and write line
		escapedLine := s.escapePDFString(line)
		stream.WriteString(fmt.Sprintf("(%s) Tj\n", escapedLine))

		// Adjust line spacing based on content
		if line == "" {
			stream.WriteString("0 -8 Td\n")
		} else if strings.Contains(line, "=====") || strings.Contains(line, "-----") {
			stream.WriteString("0 -12 Td\n")
		} else {
			stream.WriteString("0 -12 Td\n")
		}
	}

	stream.WriteString("ET\n")
	return stream.String()
}

// generateQRCodeRepresentation creates a simple text representation of QR code
func (s *PDFService) generateQRCodeRepresentation(qrCode string) string {
	// Generate a simple ASCII representation of QR code
	// In a real implementation, you would use a proper QR code library

	// Create a deterministic pattern based on QR code
	hash := s.simpleHash(qrCode)

	var qr strings.Builder
	qr.WriteString("┌" + strings.Repeat("─", 20) + "┐\n")

	for i := 0; i < 8; i++ {
		qr.WriteString("│")
		for j := 0; j < 20; j++ {
			if (hash+i*20+j)%3 == 0 {
				qr.WriteString("█")
			} else {
				qr.WriteString(" ")
			}
		}
		qr.WriteString("│\n")
	}

	qr.WriteString("└" + strings.Repeat("─", 20) + "┘")
	return qr.String()
}

// simpleHash creates a simple hash for QR code pattern generation
func (s *PDFService) simpleHash(input string) int {
	hash := 0
	for _, char := range input {
		hash = hash*31 + int(char)
	}
	if hash < 0 {
		hash = -hash
	}
	return hash
}

// getTicketStatusDisplay returns a human-readable ticket status
func (s *PDFService) getTicketStatusDisplay(status models.TicketStatus) string {
	switch status {
	case models.TicketActive:
		return "ACTIVE - Ready to use"
	case models.TicketUsed:
		return "USED - Already scanned"
	case models.TicketRefunded:
		return "REFUNDED - No longer valid"
	default:
		return string(status)
	}
}

// truncateString truncates a string to a maximum length
func (s *PDFService) truncateString(str string, maxLen int) string {
	if len(str) <= maxLen {
		return str
	}
	return str[:maxLen-3] + "..."
}

// escapePDFString escapes special characters for PDF
func (s *PDFService) escapePDFString(str string) string {
	str = strings.ReplaceAll(str, "\\", "\\\\")
	str = strings.ReplaceAll(str, "(", "\\(")
	str = strings.ReplaceAll(str, ")", "\\)")
	str = strings.ReplaceAll(str, "\r", "")
	return str
}

// GenerateTicketQRCode generates a QR code for a ticket
func (s *PDFService) GenerateTicketQRCode(ticket *models.Ticket, event *models.Event) (string, error) {
	// Generate a simple QR code string
	// In a real implementation, you would use a proper QR code library
	qrData := fmt.Sprintf("TICKET:%d:EVENT:%d:QR:%s:TIME:%d",
		ticket.ID, event.ID, ticket.QRCode, time.Now().Unix())

	return qrData, nil
}
